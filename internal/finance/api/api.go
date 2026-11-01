// Package api is the published finance contract. In Slice 9 it carries only
// the normalized legacy wallet evidence port: integration maps the legacy
// wallet sources into these neutral records (no legacy table name, column
// name or contact data reaches finance) and the offline shadow rebuild
// consumes them. Finance publishes no command.
package api

import (
	"context"
	"time"
)

// Source coverage of one evidence source.
const (
	SourceAvailable      = "AVAILABLE"
	SourcePartial        = "PARTIAL"
	SourceMissing        = "MISSING"
	SourceUnknown        = "UNKNOWN"
	SourceNotRecoverable = "NOT_RECOVERABLE"
	SourceStale          = "STALE"
)

// Amount representations: an absent amount is never zero; an amount that is
// not an exact integer in its unit is UNREPRESENTABLE (never rounded).
const (
	Representable   = "REPRESENTABLE"
	Unrepresentable = "UNREPRESENTABLE"
	Absent          = "ABSENT"
)

// RawAmount is one raw legacy amount with its evidenced unit. Finance parses
// it exactly (no rounding, no conversion); Unit nil = UNKNOWN unit.
type RawAmount struct {
	Raw  *string
	Unit *string
}

// Wallet categories (what the legacy wallet holds, by verified writer).
const (
	// WalletCommission: a staff commission wallet (verified writers only).
	WalletCommission = "COMMISSION_PAYABLE"
	// WalletUnclassified: a wallet type no verified writer produces; its
	// meaning (entitlement, stored value, other) is NOT decided (OPD-09).
	WalletUnclassified = "UNCLASSIFIED"
)

// Commission engine routes of a legacy wallet transaction (OPD-09: reported,
// never decided; no engine is activated).
const (
	RouteLegacyRules   = "LEGACY_RULES"
	RoutePurposeMatrix = "PURPOSE_MATRIX"
	RouteBiawinProject = "BIAWIN_PROJECT"
	RouteFinanceReopen = "FINANCE_REOPEN"
	RouteUnresolved    = "UNRESOLVED"
)

// Transaction status / direction classes.
const (
	StatusApproved = "APPROVED"
	StatusUnknown  = "UNKNOWN"
	Credit         = "CREDIT"
	Debit          = "DEBIT"
	DirUnknown     = "UNKNOWN"
)

// LegacyWallet is one legacy wallet (balance evidence). Subject is the
// historical holder: the legacy user key and, when the identity published
// API resolves it, the target user. Never a phone, name or email.
type LegacyWallet struct {
	WalletKey     string
	SubjectKey    string
	SubjectUserID *string
	WalletTypeRaw string
	Category      string
	Balance       RawAmount
	TotalCredit   RawAmount
	TotalDebit    RawAmount
	UpdatedAtRaw  *string
}

// LegacyWalletTx is one legacy wallet transaction (posting evidence).
type LegacyWalletTx struct {
	TxKey, WalletKey string
	SubjectKey       string
	WalletTypeRaw    string
	Direction        string
	DirectionRaw     *string
	TypeRaw          string
	Route            string
	Status           string
	StatusRaw        *string
	Amount           RawAmount
	// InvoiceKey is the legacy invoice reference; InvoiceID the billing
	// invoice the billing published API resolves it to (nil = unresolved).
	InvoiceKey *string
	InvoiceID  *string
	// ReversesTxKey is the legacy transaction this one says it reverses
	// (from the writer's own link field); nil when the row claims none.
	ReversesTxKey *string
	// RewrittenInPlace: the legacy row's type shows it was rewritten by a
	// later legacy reversal / reopen (mutable legacy history).
	RewrittenInPlace bool
	// IdempotencyKeyPresent: the legacy writer recorded an idempotency key.
	IdempotencyKeyPresent bool
	// SourceTimeRaw is the legacy DATETIME text (site-local; zone not
	// asserted). Never converted, never replaced by an import time.
	SourceTimeRaw *string
}

// WalletEvidence is one staged legacy snapshot of wallets and transactions.
type WalletEvidence struct {
	SnapshotID string
	// Cutoff is the snapshot's extraction time (UTC).
	Cutoff            time.Time
	WalletStatus      string
	TransactionStatus string
	Reason            string
	Wallets           []LegacyWallet
	Transactions      []LegacyWalletTx
}

// WalletSource is the normalized legacy wallet evidence port, implemented by
// integration over staging. It never reads a live legacy system.
type WalletSource interface {
	WalletEvidence(ctx context.Context) (WalletEvidence, error)
}
