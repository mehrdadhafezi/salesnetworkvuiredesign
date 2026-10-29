// Package money implements the frozen Slice 0 money contract (E0 §9D).
//
//	Money { Amount int64; Unit IRR | IRT }
//
// The unit is mandatory, IRR and IRT are distinct, arithmetic is allowed only
// between equal units, there is no conversion and no rounding, overflow fails
// explicitly, and an absent amount is represented by Optional, never by zero.
// Business conversion and rounding are deferred to OPD-06.
package money

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
)

// Unit is the explicit monetary unit. The zero value is invalid.
type Unit uint8

const (
	unitInvalid Unit = iota
	// IRR is the Iranian rial.
	IRR
	// IRT is the Iranian toman. It is a distinct unit; no ratio to IRR is defined here.
	IRT
)

var (
	// ErrInvalidUnit reports a missing or unsupported unit.
	ErrInvalidUnit = errors.New("money: invalid unit")
	// ErrUnitMismatch reports an operation between different units.
	ErrUnitMismatch = errors.New("money: unit mismatch")
	// ErrOverflow reports that a result does not fit in int64.
	ErrOverflow = errors.New("money: int64 overflow")
	// ErrInvalidAmount reports an amount representation that is not a plain decimal integer.
	ErrInvalidAmount = errors.New("money: invalid amount")
)

// String returns the stable unit code.
func (u Unit) String() string {
	switch u {
	case IRR:
		return "IRR"
	case IRT:
		return "IRT"
	default:
		return "INVALID"
	}
}

// Valid reports whether u is IRR or IRT.
func (u Unit) Valid() bool { return u == IRR || u == IRT }

// ParseUnit parses an exact unit code ("IRR" or "IRT").
func ParseUnit(s string) (Unit, error) {
	switch s {
	case "IRR":
		return IRR, nil
	case "IRT":
		return IRT, nil
	default:
		return unitInvalid, fmt.Errorf("%w: %q", ErrInvalidUnit, s)
	}
}

