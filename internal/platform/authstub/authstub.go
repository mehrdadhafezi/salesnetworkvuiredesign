// Package authstub is a DEVELOPMENT/TEST-ONLY authentication stub for the
// SPA shell. It is not production authentication and makes no commitment to
// a future auth design: no passwords, sessions, tokens, OTP, persistence or
// WordPress users.
//
// A request selects one of a fixed set of obviously synthetic identities
// with the X-Dev-User header. The stub can only be constructed in the
// development or test environment; in production construction fails, so the
// header is never read.
package authstub

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
)

// Header selects a synthetic identity.
const Header = "X-Dev-User"

// Environment is the deployment environment of the API process.
type Environment string

// Environments. The zero value and anything unknown are treated as production.
const (
	Production  Environment = "production"
	Development Environment = "development"
	Test        Environment = "test"
)

var (
	// ErrProductionLockout reports an attempt to enable the stub in production.
	ErrProductionLockout = errors.New("authstub: the development auth stub cannot be enabled in production")
	// ErrUnknownEnvironment reports an unrecognised environment name.
	ErrUnknownEnvironment = errors.New("authstub: unknown environment")
	// ErrUnknownIdentity reports an X-Dev-User value that is not a synthetic identity.
	ErrUnknownIdentity = errors.New("authstub: unknown synthetic identity")
)

// ParseEnvironment parses an environment name. Empty means production
// (fail closed); anything unrecognised is an error.
func ParseEnvironment(s string) (Environment, error) {
	switch Environment(s) {
	case "", Production:
		return Production, nil
	case Development, Test:
		return Environment(s), nil
	default:
		return Production, fmt.Errorf("%w: %q", ErrUnknownEnvironment, s)
	}
}

// Principal is a synthetic, display-only identity.
type Principal struct {
	ID              string
	DisplayName     string
	CapabilityHints []string // synthetic "dev.*" display hints, never authority
}

// identities are fixed and obviously fake.
var identities = map[string]Principal{
	"dev-user-001": {ID: "dev-user-001", DisplayName: "Dev Operator", CapabilityHints: []string{"dev.shell.demo", "dev.shell.view"}},
	"dev-user-002": {ID: "dev-user-002", DisplayName: "Dev Viewer", CapabilityHints: []string{"dev.shell.view"}},
}

// Identities returns the synthetic identity ids in stable order.
func Identities() []string {
	out := make([]string, 0, len(identities))
	for id := range identities {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

type principalKey struct{}

// FromContext returns the request's synthetic principal, if any.
func FromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(Principal)
	return p, ok
}

// Stub is the enabled development auth stub.
type Stub struct{ env Environment }

// New enables the stub for env. Production is refused.
func New(env Environment) (*Stub, error) {
	if env != Development && env != Test {
		return nil, ErrProductionLockout
	}
	return &Stub{env: env}, nil
}

// Resolve maps an X-Dev-User value to a principal. Empty means anonymous.
func (s *Stub) Resolve(value string) (Principal, bool, error) {
	if value == "" {
		return Principal{}, false, nil
	}
	p, ok := identities[value]
	if !ok {
		return Principal{}, false, fmt.Errorf("%w: %q", ErrUnknownIdentity, value)
	}
	p.CapabilityHints = append([]string(nil), p.CapabilityHints...)
	return p, true, nil
}

// Middleware attaches the selected synthetic principal to the request
// context. onInvalid handles an unknown X-Dev-User value.
func (s *Stub) Middleware(onInvalid func(w http.ResponseWriter, r *http.Request, err error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok, err := s.Resolve(r.Header.Get(Header))
			if err != nil {
				onInvalid(w, r, err)
				return
			}
			if ok {
				r = r.WithContext(context.WithValue(r.Context(), principalKey{}, p))
			}
			next.ServeHTTP(w, r)
		})
	}
}
