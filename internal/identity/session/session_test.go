package session_test

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/password"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/session"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
	platformmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pgtest"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/websession"
)

// All accounts and passwords are SYNTHETIC.

type clock struct {
	mu sync.Mutex
	at time.Time
}

func (c *clock) Now() timeutil.Instant {
	c.mu.Lock()
	defer c.mu.Unlock()
	i, _ := timeutil.NewInstant(c.at)
	return i
}

func (c *clock) advance(d time.Duration) {
	c.mu.Lock()
	c.at = c.at.Add(d)
	c.mu.Unlock()
}

type env struct {
	*pgtest.Env
	svc  *session.Service
	clk  *clock
	flag atomic.Bool
	conn *pgx.Conn
}

var fast = password.Params{MemoryKiB: 64, Iterations: 1, Parallelism: 1, SaltBytes: 16, KeyBytes: 32}

func setup(t *testing.T) *env {
	t.Helper()
	e := &env{Env: pgtest.New(t), clk: &clock{at: time.Date(2026, 11, 7, 9, 0, 0, 0, time.UTC)}}
	if _, err := platformmigrations.Up(context.Background(), e.DSN, migrations.Source()); err != nil {
		t.Fatal(err)
	}
	e.flag.Store(true)
	svc, err := session.New(e.Runner(t, identity.Module), e.clk, session.DefaultConfig, func(context.Context) (bool, error) { return e.flag.Load(), nil })
	if err != nil {
		t.Fatal(err)
	}
	e.svc = svc
	c, err := pgx.Connect(context.Background(), e.DSN)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close(context.Background()) })
	e.conn = c
	return e
}

var keySeq atomic.Int64

// account inserts a synthetic shadow account and (unless pw is "") an
// ACTIVE credential. The application role cannot do this: there is no
// enrolment path (D-34); tests act as a future approved provisioning step.
func (e *env) account(t *testing.T, login, pw string) string {
	t.Helper()
	ctx := context.Background()
	uid, _ := ids.NewUUID()
	batch, _ := ids.NewUUID()
	sha := sha256.Sum256([]byte(uid))
	n := keySeq.Add(1)
	if _, err := e.conn.Exec(ctx, `INSERT INTO identity.user_account (user_id, source_system, source_spec, source_key, first_batch_id, source_batch_id,
		source_row_sha256, imported_at, refreshed_at, presence, display_name, account_state, login_enabled, version)
		VALUES ($1, 'synthetic', 'synthetic.account.v1', $2, $3, $3, $4, now(), now(), 'PRESENT', $5, 'SHADOW_IMPORTED', false, 1)`,
		uid, fmt.Sprint(1000+n), batch, sha[:], "کاربر آزمایشی "+fmt.Sprint(n)); err != nil {
		t.Fatal(err)
	}
	if pw != "" {
		h, err := password.Hash(pw, fast)
		if err != nil {
			t.Fatal(err)
		}
		cid, _ := ids.NewUUID()
		if _, err := e.conn.Exec(ctx, `INSERT INTO identity.credential (credential_id, user_id, login_name, kind, secret_hash, status, created_at, updated_at, version)
			VALUES ($1, $2, $3, 'PASSWORD_ARGON2ID', $4, 'ACTIVE', now(), now(), 1)`, cid, uid, login, h); err != nil {
			t.Fatal(err)
		}
	}
	return uid
}

