// Package session is identity's server-side session lifecycle (R1;
// Architecture V1 §5/§7: identity owns Credential and Session).
//
// What it does:
//   - Login verifies a stored Argon2id credential, then issues a fresh session.
//     A session presented at login is revoked first (fixation defence). The
//     new token is 256 random bits; only its SHA-256 digest is stored.
//   - Authenticate resolves a presented token to an active session of an
//     ACTIVE credential whose identity account is PRESENT. Idle expiry
//     slides; the absolute expiry never moves.
//   - Logout revokes the current session; LogoutAll revokes every session of
//     the caller. Both are idempotent and race-safe (conditional UPDATE).
//   - Every outcome appends an auth_event row with no secret in it.
//
// What it deliberately does not do (decisions open):
//   - no enrolment, password set/reset, recovery, MFA, OTP, initial password
//     or account activation (D-34, D-42); credentials are read-only here;
//   - no permission: a session proves identity only (authorization reads
//     ACTIVE grants, of which none exist);
//   - no customer login or auto-login (D-26); issuance_reason is
//     PASSWORD_LOGIN only;
//   - no account lockout: the throttle refuses attempts for one login key
//     inside a sliding window and recovers by itself.
package session

import (
	"context"
	"crypto/sha256"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/password"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/websession"
)

// Config holds the technical session parameters. Defaults are conservative
// engineering values pending Security sign-off; they are not business policy.
type Config struct {
	IdleTimeout      time.Duration
	AbsoluteLifetime time.Duration
	// MaxFailures failed attempts for one login key within FailureWindow
	// make further attempts THROTTLED until the window slides past them.
	MaxFailures   int
	FailureWindow time.Duration
	// TouchInterval limits last_seen writes (sliding idle expiry).
	TouchInterval time.Duration
}

// DefaultConfig: idle 30 min, absolute 12 h, 5 failures / 15 min.
var DefaultConfig = Config{IdleTimeout: 30 * time.Minute, AbsoluteLifetime: 12 * time.Hour, MaxFailures: 5, FailureWindow: 15 * time.Minute,
	TouchInterval: time.Minute}

// Validate checks the configuration bounds.
func (c Config) Validate() error {
	switch {
	case c.IdleTimeout < time.Minute, c.AbsoluteLifetime < c.IdleTimeout, c.AbsoluteLifetime > 7*24*time.Hour,
		c.MaxFailures < 1, c.MaxFailures > 100, c.FailureWindow < time.Minute, c.TouchInterval < 0, c.TouchInterval > c.IdleTimeout:
		return errors.New("session: configuration out of bounds")
	}
	return nil
}

// Flag reports whether session authentication is switched on (feature flag
// identity.session_auth, default OFF). OFF: login is BLOCKED and no
// presented session is accepted (kill switch).
type Flag func(ctx context.Context) (bool, error)

// Service runs the session lifecycle.
type Service struct {
	runner *db.Runner
	clock  timeutil.Clock
	cfg    Config
	flag   Flag
}

