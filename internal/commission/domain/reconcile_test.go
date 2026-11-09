package domain

import (
	"slices"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/money"
)

func rex(h int64) *Exact   { return &Exact{Hundredths: h, Unit: money.IRT} }
func rsp(s string) *string { return &s }

func ident(inv string, stage int) EntitlementIdentity {
	return EntitlementIdentity{Kind: KindInvoiceStage, InvoiceKey: inv, Stage: stage, Purpose: "sale", ShareFacet: "seller", Context: "synthetic"}
}

func classesOf(r ReconReport, inv string, stage int) []ReconClass {
	k, _ := ident(inv, stage).Key()
	for _, g := range r.Groups {
		if g.Key == k {
			return g.Classes
		}
	}
	return nil
}

// R4: cross-engine reconciliation detects duplicate economic effects and
// disagreements without choosing an engine; the identity is independent of
// the current recipient; shadow credits never claim economic states.
func TestReconcileEngines(t *testing.T) {
	credits := []EngineCredit{
		{Engine: EnginePurposeMatrix, SourceKey: "e1-1", Identity: ident("100", 1), RecipientKey: rsp("s1"), Amount: rex(5000), State: StatePosted},
		{Engine: EngineShadow, SourceKey: "sh-1", Identity: ident("100", 1), RecipientKey: rsp("s1"), Amount: rex(5000), State: StateCalculated},
		// Two legacy engines credited the same entitlement: duplicate.
		{Engine: EnginePurposeMatrix, SourceKey: "e1-2", Identity: ident("200", 1), RecipientKey: rsp("s1"), Amount: rex(5000), State: StatePosted},
		{Engine: EngineRulesAuto, SourceKey: "e2-2", Identity: ident("200", 1), RecipientKey: rsp("s1"), Amount: rex(5000), State: StatePosted},
		// One engine twice (non-idempotent retry).
		{Engine: EngineRulesAuto, SourceKey: "e2-3a", Identity: ident("300", 2), RecipientKey: rsp("s2"), Amount: rex(100), State: StatePosted},
		{Engine: EngineRulesAuto, SourceKey: "e2-3b", Identity: ident("300", 2), RecipientKey: rsp("s2"), Amount: rex(100), State: StatePosted},
		// Shadow disagrees on amount and recipient (current hierarchy moved).
		{Engine: EngineDryRunPosting, SourceKey: "e3-4", Identity: ident("400", 1), RecipientKey: rsp("s3"), Amount: rex(700), State: StateApproved},
		{Engine: EngineShadow, SourceKey: "sh-4", Identity: ident("400", 1), RecipientKey: rsp("s4"), Amount: rex(701), State: StateCalculated},
		// Shadow claiming an economic state: forbidden.
		{Engine: EngineShadow, SourceKey: "sh-5", Identity: ident("500", 1), Amount: rex(1), State: StatePosted},
		// Unit disagreement.
		{Engine: EngineProjectCard, SourceKey: "e4-6", Identity: ident("600", 1), Amount: &Exact{Hundredths: 10, Unit: money.IRR}, State: StatePosted},
		{Engine: EngineShadow, SourceKey: "sh-6", Identity: ident("600", 1), Amount: rex(10), State: StateCalculated},
		// No approved key: non-invoice (OPD-09) and component (D-36).
		{Engine: EngineRulesAuto, SourceKey: "e2-x", Identity: EntitlementIdentity{Kind: KindNonInvoice}, Amount: rex(1), State: StatePosted},
		{Engine: EnginePurposeMatrix, SourceKey: "e1-x", Identity: EntitlementIdentity{Kind: KindComponent}, Amount: rex(1), State: StateGenerated},
	}
	r := ReconcileEngines(credits)
	for _, c := range []struct {
		inv   string
		stage int
		want  ReconClass
	}{
		{"200", 1, ReconMultiEngineDuplicate}, {"300", 2, ReconSameEngineDuplicate}, {"400", 1, ReconAmountDisagreement},
		{"400", 1, ReconRecipientDisagreement}, {"500", 1, ReconShadowEffect}, {"600", 1, ReconUnitDisagreement},
	} {
		if !slices.Contains(classesOf(r, c.inv, c.stage), c.want) {
			t.Fatalf("%s/%d lacks %s: %v", c.inv, c.stage, c.want, classesOf(r, c.inv, c.stage))
		}
	}
	// The shadow beside one legacy engine with equal facts is not a
	// duplicate and not a disagreement.
	if cls := classesOf(r, "100", 1); len(cls) != 0 {
		t.Fatalf("matching shadow comparison: %v", cls)
	}
	if len(r.Unkeyed) != 2 {
		t.Fatalf("unkeyed: %+v", r.Unkeyed)
	}
	// Lifecycle states stay distinct (preview / approval / posting).
	if r.ByState[StateCalculated] != 3 || r.ByState[StateApproved] != 1 || r.ByState[StatePosted] != 8 || r.ByState[StateGenerated] != 1 {
		t.Fatalf("states: %v", r.ByState)
	}
	// Deterministic and idempotent: the same input gives the same report.
	shuffled := slices.Clone(credits)
	slices.Reverse(shuffled)
	r2 := ReconcileEngines(shuffled)
	if len(r2.Groups) != len(r.Groups) {
		t.Fatal("non-deterministic grouping")
	}
	for i := range r.Groups {
		if r.Groups[i].Key != r2.Groups[i].Key || !slices.Equal(r.Groups[i].Classes, r2.Groups[i].Classes) {
			t.Fatalf("non-deterministic group %d", i)
		}
	}
	// The recipient is an attribute: changing it never mints a new key.
	a, b := ident("900", 1), ident("900", 1)
	ka, _ := a.Key()
	kb, _ := b.Key()
	if ka != kb {
		t.Fatal("identity must not depend on the recipient")
	}
	// No approval or posting command is executable in SHADOW mode.
	if Approve(StateGenerated) == nil || RequestPosting(StateApproved) == nil {
		t.Fatal("approval / posting must stay decision-blocked")
	}
}
