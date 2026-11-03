package api

import "context"

// PaymentTarget is the billing evidence the payments module needs to decide
// whether a manual payment submission may be accepted for an invoice's
// current stage (Slice 12, Architecture V1 §15.2 rule 3). It is read-only
// published evidence: payments never reads billing tables and billing never
// changes a stage or an invoice because of a submission.
type PaymentTarget struct {
	InvoiceID string
	// Source is the invoice's legacy provenance (system, spec, key); the
	// write-back connector needs it to address the legacy invoice.
	SourceSystem, SourceSpec, SourceKey string
	Present                             bool
	SellerUserID                        *string
	IssuedByUserID                      *string
	// MainStatusRaw / MainStatusMapping are the legacy invoice main status
	// evidence (namespaced; no target lifecycle is implied).
	MainStatusRaw     *string
	MainStatusMapping Vocabulary
	CurrentStageRaw   *string
	PaymentPlanRaw    *string
	StagesComplete    bool
	// Stage is the current stage named by CurrentStageRaw, nil when the
	// invoice names none or no single PRESENT stage row matches it.
	Stage *PaymentTargetStage
	// StageAmbiguous is true when several stage rows match the current stage.
	StageAmbiguous bool
	// Remaining is the declared REMAINING_AMOUNT evidence (never derived).
	Remaining Money
	Version   int64
}

// PaymentTargetStage is one payment stage as billing holds it.
type PaymentTargetStage struct {
	StageID      string
	StageNo      int
	StateRaw     *string
	StateMapping string
	StateTarget  *string
	Requested    Money
}

// Money is a representable integer amount in one explicit unit, or not.
type Money struct {
	Amount        int64
	Unit          string
	Representable bool
}

// PaymentTargetReader is the published read payments uses. ok is false for
// an unknown invoice.
type PaymentTargetReader interface {
	PaymentTarget(ctx context.Context, invoiceID string) (t PaymentTarget, ok bool, err error)
}
