package authorization

import (
	"context"
	"errors"

	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
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
//   - dev-user-001 ("Dev Operator"): the HR role (Slice 4 HR read) and a
//     direct synthetic grant of the Slice 5 customer read, Slice 6 sales
//     read and Slice 7 billing/payments read capabilities (no target role
//     carries them; D-30, OPD-01, D-25/26/28 are open);
//   - dev-user-002 ("Dev Viewer"): nothing (every read is denied).
var devGrants = map[string][]identityapi.Role{
	"dev-user-001": {identityapi.RoleHR},
	"dev-user-002": nil,
}

var devCapabilities = map[string][]identityapi.Capability{
	"dev-user-001": {identityapi.CapCustomerRead, identityapi.CapMergeCaseRead, identityapi.CapIdentityReportRead,
		identityapi.CapCaseRead, identityapi.CapCustodyRead, identityapi.CapSalesReportRead,
		identityapi.CapInvoiceRead, identityapi.CapPaymentAttemptRead, identityapi.CapBillingReportRead,
		identityapi.CapReportingMetricsRead, identityapi.CapReportingMISRead, identityapi.CapInvoiceReviewRead,
		identityapi.CapFinanceShadowRead, identityapi.CapFinanceLedgerRead, identityapi.CapFinanceReconciliationRead,
		identityapi.CapBiawinSubscriptionRead, identityapi.CapFulfillmentCaseRead, identityapi.CapBiawinParityRead,
		identityapi.CapContactOutcomeRecord, identityapi.CapHRRequestCreate, identityapi.CapHRRequestReview, identityapi.CapHRRequestRead,
		identityapi.CapWritebackRead, identityapi.CapPaymentUploadStage, identityapi.CapPaymentBatchSubmit, identityapi.CapPaymentSubmissionRead,
		identityapi.CapPaymentEvidenceRead, identityapi.CapFinanceReviewRead,
		identityapi.CapCommissionShadowRead, identityapi.CapCommissionPolicyRead, identityapi.CapCommissionOwnRead,
		identityapi.CapWorkRead, identityapi.CapWorkAssign, identityapi.CapWorkExecute, identityapi.CapActivationRead,
		identityapi.CapProviderOperationRead, identityapi.CapProviderOperationRecon},
}

// DevGrantBasis marks synthetic dev/test grants.
const DevGrantBasis = "SYNTHETIC_DEV_STUB"

// SyntheticDevRole names a matched synthetic direct-capability grant.
const SyntheticDevRole identityapi.Role = "synthetic_dev_stub"

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
	if caps := devCapabilities[p.ID]; len(caps) > 0 {
		s.Grants = append(s.Grants, identityapi.Grant{UserID: p.ID, LegacyValue: "synthetic", Status: identityapi.GrantActive, Basis: DevGrantBasis,
			Capabilities: append([]identityapi.Capability(nil), caps...)})
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
	decide   Decider
}

// Decider decides one request. Every production guard uses Evaluate.
type Decider func(Mode, Subject, Request) Decision

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
	return &Guard{enabled: enabled, subjects: subjects, decide: Evaluate}, nil
}

// ErrNoDecider reports a seam guard without a decider.
var ErrNoDecider = errors.New("authorization: guard requires a decider")

// NewGuardWithDecider returns a guard whose per-request decision comes from
// decide instead of Evaluate. It is a TEST SEAM ONLY (RM-E04B, NT-07): it
// lets a test hand a handler a synthetic decision, e.g. an allowed decision
// with a scope the evaluator cannot produce, to prove that the handler
// refuses it before any query. The architecture lint (authz-decision-seam)
// allows a call only from _test.go files; production composition always
// uses NewGuard. allowed_actions hints keep using Evaluate.
func NewGuardWithDecider(enabled FlagFunc, subjects SubjectResolver, decide Decider) (*Guard, error) {
	if decide == nil {
		return nil, ErrNoDecider
	}
	g, err := NewGuard(enabled, subjects)
	if err != nil {
		return nil, err
	}
	g.decide = decide
	return g, nil
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
	d := g.decide(Runtime, s, Request{Capability: capability})
	gate := GateDenied
	if d.Allowed {
		gate = GateAllowed
	}
	return Check{Gate: gate, Decision: d, Decisions: all}, nil
}

// CheckRead guards one data-returning read (RM-E04B, SR-03, closes F-02).
// It is Check followed by RequireScope: supported lists the evaluator
// scopes the calling handler explicitly implements. Today every reader
// implements ORGANIZATION only. An allowed decision with any other scope
// becomes GateDenied with UNAUTHORIZED / ROW_SCOPE_DENIED, so the handler's
// existing denied branch answers 403 before any module query runs. An
// empty supported list refuses every allowed decision (fail closed).
func (g *Guard) CheckRead(ctx context.Context, capability identityapi.Capability, supported ...Scope) (Check, error) {
	c, err := g.Check(ctx, capability)
	if err != nil {
		return c, err
	}
	return RequireScope(c, supported...), nil
}

// RequireScope enforces the supported-scope contract on one check. It
// never broadens: a disabled or denied check is returned unchanged; an
// allowed check passes only when its decision is allowed and its scope is
// one of supported. NONE and the empty scope never pass, even if listed,
// because they grant no rows. Nothing is coerced to ORGANIZATION. The
// refusal keeps the established read convention (UNAUTHORIZED /
// ROW_SCOPE_DENIED, scope NONE). allowed_actions hints are untouched.
func RequireScope(c Check, supported ...Scope) Check {
	if c.Gate != GateAllowed {
		return c
	}
	if c.Decision.Allowed && scopeSupported(c.Decision.Scope, supported) {
		return c
	}
	code := reasoncodes.RowScopeDenied
	d := c.Decision
	d.Allowed, d.Outcome, d.Reason, d.Scope = false, result.Unauthorized, &code, ScopeNone
	c.Gate, c.Decision = GateDenied, d
	return c
}

func scopeSupported(s Scope, supported []Scope) bool {
	if s == ScopeNone || s == "" {
		return false
	}
	for _, x := range supported {
		if x == s {
			return true
		}
	}
	return false
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

// Availability reports, for one request, whether the feature that guards
// each capability is enabled (RM-ACCESS-01). It is availability only and
// never authority: a capability absent from the map, or mapped to false, is
// unavailable, and an error fails the whole evaluation closed. Each
// capability's entry must be the flag of the guard that enforces it, so the
// effective-access view never reports a capability as allowed whose own
// endpoint answers BLOCKED / FEATURE_DISABLED, and one module's switch never
// decides another module's capability.
type Availability func(ctx context.Context) (map[identityapi.Capability]bool, error)

// Subject returns the caller's subject for row-scope checks of a command
// that has already passed Check. It grants nothing.
func (g *Guard) Subject(ctx context.Context) Subject { return g.subjects.Subject(ctx) }

// Enabled reports the flag state.
func (g *Guard) Enabled(ctx context.Context) (bool, error) { return g.enabled(ctx) }
