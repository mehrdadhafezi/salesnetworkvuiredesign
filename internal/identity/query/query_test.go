package query_test

import (
	"context"
	"errors"
	"sort"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	identitymigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pgtest"
)

// The stored capability vocabulary and role bundles are exactly the code ones.
func TestStoredVocabularyMatchesCode(t *testing.T) {
	e := pgtest.New(t)
	if _, err := migrations.Up(context.Background(), e.DSN, identitymigrations.Source()); err != nil {
		t.Fatal(err)
	}
	s := query.New(e.Runner(t, identity.Module))
	caps, bundles, err := s.Vocabulary(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(caps) != len(api.Capabilities()) {
		t.Fatalf("stored %d capabilities, code %d", len(caps), len(api.Capabilities()))
	}
	for _, c := range api.Capabilities() {
		if caps[string(c.Key)] != c.Description {
			t.Fatalf("capability %s differs", c.Key)
		}
	}
	if len(bundles) != len(api.Roles()) {
		t.Fatalf("stored %d roles, code %d", len(bundles), len(api.Roles()))
	}
	for _, r := range api.Roles() {
		var want []string
		for _, c := range r.Capabilities {
			want = append(want, string(c))
		}
		sort.Strings(want)
		got := bundles[string(r.Key)]
		if len(got) != len(want) {
			t.Fatalf("role %s bundle %v, code %v", r.Key, got, want)
		}
		for i := range got {
			if got[i] != want[i] {
				t.Fatalf("role %s bundle %v, code %v", r.Key, got, want)
			}
		}
	}
	if _, err := s.ListUsers(context.Background(), query.ListParams{Sort: "login"}); !errors.Is(err, query.ErrInvalidQuery) {
		t.Fatalf("unknown sort accepted: %v", err)
	}
}
