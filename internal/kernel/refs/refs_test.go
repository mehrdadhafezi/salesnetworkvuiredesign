package refs

import (
	"encoding/json"
	"errors"
	"math/rand"
	"strconv"
	"testing"
	"testing/quick"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
)

func mustNative(t *testing.T, d Domain, id string) NativeRef {
	t.Helper()
	r, err := NewNativeRef(d, id)
	if err != nil {
		t.Fatalf("NewNativeRef(%s,%s): %v", d, id, err)
	}
	return r
}

func TestValidateIDRejects(t *testing.T) {
	cases := map[string]error{
		"":                      ErrInvalidID,
		"0":                     ErrInvalidID,
		"00":                    ErrInvalidID,
		"007":                   ErrInvalidID,
		"-1":                    ErrInvalidID,
		"1.5":                   ErrInvalidID,
		"12a":                   ErrInvalidID,
		" 12":                   ErrInvalidID,
		"۱۲":                    ErrInvalidID,
		"abc-token":             ErrInvalidID,
		"Ali Rezaei":            ErrInvalidID,
		"18446744073709551616":  ErrInvalidID, // > uint64
		"123456789012345678901": ErrInvalidID, // too long
		"09123456789":           ErrPhoneShapedID,
		"9123456789":            ErrPhoneShapedID,
		"989123456789":          ErrPhoneShapedID,
		"+989123456789":         ErrPhoneShapedID,
		"00989123456789":        ErrPhoneShapedID,
	}
	for id, want := range cases {
		if err := ValidateID(id); !errors.Is(err, want) {
			t.Fatalf("ValidateID(%q) = %v, want %v", id, err, want)
		}
	}
	for _, id := range []string{"1", "42", "123456789", "18446744073709551615", "8123456789", "91234567890"} {
		if err := ValidateID(id); err != nil {
			t.Fatalf("ValidateID(%q) = %v, want nil", id, err)
		}
	}
}

func TestNativeRefDomainsAndEquality(t *testing.T) {
	for _, d := range Domains() {
		if !d.Valid() {
			t.Fatalf("declared domain %s invalid", d)
		}
	}
	for _, d := range []Domain{"", "invoice", "CUSTOMER", "PHONE", "INVOICE "} {
		if _, err := NewNativeRef(d, "1"); !errors.Is(err, ErrInvalidDomain) {
			t.Fatalf("domain %q err = %v", d, err)
		}
	}
	inv, stage := mustNative(t, DomainInvoice, "42"), mustNative(t, DomainPaymentStage, "42")
	if inv.Equal(stage) {
		t.Fatal("equal IDs in different domains must be unequal")
	}
	if !inv.Equal(mustNative(t, DomainInvoice, "42")) {
		t.Fatal("same domain and id must be equal")
	}
	if inv.Key() != "INVOICE:42" {
		t.Fatalf("Key = %q", inv.Key())
	}
}

func TestNativeRefJSON(t *testing.T) {
	r := mustNative(t, DomainLegacyLead, "7")
	b, err := json.Marshal(r)
	if err != nil || string(b) != `{"domain":"LEGACY_LEAD","id":"7"}` {
		t.Fatalf("Marshal = %s, %v", b, err)
	}
	var back NativeRef
	if err := json.Unmarshal(b, &back); err != nil || !back.Equal(r) {
		t.Fatalf("round trip = %v, %v", back, err)
	}
	for _, s := range []string{`{"domain":"LEGACY_LEAD","id":7}`, `{"domain":"LEGACY_LEAD","id":"09123456789"}`, `{"domain":"X","id":"1"}`, `[]`} {
		if err := json.Unmarshal([]byte(s), &back); err == nil {
			t.Fatalf("Unmarshal(%s) must fail", s)
		}
	}
	if _, err := json.Marshal(NativeRef{}); err == nil {
		t.Fatal("Marshal of zero NativeRef must fail")
	}
}

