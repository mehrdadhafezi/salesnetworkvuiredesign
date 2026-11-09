package db

import (
	"sort"
	"sync"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
)

// PoolStat is a point-in-time view of one open pool, labelled by the owner
// module it serves (R3 observability). It holds counts only: no DSN, role
// or host.
type PoolStat struct {
	Owner            string
	Max, Total, Idle int32
	Acquired         int32
	// EmptyAcquire counts acquisitions that had to wait for a connection
	// (saturation signal).
	EmptyAcquire int64
}

// registry tracks open pools for observability only; it never hands a
// pool out.
var registry = struct {
	sync.Mutex
	pools map[*Pool]owner.Module
}{pools: map[*Pool]owner.Module{}}

func track(p *Pool, m owner.Module) {
	registry.Lock()
	defer registry.Unlock()
	if prev, ok := registry.pools[p]; !ok || prev == "" {
		registry.pools[p] = m
	}
}

func untrack(p *Pool) {
	registry.Lock()
	defer registry.Unlock()
	delete(registry.pools, p)
}

// Stats returns one entry per owner, summing every open pool that owner
// holds (a process may open several pools for one owner, e.g. one per API
// surface), sorted by owner, so each owner is exactly one metric series.
// Pools not yet bound to an owner report "unbound".
func Stats() []PoolStat {
	registry.Lock()
	defer registry.Unlock()
	byOwner := map[string]*PoolStat{}
	for p, m := range registry.pools {
		if p.pool == nil {
			continue
		}
		s := p.pool.Stat()
		name := string(m)
		if name == "" {
			name = "unbound"
		}
		agg := byOwner[name]
		if agg == nil {
			agg = &PoolStat{Owner: name}
			byOwner[name] = agg
		}
		agg.Max += s.MaxConns()
		agg.Total += s.TotalConns()
		agg.Idle += s.IdleConns()
		agg.Acquired += s.AcquiredConns()
		agg.EmptyAcquire += s.EmptyAcquireCount()
	}
	out := make([]PoolStat, 0, len(byOwner))
	for _, agg := range byOwner {
		out = append(out, *agg)
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Owner < out[b].Owner })
	return out
}
