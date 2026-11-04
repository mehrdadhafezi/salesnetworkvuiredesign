package domain

import (
	"sort"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/money"
)

// ParityClass is one parity outcome of a legacy entitlement vs the CRM
// recalculation of it.
type ParityClass string

const (
	ParityMatch                ParityClass = "MATCH"
	ParityAmountMismatch       ParityClass = "AMOUNT_MISMATCH"
	ParityRecipientMismatch    ParityClass = "RECIPIENT_MISMATCH"
	ParityPolicyMismatch       ParityClass = "POLICY_MISMATCH"
	ParityAttributionUnknown   ParityClass = "ATTRIBUTION_UNKNOWN"
	ParityDuplicateRisk        ParityClass = "DUPLICATE_RISK"
	ParityEngineRoutingUnknown ParityClass = "ENGINE_ROUTING_UNKNOWN"
	ParitySourceIncomplete     ParityClass = "SOURCE_INCOMPLETE"
	ParityNotComparable        ParityClass = "NOT_COMPARABLE"
)

// ParityClasses lists every class in report order.
func ParityClasses() []ParityClass {
	return []ParityClass{ParityMatch, ParityAmountMismatch, ParityRecipientMismatch, ParityPolicyMismatch, ParityAttributionUnknown,
		ParityDuplicateRisk, ParityEngineRoutingUnknown, ParitySourceIncomplete, ParityNotComparable}
}

// Legacy is the legacy side of one entitlement, from the writer's own
// event-time snapshot (never recomputed from today's hierarchy or rates).
type Legacy struct {
	SourceKey       string
	Engine          Engine
	EntitlementKey  *string
	Amount          *Exact
	AmountRaw       *string
	RecipientKey    *string // the credited wallet holder (credit owner)
	SnapshotKey     *string // the writer's recipient snapshot (meta)
	RateSnapshot    *Rate
	RateObserved    *Rate // the CURRENT observed policy value (unversioned)
	Base            *Exact
	Comparable      bool   // false: D-36 / non-invoice / unsupported
	NotComparableBy string // why
}

// Field is one compared field.
type Field struct {
	Name   string  `json:"name"`
	Legacy *string `json:"legacy"`
	Target *string `json:"target"`
	Equal  *bool   `json:"equal"` // nil: not comparable
}

// ParityRow is the parity of one legacy entitlement.
type ParityRow struct {
	SourceKey      string      `json:"source_key"`
	EntitlementKey *string     `json:"entitlement_key"`
	Engine         Engine      `json:"engine"`
	Class          ParityClass `json:"class"`
	Fields         []Field     `json:"fields"`
	Reasons        []string    `json:"reasons"`
}

func ptr(s string) *string { return &s }
func bptr(b bool) *bool    { return &b }

