package db

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
)

// R3: several pools held for one owner (e.g. one per API surface) are
// reported as ONE entry per owner, so the metric exposition never carries
// duplicate series. pgxpool connects lazily, so no database is needed.
func TestStatsAggregatesPoolsByOwner(t *testing.T) {
	open := func(max int32) *Pool {
		pc, err := pgxpool.ParseConfig("postgres://stats@127.0.0.1:1/stats")
		if err != nil {
			t.Fatal(err)
		}
		pc.MaxConns = max
		pc.MinConns = 0
		pp, err := pgxpool.NewWithConfig(context.Background(), pc)
		if err != nil {
			t.Fatal(err)
		}
		p := &Pool{pool: pp}
		t.Cleanup(p.Close)
		return p
	}
	const a, b = owner.Module("r3_stats_a"), owner.Module("r3_stats_b")
	track(open(3), a)
	track(open(4), a)
	track(open(5), b)

	seen := map[string]PoolStat{}
	for _, s := range Stats() {
		if _, dup := seen[s.Owner]; dup {
			t.Fatalf("owner %q reported twice", s.Owner)
		}
		seen[s.Owner] = s
	}
	if got := seen[string(a)].Max; got != 7 {
		t.Fatalf("owner a max = %d, want 7 (3+4)", got)
	}
	if got := seen[string(b)].Max; got != 5 {
		t.Fatalf("owner b max = %d, want 5", got)
	}
}
