package ids

import (
	"errors"
	"testing"
)

func TestNewUUID(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		id, err := NewUUID()
		if err != nil {
			t.Fatal(err)
		}
		if err := ValidateUUID(id); err != nil {
			t.Fatalf("generated %q invalid: %v", id, err)
		}
		if id[14] != '4' || (id[19] != '8' && id[19] != '9' && id[19] != 'a' && id[19] != 'b') {
			t.Fatalf("%q is not a v4 RFC 9562 UUID", id)
		}
		if seen[id] {
			t.Fatalf("duplicate %q", id)
		}
		seen[id] = true
	}
	for _, bad := range []string{"", "09123456789", "00000000-0000-0000-0000-00000000000", "00000000-0000-4000-8000-00000000000G", "AAAAAAAA-0000-4000-8000-000000000000"} {
		if err := ValidateUUID(bad); !errors.Is(err, ErrInvalidUUID) {
			t.Fatalf("%q accepted", bad)
		}
	}
}
