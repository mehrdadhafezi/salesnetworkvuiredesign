package posting

import (
	"sync"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/money"
)

// Journal is an in-memory SYNTHETIC append-only store (tests, contract
// proofs). It is not a ledger of record and has no persistence.
type Journal struct {
	mu       sync.Mutex
	byKey    map[string]Recorded
	order    []string
	reversed map[string]bool
}

// NewJournal returns an empty synthetic journal.
func NewJournal() *Journal {
	return &Journal{byKey: map[string]Recorded{}, reversed: map[string]bool{}}
}

// Lookup implements Store.
func (j *Journal) Lookup(key string) (Recorded, bool, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	r, ok := j.byKey[key]
	return r, ok, nil
}

// Reversed implements Store.
func (j *Journal) Reversed(key string) (bool, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.reversed[key], nil
}

// Append implements Store.
func (j *Journal) Append(r Recorded) (bool, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if _, ok := j.byKey[r.Intent.BusinessKey]; ok {
		return false, nil
	}
	if r.Intent.Reverses != nil {
		if j.reversed[*r.Intent.Reverses] {
			return false, nil
		}
		j.reversed[*r.Intent.Reverses] = true
	}
	j.byKey[r.Intent.BusinessKey] = r
	j.order = append(j.order, r.Intent.BusinessKey)
	return true, nil
}

// Len is the number of recorded intents.
func (j *Journal) Len() int {
	j.mu.Lock()
	defer j.mu.Unlock()
	return len(j.order)
}

// Net returns debit minus credit of account in unit (overflow reported).
func (j *Journal) Net(account string, unit money.Unit) (int64, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	total := money.Money{Unit: unit}
	for _, k := range j.order {
		r := j.byKey[k]
		if r.Intent.Unit != unit {
			continue
		}
		for _, l := range r.Intent.Lines {
			if l.Account != account {
				continue
			}
			v := money.Money{Amount: l.Amount, Unit: unit}
			var err error
			if l.Side == Credit {
				if v, err = money.Neg(v); err != nil {
					return 0, err
				}
			}
			if total, err = money.Add(total, v); err != nil {
				return 0, err
			}
		}
	}
	return total.Amount, nil
}
