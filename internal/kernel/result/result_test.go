package result

import (
	"errors"
	"math/rand"
	"strings"
	"testing"
	"testing/quick"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/refs"
)

func TestStatusesClosedSet(t *testing.T) {
	want := []string{"SUCCESS", "PARTIAL", "FAILED", "UNKNOWN", "CONFLICT", "BLOCKED", "UNAUTHORIZED"}
	got := Statuses()
	if len(got) != len(want) {
		t.Fatalf("got %d statuses", len(got))
	}
	for i, s := range got {
		if s.String() != want[i] || !s.Valid() {
			t.Fatalf("status %d = %s", i, s)
		}
		parsed, err := ParseStatus(want[i])
		if err != nil || parsed != s {
			t.Fatalf("ParseStatus(%s) = %v, %v", want[i], parsed, err)
		}
		b, err := s.MarshalText()
		if err != nil || string(b) != want[i] {
			t.Fatalf("MarshalText(%s)", s)
		}
		var back Status
		if err := back.UnmarshalText(b); err != nil || back != s {
			t.Fatalf("UnmarshalText(%s)", b)
		}
	}
	for _, bad := range []string{"", "success", "OK", "ERROR", "TIMEOUT", "INVALID", "DENIED"} {
		if _, err := ParseStatus(bad); !errors.Is(err, ErrInvalidStatus) {
			t.Fatalf("ParseStatus(%q) err = %v", bad, err)
		}
	}
	for _, s := range []Status{0, 8, 255} {
		if s.Valid() || s.String() != "INVALID" {
			t.Fatalf("status %d must be invalid", s)
		}
		if _, err := s.MarshalText(); !errors.Is(err, ErrInvalidStatus) {
			t.Fatalf("invalid status %d marshalled", s)
		}
	}
	var s Status
	if err := s.UnmarshalText([]byte("nope")); err == nil {
		t.Fatal("UnmarshalText(nope) must fail")
	}
}

func TestStatusesAreNotCollapsed(t *testing.T) {
	pairs := [][2]Status{{Unknown, Failed}, {Partial, Success}, {Conflict, Failed}, {Unauthorized, Blocked}, {Unknown, Success}}
	for _, p := range pairs {
		if p[0] == p[1] || p[0].String() == p[1].String() {
			t.Fatalf("%s and %s must be distinct", p[0], p[1])
		}
	}
}

func TestAggregateRules(t *testing.T) {
	if _, err := Aggregate(nil); !errors.Is(err, ErrEmptyAggregate) {
		t.Fatalf("empty aggregate err = %v", err)
	}
	// All identical → that class, for every one of the seven statuses.
	for _, s := range Statuses() {
		got, err := Aggregate([]Status{s, s, s})
		if err != nil || got != s {
			t.Fatalf("Aggregate(all %s) = %v, %v", s, got, err)
		}
	}
	cases := []struct {
		in   []Status
		want Status
	}{
		{[]Status{Success, Failed}, Partial},
		{[]Status{Success, Unknown}, Partial},
		{[]Status{Success, Partial}, Partial},
		{[]Status{Failed, Unknown}, Partial}, // UNKNOWN wins only if every item is UNKNOWN
		{[]Status{Unknown, Unknown, Conflict}, Partial},
		{[]Status{Blocked, Unauthorized}, Partial},
		{[]Status{Conflict, Failed}, Partial},
		{[]Status{Unknown}, Unknown},
	}
	for _, c := range cases {
		got, err := Aggregate(c.in)
		if err != nil || got != c.want {
			t.Fatalf("Aggregate(%v) = %v, %v; want %v", c.in, got, err, c.want)
		}
	}
	if _, err := Aggregate([]Status{Success, 0}); !errors.Is(err, ErrInvalidStatus) {
		t.Fatalf("invalid member err = %v", err)
	}
	if _, err := Aggregate([]Status{9}); !errors.Is(err, ErrInvalidStatus) {
		t.Fatalf("invalid first err = %v", err)
	}
}