// MarshalText implements encoding.TextMarshaler.
func (u Unit) MarshalText() ([]byte, error) {
	if !u.Valid() {
		return nil, ErrInvalidUnit
	}
	return []byte(u.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (u *Unit) UnmarshalText(b []byte) error {
	parsed, err := ParseUnit(string(b))
	if err != nil {
		return err
	}
	*u = parsed
	return nil
}

// Money is an exact signed amount in an explicit unit.
// The zero value has an invalid unit and is rejected by every operation.
type Money struct {
	Amount int64
	Unit   Unit
}

// New returns Money with a validated unit.
func New(amount int64, unit Unit) (Money, error) {
	if !unit.Valid() {
		return Money{}, ErrInvalidUnit
	}
	return Money{Amount: amount, Unit: unit}, nil
}

// Validate reports whether m carries a valid unit.
func (m Money) Validate() error {
	if !m.Unit.Valid() {
		return ErrInvalidUnit
	}
	return nil
}

// IsZero reports whether the amount is zero. A zero amount is still present money.
func (m Money) IsZero() bool { return m.Amount == 0 }

// Sign returns -1, 0 or +1.
func (m Money) Sign() int {
	switch {
	case m.Amount < 0:
		return -1
	case m.Amount > 0:
		return 1
	default:
		return 0
	}
}

func sameUnit(a, b Money) error {
	if err := a.Validate(); err != nil {
		return err
	}
	if err := b.Validate(); err != nil {
		return err
	}
	if a.Unit != b.Unit {
		return fmt.Errorf("%w: %s vs %s", ErrUnitMismatch, a.Unit, b.Unit)
	}
	return nil
}

// Add returns a+b. Units must match; overflow is an error.
func Add(a, b Money) (Money, error) {
	if err := sameUnit(a, b); err != nil {
		return Money{}, err
	}
	if (b.Amount > 0 && a.Amount > math.MaxInt64-b.Amount) || (b.Amount < 0 && a.Amount < math.MinInt64-b.Amount) {
		return Money{}, ErrOverflow
	}
	return Money{Amount: a.Amount + b.Amount, Unit: a.Unit}, nil
}

// Sub returns a-b. Units must match; overflow is an error.
func Sub(a, b Money) (Money, error) {
	if err := sameUnit(a, b); err != nil {
		return Money{}, err
	}
	if (b.Amount < 0 && a.Amount > math.MaxInt64+b.Amount) || (b.Amount > 0 && a.Amount < math.MinInt64+b.Amount) {
		return Money{}, ErrOverflow
	}
	return Money{Amount: a.Amount - b.Amount, Unit: a.Unit}, nil
}

// Neg returns -m. Negating math.MinInt64 is an overflow.
func Neg(m Money) (Money, error) {
	if err := m.Validate(); err != nil {
		return Money{}, err
	}
	if m.Amount == math.MinInt64 {
		return Money{}, ErrOverflow
	}
	return Money{Amount: -m.Amount, Unit: m.Unit}, nil
}

// Sum adds values that all share one unit. An empty sum is an error because
// the unit of nothing is unknown.
func Sum(values ...Money) (Money, error) {
	if len(values) == 0 {
		return Money{}, ErrInvalidUnit
	}
	total := values[0]
	if err := total.Validate(); err != nil {
		return Money{}, err
	}
	for _, v := range values[1:] {
		next, err := Add(total, v)
		if err != nil {
			return Money{}, err
		}
		total = next
	}
	return total, nil
}

// Compare returns -1, 0 or +1. Units must match.
func Compare(a, b Money) (int, error) {
	if err := sameUnit(a, b); err != nil {
		return 0, err
	}
	switch {
	case a.Amount < b.Amount:
		return -1, nil
	case a.Amount > b.Amount:
		return 1, nil
	default:
		return 0, nil
	}
}

// Equal reports whether a and b have the same unit and amount.
func Equal(a, b Money) bool { return a.Unit == b.Unit && a.Amount == b.Amount && a.Unit.Valid() }

// String renders "<amount> <unit>" for diagnostics.
func (m Money) String() string { return strconv.FormatInt(m.Amount, 10) + " " + m.Unit.String() }

// ParseAmount parses a plain base-10 integer: optional leading '-', digits only,
// no '+', no leading zeros, no fraction, exponent, separators or whitespace.
func ParseAmount(s string) (int64, error) {
	if !plainInteger(s) {
		return 0, fmt.Errorf("%w: %q", ErrInvalidAmount, s)
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %q", ErrOverflow, s)
	}
	return v, nil
}

func plainInteger(s string) bool {
	digits := s
	if len(digits) > 0 && digits[0] == '-' {
		digits = digits[1:]
	}
	if digits == "" {
		return false
	}
	if len(digits) > 1 && digits[0] == '0' {
		return false
	}
	if s == "-0" {
		return false
	}
	for i := 0; i < len(digits); i++ {
		if digits[i] < '0' || digits[i] > '9' {
			return false
		}
	}
	return true
}

type wireMoney struct {
	Amount *string `json:"amount"`
	Unit   *string `json:"unit"`
}

// MarshalJSON renders {"amount":"<int64>","unit":"IRR|IRT"}. The amount is a
// string so that no consumer reads it as a floating-point number.
func (m Money) MarshalJSON() ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	amount := strconv.FormatInt(m.Amount, 10)
	unit := m.Unit.String()
	return json.Marshal(wireMoney{Amount: &amount, Unit: &unit})
}

// UnmarshalJSON accepts only the MarshalJSON shape; both fields are required.
func (m *Money) UnmarshalJSON(b []byte) error {
	var w wireMoney
	if err := json.Unmarshal(b, &w); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidAmount, err)
	}
	if w.Amount == nil {
		return fmt.Errorf("%w: missing amount", ErrInvalidAmount)
	}
	if w.Unit == nil {
		return fmt.Errorf("%w: missing unit", ErrInvalidUnit)
	}
	amount, err := ParseAmount(*w.Amount)
	if err != nil {
		return err
	}
	unit, err := ParseUnit(*w.Unit)
	if err != nil {
		return err
	}
	*m = Money{Amount: amount, Unit: unit}
	return nil
}

// Optional distinguishes absent money from present money (including zero).
// The zero value is absent.
type Optional struct {
	value   Money
	present bool
}

// Some returns a present Optional. The money must be valid.
func Some(m Money) (Optional, error) {
	if err := m.Validate(); err != nil {
		return Optional{}, err
	}
	return Optional{value: m, present: true}, nil
}

// None returns an absent Optional.
func None() Optional { return Optional{} }

// Present reports whether a value is present.
func (o Optional) Present() bool { return o.present }

// Get returns the value and whether it is present.
func (o Optional) Get() (Money, bool) { return o.value, o.present }

// MarshalJSON renders null when absent.
func (o Optional) MarshalJSON() ([]byte, error) {
	if !o.present {
		return []byte("null"), nil
	}
	return o.value.MarshalJSON()
}

// UnmarshalJSON maps null to absent and anything else to present Money.
func (o *Optional) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*o = Optional{}
		return nil
	}
	var m Money
	if err := m.UnmarshalJSON(b); err != nil {
		return err
	}
	*o = Optional{value: m, present: true}
	return nil
}
