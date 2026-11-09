package domain

import (
	"sort"
)

// R4: cross-engine commission RECONCILIATION (Gate-0 INV-032; OPD-09 and
// D-36 open). Legacy credits come from four engines (E1–E4) and the CRM
// shadow computes its own; no engine is assumed to be the approved one.
// Reconciliation groups every observed credit by its entitlement key and
// classifies disagreements. It never picks a winner, never merges, never
// posts and never changes a legacy calculation.

// ReconClass classifies one entitlement key across engines.
type ReconClass string

const (
	// ReconSingleSource: exactly one credit for the key.
	ReconSingleSource ReconClass = "SINGLE_SOURCE"
	// ReconMultiEngineDuplicate: two or more DIFFERENT engines credited the
	// same entitlement: a duplicate economic effect risk (OPD-09).
	ReconMultiEngineDuplicate ReconClass = "MULTI_ENGINE_DUPLICATE"
	// ReconSameEngineDuplicate: one engine credited the same entitlement
	// more than once (a retry that was not idempotent).
	ReconSameEngineDuplicate ReconClass = "SAME_ENGINE_DUPLICATE"
	// ReconAmountDisagreement / ReconUnitDisagreement: the credits of one
	// key disagree on the exact amount or on the unit.
	ReconAmountDisagreement ReconClass = "AMOUNT_DISAGREEMENT"
	ReconUnitDisagreement   ReconClass = "UNIT_DISAGREEMENT"
	// ReconRecipientDisagreement: the credits disagree on the credit owner
	// (historical credit ownership is never recomputed from the current
	// hierarchy; this is a finding, not a correction).
	ReconRecipientDisagreement ReconClass = "RECIPIENT_DISAGREEMENT"
	// ReconAmountUnknown: a credit has no exact amount.
	ReconAmountUnknown ReconClass = "AMOUNT_UNKNOWN"
	// ReconShadowEffect: a CRM shadow credit claims an economic state
	// (posted / settled / posting requested): forbidden in SHADOW mode.
	ReconShadowEffect ReconClass = "SHADOW_ECONOMIC_EFFECT"
)

// EngineShadow names the CRM shadow calculation as a reconciliation source
// (it is not a legacy engine and not an approved engine).
const EngineShadow Engine = "CRM_SHADOW"

// EngineCredit is one observed credit / entitlement from one source.
type EngineCredit struct {
	Engine    Engine
	SourceKey string
	Identity  EntitlementIdentity
	// RecipientKey is the credit owner snapshot (an attribute, never part
	// of the identity).
	RecipientKey *string
	Amount       *Exact
	State        RunState
}

// ReconGroup is the reconciliation of one entitlement key.
type ReconGroup struct {
	Key     string
	Credits []EngineCredit
	Engines []Engine
	Classes []ReconClass
}

// Unkeyed is a credit for which no approved entitlement key exists.
type Unkeyed struct {
	Credit EngineCredit
	Reason string
}

// ReconReport is the full cross-engine reconciliation.
type ReconReport struct {
	Groups  []ReconGroup
	Unkeyed []Unkeyed
	// ByState counts credits per lifecycle state: preview (calculated /
	// generated), approval, posting and settlement stay distinct.
	ByState map[RunState]int
	// ClassCounts counts groups per class.
	ClassCounts map[ReconClass]int
}

// economic states a SHADOW credit may never claim.
var economic = map[RunState]bool{StatePostingRequested: true, StatePosted: true, StateSettled: true, StateReversed: true, StateCorrected: true}

// ReconcileEngines reconciles credits from every engine. The result is
// deterministic (sorted by key, then engine, then source key).
func ReconcileEngines(credits []EngineCredit) ReconReport {
	r := ReconReport{ByState: map[RunState]int{}, ClassCounts: map[ReconClass]int{}}
	byKey := map[string][]EngineCredit{}
	for _, c := range credits {
		r.ByState[c.State]++
		k, err := c.Identity.Key()
		if err != nil {
			r.Unkeyed = append(r.Unkeyed, Unkeyed{Credit: c, Reason: err.Error()})
			continue
		}
		byKey[k] = append(byKey[k], c)
	}
	keys := make([]string, 0, len(byKey))
	for k := range byKey {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		cs := byKey[k]
		sort.Slice(cs, func(i, j int) bool {
			if cs[i].Engine != cs[j].Engine {
				return cs[i].Engine < cs[j].Engine
			}
			return cs[i].SourceKey < cs[j].SourceKey
		})
		g := ReconGroup{Key: k, Credits: cs}
		classes := map[ReconClass]bool{}
		perEngine := map[Engine]int{}
		for _, c := range cs {
			perEngine[c.Engine]++
			if c.Engine == EngineShadow && economic[c.State] {
				classes[ReconShadowEffect] = true
			}
			if c.Amount == nil {
				classes[ReconAmountUnknown] = true
			}
		}
		for e, n := range perEngine {
			g.Engines = append(g.Engines, e)
			if n > 1 {
				classes[ReconSameEngineDuplicate] = true
			}
		}
		sort.Slice(g.Engines, func(i, j int) bool { return g.Engines[i] < g.Engines[j] })
		// Legacy engines crediting the same key twice is a duplicate
		// economic effect; the shadow beside one legacy engine is a
		// comparison, not a duplicate.
		legacy := 0
		for _, e := range g.Engines {
			if e != EngineShadow {
				legacy++
			}
		}
		if legacy > 1 {
			classes[ReconMultiEngineDuplicate] = true
		}
		for i := 1; i < len(cs); i++ {
			a, b := cs[0], cs[i]
			if a.Amount != nil && b.Amount != nil {
				if a.Amount.Unit != b.Amount.Unit {
					classes[ReconUnitDisagreement] = true
				} else if a.Amount.Hundredths != b.Amount.Hundredths {
					classes[ReconAmountDisagreement] = true
				}
			}
			if (a.RecipientKey == nil) != (b.RecipientKey == nil) || a.RecipientKey != nil && *a.RecipientKey != *b.RecipientKey {
				classes[ReconRecipientDisagreement] = true
			}
		}
		if len(cs) == 1 && len(classes) == 0 {
			classes[ReconSingleSource] = true
		}
		for _, c := range []ReconClass{ReconShadowEffect, ReconMultiEngineDuplicate, ReconSameEngineDuplicate, ReconUnitDisagreement,
			ReconAmountDisagreement, ReconRecipientDisagreement, ReconAmountUnknown, ReconSingleSource} {
			if classes[c] {
				g.Classes = append(g.Classes, c)
				r.ClassCounts[c]++
			}
		}
		r.Groups = append(r.Groups, g)
	}
	return r
}