func (e *env) count(t *testing.T, q string, args ...any) int {
	t.Helper()
	var n int
	if err := e.conn.QueryRow(context.Background(), q, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	return n
}

func (e *env) login(t *testing.T, name, pw string) session.Result {
	t.Helper()
	r, err := e.svc.Login(context.Background(), session.Login{Name: name, Password: pw, CorrelationID: "corr-r1-test"})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// 1 / 2 / 9 / 12 / 17: login, invalid credentials, inactive, flag OFF, history.
func TestLoginOutcomes(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	uid := e.account(t, "seller.one", "synthetic-Pass-1")
	r := e.login(t, "  Seller.One ", "synthetic-Pass-1")
	if r.Status != session.OK || r.Issued == nil || r.Issued.Principal.UserID != uid || len(r.Issued.Token) != 43 {
		t.Fatalf("login: %+v", r)
	}
	// Only the digest is stored, never the token.
	digest, _ := websession.Digest(r.Issued.Token)
	if e.count(t, `SELECT count(*) FROM identity.session WHERE token_sha256 = $1`, digest[:]) != 1 ||
		e.count(t, `SELECT count(*) FROM identity.session WHERE encode(token_sha256, 'escape') LIKE '%' || $1 || '%'`, r.Issued.Token) != 0 {
		t.Fatal("session token must be stored only as its digest")
	}
	p, ok, err := e.svc.Authenticate(ctx, r.Issued.Token)
	if !ok || err != nil || p.UserID != uid || p.CSRF != websession.CSRFToken(r.Issued.Token) {
		t.Fatalf("authenticate: %+v %v %v", p, ok, err)
	}
	// Wrong password, unknown login and an unusable login are indistinguishable.
	for _, c := range [][2]string{{"seller.one", "wrong"}, {"nobody.here", "synthetic-Pass-1"}, {"x", "y"}, {"", ""}} {
		if r := e.login(t, c[0], c[1]); r.Status != session.Invalid || r.Issued != nil {
			t.Fatalf("%v: %+v", c, r)
		}
	}
	// Inactive: disabled credential, absent account, no credential at all.
	disabled := e.account(t, "seller.disabled", "synthetic-Pass-2")
	_, _ = e.conn.Exec(ctx, `UPDATE identity.credential SET status = 'DISABLED' WHERE user_id = $1`, disabled)
	absent := e.account(t, "seller.absent", "synthetic-Pass-3")
	_, _ = e.conn.Exec(ctx, `UPDATE identity.user_account SET presence = 'ABSENT_IN_LATEST' WHERE user_id = $1`, absent)
	e.account(t, "", "") // shadow account without a credential cannot log in by any name
	for _, c := range [][2]string{{"seller.disabled", "synthetic-Pass-2"}, {"seller.absent", "synthetic-Pass-3"}} {
		if r := e.login(t, c[0], c[1]); r.Status != session.Invalid {
			t.Fatalf("inactive %s: %+v", c[0], r)
		}
	}
	if e.count(t, `SELECT count(*) FROM identity.auth_event WHERE kind = 'LOGIN_FAILED' AND reason = 'CREDENTIAL_DISABLED'`) != 1 ||
		e.count(t, `SELECT count(*) FROM identity.auth_event WHERE kind = 'LOGIN_FAILED' AND reason = 'ACCOUNT_NOT_PRESENT'`) != 1 {
		t.Fatal("inactive reasons must be recorded internally")
	}
	// Flag OFF: login BLOCKED and existing sessions are not accepted (kill switch).
	e.flag.Store(false)
	if r := e.login(t, "seller.one", "synthetic-Pass-1"); r.Status != session.Disabled {
		t.Fatalf("flag off: %+v", r)
	}
	if _, ok, _ := e.svc.Authenticate(ctx, r.Issued.Token); ok {
		t.Fatal("flag off must reject existing sessions")
	}
	e.flag.Store(true)
	// 17: history, with no secret in it.
	if e.count(t, `SELECT count(*) FROM identity.auth_event WHERE kind = 'LOGIN_SUCCEEDED' AND user_id = $1`, uid) != 1 ||
		e.count(t, `SELECT count(*) FROM identity.auth_event WHERE kind = 'LOGIN_FAILED'`) != 6 {
		t.Fatal("auth events")
	}
	var dump string
	_ = e.conn.QueryRow(ctx, `SELECT coalesce(string_agg(t::text, '|'), '') FROM identity.auth_event t`).Scan(&dump)
	var dump2 string
	_ = e.conn.QueryRow(ctx, `SELECT coalesce(string_agg(t::text, '|'), '') FROM identity.session t`).Scan(&dump2)
	for _, secret := range []string{"synthetic-Pass-1", "wrong", r.Issued.Token, websession.CSRFToken(r.Issued.Token), "seller.one", "nobody.here"} {
		if strings.Contains(dump, secret) || strings.Contains(dump2, secret) {
			t.Fatalf("secret or raw login %q stored", secret)
		}
	}
	// Append-only evidence.
	if _, err := e.conn.Exec(ctx, `UPDATE identity.auth_event SET reason = 'X'`); err == nil || !strings.Contains(err.Error(), "append-only") {
		t.Fatalf("auth_event must be append-only: %v", err)
	}
	if _, err := e.conn.Exec(ctx, `DELETE FROM identity.session`); err == nil {
		t.Fatal("sessions must never be deleted")
	}
}

// 8: repeated attempts are throttled per login key (no lockout: the window slides).
func TestThrottle(t *testing.T) {
	e := setup(t)
	e.account(t, "seller.throttle", "synthetic-Pass-1")
	for i := 0; i < session.DefaultConfig.MaxFailures; i++ {
		if r := e.login(t, "seller.throttle", "bad"); r.Status != session.Invalid {
			t.Fatalf("attempt %d: %+v", i, r)
		}
	}
	// Even the right password is refused while throttled.
	if r := e.login(t, "seller.throttle", "synthetic-Pass-1"); r.Status != session.Throttled {
		t.Fatalf("throttled: %+v", r)
	}
	// Another login key is unaffected.
	e.account(t, "seller.other", "synthetic-Pass-9")
	if r := e.login(t, "seller.other", "synthetic-Pass-9"); r.Status != session.OK {
		t.Fatalf("other key: %+v", r)
	}
	e.clk.advance(session.DefaultConfig.FailureWindow + time.Second)
	if r := e.login(t, "seller.throttle", "synthetic-Pass-1"); r.Status != session.OK {
		t.Fatalf("window slid: %+v", r)
	}
	// Concurrent guessing cannot exceed the budget (attempts are serialised per key).
	e.account(t, "seller.burst", "synthetic-Pass-B")
	var wg sync.WaitGroup
	var invalid atomic.Int64
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := e.svc.Login(context.Background(), session.Login{Name: "seller.burst", Password: "bad", CorrelationID: "corr-burst-test"})
			if err == nil && r.Status == session.Invalid {
				invalid.Add(1)
			}
		}()
	}
	wg.Wait()
	if invalid.Load() != int64(session.DefaultConfig.MaxFailures) {
		t.Fatalf("burst evaluated %d passwords, budget %d", invalid.Load(), session.DefaultConfig.MaxFailures)
	}
}

