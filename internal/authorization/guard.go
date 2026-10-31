package authorization

import (
	"context"
	"errors"

	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/authstub"
)

// SubjectResolver turns a request into an evaluated subject.
type SubjectResolver interface {
	Subject(ctx context.Context) Subject
}

// Anonymous resolves every request to the anonymous subject. It is the
// production resolver of Slice 4: there is no production authentication,
// so every Slice 4 read is denied outside development/test.
type Anonymous struct{}

// Subject implements SubjectResolver.
func (Anonymous) Subject(context.Context) Subject { return Subject{} }

// DevSubjects maps the DEV/TEST stub's SYNTHETIC principals to synthetic
// ACTIVE grants so the HR read surfaces can be exercised locally. It is
// constructed only when the dev auth stub is enabled, which the server
// refuses in production (authstub.ErrProductionLockout). It is not
// production authorization and grants nothing to real accounts.
type DevSubjects struct{}

// devGrants are the synthetic grants per stub identity:
//   - dev-user-001 ("Dev Operator"): the HR role (Slice 4 HR read);
//   - dev-user-002 ("Dev Viewer"): nothing (every HR read is denied).
var devGrants = map[string][]identityapi.Role{
	"dev-user-001": {identityapi.RoleHR},
	"dev-user-002": nil,
}

// DevGrantBasis marks synthetic dev/test grants.
const DevGrantBasis = "SYNTHETIC_DEV_STUB"

// Subject implements SubjectResolver.
func (DevSubjects) Subject(ctx context.Context) Subject {
	p, ok := authstub.FromContext(ctx)
	if !ok {
		return Subject{}
	}
	s := Subject{PrincipalID: p.ID}
	for _, r := range devGrants[p.ID] {
		role := r
		s.Grants = append(s.Grants, identityapi.Grant{UserID: p.ID, RoleKey: &role, LegacyValue: "synthetic", Status: identityapi.GrantActive, Basis: DevGrantBasis})
	}
	return s
}

// FlagFunc reports whether the Slice 4 HR read surface is enabled.
type FlagFunc func(ctx context.Context) (bool, error)

// Gate is the result of guarding one request.
type Gate uint8

const (
	// GateDisabled: the feature flag is off (BLOCKED / FEATURE_DISABLED).
	GateDisabled Gate = iota
	// GateDenied: the subject lacks the capability (UNAUTHORIZED).
	GateDenied
	// GateAllowed: the read may proceed.
	GateAllowed
)

// Guard evaluates every Slice 4 read request: flag first, then the
// evaluator in Runtime mode (ACTIVE grants only). allowed_actions hints are
// produced by the same evaluator.
type Guard struct {
	enabled  FlagFunc
	subjects SubjectResolver
}

// ErrNoFlag reports a guard without a flag source.
var ErrNoFlag = errors.New("authorization: guard requires a flag source")

// NewGuard returns a guard. A nil resolver means Anonymous.
func NewGuard(enabled FlagFunc, subjects SubjectResolver) (*Guard, error) {
	if enabled == nil {
		return nil, ErrNoFlag
	}
	if subjects == nil {
		subjects = Anonymous{}
	}
	return &Guard{enabled: enabled, subjects: subjects}, nil
}

// Check is the outcome of guarding one request.
type Check struct {
	Gate      Gate
	Decision  Decision
	Decisions map[identityapi.Capability]Decision
}

// Can reports whether capability c is allowed in this check.
func (c Check) Can(cap identityapi.Capability) bool { return c.Decisions[cap].Allowed }

// Check guards one read of capability.
func (g *Guard) Check(ctx context.Context, capability identityapi.Capability) (Check, error) {
	on, err := g.enabled(ctx)
	if err != nil {
		return Check{}, err
	}
	if !on {
		return Check{Gate: GateDisabled}, nil
	}
	s := g.subjects.Subject(ctx)
	all := EvaluateAll(Runtime, s)
	d := Evaluate(Runtime, s, Request{Capability: capability})
	gate := GateDenied
	if d.Allowed {
		gate = GateAllowed
	}
	return Check{Gate: gate, Decision: d, Decisions: all}, nil
}

// Effective evaluates every capability for the caller.
func (g *Guard) Effective(ctx context.Context) (enabled bool, authenticated bool, decisions map[identityapi.Capability]Decision, err error) {
	on, err := g.enabled(ctx)
	if err != nil {
		return false, false, nil, err
	}
	s := g.subjects.Subject(ctx)
	return on, s.PrincipalID != "", EvaluateAll(Runtime, s), nil
}

// Enabled reports the flag state.
func (g *Guard) Enabled(ctx context.Context) (bool, error) { return g.enabled(ctx) }
