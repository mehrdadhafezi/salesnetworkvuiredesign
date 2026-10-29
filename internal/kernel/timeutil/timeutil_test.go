package timeutil

import (
	"errors"
	"testing"
	"time"
	_ "time/tzdata" // deterministic IANA data in tests, independent of the host OS
)

func loadZone(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatalf("LoadLocation(%q): %v", name, err)
	}
	return loc
}

func TestNewInstantNormalizesToUTC(t *testing.T) {
	tehran := time.FixedZone("x", 3*3600+1800)
	in := time.Date(2024, 3, 20, 10, 0, 0, 0, tehran)
	i, err := NewInstant(in)
	if err != nil {
		t.Fatal(err)
	}
	if i.Time().Location() != time.UTC {
		t.Fatalf("location = %v, want UTC", i.Time().Location())
	}
	if want := time.Date(2024, 3, 20, 6, 30, 0, 0, time.UTC); !i.Time().Equal(want) || i.Time() != want {
		t.Fatalf("instant = %v, want %v", i.Time(), want)
	}
	if i.String() != "2024-03-20T06:30:00Z" {
		t.Fatalf("String = %q", i.String())
	}
	if _, err := NewInstant(time.Time{}); !errors.Is(err, ErrInvalidTime) {
		t.Fatalf("zero time err = %v", err)
	}
	var zero Instant
	if !zero.IsZero() {
		t.Fatal("zero Instant must be zero")
	}
	if _, err := zero.MarshalText(); !errors.Is(err, ErrInvalidTime) {
		t.Fatalf("zero MarshalText err = %v", err)
	}
}

func TestParseRFC3339ExplicitOffset(t *testing.T) {
	cases := map[string]string{
		"2024-03-20T10:00:00+03:30":     "2024-03-20T06:30:00Z",
		"2024-03-20T06:30:00Z":          "2024-03-20T06:30:00Z",
		"2024-03-20T01:30:00-05:00":     "2024-03-20T06:30:00Z",
		"2024-03-20T10:00:00.123+03:30": "2024-03-20T06:30:00.123Z",
	}
	for in, want := range cases {
		i, err := ParseRFC3339(in)
		if err != nil || i.String() != want {
			t.Fatalf("ParseRFC3339(%q) = %v, %v; want %s", in, i, err, want)
		}
	}
	for _, bad := range []string{"", "2024-03-20T10:00:00", "2024-03-20 10:00:00", "2024-13-01T00:00:00Z", "1403/01/01", "not a time"} {
		if _, err := ParseRFC3339(bad); !errors.Is(err, ErrInvalidTime) {
			t.Fatalf("ParseRFC3339(%q) err = %v, want ErrInvalidTime", bad, err)
		}
	}
}

func TestInstantTextRoundTrip(t *testing.T) {
	i, _ := ParseRFC3339("2025-01-02T03:04:05.000000006+03:30")
	b, err := i.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	var back Instant
	if err := back.UnmarshalText(b); err != nil || !back.Equal(i) {
		t.Fatalf("round trip %s -> %v, %v", b, back, err)
	}
	if err := back.UnmarshalText([]byte("2025-01-02T03:04:05")); !errors.Is(err, ErrInvalidTime) {
		t.Fatalf("offsetless UnmarshalText err = %v", err)
	}
}

func TestParseLocalRequiresExplicitZone(t *testing.T) {
	const layout = "2006-01-02 15:04:05"
	if _, err := ParseLocal(layout, "2024-03-20 10:00:00", nil); !errors.Is(err, ErrZoneRequired) {
		t.Fatalf("nil zone err = %v", err)
	}
	if _, err := ParseLocal(layout, "2024-03-20 10:00:00", time.Local); !errors.Is(err, ErrZoneRequired) {
		t.Fatalf("machine-local zone err = %v", err)
	}
	if _, err := ParseLocal(time.RFC3339, "2024-03-20T10:00:00Z", time.UTC); !errors.Is(err, ErrInvalidTime) {
		t.Fatalf("layout with offset must be rejected, err = %v", err)
	}
	if _, err := ParseLocal(layout, "2024-03-20T10:00", time.UTC); !errors.Is(err, ErrInvalidTime) {
		t.Fatalf("malformed text err = %v", err)
	}
}

