package authorization

import (
	"context"
	"errors"
	"testing"

	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
)

// RM-E04B: RequireScope / CheckRead (SR-03, closes F-02).

func allowedCheck(s Scope) Check {
	return Check{Gate: GateAllowed, Decision: Decision{Allowed: true, Outcome: result.Success, Module: "cases", Scope: s}}
}

func TestRequireScopeSupportedPasses(t *testing.T) {
	in := allowedCheck(ScopeOrganization)
	out := RequireScope(in, ScopeOrganization)
	if out.Gate != GateAllowed || !out.Decision.Allowed || out.Decision.Scope != ScopeOrganization || out.Decision.Reason != nil {
		t.Fatalf("supported ORGANIZATION changed: %+v", out)
	}
}

func TestRequireScopeRefusesUnsupported(t *testing.T) {
	for _, tc := range []struct {
		name      string
		scope     Scope
		supported []Scope
	}{
		{"unknown scope", "OWN_TEST", []Scope{ScopeOrganization}},
		{"case-variant is not coerced", "organization", []Scope{ScopeOrganization}},
		{"NONE never passes", ScopeNone, []Scope{ScopeOrganization}},
		{"NONE never passes even if listed", ScopeNone, []Scope{ScopeNone, ScopeOrganization}},
		{"empty never passes", "", []Scope{ScopeOrganization}},
		{"empty never passes even if listed", "", []Scope{""}},
		{"no supported scope fails closed", ScopeOrganization, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := RequireScope(allowedCheck(tc.scope), tc.supported...)
			if out.Gate != GateDenied || out.Decision.Allowed || out.Decision.Outcome != result.Unauthorized ||
				out.Decision.Reason == nil || *out.Decision.Reason != reasoncodes.RowScopeDenied || out.Decision.Scope != ScopeNone {
				t.Fatalf("not refused as UNAUTHORIZED / ROW_SCOPE_DENIED: %+v", out)
			}
		})
	}
}

func TestRequireScopeRefusesInconsistentAllowedGate(t *testing.T) {
	in := allowedCheck(ScopeOrganization)
	in.Decision.Allowed = false
	if out := RequireScope(in, ScopeOrganization); out.Gate != GateDenied {
		t.Fatalf("GateAllowed with a not-allowed decision passed: %+v", out)
	}
}

func TestRequireScopeLeavesDisabledAndDeniedUnchanged(t *testing.T) {
	code := reasoncodes.ActionDenied
	denied := Check{Gate: GateDenied, Decision: Decision{Outcome: result.Unauthorized, Reason: &code, Scope: ScopeNone}}
	if out := RequireScope(denied, ScopeOrganization); out.Gate != GateDenied || *out.Decision.Reason != reasoncodes.ActionDenied {
		t.Fatalf("denied decision rewritten: %+v", out)
	}
	disabled := Check{Gate: GateDisabled}
	if out := RequireScope(disabled, ScopeOrganization); out.Gate != GateDisabled {
		t.Fatalf("disabled check rewritten: %+v", out)
	}
}

func TestRequireScopeKeepsHints(t *testing.T) {
	in := allowedCheck("OWN_TEST")
	in.Decisions = map[identityapi.Capability]Decision{identityapi.CapCaseRead: {Allowed: true}}
	out := RequireScope(in, ScopeOrganization)
	if !out.Decisions[identityapi.CapCaseRead].Allowed {
		t.Fatal("allowed_actions hints were modified")
	}
}

func on(context.Context) (bool, error) { return true, nil }

func TestCheckReadWithRealEvaluator(t *testing.T) {
	g, err := NewGuard(on, DevSubjects{})
	if err != nil {
		t.Fatal(err)
	}
	// Anonymous (no stub principal in the context): denied, unchanged.
	c, err := g.CheckRead(context.Background(), identityapi.CapCaseRead, ScopeOrganization)
	if err != nil || c.Gate != GateDenied || *c.Decision.Reason != reasoncodes.ActionDenied {
		t.Fatalf("anonymous: %+v %v", c, err)
	}
}

func TestCheckReadWithSeam(t *testing.T) {
	sub := stubSubjects{Subject{PrincipalID: "syn", UserID: "u"}}
	decide := func(s Scope) Decider {
		return func(Mode, Subject, Request) Decision {
			return Decision{Allowed: true, Outcome: result.Success, Scope: s}
		}
	}
	g, err := NewGuardWithDecider(on, sub, decide(ScopeOrganization))
	if err != nil {
		t.Fatal(err)
	}
	if c, _ := g.CheckRead(context.Background(), identityapi.CapCaseRead, ScopeOrganization); c.Gate != GateAllowed {
		t.Fatalf("ORGANIZATION refused: %+v", c)
	}
	g, _ = NewGuardWithDecider(on, sub, decide("OWN_TEST"))
	if c, _ := g.CheckRead(context.Background(), identityapi.CapCaseRead, ScopeOrganization); c.Gate != GateDenied || *c.Decision.Reason != reasoncodes.RowScopeDenied {
		t.Fatalf("unknown scope not refused: %+v", c)
	}
	// Plain Check is unchanged (commands keep their own row checks).
	if c, _ := g.Check(context.Background(), identityapi.CapCaseRead); c.Gate != GateAllowed {
		t.Fatalf("Check changed: %+v", c)
	}
	boom := errors.New("flag")
	g, _ = NewGuardWithDecider(func(context.Context) (bool, error) { return false, boom }, sub, decide(ScopeOrganization))
	if _, err := g.CheckRead(context.Background(), identityapi.CapCaseRead, ScopeOrganization); !errors.Is(err, boom) {
		t.Fatalf("flag error not propagated: %v", err)
	}
	if _, err := NewGuardWithDecider(on, sub, nil); !errors.Is(err, ErrNoDecider) {
		t.Fatalf("nil decider accepted: %v", err)
	}
	if _, err := NewGuardWithDecider(nil, sub, decide(ScopeOrganization)); !errors.Is(err, ErrNoFlag) {
		t.Fatalf("nil flag accepted: %v", err)
	}
}

type stubSubjects struct{ s Subject }

func (r stubSubjects) Subject(context.Context) Subject { return r.s }

// The evaluator scope model is unchanged: allowed is ORGANIZATION, denied
// is NONE (RM-E04B adds no scope).
func TestEvaluatorScopeModelUnchanged(t *testing.T) {
	g := identityapi.Grant{UserID: "syn", LegacyValue: "synthetic", Status: identityapi.GrantActive, Basis: DevGrantBasis,
		Capabilities: []identityapi.Capability{identityapi.CapCaseRead}}
	s := Subject{PrincipalID: "syn", Grants: []identityapi.Grant{g}}
	if d := Evaluate(Runtime, s, Request{Capability: identityapi.CapCaseRead}); !d.Allowed || d.Scope != ScopeOrganization {
		t.Fatalf("allowed decision scope %q", d.Scope)
	}
	if d := Evaluate(Runtime, s, Request{Capability: identityapi.CapInvoiceRead}); d.Allowed || d.Scope != ScopeNone {
		t.Fatalf("denied decision scope %q", d.Scope)
	}
}
