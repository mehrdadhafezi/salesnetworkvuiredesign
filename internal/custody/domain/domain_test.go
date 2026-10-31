package domain

import (
	"strconv"
	"testing"
)

func s(v string) *string { return &v }

func e(n int, kind string, from, to *string, at string) Event {
	var t *string
	if at != "" {
		t = &at
	}
	return Event{ID: "e" + strconv.Itoa(n), Ordinal: strconv.Itoa(n), Kind: kind, Action: "synthetic_" + kind, From: from, To: to, OccurredRaw: t}
}

func kinds(r Result) map[string]int {
	m := map[string]int{}
	for _, f := range r.Findings {
		m[f.Kind]++
	}
	return m
}

// Given ordered events, the expected current holder is derived; no mutable
// shortcut is used. Insertion order, not slice order, decides.
func TestDeriveOrderedHolder(t *testing.T) {
	r := Derive([]Event{
		e(3, HolderAssigned, s("20"), s("30"), "2026-01-03 10:00:00"),
		e(1, HolderAssigned, nil, s("10"), "2026-01-01 10:00:00"),
		e(2, HolderAssigned, s("10"), s("20"), "2026-01-02 10:00:00"),
		e(4, NoHolderChange, s("30"), s("30"), "2026-01-04 10:00:00"),
	})
	if r.State != Derived || *r.Holder != "30" || len(r.Findings) != 0 || r.LastID != "e4" {
		t.Fatalf("derive = %+v", r)
	}
	if Classify(r, s("30")) != Match || Classify(r, s("99")) != Conflict || Classify(r, nil) != LegacyUnknown {
		t.Fatal("classification")
	}
}

func TestReleaseAndGaps(t *testing.T) {
	r := Derive([]Event{e(1, HolderAssigned, nil, s("10"), ""), e(2, HolderReleased, s("10"), nil, "")})
	if r.State != Released || Classify(r, nil) != Match || Classify(r, s("10")) != Conflict {
		t.Fatalf("release = %+v", r)
	}
	// A transition whose prior holder is not the derived one: HISTORY_GAP,
	// never a synthesised missing event.
	g := Derive([]Event{e(1, HolderAssigned, nil, s("10"), ""), e(2, HolderAssigned, s("15"), s("20"), "")})
	if kinds(g)[FindHistoryGap] != 1 || *g.Holder != "20" || Classify(g, s("20")) != HistoryGap {
		t.Fatalf("gap = %+v", g)
	}
	// History that starts mid-stream.
	m := Derive([]Event{e(5, NoHolderChange, s("40"), s("40"), "")})
	if m.State != None || kinds(m)[FindHistoryGap] != 1 || Classify(m, s("40")) != TargetUnknown {
		t.Fatalf("mid-stream = %+v", m)
	}
}

// Contradictory equally-timed holders are MULTIPLE_CURRENT; never picked.
func TestMultipleCurrent(t *testing.T) {
	r := Derive([]Event{
		e(1, HolderAssigned, nil, s("10"), "2026-01-01 09:00:00"),
		e(2, HolderAssigned, s("10"), s("20"), "2026-01-02 10:00:00"),
		e(3, HolderAssigned, s("10"), s("30"), "2026-01-02 10:00:00"),
	})
	if r.State != Multiple || r.Holder != nil || Classify(r, s("30")) != MultipleCurrent || kinds(r)[FindMultipleCurrent] != 1 {
		t.Fatalf("multiple = %+v", r)
	}
}

func TestUnknownActionAndMissingHistory(t *testing.T) {
	r := Derive([]Event{e(1, HolderAssigned, nil, s("10"), ""), e(2, KindUnknown, s("10"), s("11"), "")})
	if r.State != Unknown || Classify(r, s("11")) != TargetUnknown || Classify(r, nil) != NotRecoverable {
		t.Fatalf("unknown = %+v", r)
	}
	n := Derive(nil)
	if n.State != None || kinds(n)[FindMissingHistory] != 1 || Classify(n, nil) != NotRecoverable || Classify(n, s("1")) != TargetUnknown {
		t.Fatalf("none = %+v", n)
	}
	// Assignment without a target holder: UNKNOWN, not a guess.
	m := Derive([]Event{e(1, HolderAssigned, nil, nil, "")})
	if m.State != Unknown || kinds(m)[FindMissingTargetOwner] != 1 {
		t.Fatalf("missing target = %+v", m)
	}
}

func TestOrderingContradictionIsReported(t *testing.T) {
	r := Derive([]Event{
		e(1, HolderAssigned, nil, s("10"), "2026-02-01 10:00:00"),
		e(2, HolderAssigned, s("10"), s("20"), "2026-01-01 10:00:00"),
	})
	if kinds(r)[FindOrdering] != 1 || Classify(r, s("20")) != HistoryGap {
		t.Fatalf("ordering = %+v", r)
	}
	// Zero and unreadable dates never order anything.
	z := Derive([]Event{e(1, HolderAssigned, nil, s("10"), "0000-00-00 00:00:00"), e(2, HolderAssigned, s("10"), s("20"), "garbage")})
	if len(z.Findings) != 0 || Classify(z, s("20")) != Match {
		t.Fatalf("zero dates = %+v", z)
	}
}
