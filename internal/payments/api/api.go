// Package api is the published contract of the payments module: neutral
// import records built by the integration connector, and the read surface
// billing and reconciliation use. Invoice references are resolved through
// the published billing API; payments never reads or writes billing tables.
package api

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
)

// ErrInvalidImport reports a command that violates the contract.
var ErrInvalidImport = errors.New("payments: invalid import")

var (
	keyPattern  = regexp.MustCompile(`^[1-9][0-9]{0,19}$`)
	uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	specPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*\.[a-z][a-z0-9_]*\.v[1-9][0-9]*$`)
	sysPattern  = regexp.MustCompile(`^[a-z][a-z0-9_]{1,15}$`)
	nsPattern   = regexp.MustCompile(`^[a-z][a-z0-9_]*\.[a-z][a-z0-9_]*$`)
	unitPattern = regexp.MustCompile(`^(IRR|IRT)$`)
	hostPattern = regexp.MustCompile(`^[a-z0-9.-]{1,253}$`)
	extPattern  = regexp.MustCompile(`^[a-z0-9]{1,8}$`)
)

// SourceRef is the provenance of one staged source row.
type SourceRef struct {
	System, Spec, Key, BatchID string
	RowSHA256                  []byte
}

// Validate checks a source reference.
func (s SourceRef) Validate() error {
	if !sysPattern.MatchString(s.System) || !specPattern.MatchString(s.Spec) || !keyPattern.MatchString(s.Key) ||
		!uuidPattern.MatchString(s.BatchID) || len(s.RowSHA256) != 32 {
		return fmt.Errorf("%w: source ref %s/%s", ErrInvalidImport, s.Spec, s.Key)
	}
	return nil
}

// RowRef names a staged row by spec and key.
type RowRef struct{ Spec, Key string }

func (r RowRef) valid() bool { return specPattern.MatchString(r.Spec) && keyPattern.MatchString(r.Key) }

// UserRef is a legacy user reference with its identity user id if resolved.
type UserRef struct {
	LegacyKey string
	UserID    *string
}

func (u *UserRef) valid() bool {
	return u == nil || (keyPattern.MatchString(u.LegacyKey) && (u.UserID == nil || uuidPattern.MatchString(*u.UserID)))
}

// RawMoney is raw decimal text with its explicit unit; never converted.
type RawMoney struct {
	Raw  *string
	Unit string
}

// Channels (evidence-based; never collapsed).
const (
	ChannelGateway    = "GATEWAY"
	ChannelManualCard = "MANUAL_CARD"
	ChannelUnknown    = "UNKNOWN"
)

// Approval provenance of an accepted attempt (D-25 open). GATEWAY_AUTO is
// the legacy gateway auto-approval and is never a Finance review.
const (
	ProvGatewayAuto          = "GATEWAY_AUTO"
	ProvLegacyManualApproval = "LEGACY_MANUAL_APPROVAL"
	ProvNotApproved          = "NOT_APPROVED"
	ProvUnknown              = "UNKNOWN"
)

var (
	channels    = map[string]bool{ChannelGateway: true, ChannelManualCard: true, ChannelUnknown: true}
	provenances = map[string]bool{ProvGatewayAuto: true, ProvLegacyManualApproval: true, ProvNotApproved: true, ProvUnknown: true}
	attempts    = map[string]bool{"PENDING": true, "ACCEPTED": true, "REJECTED": true, "CANCELLED": true, "RECONCILIATION_REQUIRED": true}
	gateways    = map[string]bool{"INITIATED": true, "VERIFIED": true, "CANCELLED_AT_GATEWAY": true}
)

// GatewayRecord is the gateway transaction evidence of an attempt. Raw
// references are never carried: only their SHA-256.
type GatewayRecord struct {
	ProviderRaw                *string
	AuthoritySHA256, RefSHA256 []byte
	StateRaw                   *string
	StateTarget                *string
}

// AttemptRecord is one staged payment attempt.
type AttemptRecord struct {
	Source                         SourceRef
	Invoice                        RowRef
	StageNoRaw                     *string
	Channel                        string
	PayMethodRaw, PaymentSourceRaw *string
	Amount                         RawMoney
	StateNamespace                 string
	StateRaw, StateTarget          *string
	ApprovalProvenance             string
	UploadedByTypeRaw              *string
	UploadedBy                     *UserRef
	CreatedAtRaw, UpdatedAtRaw     *string
	Gateway                        *GatewayRecord
}

// Receipt states supported by Slice 7 evidence. A reference is never
// AVAILABLE without a file check (none happens in Slice 7).
const (
	ReceiptReferenceOnly = "REFERENCE_ONLY"
	ReceiptUnknown       = "UNKNOWN"
)

// ReceiptRecord is one receipt reference (metadata only; no URL or path).
type ReceiptRecord struct {
	Source     SourceRef // the stage or invoice row carrying the reference
	OwnerKind  string    // STAGE | INVOICE
	Invoice    RowRef
	StageNoRaw *string
	RefIndex   int
	RefSHA256  []byte
	Host       *string
	Extension  *string
	State      string
}

// PaymentsImport is one batch. AttemptsComplete / ReceiptSourcesComplete say
// whether the attempt and receipt-bearing source entities are COMPLETE.
type PaymentsImport struct {
	BatchID, SnapshotID, SourceSystem string
	AttemptsComplete                  bool
	ReceiptSourcesComplete            bool
	Attempts                          []AttemptRecord
	Receipts                          []ReceiptRecord
}

// Validate checks the command contract.
func (in PaymentsImport) Validate() error {
	if !uuidPattern.MatchString(in.BatchID) || in.SnapshotID == "" || !sysPattern.MatchString(in.SourceSystem) {
		return fmt.Errorf("%w: batch/snapshot/source system", ErrInvalidImport)
	}
	for _, a := range in.Attempts {
		if err := a.Source.Validate(); err != nil {
			return err
		}
		bad := a.Source.BatchID != in.BatchID || !a.Invoice.valid() || !channels[a.Channel] || !provenances[a.ApprovalProvenance] ||
			!nsPattern.MatchString(a.StateNamespace) || (a.StateTarget != nil && !attempts[*a.StateTarget]) || !a.UploadedBy.valid() ||
			(a.Amount.Raw != nil && !unitPattern.MatchString(a.Amount.Unit))
		// Provenance is only meaningful for accepted attempts.
		accepted := a.StateTarget != nil && *a.StateTarget == "ACCEPTED"
		bad = bad || (!accepted && a.ApprovalProvenance != ProvNotApproved && a.ApprovalProvenance != ProvUnknown) ||
			(a.ApprovalProvenance == ProvGatewayAuto && a.Channel != ChannelGateway)
		if g := a.Gateway; g != nil {
			bad = bad || a.Channel != ChannelGateway || (g.StateTarget != nil && !gateways[*g.StateTarget]) ||
				(g.AuthoritySHA256 != nil && len(g.AuthoritySHA256) != 32) || (g.RefSHA256 != nil && len(g.RefSHA256) != 32)
		}
		if bad {
			return fmt.Errorf("%w: attempt %s", ErrInvalidImport, a.Source.Key)
		}
	}
	for _, r := range in.Receipts {
		if err := r.Source.Validate(); err != nil {
			return err
		}
		if (r.OwnerKind != "STAGE" && r.OwnerKind != "INVOICE") || !r.Invoice.valid() || r.RefIndex < 0 ||
			(r.State != ReceiptReferenceOnly && r.State != ReceiptUnknown) || (r.RefSHA256 != nil && len(r.RefSHA256) != 32) ||
			(r.Host != nil && !hostPattern.MatchString(*r.Host)) || (r.Extension != nil && !extPattern.MatchString(*r.Extension)) {
			return fmt.Errorf("%w: receipt %s/%d", ErrInvalidImport, r.Source.Key, r.RefIndex)
		}
	}
	return nil
}

// ImportReport is the outcome of one import.
type ImportReport struct {
	Outcome result.Status
	Replay  bool
	RunID   string
	Counts  map[string]int64
}

// UnitSum is the accepted-attempt sum of one invoice in one unit.
type UnitSum struct {
	Unit     string
	Amount   *int64
	State    string // COMPLETE | INCOMPLETE | NONE | OVERFLOW
	Accepted int
}

// InvoicePayments is the accepted-attempt evidence of one invoice. Complete
// is false when the attempt source of the latest import was not COMPLETE,
// or the invoice has attempts whose state is UNKNOWN/EMPTY or whose accepted
// amount is not representable: a missing contribution is never zero.
type InvoicePayments struct {
	Units        map[string]UnitSum
	Unclassified int64
	Complete     bool
}

// RowProvenance is the stored provenance of one attempt row.
type RowProvenance struct {
	BatchID   string
	RowSHA256 []byte
	Present   bool
}

// Reader is the published payments read surface.
type Reader interface {
	// AcceptedSums returns, per invoice id and unit, the sum of attempts
	// explicitly mapped ACCEPTED. UNKNOWN attempts are never counted.
	// Invoices without any attempt are present with empty Units.
	AcceptedSums(ctx context.Context, invoiceIDs []string) (map[string]InvoicePayments, error)
	// RowProvenance maps attempt source keys of one spec to provenance.
	RowProvenance(ctx context.Context, system, spec string, keys []string) (map[string]RowProvenance, error)
	// BatchImported reports whether a payments import run exists for a batch.
	BatchImported(ctx context.Context, batchID string) (bool, error)
	// Coverage is the payments coverage report.
	Coverage(ctx context.Context) (Coverage, error)
}

// Coverage is the machine-readable payments coverage report.
type Coverage struct {
	Attempts           int64            `json:"attempts_present"`
	ByChannel          map[string]int64 `json:"attempts_by_channel"`
	ByStateMapping     map[string]int64 `json:"attempt_state_mapping"`
	ByTarget           map[string]int64 `json:"attempt_states"`
	UnknownRaw         map[string]int64 `json:"unknown_raw_attempt_states"`
	ApprovedRaw        int64            `json:"approved_raw_count"`
	ReconcileErrorRaw  int64            `json:"reconcile_error_raw_count"`
	ByProvenance       map[string]int64 `json:"approval_provenance"`
	GatewayAttempts    int64            `json:"gateway_attempts"`
	GatewayStates      map[string]int64 `json:"gateway_states"`
	WithoutInvoice     int64            `json:"attempts_without_invoice"`
	Unrepresentable    int64            `json:"unrepresentable_amounts"`
	StateRegressions   int64            `json:"attempt_state_regressions"`
	ReceiptsReferenced int64            `json:"receipts_referenced"`
	ReceiptsUnknown    int64            `json:"receipts_unknown"`
	ReceiptsMissing    int64            `json:"receipts_missing"`
}
