package domain

import "testing"

func ev(kinds ...string) []Evidence {
	out := make([]Evidence, 0, len(kinds))
	for _, k := range kinds {
		c := Catalogue[k]
		out = append(out, Evidence{Kind: k, Strength: c.Strength, Polarity: c.Polarity})
	}
	return out
}

func TestResolverRules(t *testing.T) {
	cases := []struct {
		kinds []string
		want  string
	}{
		{nil, Unknown},
		{[]string{PhoneMatch}, Unknown},                             // phone-only never resolves
		{[]string{PhoneMatch, SequentialPhoneReuse}, Unknown},       // sequential reuse: distinct, no merge
		{[]string{PhoneMatch, AccountRefMatch}, Related},            // weak relation, not equality
		{[]string{PhoneMatch, OTPContactControl}, Related},          // OTP proves control only
		{[]string{PhoneMatch, OverlappingPhoneOwnership}, Conflict}, // overlapping ownership
		{[]string{ExclusiveIdentifierShared}, Conflict},
		{[]string{VerifiedEquality}, Resolved}, // only explicit strong equality resolves
		{[]string{VerifiedEquality, VerifiedDistinct}, Conflict},
		{[]string{VerifiedEquality, OverlappingPhoneOwnership}, Conflict},
		{[]string{SourceNotRecoverable, PhoneMatch}, NotRecoverable},
	}
	for _, c := range cases {
		if got, _ := Resolve(ev(c.kinds...)); got != c.want {
			t.Errorf("Resolve(%v) = %s, want %s", c.kinds, got, c.want)
		}
	}
	// No combination of weak/medium evidence alone can ever resolve.
	nonStrong := []string{PhoneMatch, AccountRefMatch, SequentialPhoneReuse, OTPContactControl, OverlappingPhoneOwnership, ExclusiveIdentifierShared}
	for mask := 0; mask < 1<<len(nonStrong); mask++ {
		var ks []string
		for i, k := range nonStrong {
			if mask&(1<<i) != 0 {
				ks = append(ks, k)
			}
		}
		if got, _ := Resolve(ev(ks...)); got == Resolved {
			t.Fatalf("non-strong evidence %v resolved", ks)
		}
	}
	for k, c := range Catalogue {
		if c.Polarity == Equality && c.Strength != Strong {
			t.Fatalf("%s: equality evidence must be STRONG", k)
		}
		if (k == PhoneMatch || k == AccountRefMatch || k == SequentialPhoneReuse) && c.Strength != Weak {
			t.Fatalf("%s must stay WEAK", k)
		}
		if k == OTPContactControl && (c.Strength != Medium || c.Polarity == Equality) {
			t.Fatal("OTP is MEDIUM contact control, never equality")
		}
	}
}

func TestCustomerState(t *testing.T) {
	for _, c := range []struct {
		rec    bool
		states []string
		want   string
	}{
		{true, nil, Unknown},
		{true, []string{Unknown, Related}, Related},
		{true, []string{Resolved, Conflict}, Conflict},
		{false, []string{Resolved}, NotRecoverable},
	} {
		if got, _ := CustomerState(c.rec, c.states); got != c.want {
			t.Errorf("CustomerState(%v, %v) = %s, want %s", c.rec, c.states, got, c.want)
		}
	}
}

func TestNormalization(t *testing.T) {
	for raw, want := range map[string]string{
		"0900 000 0001": "09000000001", "+98 900 000 0001": "09000000001", "00989000000001": "09000000001",
		"9000000001": "09000000001", "۰۹۰۰۰۰۰۰۰۰۱": "09000000001",
	} {
		if got, st := NormalizePhone(raw); got != want || st != NormValid {
			t.Errorf("NormalizePhone(%q) = %q %s", raw, got, st)
		}
	}
	for _, raw := range []string{"TEST-PHONE-0001", "12345", "0800000001"} {
		if got, st := NormalizePhone(raw); got != "" || st != NormInvalid {
			t.Errorf("NormalizePhone(%q) = %q %s", raw, got, st)
		}
	}
	if _, st := NormalizePhone("  "); st != NormEmpty {
		t.Fatal("blank phone must be EMPTY")
	}
	if Mask("09000000001") != "09*******01" || Mask("abc") != "***" {
		t.Fatalf("mask = %q", Mask("09000000001"))
	}
	if !Overlap(ParseWindow(p("2025-01-01 00:00:00"), p("2025-06-01 00:00:00")), ParseWindow(p("2025-05-01 00:00:00"), nil)) {
		t.Fatal("overlap not detected")
	}
	if Overlap(ParseWindow(p("2025-01-01 00:00:00"), p("2025-02-01 00:00:00")), ParseWindow(p("2025-03-01 00:00:00"), nil)) {
		t.Fatal("disjoint windows overlap")
	}
	if ParseWindow(p("0000-00-00 00:00:00"), nil).Known {
		t.Fatal("zero date window known")
	}
}

func p(s string) *string { return &s }
