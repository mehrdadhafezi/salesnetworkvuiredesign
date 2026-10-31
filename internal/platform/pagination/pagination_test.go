package pagination

import (
	"errors"
	"strings"
	"testing"
)

func TestRoundTripAndSortBinding(t *testing.T) {
	c := Cursor{Sort: "display_name", Value: "Dev User — کاربر", ID: "0b9c2f5e-1111-4111-8111-111111111111"}
	s, err := Encode(c)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(s, "display_name")
	if err != nil || got != c {
		t.Fatalf("decode = %+v %v", got, err)
	}
	if _, err := Decode(s, "-display_name"); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("cursor accepted for another sort: %v", err)
	}
	for _, bad := range []string{"", "!!!", strings.Repeat("a", MaxCursorLen+1), "e30"} {
		if _, err := Decode(bad, "display_name"); !errors.Is(err, ErrInvalidCursor) {
			t.Fatalf("%q accepted", bad)
		}
	}
}

func TestLimit(t *testing.T) {
	if n, err := Limit(nil); err != nil || n != DefaultLimit {
		t.Fatalf("default = %d %v", n, err)
	}
	for _, v := range []int{0, -1, MaxLimit + 1} {
		v := v
		if _, err := Limit(&v); err == nil {
			t.Fatalf("limit %d accepted", v)
		}
	}
	v := MaxLimit
	if n, err := Limit(&v); err != nil || n != MaxLimit {
		t.Fatalf("max = %d %v", n, err)
	}
}
