package api

import (
	"context"
	"errors"
	"time"
)

// ErrTooManyFacts reports an enumeration beyond the published bound.
var ErrTooManyFacts = errors.New("billing: fact enumeration exceeds the published bound")

// MaxFacts bounds one enumeration.
const MaxFacts = 500000

// StateFact is one raw invoice state field with its vocabulary membership.
type StateFact struct {
	Raw     *string
	Mapping string
}

// InvoiceFact is the published, PII-free reporting fact of one invoice.
// Every legacy state field stays separate; the lifecycle is never claimed.
type InvoiceFact struct {
	InvoiceID, SourceSpec, SourceKey string
	CodeRaw                          *string
	Presence                         string
	CustomerLink, CaseLink           string
	SellerKey                        *string
	States                           map[StateField]StateFact
	ItemCount, StageCount            int64
	UnknownStates                    int64
	ItemsComplete, StagesComplete    bool
	FirstSeenAt                      time.Time
}

// UnitTotal is the published per-unit total evidence of one invoice. No
// value is canonical (TAD #19); units are never combined.
type UnitTotal struct {
	Unit                      string
	Declared                  map[string]*int64
	ItemSum, StageSum, PaySum PaymentUnitSum
	Class                     string
}

// FactSource is the published, read-only fact enumeration for reporting.
type FactSource interface {
	InvoiceFacts(ctx context.Context, fn func(InvoiceFact) error) error
	// InvoiceUnitTotals classifies the per-unit total evidence of invoices
	// given the payments evidence (absent = INCOMPLETE), in one snapshot.
	InvoiceUnitTotals(ctx context.Context, invoiceIDs []string, payments map[string]PaymentEvidence) (map[string][]UnitTotal, error)
}
