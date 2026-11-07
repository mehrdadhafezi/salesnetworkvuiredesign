package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity"
	identityhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/http"
	identityquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/session"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/apiserver"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/clock"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
)

// R1 production authentication (optional; needs the identity owner login
// and the flag store). Server-side sessions with password login against
// identity credentials. Nothing here creates a credential, activates an
// account or grants authority: the subject of a session holds only its
// ACTIVE role grants (none exist; D-42). Flag identity.session_auth is OFF
// by default: login answers BLOCKED and no session is accepted.
const (
	EnvIdentityAuthURL = "CRM_IDENTITY_AUTH_DATABASE_URL"
	// SessionAuthFlag is the R1 kill switch (default off).
	SessionAuthFlag = "identity.session_auth"

	EnvSessionIdle        = "CRM_AUTH_SESSION_IDLE"
	EnvSessionAbsolute    = "CRM_AUTH_SESSION_ABSOLUTE"
	EnvLoginMaxFailures   = "CRM_AUTH_LOGIN_MAX_FAILURES"
	EnvLoginFailureWindow = "CRM_AUTH_LOGIN_FAILURE_WINDOW"
)

// ErrAuthConfig reports a partial R1 configuration.
var ErrAuthConfig = errors.New("crm-api: " + EnvIdentityAuthURL + " needs " + EnvFeatureFlagsURL)

// SessionConfigFromEnv reads optional overrides of the technical session
// parameters (Go durations / integers); defaults are session.DefaultConfig.
func SessionConfigFromEnv(getenv func(string) string) (session.Config, error) {
	cfg := session.DefaultConfig
	durations := map[string]*time.Duration{EnvSessionIdle: &cfg.IdleTimeout, EnvSessionAbsolute: &cfg.AbsoluteLifetime, EnvLoginFailureWindow: &cfg.FailureWindow}
	for k, p := range durations {
		if v := getenv(k); v != "" {
			d, err := time.ParseDuration(v)
			if err != nil {
				return session.Config{}, fmt.Errorf("crm-api: %s: %w", k, err)
			}
			*p = d
		}
	}
	if v := getenv(EnvLoginMaxFailures); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return session.Config{}, fmt.Errorf("crm-api: %s: %w", EnvLoginMaxFailures, err)
		}
		cfg.MaxFailures = n
	}
	return cfg, cfg.Validate()
}

// NewAuthAPI composes the R1 surface and the production grant loader over
// the identity owner runner and the flag store.
func NewAuthAPI(identityRunner, flags *db.Runner, cfg session.Config) (apiserver.AuthAPI, authorization.GrantLoader, error) {
	if identityRunner.Module() != identity.Module || flags.Module() != owner.FeatureFlags {
		return nil, nil, errors.New("crm-api: auth runners bound to the wrong modules")
	}
	flag := flagFunc(flags, SessionAuthFlag)
	svc, err := session.New(identityRunner, clock.System{}, cfg, session.Flag(flag))
	if err != nil {
		return nil, nil, err
	}
	h, err := identityhttp.NewAuth(svc, clock.System{})
	if err != nil {
		return nil, nil, err
	}
	return h, identityquery.New(identityRunner), nil
}

// openAuth opens the R1 pools when configured. It must run before the
// other surfaces are composed: it sets the production subject resolver.
func openAuth(ctx context.Context, getenv func(string) string, devAuth bool) (apiserver.AuthAPI, func(), error) {
	iu, ffu := getenv(EnvIdentityAuthURL), getenv(EnvFeatureFlagsURL)
	if iu == "" {
		return nil, func() {}, nil
	}
	if ffu == "" {
		return nil, nil, ErrAuthConfig
	}
	if devAuth {
		return nil, nil, apiserver.ErrAuthConflict
	}
	scfg, err := SessionConfigFromEnv(getenv)
	if err != nil {
		return nil, nil, err
	}
	var pools []*db.Pool
	closeAll := func() {
		for _, p := range pools {
			p.Close()
		}
	}
	runners := map[owner.Module]*db.Runner{}
	for m, u := range map[owner.Module]string{identity.Module: iu, owner.FeatureFlags: ffu} {
		dcfg := db.DefaultConfig(u)
		dcfg.ApplicationName = "crm-api"
		pool, err := db.Open(ctx, dcfg)
		if err != nil {
			closeAll()
			return nil, nil, fmt.Errorf("crm-api: %s database: %w", m, err)
		}
		pools = append(pools, pool)
		if runners[m], err = pool.Runner(m); err != nil {
			closeAll()
			return nil, nil, err
		}
	}
	api, grants, err := NewAuthAPI(runners[identity.Module], runners[owner.FeatureFlags], scfg)
	if err != nil {
		closeAll()
		return nil, nil, err
	}
	sessionSubjects = authorization.SessionSubjects{Grants: grants}
	return api, closeAll, nil
}
