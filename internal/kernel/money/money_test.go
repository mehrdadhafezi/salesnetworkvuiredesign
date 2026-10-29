package money

import (
	"encoding/json"
	"errors"
	"math"
	"math/big"
	"math/rand"
	"testing"
	"testing/quick"
)

func mustNew(t *testing.T, amount int64, unit Unit) Money {
	t.Helper()
	m, err := New(amount, unit)
	if err != nil {
		t.Fatalf("New(%d, %v): %v", amount, unit, err)
	}
	return m
}

func TestUnitParseAndText(t *testing.T) {
	for _, u := range []Unit{IRR, IRT} {
		got, err := ParseUnit(u.String())
		if err != nil || got != u {
			t.Fatalf("ParseUnit(%q) = %v, %v", u.String(), got, err)
		}
		b, err := u.MarshalText()
		if err != nil || string(b) != u.String() {
			t.Fatalf("MarshalText(%v) = %q, %v", u, b, err)
		}
		var back Unit
		if err := back.UnmarshalText(b); err != nil || back != u {
			t.Fatalf("UnmarshalText(%q) = %v, %v", b, back, err)
		}
	}
	for _, bad := range []string{"", "irr", "Irt", "IRR ", "TOMAN", "RIAL", "USD"} {
		if _, err := ParseUnit(bad); !errors.Is(err, ErrInvalidUnit) {
			t.Fatalf("ParseUnit(%q) err = %v, want ErrInvalidUnit", bad, err)
		}
	}
	var zero Unit
	if zero.Valid() || zero.String() != "INVALID" {
		t.Fatalf("zero Unit must be invalid")
	}
	if _, err := zero.MarshalText(); !errors.Is(err, ErrInvalidUnit) {
		t.Fatalf("zero Unit MarshalText err = %v", err)
	}
	var u Unit
	if err := u.UnmarshalText([]byte("XYZ")); !errors.Is(err, ErrInvalidUnit) {
		t.Fatalf("UnmarshalText(XYZ) err = %v", err)
	}
}

func TestUnitRequired(t *testing.T) {
	if _, err := New(1, Unit(0)); !errors.Is(err, ErrInvalidUnit) {
		t.Fatalf("New with zero unit err = %v", err)
	}
	if _, err := New(1, Unit(9)); !errors.Is(err, ErrInvalidUnit) {
		t.Fatalf("New with unknown unit err = %v", err)
	}
	var zero Money
	if err := zero.Validate(); !errors.Is(err, ErrInvalidUnit) {
		t.Fatalf("zero Money must be invalid, got %v", err)
	}
	ok := mustNew(t, 1, IRR)
	if _, err := Add(zero, ok); !errors.Is(err, ErrInvalidUnit) {
		t.Fatalf("Add(zero, ok) err = %v", err)
	}
	if _, err := Add(ok, zero); !errors.Is(err, ErrInvalidUnit) {
		t.Fatalf("Add(ok, zero) err = %v", err)
	}
	if _, err := Neg(zero); !errors.Is(err, ErrInvalidUnit) {
		t.Fatalf("Neg(zero) err = %v", err)
	}
	if _, err := Compare(zero, ok); !errors.Is(err, ErrInvalidUnit) {
		t.Fatalf("Compare(zero, ok) err = %v", err)
	}
	if _, err := zero.MarshalJSON(); !errors.Is(err, ErrInvalidUnit) {
		t.Fatalf("zero MarshalJSON err = %v", err)
	}
	if Equal(zero, zero) {
		t.Fatalf("zero Money must not be Equal to itself")
	}
}

