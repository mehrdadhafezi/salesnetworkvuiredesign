package api

import (
	"context"
	"errors"
	"time"
)

// ErrTooManyFacts reports an enumeration beyond the published bound; the
// consumer must treat its projection as PARTIAL, never complete.
var ErrTooManyFacts = errors.New("cases: fact enumeration exceeds the published bound")

// MaxFacts bounds one enumeration (bounded rebuild, Architecture V1 §20).
const MaxFacts = 500000

// AttributionFact is the latest immutable attribution snapshot of one facet.
type AttributionFact struct {
	Resolution string
	SubjectKey *string
}

// CaseFact is the published, PII-free reporting fact of one Case. It names
// no phone, name or customer contact; ownership facets stay separate.
type CaseFact struct {
	CaseID, Workflow, StateNamespace string
	StateRaw                         *string
	StateMapping                     string
	CustomerLinkState                string
	Presence                         string
	SourceSpec, SourceKey            string
	AliasCount                       int64
	// FirstSeenAt is when the target import first observed the source row
	// (UTC); it is not a legacy business time.
	FirstSeenAt time.Time
	// Attribution maps a facet (ORIGINAL_OWNER, ORIGINAL_SELLER,
	// CREDIT_OWNER) to its latest snapshot; a missing facet has none.
	Attribution map[string]AttributionFact
}

// FactSource is the published, read-only fact enumeration used by reporting
// projections. One call reads one consistent read-only snapshot.
type FactSource interface {
	CaseFacts(ctx context.Context, fn func(CaseFact) error) error
}
