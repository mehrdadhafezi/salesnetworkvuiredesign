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
	// CustomerID is the Slice 5 customer only when CustomerLink is RESOLVED
	// (never guessed; Slice 9 finance keys nothing on an unresolved link).
	CustomerID                    *string
	SellerKey                     *string
	States                        map[StateField]StateFact
	ItemCount, StageCount         int64
	UnknownStates                 int64
	ItemsComplete, StagesComplete bool
	FirstSeenAt                   time.Time
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

// AmountEvidence is one unit's invoice amount evidence for the finance
// shadow ledger (Slice 9). Class compares the declared totals with the item
// and stage sums only: payment evidence is deliberately NOT compared, since
// a receipt is not invoice-amount evidence. No value is canonical (TAD #19).
// LegacyPaid / LegacyRemaining are the legacy declared paid / remaining
// figures in this unit: reconciliation evidence only, never a posting basis.
type AmountEvidence struct {
	Unit                                string
	Declared                            map[string]*int64
	ItemSum, StageSum                   PaymentUnitSum
	Class                               string
	LegacyPaid, LegacyRemaining         *int64
	LegacyPaidRepr, LegacyRemainingRepr string // REPRESENTABLE / UNREPRESENTABLE / ABSENT / OTHER_UNIT
}

// OtherUnit marks a legacy figure declared in a different unit than the
// evidence row: it is never compared or converted across units.
const OtherUnit = "OTHER_UNIT"

// AmountEvidenceSource is the published, read-only invoice amount evidence
// (Slice 9 finance shadow ledger), read in one billing snapshot.
type AmountEvidenceSource interface {
	InvoiceAmountEvidence(ctx context.Context, invoiceIDs []string) (map[string][]AmountEvidence, error)
}