// New returns the service bound to the identity owner runner.
func New(runner *db.Runner, clock timeutil.Clock, cfg Config, flag Flag) (*Service, error) {
	if runner == nil || runner.Module() != identity.Module || clock == nil || flag == nil {
		return nil, errors.New("session: identity runner, clock and flag are required")
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &Service{runner: runner, clock: clock, cfg: cfg, flag: flag}, nil
}

// Status is a login outcome. Every non-OK outcome is reported to the client
// without saying why (no account enumeration).
type Status string

// Login outcomes.
const (
	OK        Status = "OK"
	Invalid   Status = "INVALID_CREDENTIALS"
	Throttled Status = "THROTTLED"
	Disabled  Status = "FEATURE_DISABLED"
)

// Login is one login attempt.
type Login struct {
	Name     string
	Password string
	// ReplaceSessionID is the active session the browser already presented,
	// if any (revoked on success: an existing session is never upgraded or
	// reused; a client-chosen token is never accepted as a session).
	ReplaceSessionID string
	CorrelationID    string
}

// Issued is a newly issued session. Token goes only into the cookie.
type Issued struct {
	Token     string
	Principal websession.Principal
}

// Result is a login result.
type Result struct {
	Status Status
	Issued *Issued
}

// NormaliseLogin returns the canonical login name ("" if unusable).
func NormaliseLogin(s string) string {
	n := strings.ToLower(strings.TrimSpace(s))
	if len(n) < 3 || len(n) > 128 {
		return ""
	}
	return n
}

func loginKey(normalised, raw string) [32]byte {
	if normalised == "" {
		// Unusable names still get a stable key (bounded length) so their
		// attempts are throttled too; the raw value is never stored.
		if len(raw) > 256 {
			raw = raw[:256]
		}
		return sha256.Sum256([]byte("unusable:" + strings.ToLower(strings.TrimSpace(raw))))
	}
	return sha256.Sum256([]byte(normalised))
}

func corr(c string) string {
	if c == "" || len(c) > 128 {
		return "uncorrelated"
	}
	return c
}

// Login verifies a credential and issues a session. Attempts for one login
// key are serialised (advisory lock) so concurrent guesses cannot exceed the
// throttle.
func (s *Service) Login(ctx context.Context, in Login) (Result, error) {
	on, err := s.flag(ctx)
	if err != nil {
		return Result{}, err
	}
	if !on {
		return Result{Status: Disabled}, nil
	}
	name := NormaliseLogin(in.Name)
	key := loginKey(name, in.Name)
	c := corr(in.CorrelationID)
	var out Result
	err = s.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		now := s.clock.Now().Time().UTC()
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended(encode($1::bytea, 'hex'), 1415))`, key[:]); err != nil {
			return err
		}
		var failures int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM identity.auth_event WHERE kind = 'LOGIN_FAILED' AND login_key_sha256 = $1 AND occurred_at > $2`,
			key[:], now.Add(-s.cfg.FailureWindow)).Scan(&failures); err != nil {
			return err
		}
		if failures >= s.cfg.MaxFailures {
			password.Dummy(in.Password)
			out = Result{Status: Throttled}
			return s.event(ctx, tx, "LOGIN_THROTTLED", nil, nil, key[:], "THROTTLED", c, now)
		}
		var credID, userID, hash, status, presence string
		var display *string
		found := false
		if name != "" {
			err := tx.QueryRow(ctx, `SELECT c.credential_id::text, c.user_id::text, c.secret_hash, c.status, u.presence, u.display_name
				FROM identity.credential c JOIN identity.user_account u USING (user_id) WHERE c.login_name = $1`, name).
				Scan(&credID, &userID, &hash, &status, &presence, &display)
			switch {
			case err == nil:
				found = true
			case !errors.Is(err, pgx.ErrNoRows):
				return err
			}
		}
		if !found {
			password.Dummy(in.Password)
			out = Result{Status: Invalid}
			return s.event(ctx, tx, "LOGIN_FAILED", nil, nil, key[:], "INVALID_CREDENTIALS", c, now)
		}
		ok, verr := password.Verify(hash, in.Password)
		reason := ""
		switch {
		case verr != nil:
			reason = "CREDENTIAL_UNUSABLE"
		case !ok:
			reason = "INVALID_CREDENTIALS"
		case status != "ACTIVE":
			reason = "CREDENTIAL_DISABLED"
		case presence != "PRESENT":
			reason = "ACCOUNT_NOT_PRESENT"
		}
		if reason != "" {
			out = Result{Status: Invalid}
			return s.event(ctx, tx, "LOGIN_FAILED", &userID, nil, key[:], reason, c, now)
		}
		// Fixation defence: any presented session is retired, never reused.
		if in.ReplaceSessionID != "" {
			var oldID string
			err := tx.QueryRow(ctx, `UPDATE identity.session SET revoked_at = $2, revoke_reason = 'REPLACED_AT_LOGIN'
				WHERE session_id = $1::uuid AND revoked_at IS NULL RETURNING session_id::text`, in.ReplaceSessionID, now).Scan(&oldID)
			switch {
			case err == nil:
				if err := s.event(ctx, tx, "SESSION_REPLACED", &userID, &oldID, nil, "REPLACED_AT_LOGIN", c, now); err != nil {
					return err
				}
			case !errors.Is(err, pgx.ErrNoRows):
				return err
			}
		}
		token, err := websession.NewToken()
		if err != nil {
			return err
		}
		digest, _ := websession.Digest(token)
		sid, err := ids.NewUUID()
		if err != nil {
			return err
		}
		abs := now.Add(s.cfg.AbsoluteLifetime)
		idle := minTime(now.Add(s.cfg.IdleTimeout), abs)
		if _, err := tx.Exec(ctx, `INSERT INTO identity.session (session_id, token_sha256, user_id, credential_id, issuance_reason, issued_at,
			last_seen_at, idle_expires_at, absolute_expires_at, correlation_id) VALUES ($1, $2, $3, $4, 'PASSWORD_LOGIN', $5, $5, $6, $7, $8)`,
			sid, digest[:], userID, credID, now, idle, abs, c); err != nil {
			return err
		}
		dn := ""
		if display != nil {
			dn = *display
		}
		out = Result{Status: OK, Issued: &Issued{Token: token, Principal: websession.Principal{SessionID: sid, UserID: userID, DisplayName: dn,
			IdleExpiresAt: idle, AbsoluteExpiresAt: abs, CSRF: websession.CSRFToken(token)}}}
		return s.event(ctx, tx, "LOGIN_SUCCEEDED", &userID, &sid, key[:], "", c, now)
	})
	if err != nil {
		return Result{}, err
	}
	return out, nil
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