func TestProvenanceStatesAreDistinct(t *testing.T) {
	all := Provenances()
	if len(all) != 5 {
		t.Fatalf("want 5 provenance states, got %d", len(all))
	}
	seen := map[string]Provenance{}
	for _, p := range all {
		if !p.Valid() {
			t.Fatalf("%v invalid", p)
		}
		if other, dup := seen[p.String()]; dup {
			t.Fatalf("%v and %v share a name", p, other)
		}
		seen[p.String()] = p
		parsed, err := ParseProvenance(p.String())
		if err != nil || parsed != p {
			t.Fatalf("ParseProvenance(%s) = %v, %v", p, parsed, err)
		}
	}
	// Explicit distinctions required by E0 §9B and Gate-0 §3.
	if ProvenanceNotRecoverable == ProvenanceUnknown || ProvenanceNotApplicable == ProvenanceUnknown || ProvenanceNotRecoverable == ProvenanceNotApplicable {
		t.Fatal("UNKNOWN, NOT_RECOVERABLE and NOT_APPLICABLE must be distinct")
	}
	var zero Provenance
	if zero.Valid() || zero.String() != "INVALID" {
		t.Fatal("zero provenance must be invalid (never collapses into UNKNOWN)")
	}
	for _, bad := range []string{"", "unknown", "NOT RECOVERABLE", "NULL"} {
		if _, err := ParseProvenance(bad); !errors.Is(err, ErrInvalidProvenance) {
			t.Fatalf("ParseProvenance(%q) err = %v", bad, err)
		}
	}
	if _, err := zero.MarshalText(); !errors.Is(err, ErrInvalidProvenance) {
		t.Fatal("zero provenance must not marshal")
	}
	var p Provenance
	if err := p.UnmarshalText([]byte("NOT_APPLICABLE")); err != nil || p != ProvenanceNotApplicable {
		t.Fatalf("UnmarshalText = %v, %v", p, err)
	}
	if err := p.UnmarshalText([]byte("x")); err == nil {
		t.Fatal("UnmarshalText(x) must fail")
	}
}

func TestSourceRef(t *testing.T) {
	row := mustNative(t, DomainMISRow, "10")
	batch := mustNative(t, DomainEvent, "3")
	at, _ := timeutil.ParseRFC3339("2024-03-20T06:30:00Z")
	raw := "1403/01/01 10:00"
	s := SourceRef{Native: row, OriginBatch: &batch, OriginAt: &timeutil.SourceTime{UTC: at, SourceZone: "Asia/Tehran", RawText: &raw}, Provenance: ProvenanceIncomplete}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var back SourceRef
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if !back.Native.Equal(row) || back.OriginBatch == nil || !back.OriginBatch.Equal(batch) || back.OriginActor != nil ||
		back.Provenance != ProvenanceIncomplete || back.OriginAt == nil || *back.OriginAt.RawText != raw || !back.OriginAt.UTC.Equal(at) {
		t.Fatalf("round trip mismatch: %+v", back)
	}
	// Each of the three "not known" states round-trips as itself.
	for _, p := range []Provenance{ProvenanceUnknown, ProvenanceNotRecoverable, ProvenanceNotApplicable} {
		b, _ := json.Marshal(SourceRef{Native: row, Provenance: p})
		var got SourceRef
		if err := json.Unmarshal(b, &got); err != nil || got.Provenance != p {
			t.Fatalf("provenance %v round trip = %v, %v", p, got.Provenance, err)
		}
	}
	invalid := []SourceRef{
		{Native: row},
		{Native: NativeRef{}, Provenance: ProvenanceVerified},
		{Native: row, OriginActor: &NativeRef{Domain: DomainEvent, ID: "0"}, Provenance: ProvenanceVerified},
		{Native: row, OriginAt: &timeutil.SourceTime{}, Provenance: ProvenanceVerified},
	}
	for i, s := range invalid {
		if err := s.Validate(); err == nil {
			t.Fatalf("invalid SourceRef %d accepted", i)
		}
		if _, err := json.Marshal(s); err == nil {
			t.Fatalf("invalid SourceRef %d marshalled", i)
		}
	}
	if err := json.Unmarshal([]byte(`{"native":{"domain":"MIS_ROW","id":"1"}}`), &back); !errors.Is(err, ErrInvalidProvenance) {
		t.Fatalf("missing provenance err = %v", err)
	}
	if err := json.Unmarshal([]byte(`{"native":1}`), &back); err == nil {
		t.Fatal("malformed SourceRef must fail")
	}
}

func TestCanonicalIDIsNeverMinted(t *testing.T) {
	c, err := ParseCanonicalID("case-7f3a")
	if err != nil || c.String() != "case-7f3a" {
		t.Fatalf("ParseCanonicalID = %v, %v", c, err)
	}
	for _, bad := range []string{"", "has space", "tab\t", string(make([]byte, 129)), "فارسی"} {
		if _, err := ParseCanonicalID(bad); !errors.Is(err, ErrInvalidCanonicalID) {
			t.Fatalf("ParseCanonicalID(%q) err = %v", bad, err)
		}
	}
	for _, phone := range []string{"09123456789", "+989123456789", "9123456789"} {
		if _, err := ParseCanonicalID(phone); !errors.Is(err, ErrPhoneShapedID) {
			t.Fatalf("phone %q accepted as canonical id: %v", phone, err)
		}
	}
	var zero CanonicalID
	if !zero.IsZero() {
		t.Fatal("zero canonical id must be zero")
	}
	if _, err := zero.MarshalText(); err == nil {
		t.Fatal("zero canonical id must not marshal")
	}
	var back CanonicalID
	if err := back.UnmarshalText([]byte("09123456789")); err == nil {
		t.Fatal("UnmarshalText must reject phone")
	}
}