// Compare classifies one legacy entitlement against its recalculation.
// duplicate is true when another live credit shares its entitlement key.
// The FIRST failing class (in a fixed precedence) is the row class; every
// field comparison is still kept.
func Compare(l Legacy, out *Output, duplicate bool) ParityRow {
	row := ParityRow{SourceKey: l.SourceKey, EntitlementKey: l.EntitlementKey, Engine: l.Engine, Reasons: []string{}}
	f := func(name string, a, b *string) {
		fl := Field{Name: name, Legacy: a, Target: b}
		if a != nil && b != nil {
			fl.Equal = bptr(*a == *b)
		}
		row.Fields = append(row.Fields, fl)
	}
	var tgtAmount, tgtRecipient, tgtRate, tgtBase, eligibility *string
	if out != nil {
		if out.Amount != nil {
			tgtAmount = ptr(out.Amount.String())
		}
		tgtRecipient = out.Recipient.UserKey
		if out.Rate != nil {
			tgtRate = ptr(out.Rate.String())
		}
		if out.Base != nil {
			tgtBase = ptr(out.Base.String())
		}
		eligibility = ptr(string(out.Eligibility))
		row.Reasons = append(row.Reasons, out.Reasons...)
		for _, b := range out.Blocks {
			row.Reasons = append(row.Reasons, string(b))
		}
	}
	var legAmount, legRate, legBase *string
	if l.Amount != nil {
		legAmount = ptr(l.Amount.String())
	} else {
		legAmount = l.AmountRaw
	}
	if l.RateSnapshot != nil {
		legRate = ptr(l.RateSnapshot.String())
	}
	if l.Base != nil {
		legBase = ptr(l.Base.String())
	}
	f("eligibility", ptr(string(Eligible)), eligibility)
	f("base", legBase, tgtBase)
	f("rate", legRate, tgtRate)
	f("amount", legAmount, tgtAmount)
	f("unit", unitOf(l.Amount), unitOfOut(out))
	f("recipient", l.RecipientKey, tgtRecipient)
	f("recipient_snapshot", l.SnapshotKey, tgtRecipient)
	f("share_facet_identity", l.EntitlementKey, outKey(out))
	var observed *string
	if l.RateObserved != nil {
		observed = ptr(l.RateObserved.String())
	}
	f("policy_observed_rate", legRate, observed)
	f("engine", ptr(string(l.Engine)), ptr(string(l.Engine)))

	switch {
	case !l.Comparable:
		row.Class = ParityNotComparable
		row.Reasons = append(row.Reasons, l.NotComparableBy)
	case l.Engine == EngineUnknown:
		row.Class = ParityEngineRoutingUnknown
	case duplicate:
		row.Class = ParityDuplicateRisk
	case l.SnapshotKey == nil || (out != nil && out.Recipient.State != AttrSnapshot):
		row.Class = ParityAttributionUnknown
	case l.EntitlementKey == nil || l.Amount == nil || l.Base == nil || l.RateSnapshot == nil || out == nil || out.Amount == nil:
		row.Class = ParitySourceIncomplete
	case l.RecipientKey == nil || *l.RecipientKey != *l.SnapshotKey:
		row.Class = ParityRecipientMismatch
	case l.Amount.Unit != out.Amount.Unit || l.Amount.Hundredths != out.Amount.Hundredths:
		row.Class = ParityAmountMismatch
	case l.RateObserved != nil && l.RateObserved.E4 != l.RateSnapshot.E4:
		// The amount matches the EVENT-TIME rate; the current (unversioned)
		// matrix value differs: the policy changed after the event.
		row.Class = ParityPolicyMismatch
	default:
		row.Class = ParityMatch
	}
	sort.Strings(row.Reasons)
	row.Reasons = dedupe(row.Reasons)
	return row
}

func outKey(o *Output) *string {
	if o == nil {
		return nil
	}
	return o.EntitlementKey
}

func unitOf(e *Exact) *string {
	if e == nil {
		return nil
	}
	return ptr(e.Unit.String())
}

func unitOfOut(o *Output) *string {
	if o == nil || o.Amount == nil {
		return nil
	}
	return unitOf(o.Amount)
}

func dedupe(in []string) []string {
	out := []string{}
	for i, s := range in {
		if i == 0 || s != in[i-1] {
			out = append(out, s)
		}
	}
	return out
}

// Totals sums exact amounts per unit; a row without an exact amount is
// counted as missing, never as zero.
type Totals struct {
	ByUnit  map[string]int64 `json:"by_unit_hundredths"`
	Missing int64            `json:"missing"`
}

// Add accumulates one amount.
func (t *Totals) Add(e *Exact) error {
	if t.ByUnit == nil {
		t.ByUnit = map[string]int64{}
	}
	if e == nil || !e.Unit.Valid() {
		t.Missing++
		return nil
	}
	cur := Exact{Hundredths: t.ByUnit[e.Unit.String()], Unit: e.Unit}
	s, err := AddExact(cur, *e)
	if err != nil {
		return err
	}
	t.ByUnit[e.Unit.String()] = s.Hundredths
	return nil
}

// Decimal renders the totals per unit as exact decimal text.
func (t Totals) Decimal() map[string]string {
	out := map[string]string{}
	for u, v := range t.ByUnit {
		out[u] = FormatDecimal(v, AmountScale)
	}
	return out
}

// IRT is the evidenced unit of every legacy commission amount (Toman
// stage / invoice amounts, class-sn-plugin.php:31200; Slice 7 / 9 evidence).
const IRT = money.IRT