func TestSameUnitArithmetic(t *testing.T) {
	cases := []struct {
		unit Unit
		a, b int64
	}{{IRR, 1500, 250}, {IRT, 150, 25}, {IRR, -10, 3}, {IRT, 0, 0}}
	for _, c := range cases {
		a, b := mustNew(t, c.a, c.unit), mustNew(t, c.b, c.unit)
		sum, err := Add(a, b)
		if err != nil || sum.Amount != c.a+c.b || sum.Unit != c.unit {
			t.Fatalf("Add(%v,%v) = %v, %v", a, b, sum, err)
		}
		diff, err := Sub(a, b)
		if err != nil || diff.Amount != c.a-c.b || diff.Unit != c.unit {
			t.Fatalf("Sub(%v,%v) = %v, %v", a, b, diff, err)
		}
	}
}

func TestMismatchedUnitsRejected(t *testing.T) {
	r, tm := mustNew(t, 10, IRR), mustNew(t, 1, IRT)
	if _, err := Add(r, tm); !errors.Is(err, ErrUnitMismatch) {
		t.Fatalf("Add(IRR, IRT) err = %v", err)
	}
	if _, err := Sub(tm, r); !errors.Is(err, ErrUnitMismatch) {
		t.Fatalf("Sub(IRT, IRR) err = %v", err)
	}
	if _, err := Compare(r, tm); !errors.Is(err, ErrUnitMismatch) {
		t.Fatalf("Compare(IRR, IRT) err = %v", err)
	}
	if _, err := Sum(r, tm); !errors.Is(err, ErrUnitMismatch) {
		t.Fatalf("Sum(IRR, IRT) err = %v", err)
	}
	// The same amount in different units is never equal: no implicit ratio.
	if Equal(mustNew(t, 10, IRR), mustNew(t, 10, IRT)) {
		t.Fatalf("10 IRR must not equal 10 IRT")
	}
	if Equal(mustNew(t, 10, IRR), mustNew(t, 1, IRT)) {
		t.Fatalf("10 IRR must not equal 1 IRT: the kernel defines no conversion")
	}
}

func TestOverflowAndUnderflow(t *testing.T) {
	maxM, minM := mustNew(t, math.MaxInt64, IRR), mustNew(t, math.MinInt64, IRR)
	one, minusOne := mustNew(t, 1, IRR), mustNew(t, -1, IRR)
	checks := []struct {
		name string
		fn   func() (Money, error)
	}{
		{"max+1", func() (Money, error) { return Add(maxM, one) }},
		{"min-1 via add", func() (Money, error) { return Add(minM, minusOne) }},
		{"min-1 via sub", func() (Money, error) { return Sub(minM, one) }},
		{"max-(-1)", func() (Money, error) { return Sub(maxM, minusOne) }},
		{"neg(min)", func() (Money, error) { return Neg(minM) }},
		{"sum overflow", func() (Money, error) { return Sum(maxM, one) }},
	}
	for _, c := range checks {
		if _, err := c.fn(); !errors.Is(err, ErrOverflow) {
			t.Fatalf("%s: err = %v, want ErrOverflow", c.name, err)
		}
	}
	if got, err := Add(maxM, minusOne); err != nil || got.Amount != math.MaxInt64-1 {
		t.Fatalf("max+(-1) = %v, %v", got, err)
	}
	if got, err := Neg(maxM); err != nil || got.Amount != -math.MaxInt64 {
		t.Fatalf("neg(max) = %v, %v", got, err)
	}
}

func TestZeroAndNegative(t *testing.T) {
	zero := mustNew(t, 0, IRT)
	if !zero.IsZero() || zero.Sign() != 0 || zero.Validate() != nil {
		t.Fatalf("zero money must be valid and zero")
	}
	neg := mustNew(t, -5, IRT)
	if neg.Sign() != -1 || neg.IsZero() {
		t.Fatalf("negative money sign")
	}
	if mustNew(t, 7, IRT).Sign() != 1 {
		t.Fatalf("positive money sign")
	}
	c, err := Compare(neg, zero)
	if err != nil || c != -1 {
		t.Fatalf("Compare(-5, 0) = %d, %v", c, err)
	}
	if c, _ := Compare(zero, neg); c != 1 {
		t.Fatalf("Compare(0, -5) = %d", c)
	}
	if c, _ := Compare(zero, zero); c != 0 {
		t.Fatalf("Compare(0, 0) = %d", c)
	}
	n, err := Neg(neg)
	if err != nil || n.Amount != 5 {
		t.Fatalf("Neg(-5) = %v, %v", n, err)
	}
	if neg.String() != "-5 IRT" {
		t.Fatalf("String = %q", neg.String())
	}
}