func TestCaseRefResolution(t *testing.T) {
	lead, dot := mustNative(t, DomainLegacyLead, "5"), mustNative(t, DomainDotCase, "9")
	canonical, _ := ParseCanonicalID("case-1")

	// RESOLVED requires an existing canonical id.
	if _, err := NewCaseRef([]NativeRef{lead, dot}, nil, nil, ResolutionResolved); !errors.Is(err, ErrResolvedWithoutCanonical) {
		t.Fatalf("RESOLVED without canonical err = %v", err)
	}
	resolved, err := NewCaseRef([]NativeRef{dot, lead}, nil, &canonical, ResolutionResolved)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Natives[0].Key() != "DOT_CASE:9" {
		t.Fatalf("natives must be sorted deterministically: %v", resolved.Natives)
	}
	// A proven grouping without a canonical id is RELATED, never RESOLVED.
	for _, r := range []Resolution{ResolutionRelated, ResolutionUnknown, ResolutionConflict} {
		c, err := NewCaseRef([]NativeRef{lead, dot}, nil, nil, r)
		if err != nil || c.CanonicalID != nil || c.Resolution != r {
			t.Fatalf("%v without canonical = %+v, %v", r, c, err)
		}
		// An existing canonical id may still be supplied for these states.
		if _, err := NewCaseRef([]NativeRef{lead}, nil, &canonical, r); err != nil {
			t.Fatalf("%v with supplied canonical: %v", r, err)
		}
	}
	if _, err := NewCaseRef(nil, nil, nil, ResolutionUnknown); !errors.Is(err, ErrEmptyNatives) {
		t.Fatalf("empty natives err = %v", err)
	}
	if _, err := NewCaseRef([]NativeRef{lead, lead}, nil, nil, ResolutionRelated); !errors.Is(err, ErrDuplicateNative) {
		t.Fatalf("duplicate native err = %v", err)
	}
	if _, err := NewCaseRef([]NativeRef{lead}, nil, nil, Resolution(0)); !errors.Is(err, ErrInvalidResolution) {
		t.Fatalf("zero resolution err = %v", err)
	}
	if _, err := NewCaseRef([]NativeRef{lead}, nil, &CanonicalID{}, ResolutionRelated); !errors.Is(err, ErrInvalidCanonicalID) {
		t.Fatalf("empty canonical err = %v", err)
	}
	if _, err := NewCaseRef([]NativeRef{{Domain: DomainInvoice, ID: "09123456789"}}, nil, nil, ResolutionUnknown); !errors.Is(err, ErrPhoneShapedID) {
		t.Fatalf("phone native err = %v", err)
	}
	if _, err := NewCaseRef([]NativeRef{lead}, []SourceRef{{Native: lead}}, nil, ResolutionUnknown); !errors.Is(err, ErrInvalidProvenance) {
		t.Fatalf("invalid source err = %v", err)
	}
	for _, r := range Resolutions() {
		parsed, err := ParseResolution(r.String())
		if err != nil || parsed != r {
			t.Fatalf("ParseResolution(%s) = %v, %v", r, parsed, err)
		}
	}
	if _, err := ParseResolution("resolved"); !errors.Is(err, ErrInvalidResolution) {
		t.Fatal("lowercase resolution must be rejected")
	}
	var zero Resolution
	if zero.String() != "INVALID" {
		t.Fatal("zero resolution name")
	}
	if _, err := zero.MarshalText(); err == nil {
		t.Fatal("zero resolution must not marshal")
	}
}