// Authenticate resolves a presented token (websession.Authenticator).
func (s *Service) Authenticate(ctx context.Context, token string) (websession.Principal, bool, error) {
	digest, ok := websession.Digest(token)
	if !ok {
		return websession.Principal{}, false, nil
	}
	on, err := s.flag(ctx)
	if err != nil || !on {
		return websession.Principal{}, false, err
	}
	var p websession.Principal
	found := false
	err = s.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		now := s.clock.Now().Time().UTC()
		var lastSeen time.Time
		var display *string
		err := tx.QueryRow(ctx, `SELECT s.session_id::text, s.user_id::text, u.display_name, s.last_seen_at, s.idle_expires_at, s.absolute_expires_at
			FROM identity.session s JOIN identity.credential c ON c.credential_id = s.credential_id JOIN identity.user_account u ON u.user_id = s.user_id
			WHERE s.token_sha256 = $1 AND s.revoked_at IS NULL AND s.idle_expires_at > $2 AND s.absolute_expires_at > $2
			  AND c.status = 'ACTIVE' AND c.user_id = s.user_id AND u.presence = 'PRESENT'`, digest[:], now).
			Scan(&p.SessionID, &p.UserID, &display, &lastSeen, &p.IdleExpiresAt, &p.AbsoluteExpiresAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		found = true
		if display != nil {
			p.DisplayName = *display
		}
		if now.Sub(lastSeen) >= s.cfg.TouchInterval {
			idle := minTime(now.Add(s.cfg.IdleTimeout), p.AbsoluteExpiresAt)
			tag, err := tx.Exec(ctx, `UPDATE identity.session SET last_seen_at = $2, idle_expires_at = $3
				WHERE session_id = $1 AND revoked_at IS NULL AND last_seen_at < $2`, p.SessionID, now, idle)
			if err != nil {
				return err
			}
			if tag.RowsAffected() == 1 {
				p.IdleExpiresAt = idle
			}
		}
		return nil
	})
	if err != nil || !found {
		return websession.Principal{}, false, err
	}
	p.CSRF = websession.CSRFToken(token)
	return p, true, nil
}

// Logout revokes one session (idempotent). It reports whether this call
// revoked it.
func (s *Service) Logout(ctx context.Context, p websession.Principal, correlationID string) (bool, error) {
	revoked := false
	err := s.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		now := s.clock.Now().Time().UTC()
		tag, err := tx.Exec(ctx, `UPDATE identity.session SET revoked_at = $2, revoke_reason = 'LOGOUT' WHERE session_id = $1 AND revoked_at IS NULL`,
			p.SessionID, now)
		if err != nil || tag.RowsAffected() == 0 {
			return err
		}
		revoked = true
		return s.event(ctx, tx, "LOGOUT", &p.UserID, &p.SessionID, nil, "", corr(correlationID), now)
	})
	return revoked, err
}

// LogoutAll revokes every active (unexpired, unrevoked) session of the
// caller. It returns how many it revoked.
func (s *Service) LogoutAll(ctx context.Context, p websession.Principal, correlationID string) (int64, error) {
	var n int64
	err := s.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		now := s.clock.Now().Time().UTC()
		tag, err := tx.Exec(ctx, `UPDATE identity.session SET revoked_at = $2, revoke_reason = 'LOGOUT_ALL'
			WHERE user_id = $1 AND revoked_at IS NULL AND idle_expires_at > $2 AND absolute_expires_at > $2`,
			p.UserID, now)
		if err != nil {
			return err
		}
		n = tag.RowsAffected()
		return s.event(ctx, tx, "LOGOUT_ALL", &p.UserID, &p.SessionID, nil, "ALL_SESSIONS", corr(correlationID), now)
	})
	return n, err
}

// RecordCSRFRejection appends a CSRF_REJECTED event (websession.Authenticator).
func (s *Service) RecordCSRFRejection(ctx context.Context, p websession.Principal) error {
	return s.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		return s.event(ctx, tx, "CSRF_REJECTED", &p.UserID, &p.SessionID, nil, "CSRF_TOKEN_MISMATCH", "uncorrelated", s.clock.Now().Time().UTC())
	})
}

// LoginEnabled reports the flag (session navigation hint).
func (s *Service) LoginEnabled(ctx context.Context) bool {
	on, err := s.flag(ctx)
	return err == nil && on
}

func (s *Service) event(ctx context.Context, tx db.Tx, kind string, userID, sessionID *string, key []byte, reason, correlationID string, at time.Time) error {
	id, err := ids.NewUUID()
	if err != nil {
		return err
	}
	var r any
	if reason != "" {
		r = reason
	}
	var k any
	if key != nil {
		k = key
	}
	_, err = tx.Exec(ctx, `INSERT INTO identity.auth_event (event_id, kind, user_id, session_id, login_key_sha256, reason, correlation_id, occurred_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, id, kind, userID, sessionID, k, r, correlationID, at)
	return err
}
