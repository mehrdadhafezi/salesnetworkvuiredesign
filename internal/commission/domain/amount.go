// Package domain is the commission module's PURE, deterministic core: exact
// amounts, the typed registry of evidenced calculation functions, the
// entitlement identity, attribution facets, the run lifecycle, the
// non-dispatching posting contract and parity classification. It performs
// no I/O, reads no clock, never uses floating point and never reads the
// current hierarchy: every input is an immutable snapshot passed in.
package domain

import (
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/money"
)

// AmountScale is the fixed number of decimal places an Exact amount keeps.
// Legacy commission amounts are DECIMAL(18,2) (class-sn-plugin.php:35554);
// the kernel Money contract has no fraction, so commission carries its own
// exact scaled integer instead of rounding into Money.
const AmountScale = 2

// RateScale is the fixed number of decimal places of a percent rate.
// Legacy rates are DECIMAL(7,4) / DECIMAL(10,4) percent.
const RateScale = 4

var (
	// ErrNotExact reports a decimal text that is not an exact representable value.
	ErrNotExact = errors.New("commission: value is not an exact decimal at the fixed scale")
	// ErrOverflow reports an int64 overflow.
	ErrOverflow = errors.New("commission: int64 overflow")
	// ErrUnitMismatch reports arithmetic across units.
	ErrUnitMismatch = errors.New("commission: unit mismatch")
)

// Exact is an exact signed amount in hundredths of an explicit unit.
type Exact struct {
	Hundredths int64
	Unit       money.Unit
}

// Rate is an exact percent rate in ten-thousandths of a percent
// (12.5% = 125000).
type Rate struct{ E4 int64 }

// ParseDecimal parses plain decimal text ("1234", "-12.5", "0.0100") into an
// integer scaled by 10^scale. More fractional digits than scale are accepted
// only when they are zeros; anything else is ErrNotExact (never rounded).
func ParseDecimal(s string, scale int) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, ErrNotExact
	}
	neg := false
	switch s[0] {
	case '-':
		neg, s = true, s[1:]
	case '+':
		s = s[1:]
	}
	intPart, frac, _ := strings.Cut(s, ".")
	if intPart == "" && frac == "" {
		return 0, ErrNotExact
	}
	for _, r := range intPart + frac {
		if r < '0' || r > '9' {
			return 0, ErrNotExact
		}
	}
	if len(frac) > scale {
		if strings.Trim(frac[scale:], "0") != "" {
			return 0, ErrNotExact
		}
		frac = frac[:scale]
	}
	frac += strings.Repeat("0", scale-len(frac))
	if intPart == "" {
		intPart = "0"
	}
	v, ok := new(big.Int).SetString(intPart+frac, 10)
	if !ok {
		return 0, ErrNotExact
	}
	if neg {
		v.Neg(v)
	}
	if !v.IsInt64() {
		return 0, ErrOverflow
	}
	return v.Int64(), nil
}

// FormatDecimal renders a scaled integer as plain decimal text.
func FormatDecimal(v int64, scale int) string {
	b := new(big.Int).SetInt64(v)
	neg := b.Sign() < 0
	b.Abs(b)
	s := b.String()
	if len(s) <= scale {
		s = strings.Repeat("0", scale-len(s)+1) + s
	}
	out := s[:len(s)-scale]
	if scale > 0 {
		out += "." + s[len(s)-scale:]
	}
	if neg {
		out = "-" + out
	}
	return out
}

// ParseExact parses decimal text in an explicit unit.
func ParseExact(raw string, unit money.Unit) (Exact, error) {
	if !unit.Valid() {
		return Exact{}, money.ErrInvalidUnit
	}
	v, err := ParseDecimal(raw, AmountScale)
	if err != nil {
		return Exact{}, err
	}
	return Exact{Hundredths: v, Unit: unit}, nil
}

// ParseRate parses a percent rate as decimal text.
func ParseRate(raw string) (Rate, error) {
	v, err := ParseDecimal(raw, RateScale)
	if err != nil {
		return Rate{}, err
	}
	return Rate{E4: v}, nil
}

// String renders the amount as "<decimal> <unit>".
func (e Exact) String() string {
	return FormatDecimal(e.Hundredths, AmountScale) + " " + e.Unit.String()
}

// Decimal renders the amount's decimal text only.
func (e Exact) Decimal() string { return FormatDecimal(e.Hundredths, AmountScale) }

// String renders the rate as decimal percent text.
func (r Rate) String() string { return FormatDecimal(r.E4, RateScale) }

// AddExact adds two exact amounts of the same unit.
func AddExact(a, b Exact) (Exact, error) {
	if !a.Unit.Valid() || !b.Unit.Valid() {
		return Exact{}, money.ErrInvalidUnit
	}
	if a.Unit != b.Unit {
		return Exact{}, fmt.Errorf("%w: %s vs %s", ErrUnitMismatch, a.Unit, b.Unit)
	}
	s := new(big.Int).Add(big.NewInt(a.Hundredths), big.NewInt(b.Hundredths))
	if !s.IsInt64() {
		return Exact{}, ErrOverflow
	}
	return Exact{Hundredths: s.Int64(), Unit: a.Unit}, nil
}

// Rounding names a rounding behaviour. There is NO default: a calculation
// without an explicit rounding is BLOCKED (OPD-06).
type Rounding string

const (
	// RoundingNone means "no rounding policy given" and blocks every
	// calculation that needs to round.
	RoundingNone Rounding = ""
	// RoundingLegacyHalfAwayTwoDP reproduces the legacy writers'
	// round(x, 2) (PHP: half away from zero) on the EXACT product. It is
	// evidenced legacy behaviour (class-sn-purpose-commission.php:606,
	// class-sn-plugin.php:32905-32912) usable for PARITY only; it is not an
	// approved target rounding (OPD-06 open). PHP evaluates on binary
	// floats, so a half-cent tie can differ: such a row surfaces as
	// AMOUNT_MISMATCH, never silently adjusted.
	RoundingLegacyHalfAwayTwoDP Rounding = "LEGACY_PHP_ROUND_HALF_AWAY_2DP"
)

// Known reports whether r is a registered rounding.
func (r Rounding) Known() bool { return r == RoundingLegacyHalfAwayTwoDP }

// divRound divides n by d (d > 0) rounding half away from zero.
func divRound(n, d *big.Int) *big.Int {
	q, m := new(big.Int).QuoRem(n, d, new(big.Int))
	twice := new(big.Int).Mul(new(big.Int).Abs(m), big.NewInt(2))
	if twice.Cmp(d) >= 0 {
		if n.Sign() < 0 {
			q.Sub(q, big.NewInt(1))
		} else {
			q.Add(q, big.NewInt(1))
		}
	}
	return q
}

// PercentOf returns base * rate% exactly, rounded to AmountScale with r.
// It fails for an unknown rounding (never a silent default) or overflow.
func PercentOf(base Exact, rate Rate, r Rounding) (Exact, error) {
	if !base.Unit.Valid() {
		return Exact{}, money.ErrInvalidUnit
	}
	if !r.Known() {
		return Exact{}, fmt.Errorf("commission: rounding %q is not a registered rounding", string(r))
	}
	// hundredths * E4 / (100 * 10^4) keeps hundredths.
	n := new(big.Int).Mul(big.NewInt(base.Hundredths), big.NewInt(rate.E4))
	q := divRound(n, big.NewInt(1_000_000))
	if !q.IsInt64() {
		return Exact{}, ErrOverflow
	}
	return Exact{Hundredths: q.Int64(), Unit: base.Unit}, nil
}