var quickConfig = &quick.Config{MaxCount: 3000, Rand: rand.New(rand.NewSource(20261029))}

func toStatuses(raw []uint8) []Status {
	out := make([]Status, len(raw))
	for i, r := range raw {
		out[i] = Statuses()[int(r)%7]
	}
	return out
}

// Property: aggregation is permutation-invariant.
func TestPropertyPermutationInvariant(t *testing.T) {
	r := rand.New(rand.NewSource(7))
	f := func(raw []uint8) bool {
		if len(raw) == 0 {
			return true
		}
		in := toStatuses(raw)
		a, errA := Aggregate(in)
		shuffled := append([]Status(nil), in...)
		r.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
		b, errB := Aggregate(shuffled)
		return errA == nil && errB == nil && a == b
	}
	if err := quick.Check(f, quickConfig); err != nil {
		t.Fatal(err)
	}
}

// Property: UNKNOWN never becomes SUCCESS or FAILED, and SUCCESS needs all SUCCESS.
func TestPropertyUnknownNeverCoerced(t *testing.T) {
	f := func(raw []uint8) bool {
		in := append(toStatuses(raw), Unknown)
		got, err := Aggregate(in)
		if err != nil || got == Success || got == Failed {
			return false
		}
		allUnknown := true
		for _, s := range in {
			allUnknown = allUnknown && s == Unknown
		}
		return (got == Unknown) == allUnknown && (allUnknown || got == Partial)
	}
	if err := quick.Check(f, quickConfig); err != nil {
		t.Fatal(err)
	}
	g := func(raw []uint8) bool {
		if len(raw) == 0 {
			return true
		}
		in := toStatuses(raw)
		got, _ := Aggregate(in)
		allSuccess := true
		for _, s := range in {
			allSuccess = allSuccess && s == Success
		}
		return (got == Success) == allSuccess
	}
	if err := quick.Check(g, quickConfig); err != nil {
		t.Fatal(err)
	}
}

func nativeRef(t *testing.T) *ItemRef {
	t.Helper()
	n, err := refs.NewNativeRef(refs.DomainInvoice, "42")
	if err != nil {
		t.Fatal(err)
	}
	return &ItemRef{Native: &n}
}

func strp(s string) *string { return &s }