func TestParseLocalPreservesSourceZoneAndRaw(t *testing.T) {
	tehran := loadZone(t, "Asia/Tehran")
	raw := "2024-03-20 10:00:00"
	st, err := ParseLocal("2006-01-02 15:04:05", raw, tehran)
	if err != nil {
		t.Fatal(err)
	}
	if st.UTC.String() != "2024-03-20T06:30:00Z" {
		t.Fatalf("UTC = %s", st.UTC)
	}
	if st.SourceZone != "Asia/Tehran" {
		t.Fatalf("SourceZone = %q", st.SourceZone)
	}
	if st.RawText == nil || *st.RawText != raw {
		t.Fatalf("RawText not preserved: %v", st.RawText)
	}
	if err := st.Validate(); err != nil {
		t.Fatal(err)
	}
	back, err := st.InZone(tehran)
	if err != nil || back.Format("2006-01-02 15:04:05") != raw {
		t.Fatalf("InZone = %v, %v", back, err)
	}
	if _, err := st.InZone(time.UTC); !errors.Is(err, ErrZoneRequired) {
		t.Fatalf("InZone with a different zone must fail, err = %v", err)
	}
	if _, err := st.InZone(nil); !errors.Is(err, ErrZoneRequired) {
		t.Fatalf("InZone(nil) err = %v", err)
	}
}

func TestParseLocalDSTGapAndOverlap(t *testing.T) {
	ny := loadZone(t, "America/New_York")
	const layout = "2006-01-02 15:04"
	if _, err := ParseLocal(layout, "2021-03-14 02:30", ny); !errors.Is(err, ErrNonexistentLocalTime) {
		t.Fatalf("spring-forward gap err = %v", err)
	}
	if _, err := ParseLocal(layout, "2021-11-07 01:30", ny); !errors.Is(err, ErrAmbiguousLocalTime) {
		t.Fatalf("fall-back overlap err = %v", err)
	}
	st, err := ParseLocal(layout, "2021-11-07 03:30", ny)
	if err != nil || st.UTC.String() != "2021-11-07T08:30:00Z" {
		t.Fatalf("unambiguous after overlap = %v, %v", st.UTC, err)
	}
}

func TestSourceTimeValidate(t *testing.T) {
	i, _ := ParseRFC3339("2024-01-01T00:00:00Z")
	if err := (SourceTime{}).Validate(); !errors.Is(err, ErrInvalidTime) {
		t.Fatalf("zero SourceTime err = %v", err)
	}
	if err := (SourceTime{UTC: i}).Validate(); !errors.Is(err, ErrZoneRequired) {
		t.Fatalf("missing zone err = %v", err)
	}
	if err := (SourceTime{UTC: i, SourceZone: "Local"}).Validate(); !errors.Is(err, ErrZoneRequired) {
		t.Fatalf("Local zone err = %v", err)
	}
	st, err := FromInstant(i, loadZone(t, "Asia/Tehran"))
	if err != nil || st.RawText != nil || st.SourceZone != "Asia/Tehran" {
		t.Fatalf("FromInstant = %+v, %v", st, err)
	}
	if _, err := FromInstant(Instant{}, time.UTC); !errors.Is(err, ErrInvalidTime) {
		t.Fatalf("FromInstant(zero) err = %v", err)
	}
	if _, err := FromInstant(i, time.Local); !errors.Is(err, ErrZoneRequired) {
		t.Fatalf("FromInstant(Local) err = %v", err)
	}
}

func TestFixedClock(t *testing.T) {
	i, _ := ParseRFC3339("2026-10-29T00:00:00Z")
	var c Clock = FixedClock{At: i}
	if !c.Now().Equal(i) || !c.Now().Equal(c.Now()) {
		t.Fatal("FixedClock must be deterministic")
	}
}

func FuzzParseRFC3339(f *testing.F) {
	for _, s := range []string{"2024-03-20T10:00:00+03:30", "2024-03-20T10:00:00Z", "2024-03-20T10:00:00", ""} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		i, err := ParseRFC3339(s)
		if err != nil {
			return
		}
		if i.Time().Location() != time.UTC {
			t.Fatalf("ParseRFC3339(%q) not UTC", s)
		}
	})
}