func TestSum(t *testing.T) {
	if _, err := Sum(); !errors.Is(err, ErrInvalidUnit) {
		t.Fatalf("empty Sum err = %v", err)
	}
	if _, err := Sum(Money{}); !errors.Is(err, ErrInvalidUnit) {
		t.Fatalf("Sum(invalid) err = %v", err)
	}
	got, err := Sum(mustNew(t, 1, IRR), mustNew(t, 2, IRR), mustNew(t, 3, IRR))
	if err != nil || !Equal(got, mustNew(t, 6, IRR)) {
		t.Fatalf("Sum = %v, %v", got, err)
	}
}

func TestOptionalAbsentIsNotZero(t *testing.T) {
	absent := None()
	if absent.Present() {
		t.Fatalf("None must be absent")
	}
	var zeroValue Optional
	if zeroValue.Present() {
		t.Fatalf("zero Optional must be absent")
	}
	present, err := Some(mustNew(t, 0, IRR))
	if err != nil || !present.Present() {
		t.Fatalf("Some(0 IRR) must be present: %v", err)
	}
	if m, ok := present.Get(); !ok || !m.IsZero() {
		t.Fatalf("Get = %v, %v", m, ok)
	}
	if _, ok := absent.Get(); ok {
		t.Fatalf("absent Get must report false")
	}
	if _, err := Some(Money{}); !errors.Is(err, ErrInvalidUnit) {
		t.Fatalf("Some(invalid) err = %v", err)
	}
	b, _ := json.Marshal(absent)
	if string(b) != "null" {
		t.Fatalf("absent JSON = %s", b)
	}
	b, _ = json.Marshal(present)
	if string(b) != `{"amount":"0","unit":"IRR"}` {
		t.Fatalf("present zero JSON = %s", b)
	}
	var back Optional
	if err := json.Unmarshal([]byte("null"), &back); err != nil || back.Present() {
		t.Fatalf("null must decode to absent: %v", err)
	}
	if err := json.Unmarshal([]byte(`{"amount":"0","unit":"IRR"}`), &back); err != nil || !back.Present() {
		t.Fatalf("zero must decode to present: %v", err)
	}
	if err := json.Unmarshal([]byte(`{"amount":"x","unit":"IRR"}`), &back); err == nil {
		t.Fatalf("invalid optional must fail")
	}
}

func TestParseAmount(t *testing.T) {
	good := map[string]int64{"0": 0, "7": 7, "-7": -7, "9223372036854775807": math.MaxInt64, "-9223372036854775808": math.MinInt64}
	for s, want := range good {
		got, err := ParseAmount(s)
		if err != nil || got != want {
			t.Fatalf("ParseAmount(%q) = %d, %v", s, got, err)
		}
	}
	for _, s := range []string{"", "-", "-0", "+1", "01", "1.0", "1e3", "1,000", " 1", "1 ", "۱۲", "0x10", "1_000", "NaN"} {
		if _, err := ParseAmount(s); !errors.Is(err, ErrInvalidAmount) {
			t.Fatalf("ParseAmount(%q) err = %v, want ErrInvalidAmount", s, err)
		}
	}
	for _, s := range []string{"9223372036854775808", "-9223372036854775809"} {
		if _, err := ParseAmount(s); !errors.Is(err, ErrOverflow) {
			t.Fatalf("ParseAmount(%q) err = %v, want ErrOverflow", s, err)
		}
	}
}

