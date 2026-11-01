// Package domain is the billing module's pure, deterministic logic over
// money evidence: parsing raw legacy decimal text into the frozen Money
// contract (int64 + explicit unit, no rounding, no conversion), summing per
// unit with overflow checks, and classifying per-invoice total evidence.
//
// TAD #19 (canonical invoice-total resolver) is deferred: nothing here
// selects a canonical amount. Absent amounts are never zero; amounts in
// different units are never combined; there is no IRR/IRT ratio anywhere.
package domain

import (
	"errors"
	"regexp"
	"sort"
	"strconv"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/money"
)

// Representation of one raw amount.
const (
	Representable   = "REPRESENTABLE"
	Unrepresentable = "UNREPRESENTABLE"
	Absent          = "ABSENT"
)

// decimalPattern accepts a plain legacy DECIMAL(18,2) text: optional sign,
// digits, optional fraction of up to two digits.
var decimalPattern = regexp.MustCompile(`^(-?)([0-9]{1,18})(?:\.([0-9]{1,2}))?$`)

// Amount is one parsed money evidence.
type Amount struct {
	Raw            *string
	Money          *money.Money
	Representation string
}

// ParseAmount turns raw legacy decimal text and an explicit unit into Money
// only when the value is integral in that unit. A non-zero fraction, an
// unparseable value or an unknown unit is UNREPRESENTABLE (never rounded,
// never converted); NULL / empty is ABSENT (never zero).
func ParseAmount(raw *string, unit string) Amount {
	a := Amount{Raw: raw}
	if raw == nil || *raw == "" {
		a.Representation = Absent
		return a
	}
	u, err := money.ParseUnit(unit)
	m := decimalPattern.FindStringSubmatch(*raw)
	if err != nil || m == nil {
		a.Representation = Unrepresentable
		return a
	}
	if frac := m[3]; frac != "" && frac != "0" && frac != "00" {
		a.Representation = Unrepresentable
		return a
	}
	n, err := strconv.ParseInt(m[1]+m[2], 10, 64)
	if err != nil {
		a.Representation = Unrepresentable
		return a
	}
	mm, err := money.New(n, u)
	if err != nil {
		a.Representation = Unrepresentable
		return a
	}
	a.Money, a.Representation = &mm, Representable
	return a
}

// SumState is the state of one per-unit sum.
const (
	SumComplete   = "COMPLETE"   // every contributing row representable
	SumIncomplete = "INCOMPLETE" // some row unrepresentable/absent, or the source is incomplete
	SumNone       = "NONE"       // no contributing row at all (absent is not zero)
	SumOverflow   = "OVERFLOW"   // int64 overflow; no value claimed
)

// Sum is one per-unit sum.
type Sum struct {
	Unit   string
	Amount *int64
	State  string
	Rows   int
}

// SumByUnit sums amounts grouped by unit. Rows whose unit is unknown (not
// representable) make every unit's sum INCOMPLETE because their unit cannot
// be attributed. sourceComplete=false marks every sum INCOMPLETE.
func SumByUnit(amounts []Amount, sourceComplete bool) map[string]Sum {
	out := map[string]Sum{}
	unattributable := false
	for _, a := range amounts {
		if a.Representation != Representable {
			unattributable = true
			continue
		}
		u := a.Money.Unit.String()
		s, ok := out[u]
		if !ok {
			zero := int64(0)
			s = Sum{Unit: u, Amount: &zero, State: SumComplete}
		}
		s.Rows++
		if s.State != SumOverflow {
			acc, err := money.Add(money.Money{Amount: *s.Amount, Unit: a.Money.Unit}, *a.Money)
			if errors.Is(err, money.ErrOverflow) || err != nil {
				s.State, s.Amount = SumOverflow, nil
			} else {
				v := acc.Amount
				s.Amount = &v
			}
		}
		out[u] = s
	}
	for u, s := range out {
		if s.State == SumComplete && (unattributable || !sourceComplete) {
			s.State = SumIncomplete
		}
		out[u] = s
	}
	return out
}

// Comparison classes per invoice and unit.
const (
	Match      = "MATCH"
	Mismatch   = "MISMATCH"
	Unknown    = "UNKNOWN"
	Incomplete = "INCOMPLETE"
	Conflict   = "CONFLICT"
)

// Evidence is one compared value for a unit: a declared amount, or a sum.
type Evidence struct {
	Name   string // e.g. declared:final_total, item_sum, stage_sum, payment_sum
	Amount *int64
	State  string // REPRESENTABLE / UNREPRESENTABLE / ABSENT or a Sum state
}

// Declared reports whether the evidence is a declared invoice amount.
func (e Evidence) Declared() bool { return len(e.Name) > 9 && e.Name[:9] == "declared:" }

// Compare classifies the evidence of one unit. It never picks a winner:
//   - CONFLICT: two declared total evidences disagree with each other;
//   - INCOMPLETE: some evidence is unrepresentable, incomplete or overflowed;
//   - UNKNOWN: fewer than two comparable values, or no computed sum at all
//     (declared fields agreeing with each other corroborate nothing);
//   - MATCH / MISMATCH: all comparable values equal / not.
func Compare(ev []Evidence) string {
	var vals []int64
	var declared []int64
	computed := 0
	incomplete := false
	for _, e := range ev {
		switch e.State {
		case Representable, SumComplete:
			if e.Amount == nil {
				incomplete = true
				continue
			}
			vals = append(vals, *e.Amount)
			if e.Declared() {
				declared = append(declared, *e.Amount)
			} else {
				computed++
			}
		case Absent, SumNone:
			// not evidence; absent is not zero
		default:
			incomplete = true
		}
	}
	for i := 1; i < len(declared); i++ {
		if declared[i] != declared[0] {
			return Conflict
		}
	}
	if incomplete {
		return Incomplete
	}
	if len(vals) < 2 || computed == 0 {
		return Unknown
	}
	for _, v := range vals[1:] {
		if v != vals[0] {
			return Mismatch
		}
	}
	return Match
}

// Units lists the units appearing in any of the maps, sorted.
func Units(maps ...map[string]Sum) []string {
	set := map[string]bool{}
	for _, m := range maps {
		for u := range m {
			set[u] = true
		}
	}
	out := make([]string, 0, len(set))
	for u := range set {
		out = append(out, u)
	}
	sort.Strings(out)
	return out
}