func TestItemOutcomeValidation(t *testing.T) {
	ok := ItemOutcome{ItemRef: nativeRef(t), Status: Success}
	if err := ok.Validate(); err != nil {
		t.Fatal(err)
	}
	// A nil ItemRef requires a missing-ref reason.
	if err := (ItemOutcome{Status: Unknown}).Validate(); !errors.Is(err, ErrMissingRefReason) {
		t.Fatalf("nil ref without reason err = %v", err)
	}
	if err := (ItemOutcome{Status: Unknown, ReasonCodes: []reasoncodes.Code{reasoncodes.SourceUnavailable}}).Validate(); err != nil {
		t.Fatalf("nil ref with reason: %v", err)
	}
	lead, _ := refs.NewNativeRef(refs.DomainLegacyLead, "5")
	caseRef, _ := refs.NewCaseRef([]refs.NativeRef{lead}, nil, nil, refs.ResolutionUnknown)
	if err := (ItemOutcome{ItemRef: &ItemRef{Case: &caseRef}, Status: Conflict}).Validate(); err != nil {
		t.Fatalf("case item ref: %v", err)
	}
	zeroOnly := &refs.NativeRef{}
	invalid := []ItemOutcome{
		{ItemRef: nativeRef(t)},                                             // zero status
		{ItemRef: &ItemRef{}, Status: Success},                              // neither ref
		{ItemRef: &ItemRef{Native: &lead, Case: &caseRef}, Status: Success}, // both refs
		{ItemRef: &ItemRef{Native: zeroOnly}, Status: Success},              // invalid native
		{ItemRef: &ItemRef{Case: &refs.CaseRef{}}, Status: Success},         // invalid case
		{ItemRef: nativeRef(t), Status: Success, ReasonCodes: []reasoncodes.Code{"NOPE"}},
		{ItemRef: nativeRef(t), Status: Success, BeforeRef: zeroOnly},
		{ItemRef: nativeRef(t), Status: Success, AfterRef: zeroOnly},
		{ItemRef: nativeRef(t), Status: Success, CorrelationID: strp("")},
		{ItemRef: nativeRef(t), Status: Success, CorrelationID: strp("has space")},
		{ItemRef: nativeRef(t), Status: Success, CorrelationID: strp(strings.Repeat("x", 129))},
		{ItemRef: nativeRef(t), Status: Success, Metadata: Metadata{Disposition: 99}},
		{ItemRef: nativeRef(t), Status: Success, Metadata: Metadata{Coverage: 99}},
		{ItemRef: nativeRef(t), Status: Success, Metadata: Metadata{EvaluationType: "Has-Caps"}},
		{ItemRef: nativeRef(t), Status: Success, Metadata: Metadata{EvaluationType: strings.Repeat("a", 65)}},
		{ItemRef: nativeRef(t), Status: Success, Metadata: Metadata{Disposition: DispositionSkipped}}, // SKIPPED+SUCCESS needs reason
	}
	for i, o := range invalid {
		if err := o.Validate(); err == nil {
			t.Fatalf("invalid item %d accepted: %+v", i, o)
		}
	}
	known := true
	full := ItemOutcome{
		ItemRef: nativeRef(t), Status: Success, ReasonCodes: []reasoncodes.Code{reasoncodes.StaleState},
		Message: strp("no-op"), CorrelationID: strp("corr-1"), BeforeRef: &lead, AfterRef: &lead,
		Metadata: Metadata{EvaluationType: "dependency_check", Disposition: DispositionSkipped, Coverage: CoverageComplete, EffectsKnown: &known},
	}
	if err := full.Validate(); err != nil {
		t.Fatalf("full item: %v", err)
	}
}

func TestNewResult(t *testing.T) {
	if _, err := New(nil, nil); !errors.Is(err, ErrEmptyAggregate) {
		t.Fatalf("empty New err = %v", err)
	}
	items := []ItemOutcome{
		{ItemRef: nativeRef(t), Status: Success},
		{Status: Unknown, ReasonCodes: []reasoncodes.Code{reasoncodes.OutcomeUnknown}},
	}
	r, err := New(items, strp("op-1"))
	if err != nil || r.Status != Partial || len(r.Items) != 2 {
		t.Fatalf("New = %+v, %v", r, err)
	}
	counts := r.Counts()
	if counts[Success] != 1 || counts[Unknown] != 1 {
		t.Fatalf("per-item classes must be preserved: %v", counts)
	}
	items[0].Status = Failed // the result holds its own copy
	if r.Items[0].Status != Success {
		t.Fatal("Result must copy items")
	}
	if _, err := New([]ItemOutcome{{Status: Success}}, nil); !errors.Is(err, ErrMissingRefReason) {
		t.Fatalf("invalid item err = %v", err)
	}
	if _, err := New(items, strp("")); !errors.Is(err, ErrInvalidCorrelation) {
		t.Fatalf("invalid correlation err = %v", err)
	}
	all, err := New([]ItemOutcome{{ItemRef: nativeRef(t), Status: Blocked}}, nil)
	if err != nil || all.Status != Blocked {
		t.Fatalf("single BLOCKED = %v, %v", all.Status, err)
	}
}

func TestDispositionAndCoverageBounds(t *testing.T) {
	for d := DispositionNone; d <= DispositionUnprocessed; d++ {
		if !d.Valid() {
			t.Fatalf("disposition %d invalid", d)
		}
	}
	if Disposition(DispositionUnprocessed + 1).Valid() {
		t.Fatal("disposition out of range must be invalid")
	}
	if !CoverageIncomplete.Valid() || Coverage(CoverageIncomplete+1).Valid() {
		t.Fatal("coverage bounds")
	}
}