// 3 / 4 / 5 / 6 / 14 / 15 / 16: expiry, revocation, malformed tokens,
// fixation, refresh, concurrent logout.
func TestSessionLifecycle(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	uid := e.account(t, "seller.life", "synthetic-Pass-1")
	r := e.login(t, "seller.life", "synthetic-Pass-1")
	tok := r.Issued.Token

	// 14: a later request (browser refresh) with a valid session slides idle expiry.
	e.clk.advance(20 * time.Minute)
	p, ok, _ := e.svc.Authenticate(ctx, tok)
	if !ok || !p.IdleExpiresAt.After(r.Issued.Principal.IdleExpiresAt) || !p.AbsoluteExpiresAt.Equal(r.Issued.Principal.AbsoluteExpiresAt) {
		t.Fatalf("slide: %+v", p)
	}
	// 3 / 15: idle expiry.
	e.clk.advance(session.DefaultConfig.IdleTimeout + time.Second)
	if _, ok, _ := e.svc.Authenticate(ctx, tok); ok {
		t.Fatal("idle-expired session accepted")
	}
	// Absolute expiry even when active.
	r2 := e.login(t, "seller.life", "synthetic-Pass-1")
	for i := 0; i < 26; i++ {
		e.clk.advance(29 * time.Minute)
		if _, ok, _ := e.svc.Authenticate(ctx, r2.Issued.Token); !ok {
			if i < 24 {
				t.Fatalf("expired early at step %d", i)
			}
			break
		}
		if i == 25 {
			t.Fatal("absolute lifetime not enforced")
		}
	}

	// 5: malformed, foreign and tampered tokens never resolve.
	r3 := e.login(t, "seller.life", "synthetic-Pass-1")
	tampered := []byte(r3.Issued.Token)
	tampered[5] ^= 1
	if tampered[5] == '-' || tampered[5] == '_' {
		tampered[5] = 'A'
	}
	for _, bad := range []string{"", "x", strings.Repeat("A", 43), string(tampered), r3.Issued.Token + "A", "' OR 1=1 --", strings.Repeat("A", 4096)} {
		if _, ok, err := e.svc.Authenticate(ctx, bad); ok || err != nil {
			t.Fatalf("token %q: ok=%v err=%v", bad, ok, err)
		}
	}

	// 6: fixation. A client-chosen token is never a session, and logging in
	// while holding a session retires it.
	chosen, _ := websession.NewToken()
	if _, ok, _ := e.svc.Authenticate(ctx, chosen); ok {
		t.Fatal("client-chosen token accepted")
	}
	held, _, _ := e.svc.Authenticate(ctx, r3.Issued.Token)
	r4, err := e.svc.Login(ctx, session.Login{Name: "seller.life", Password: "synthetic-Pass-1", ReplaceSessionID: held.SessionID, CorrelationID: "corr-fix-test"})
	if err != nil || r4.Status != session.OK || r4.Issued.Token == r3.Issued.Token || r4.Issued.Principal.SessionID == held.SessionID {
		t.Fatalf("relogin: %+v %v", r4, err)
	}
	if _, ok, _ := e.svc.Authenticate(ctx, r3.Issued.Token); ok {
		t.Fatal("pre-login session survived login")
	}
	if e.count(t, `SELECT count(*) FROM identity.session WHERE session_id = $1::uuid AND revoke_reason = 'REPLACED_AT_LOGIN'`, held.SessionID) != 1 {
		t.Fatal("replaced session not revoked")
	}

	// 4 / 16: revocation, concurrent logout and requests.
	cur, _, _ := e.svc.Authenticate(ctx, r4.Issued.Token)
	var wg sync.WaitGroup
	var revoked atomic.Int64
	for i := 0; i < 8; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			if ok, err := e.svc.Logout(ctx, cur, "corr-logout-test"); err == nil && ok {
				revoked.Add(1)
			}
		}()
		go func() {
			defer wg.Done()
			_, _, _ = e.svc.Authenticate(ctx, r4.Issued.Token)
		}()
	}
	wg.Wait()
	if revoked.Load() != 1 || e.count(t, `SELECT count(*) FROM identity.auth_event WHERE kind = 'LOGOUT' AND session_id = $1::uuid`, cur.SessionID) != 1 {
		t.Fatalf("logout must revoke exactly once: %d", revoked.Load())
	}
	if _, ok, _ := e.svc.Authenticate(ctx, r4.Issued.Token); ok {
		t.Fatal("revoked session accepted")
	}

	// Logout-all revokes every session of the caller only.
	other := e.account(t, "seller.bystander", "synthetic-Pass-5")
	b := e.login(t, "seller.bystander", "synthetic-Pass-5")
	s1, s2 := e.login(t, "seller.life", "synthetic-Pass-1"), e.login(t, "seller.life", "synthetic-Pass-1")
	p1, _, _ := e.svc.Authenticate(ctx, s1.Issued.Token)
	if n, err := e.svc.LogoutAll(ctx, p1, "corr-logoutall-test"); err != nil || n != 2 {
		t.Fatalf("logout all: %d %v", n, err)
	}
	if _, ok, _ := e.svc.Authenticate(ctx, s2.Issued.Token); ok {
		t.Fatal("logout-all left a session")
	}
	if bp, ok, _ := e.svc.Authenticate(ctx, b.Issued.Token); !ok || bp.UserID != other {
		t.Fatal("logout-all touched another user")
	}

	// A disabled credential or an account absent from the latest snapshot
	// ends every active session at the next request.
	d := e.login(t, "seller.life", "synthetic-Pass-1")
	_, _ = e.conn.Exec(ctx, `UPDATE identity.credential SET status = 'DISABLED' WHERE user_id = $1`, uid)
	if _, ok, _ := e.svc.Authenticate(ctx, d.Issued.Token); ok {
		t.Fatal("session of a disabled credential accepted")
	}
}
