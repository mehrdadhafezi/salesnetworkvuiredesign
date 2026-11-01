package api

import (
	"context"
	"errors"
)

// ErrTooManyFacts reports an enumeration beyond the published bound.
var ErrTooManyFacts = errors.New("custody: fact enumeration exceeds the published bound")

// MaxFacts bounds one enumeration.
const MaxFacts = 500000

// AssignmentFact is the published reporting fact of one derived custody
// projection. Current custody (derived holder) and the legacy holder
// evidence stay separate; parity is not a holder.
type AssignmentFact struct {
	CaseID, Derivation, Parity string
	DerivedKey, LegacyKey      *string
	EventCount                 int
	FindingCount               int64
	Presence                   string
}

// FactSource is the published, read-only fact enumeration for reporting.
type FactSource interface {
	AssignmentFacts(ctx context.Context, fn func(AssignmentFact) error) error
}
