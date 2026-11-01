package api

import (
	"context"
	"errors"
	"time"
)

// ErrTooManyFacts reports an enumeration beyond the published bound.
var ErrTooManyFacts = errors.New("payments: fact enumeration exceeds the published bound")

// MaxFacts bounds one enumeration.
const MaxFacts = 500000

// AttemptFact is the published reporting fact of one payment attempt. The
// attempt, gateway state and approval provenance stay separate; no gateway
// reference is exposed.
type AttemptFact struct {
	AttemptID            string
	InvoiceID            *string
	InvoiceLink, Channel string
	StateRaw             *string
	StateMapping         string
	StateTarget          *string
	Provenance           string
	AmountRaw            *string
	Amount               *int64
	Unit                 *string
	AmountRepr           string
	GatewayState         *string
	Presence             string
	FirstSeenAt          time.Time
}

// FactSource is the published, read-only fact enumeration for reporting.
type FactSource interface {
	AttemptFacts(ctx context.Context, fn func(AttemptFact) error) error
}
