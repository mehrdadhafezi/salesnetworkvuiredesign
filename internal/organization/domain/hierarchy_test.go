package domain

import "testing"

func s(v string) *string { return &v }
func b(v bool) *bool     { return &v }

func kinds(fs []Finding) map[string]int {
	m := map[string]int{}
	for _, f := range fs {
		m[f.Kind]++
	}
	return m
}

func TestLegacyDates(t *testing.T) {
	if _, ok := LegacyDateTime(s("0000-00-00 00:00:00")); ok {
		t.Fatal("zero datetime read")
	}
	if _, ok := LegacyDateTime(s("2026-01-02T03:04:05Z")); ok {
		t.Fatal("non-legacy format read")
	}
	if v, ok := LegacyDateTime(s("2026-01-02 03:04:05")); !ok || v.Hour() != 3 {
		t.Fatal("valid datetime not read")
	}
	if _, ok := LegacyDate(s("0000-00-00")); ok {
		t.Fatal("zero date read")
	}
	if _, ok := LegacyDate(nil); ok {
		t.Fatal("nil date read")
	}
}

func TestAnalyzeIsDeterministicAndHonest(t *testing.T) {
	in := Input{
		Assignments: []AssignmentNode{
			{ID: "a1", Key: "1", ChildID: s("e1"), ChildKey: "10", ParentID: s("e2"), ParentKey: s("20"), ParentState: RefResolved, Kind: "REPORTS_TO", EffectiveFromRaw: s("2026-01-01 00:00:00"), IsCurrent: b(true)},
			{ID: "a2", Key: "2", ChildID: s("e2"), ChildKey: "20", ParentID: s("e1"), ParentKey: s("10"), ParentState: RefResolved, Kind: "REPORTS_TO", EffectiveFromRaw: s("2026-01-01 00:00:00"), IsCurrent: b(true)},
			{ID: "a3", Key: "3", ChildID: s("e3"), ChildKey: "30", ParentKey: s("99"), ParentState: RefMissing, Kind: "REPORTS_TO", IsCurrent: b(true)},
			{ID: "a4", Key: "4", ChildID: s("e4"), ChildKey: "40", ParentID: s("e4"), ParentKey: s("40"), ParentState: RefResolved, Kind: "REPORTS_TO", IsCurrent: b(false)},
		},
	}
	first := Analyze(in)
	k := kinds(first)
	if k[KindCycle] != 2 || k[KindMissingParent] != 1 || k[KindSelfParent] != 1 || k[KindCurrentFlagInconsistent] != 1 {
		t.Fatalf("findings %v", k)
	}
	again := Analyze(in)
	for i := range first {
		if first[i].Kind != again[i].Kind || first[i].Detail != again[i].Detail {
			t.Fatal("analysis is not deterministic")
		}
	}
	// Nothing is repaired: the input is unchanged.
	if in.Assignments[2].ParentID != nil || in.Assignments[3].ParentID == nil {
		t.Fatal("analysis changed its input")
	}
}

func TestNoFindingsForCleanTree(t *testing.T) {
	in := Input{
		Units:       []UnitNode{{ID: "d", Kind: "DEPARTMENT", Key: "1", ParentState: "NOT_APPLICABLE"}, {ID: "u", Kind: "UNIT", Key: "1", ParentID: s("d"), ParentKey: s("1"), ParentState: RefResolved}},
		Employments: []EmploymentNode{{ID: "e1", Key: "10", UserLink: "LINKED", UserKey: s("1"), Department: RefState{ID: s("d"), State: RefResolved}, Unit: RefState{ID: s("u"), State: RefResolved}}},
		Assignments: []AssignmentNode{
			{ID: "a1", Key: "1", ChildID: s("e1"), ChildKey: "10", ParentID: s("e2"), ParentKey: s("20"), ParentState: RefResolved, Kind: "REPORTS_TO",
				EffectiveFromRaw: s("2025-01-01 00:00:00"), EffectiveToRaw: s("2025-06-01 00:00:00"), IsCurrent: b(false)},
			{ID: "a2", Key: "2", ChildID: s("e1"), ChildKey: "10", ParentID: s("e3"), ParentKey: s("30"), ParentState: RefResolved, Kind: "REPORTS_TO",
				EffectiveFromRaw: s("2025-06-01 00:00:00"), IsCurrent: b(true)},
		},
		History: []HistoryNode{{ID: "h", Key: "1", ChildKey: s("10"), OldParentKey: s("20"), NewParentKey: s("30"), RecordedRaw: s("2025-06-01 00:00:00")}},
	}
	if fs := Analyze(in); len(fs) != 0 {
		t.Fatalf("clean tree reported %+v", fs)
	}
}