func TestCaseRefJSON(t *testing.T) {
	lead := mustNative(t, DomainLegacyLead, "5")
	c, _ := NewCaseRef([]NativeRef{lead}, []SourceRef{{Native: lead, Provenance: ProvenanceNotRecoverable}}, nil, ResolutionRelated)
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"natives":[{"domain":"LEGACY_LEAD","id":"5"}],"sources":[{"native":{"domain":"LEGACY_LEAD","id":"5"},"provenance":"NOT_RECOVERABLE"}],"canonical_id":null,"resolution":"RELATED"}`
	if string(b) != want {
		t.Fatalf("Marshal =\n%s\nwant\n%s", b, want)
	}
	var back CaseRef
	if err := json.Unmarshal(b, &back); err != nil || back.Resolution != ResolutionRelated || back.CanonicalID != nil {
		t.Fatalf("round trip = %+v, %v", back, err)
	}
	if err := json.Unmarshal([]byte(`{"natives":[{"domain":"LEGACY_LEAD","id":"5"}],"canonical_id":null,"resolution":"RESOLVED"}`), &back); !errors.Is(err, ErrResolvedWithoutCanonical) {
		t.Fatalf("RESOLVED/null JSON err = %v", err)
	}
	if err := json.Unmarshal([]byte(`{"natives":[]}`), &back); err == nil {
		t.Fatal("empty natives JSON must fail")
	}
	if _, err := json.Marshal(CaseRef{}); err == nil {
		t.Fatal("zero CaseRef must not marshal")
	}
}

func TestLegacyRef(t *testing.T) {
	r, err := NewLegacyRef(LegacyWP, "users", "12")
	if err != nil || r.System() != LegacyWP || r.Entity() != "users" || r.ID() != "12" || r.IsZero() {
		t.Fatalf("NewLegacyRef = %+v, %v", r, err)
	}
	other, _ := NewLegacyRef(LegacyWoo, "users", "12")
	if r.Equal(other) {
		t.Fatal("different systems must be unequal")
	}
	if (LegacyRef{}).IsZero() != true {
		t.Fatal("zero LegacyRef must be zero")
	}
	if _, err := NewLegacyRef("mysql", "users", "1"); !errors.Is(err, ErrInvalidLegacySystem) {
		t.Fatalf("system err = %v", err)
	}
	for _, e := range []string{"", "Users", "1users", "user-s", string(make([]byte, 65))} {
		if _, err := NewLegacyRef(LegacyGateway, e, "1"); !errors.Is(err, ErrInvalidLegacyEntity) {
			t.Fatalf("entity %q err = %v", e, err)
		}
	}
	if _, err := NewLegacyRef(LegacyBiawin, "orders", "09123456789"); !errors.Is(err, ErrPhoneShapedID) {
		t.Fatalf("phone legacy id err = %v", err)
	}
	if _, err := NewLegacyRef(LegacyBiawin, "orders", "tok_abc"); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("token legacy id err = %v", err)
	}
}

var quickConfig = &quick.Config{MaxCount: 2000, Rand: rand.New(rand.NewSource(20261029))}

// Property: every accepted ID round-trips through uint64 unchanged and is never phone-shaped.
func TestPropertyAcceptedIDsAreCanonical(t *testing.T) {
	f := func(n uint64) bool {
		id := strconv.FormatUint(n, 10)
		err := ValidateID(id)
		if n == 0 {
			return errors.Is(err, ErrInvalidID)
		}
		if looksLikePhone(id) {
			return errors.Is(err, ErrPhoneShapedID)
		}
		return err == nil
	}
	if err := quick.Check(f, quickConfig); err != nil {
		t.Fatal(err)
	}
}

// Property: CaseRef JSON round trip preserves resolution and canonical presence.
func TestPropertyCaseRefRoundTrip(t *testing.T) {
	f := func(a, b uint32, res uint8, withCanonical bool) bool {
		natives := []NativeRef{{DomainInvoice, strconv.FormatUint(uint64(a)+1, 10)}, {DomainEvent, strconv.FormatUint(uint64(b)+1, 10)}}
		r := Resolutions()[int(res)%len(Resolutions())]
		var canonical *CanonicalID
		if withCanonical || r == ResolutionResolved {
			c, _ := ParseCanonicalID("c-" + strconv.FormatUint(uint64(a), 10))
			canonical = &c
		}
		c, err := NewCaseRef(natives, nil, canonical, r)
		if err != nil {
			return false
		}
		data, err := json.Marshal(c)
		if err != nil {
			return false
		}
		var back CaseRef
		if json.Unmarshal(data, &back) != nil || back.Resolution != r || (back.CanonicalID == nil) != (canonical == nil) {
			return false
		}
		return back.Resolution != ResolutionResolved || back.CanonicalID != nil
	}
	if err := quick.Check(f, quickConfig); err != nil {
		t.Fatal(err)
	}
}

func FuzzValidateID(f *testing.F) {
	for _, s := range []string{"1", "0", "09123456789", "+989123456789", "18446744073709551615", "abc", ""} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if ValidateID(s) != nil {
			return
		}
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil || n == 0 || strconv.FormatUint(n, 10) != s || looksLikePhone(s) {
			t.Fatalf("ValidateID accepted %q", s)
		}
	})
}
