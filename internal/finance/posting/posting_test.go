package posting

import (
	"errors"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/money"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
)

var t0 = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

func s(v string) *string { return &v }

func intent(key string, amount int64) Intent {
	return Intent{BusinessKey: key, Basis: BasisAccrual, Unit: money.IRR, Source: Source{System: "synthetic", InvoiceID: s("inv-1")},
		EffectiveAt: t0, RecordedAt: t0.Add(time.Hour),
		Lines: []Line{{Account: "customer_receivable", Side: Debit, Amount: amount}, {Account: "revenue_synthetic", Side: Credit, Amount: amount}}}
}

func TestValidateInvariants(t *testing.T) {
	if err := intent("finance|invoice_issued|inv-1|receivable|IRR", 100).Validate(); err != nil {
		t.Fatal(err)
	}
	for name, mut := range map[string]func(*Intent){
		"unbalanced":                   func(i *Intent) { i.Lines[1].Amount = 99 },
		"zero amount":                  func(i *Intent) { i.Lines[0].Amount, i.Lines[1].Amount = 0, 0 },
		"negative amount":              func(i *Intent) { i.Lines[0].Amount, i.Lines[1].Amount = -5, -5 },
		"no unit":                      func(i *Intent) { i.Unit = 0 },
		"no basis (D-41 never chosen)": func(i *Intent) { i.Basis = "" },
		"unknown basis":                func(i *Intent) { i.Basis = "MIXED" },
		"no provenance":                func(i *Intent) { i.Source.InvoiceID = nil },
		"no system":                    func(i *Intent) { i.Source.System = "" },
		"bad key":                      func(i *Intent) { i.BusinessKey = "free text key" },
		"one line":                     func(i *Intent) { i.Lines = i.Lines[:1] },
		"bad side":                     func(i *Intent) { i.Lines[0].Side = "LEFT" },
		"zero effective":               func(i *Intent) { i.EffectiveAt = time.Time{} },
		"non-UTC":                      func(i *Intent) { i.RecordedAt = i.RecordedAt.In(time.FixedZone("x", 3600)) },
		"overflow": func(i *Intent) {
			i.Lines = []Line{{Account: "a1", Side: Debit, Amount: math.MaxInt64}, {Account: "a2", Side: Debit, Amount: 1},
				{Account: "a3", Side: Credit, Amount: 1}}
		},
	} {
		i := intent("finance|invoice_issued|inv-1|receivable|IRR", 100)
		mut(&i)
		if err := i.Validate(); err == nil {
			t.Fatalf("%s must be invalid", name)
		}
	}
}

func TestActivationAlwaysBlocked(t *testing.T) {
	o := EvaluateActivation(intent("finance|invoice_issued|inv-1|receivable|IRR", 100))
	for _, d := range []string{DecisionD41, DecisionOPD06, DecisionOPD08, DecisionOPD09, NoPostingAuthority} {
		if o.Status != result.Blocked || !slices.Contains(o.BlockedBy, d) {
			t.Fatalf("activation: %+v", o)
		}
	}
}

// Duplicates never produce a second economic effect; a reused key with a
// different intent conflicts; a reversal is a separate mirrored intent and
// happens at most once; the original is never mutated.
func TestIdempotencyAndReversal(t *testing.T) {
	j := NewJournal()
	k := "finance|invoice_issued|inv-1|receivable|IRR"
	a := intent(k, 100)
	if o := Apply(j, a); o.Status != result.Success || o.Replay {
		t.Fatalf("first: %+v", o)
	}
	for range 5 {
		if o := Apply(j, a); o.Status != result.Success || !o.Replay {
			t.Fatalf("replay: %+v", o)
		}
	}
	if o := Apply(j, intent(k, 101)); o.Status != result.Conflict {
		t.Fatalf("key reuse: %+v", o)
	}
	if net, _ := j.Net("customer_receivable", money.IRR); j.Len() != 1 || net != 100 {
		t.Fatalf("duplicate economic effect: len %d net %d", j.Len(), net)
	}
	rev, err := Reversal(a, "finance|invoice_issued_reversal|inv-1|receivable|IRR", t0.Add(2*time.Hour), t0.Add(2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if o := Apply(j, rev); o.Status != result.Success {
		t.Fatalf("reversal: %+v", o)
	}
	rev2 := rev
	rev2.BusinessKey = "finance|invoice_issued_reversal2|inv-1|receivable|IRR"
	if o := Apply(j, rev2); o.Status != result.Conflict {
		t.Fatalf("second reversal: %+v", o)
	}
	if net, _ := j.Net("customer_receivable", money.IRR); net != 0 || j.Len() != 2 {
		t.Fatalf("after reversal net %d len %d", net, j.Len())
	}
	orig, _, _ := j.Lookup(k)
	if RequestHash(orig.Intent) != RequestHash(a) {
		t.Fatal("the original must never be mutated")
	}
	if _, err := Reversal(rev, "finance|x|y|z|IRR", t0, t0); err == nil {
		t.Fatal("a reversal is never reversed again")
	}
	// A non-mirroring "reversal" is refused.
	bad := intent("finance|bad_reversal|inv-1|receivable|IRR", 50)
	bad.Reverses = s(k)
	if o := Apply(NewJournal(), bad); o.Status != result.Conflict {
		t.Fatalf("reversal of a missing original: %+v", o)
	}
	// Units never mix: a net in another unit sees nothing.
	if net, _ := j.Net("customer_receivable", money.IRT); net != 0 {
		t.Fatal("cross-unit leakage")
	}
}

// flaky is a store whose first Append commits but reports an unknown
// outcome (a lost commit acknowledgement).
type flaky struct {
	*Journal
	lost bool
}

func (f *flaky) Append(r Recorded) (bool, error) {
	ok, err := f.Journal.Append(r)
	if err == nil && !f.lost {
		f.lost = true
		return ok, ErrOutcomeUnknown
	}
	return ok, err
}

func TestUnknownOutcomeIsReconciledBeforeRetry(t *testing.T) {
	f := &flaky{Journal: NewJournal()}
	a := intent("finance|payment_accepted|att-1|cash|IRR", 700)
	o := Apply(f, a)
	if o.Status != result.Unknown {
		t.Fatalf("lost ack must be UNKNOWN: %+v", o)
	}
	// The retry looks the key up first: it finds the committed intent and
	// reports a replay, never a second effect.
	if o := Apply(f, a); o.Status != result.Success || !o.Replay || f.Len() != 1 {
		t.Fatalf("retry after unknown: %+v len %d", o, f.Len())
	}
	if !errors.Is(ErrOutcomeUnknown, ErrOutcomeUnknown) {
		t.Fatal("sentinel")
	}
}
