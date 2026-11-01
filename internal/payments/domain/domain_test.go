package domain

// Payments mirrors billing's money rules without importing billing (module
// boundary); the same guarantees are proven here independently.

import (
	"math"
	"strconv"
	"testing"
)

func s(v string) *string { return &v }
func i(v int64) *int64   { return &v }

func TestParseAmountNeverRoundsOrConverts(t *testing.T) {
	for raw, want := range map[string]int64{"1500000.00": 1500000, "7": 7, "0.00": 0, "-25.0": -25, "999999999999999999": 999999999999999999} {
		a := ParseAmount(s(raw), "IRT")
		if a.Representation != Representable || a.Money.Amount != want || a.Money.Unit.String() != "IRT" {
			t.Fatalf("%s = %+v", raw, a)
		}
	}
	// Fractions are never rounded; garbage and unknown units are never guessed.
	for _, c := range []struct{ raw, unit string }{{"10.50", "IRT"}, {"1,000", "IRT"}, {"abc", "IRT"}, {"100", "TOMAN"}, {"100", ""}} {
		if a := ParseAmount(s(c.raw), c.unit); a.Representation != Unrepresentable || a.Money != nil {
			t.Fatalf("%v = %+v", c, a)
		}
	}
	if a := ParseAmount(nil, "IRT"); a.Representation != Absent || a.Money != nil {
		t.Fatal("NULL must be ABSENT, never zero")
	}
	// The same digits in IRR and IRT stay distinct units: no *10 or /10.
	if a, b := ParseAmount(s("100"), "IRR"), ParseAmount(s("100"), "IRT"); a.Money.Amount != 100 || b.Money.Amount != 100 || a.Money.Unit == b.Money.Unit {
		t.Fatal("units must not be converted")
	}
}

func TestSumByUnitSeparatesUnitsAndChecksOverflow(t *testing.T) {
	sums := SumByUnit([]Amount{ParseAmount(s("100"), "IRT"), ParseAmount(s("50"), "IRT"), ParseAmount(s("1000"), "IRR")}, true)
	if *sums["IRT"].Amount != 150 || *sums["IRR"].Amount != 1000 || sums["IRT"].State != SumComplete || sums["IRR"].State != SumComplete {
		t.Fatalf("sums = %+v", sums)
	}
	inc := SumByUnit([]Amount{ParseAmount(s("100"), "IRT"), ParseAmount(s("1.5"), "IRT")}, true)
	if inc["IRT"].State != SumIncomplete {
		t.Fatalf("unrepresentable line must make the sum INCOMPLETE: %+v", inc)
	}
	if SumByUnit([]Amount{ParseAmount(s("1"), "IRT")}, false)["IRT"].State != SumIncomplete {
		t.Fatal("incomplete source must make the sum INCOMPLETE")
	}
	big := strconv.FormatInt(math.MaxInt64, 10)
	over := SumByUnit([]Amount{ParseAmount(s(big[:18]), "IRT"), ParseAmount(s(big[:18]), "IRT"), ParseAmount(s(big[:18]), "IRT"),
		ParseAmount(s(big[:18]), "IRT"), ParseAmount(s(big[:18]), "IRT"), ParseAmount(s(big[:18]), "IRT"), ParseAmount(s(big[:18]), "IRT"),
		ParseAmount(s(big[:18]), "IRT"), ParseAmount(s(big[:18]), "IRT"), ParseAmount(s(big[:18]), "IRT"), ParseAmount(s(big[:18]), "IRT")}, true)
	if over["IRT"].State != SumOverflow || over["IRT"].Amount != nil {
		t.Fatalf("overflow = %+v", over)
	}
	if len(SumByUnit(nil, true)) != 0 {
		t.Fatal("no rows is no sum (absent is not zero)")
	}
}

