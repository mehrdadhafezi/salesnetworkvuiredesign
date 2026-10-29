package timeutil

import (
	"errors"
	"testing"
	"time"
)

// Fixed date table: Nowruz boundaries, Esfand 29/30 in common and leap years,
// and the Gregorian leap day.
var jalaliTable = []struct {
	g time.Time
	j JalaliDate
}{
	{time.Date(2020, 3, 20, 0, 0, 0, 0, time.UTC), JalaliDate{1399, 1, 1}},
	{time.Date(2021, 3, 20, 0, 0, 0, 0, time.UTC), JalaliDate{1399, 12, 30}}, // 1399 leap
	{time.Date(2021, 3, 21, 0, 0, 0, 0, time.UTC), JalaliDate{1400, 1, 1}},
	{time.Date(2023, 3, 20, 0, 0, 0, 0, time.UTC), JalaliDate{1401, 12, 29}}, // 1401 common
	{time.Date(2023, 3, 21, 0, 0, 0, 0, time.UTC), JalaliDate{1402, 1, 1}},
	{time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC), JalaliDate{1402, 12, 10}}, // Gregorian leap day
	{time.Date(2024, 3, 19, 0, 0, 0, 0, time.UTC), JalaliDate{1402, 12, 29}},
	{time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC), JalaliDate{1403, 1, 1}},
	{time.Date(2025, 3, 20, 0, 0, 0, 0, time.UTC), JalaliDate{1403, 12, 30}}, // 1403 leap
	{time.Date(2025, 3, 21, 0, 0, 0, 0, time.UTC), JalaliDate{1404, 1, 1}},
	{time.Date(2024, 9, 22, 0, 0, 0, 0, time.UTC), JalaliDate{1403, 7, 1}},
	{time.Date(2024, 9, 21, 0, 0, 0, 0, time.UTC), JalaliDate{1403, 6, 31}},
}

func TestJalaliTable(t *testing.T) {
	for _, row := range jalaliTable {
		got, err := ToJalali(row.g)
		if err != nil || got != row.j {
			t.Fatalf("ToJalali(%s) = %v, %v; want %v", row.g.Format("2006-01-02"), got, err, row.j)
		}
		back, err := FromJalali(row.j)
		if err != nil || back.Year != row.g.Year() || back.Month != row.g.Month() || back.Day != row.g.Day() {
			t.Fatalf("FromJalali(%v) = %+v, %v", row.j, back, err)
		}
	}
}

func TestJalaliLeapYears(t *testing.T) {
	leap := map[int]bool{1395: true, 1399: true, 1403: true, 1408: true, 1400: false, 1401: false, 1402: false, 1404: false}
	for y, want := range leap {
		got, err := IsJalaliLeapYear(y)
		if err != nil || got != want {
			t.Fatalf("IsJalaliLeapYear(%d) = %v, %v; want %v", y, got, err, want)
		}
		n, _ := JalaliMonthLength(y, 12)
		if (n == 30) != want {
			t.Fatalf("Esfand %d length = %d", y, n)
		}
	}
	if n, _ := JalaliMonthLength(1403, 6); n != 31 {
		t.Fatalf("Shahrivar length = %d", n)
	}
	if n, _ := JalaliMonthLength(1403, 7); n != 30 {
		t.Fatalf("Mehr length = %d", n)
	}
}

func TestJalaliInvalid(t *testing.T) {
	bad := []JalaliDate{{1402, 12, 30}, {1403, 0, 1}, {1403, 13, 1}, {1403, 7, 31}, {1403, 1, 0}, {-100, 1, 1}, {4000, 1, 1}}
	for _, d := range bad {
		if _, err := FromJalali(d); !errors.Is(err, ErrInvalidJalaliDate) {
			t.Fatalf("FromJalali(%v) err = %v", d, err)
		}
	}
	if _, err := IsJalaliLeapYear(5000); !errors.Is(err, ErrInvalidJalaliDate) {
		t.Fatalf("out-of-range year err = %v", err)
	}
	if _, err := ToJalali(time.Date(4500, 1, 1, 0, 0, 0, 0, time.UTC)); !errors.Is(err, ErrInvalidJalaliDate) {
		t.Fatalf("out-of-range Gregorian err = %v", err)
	}
	if (JalaliDate{1403, 1, 1}).String() != "1403/01/01" {
		t.Fatal("String format")
	}
}

// Property: every day in a wide range round-trips Gregorian → Jalali → Gregorian.
func TestJalaliRoundTripRange(t *testing.T) {
	start := time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)
	for d := start; d.Year() < 2100; d = d.AddDate(0, 0, 1) {
		j, err := ToJalali(d)
		if err != nil {
			t.Fatalf("ToJalali(%s): %v", d.Format("2006-01-02"), err)
		}
		if err := j.Validate(); err != nil {
			t.Fatalf("ToJalali(%s) produced invalid %v", d.Format("2006-01-02"), j)
		}
		g, err := FromJalali(j)
		if err != nil || g.Year != d.Year() || g.Month != d.Month() || g.Day != d.Day() {
			t.Fatalf("round trip %s -> %v -> %+v (%v)", d.Format("2006-01-02"), j, g, err)
		}
	}
}