func TestJSON(t *testing.T) {
	m := mustNew(t, -9223372036854775808, IRT)
	b, err := json.Marshal(m)
	if err != nil || string(b) != `{"amount":"-9223372036854775808","unit":"IRT"}` {
		t.Fatalf("Marshal = %s, %v", b, err)
	}
	var back Money
	if err := json.Unmarshal(b, &back); err != nil || !Equal(back, m) {
		t.Fatalf("round trip = %v, %v", back, err)
	}
	bad := []string{
		`{"amount":"1"}`, `{"unit":"IRR"}`, `{"amount":1,"unit":"IRR"}`, `{"amount":"1.5","unit":"IRR"}`,
		`{"amount":"1","unit":"irr"}`, `{"amount":"1","unit":""}`, `[]`, `"1 IRR"`, `{"amount":"1e2","unit":"IRT"}`,
	}
	for _, s := range bad {
		var x Money
		if err := json.Unmarshal([]byte(s), &x); err == nil {
			t.Fatalf("Unmarshal(%s) must fail", s)
		}
	}
}

var quickConfig = &quick.Config{MaxCount: 2000, Rand: rand.New(rand.NewSource(20261029))}

// Property: Add either matches exact big-integer arithmetic or reports overflow.
func TestPropertyAddExact(t *testing.T) {
	f := func(a, b int64, irt bool) bool {
		unit := IRR
		if irt {
			unit = IRT
		}
		got, err := Add(Money{a, unit}, Money{b, unit})
		exact := new(big.Int).Add(big.NewInt(a), big.NewInt(b))
		if !exact.IsInt64() {
			return errors.Is(err, ErrOverflow)
		}
		return err == nil && got.Amount == exact.Int64() && got.Unit == unit
	}
	if err := quick.Check(f, quickConfig); err != nil {
		t.Fatal(err)
	}
}

// Property: Sub either matches exact big-integer arithmetic or reports overflow.
func TestPropertySubExact(t *testing.T) {
	f := func(a, b int64) bool {
		got, err := Sub(Money{a, IRT}, Money{b, IRT})
		exact := new(big.Int).Sub(big.NewInt(a), big.NewInt(b))
		if !exact.IsInt64() {
			return errors.Is(err, ErrOverflow)
		}
		return err == nil && got.Amount == exact.Int64()
	}
	if err := quick.Check(f, quickConfig); err != nil {
		t.Fatal(err)
	}
}

// Property: Add is commutative and cross-unit operations always fail.
func TestPropertyCommutativeAndUnitIsolation(t *testing.T) {
	f := func(a, b int64) bool {
		x, errX := Add(Money{a, IRR}, Money{b, IRR})
		y, errY := Add(Money{b, IRR}, Money{a, IRR})
		if (errX == nil) != (errY == nil) || (errX == nil && x != y) {
			return false
		}
		_, mixed := Add(Money{a, IRR}, Money{b, IRT})
		return errors.Is(mixed, ErrUnitMismatch)
	}
	if err := quick.Check(f, quickConfig); err != nil {
		t.Fatal(err)
	}
}

// Property: JSON round-trips exactly at every int64 value.
func TestPropertyJSONRoundTrip(t *testing.T) {
	f := func(a int64) bool {
		m := Money{a, IRT}
		b, err := json.Marshal(m)
		if err != nil {
			return false
		}
		var back Money
		return json.Unmarshal(b, &back) == nil && Equal(back, m)
	}
	if err := quick.Check(f, quickConfig); err != nil {
		t.Fatal(err)
	}
}

func FuzzParseAmount(f *testing.F) {
	for _, s := range []string{"0", "-1", "9223372036854775807", "-9223372036854775808", "01", "1.0", "+1", ""} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		v, err := ParseAmount(s)
		if err != nil {
			return
		}
		// Accepted input must be canonical: re-rendering yields the same text.
		if got := (Money{v, IRR}).String(); got != s+" IRR" {
			t.Fatalf("ParseAmount(%q) accepted non-canonical input -> %q", s, got)
		}
	})
}
