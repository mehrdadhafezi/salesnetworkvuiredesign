// Package domain is the payments module's pure money logic: parsing raw
// legacy decimal text into the frozen Money contract (int64 + explicit unit,
// no rounding, no conversion) and summing per unit with overflow checks.
// It mirrors billing's rules without importing billing (module boundary).
// Absent amounts are never zero; units are never combined or converted.
package domain

import (
	"errors"
	"regexp"
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
