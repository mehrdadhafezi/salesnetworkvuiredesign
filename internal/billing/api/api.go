// Package api is the published contract of the billing module: neutral
// import records built by the integration connector, and the read surface
// payments and reconciliation use. No legacy table or column names; legacy
// keys are provenance only.
package api

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
)

// ErrInvalidImport reports a command that violates the contract.
var ErrInvalidImport = errors.New("billing: invalid import")

var (
	keyPattern  = regexp.MustCompile(`^[1-9][0-9]{0,19}$`)
	uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	specPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*\.[a-z][a-z0-9_]*\.v[1-9][0-9]*$`)
	sysPattern  = regexp.MustCompile(`^[a-z][a-z0-9_]{1,15}$`)
	nsPattern   = regexp.MustCompile(`^[a-z][a-z0-9_]*\.[a-z][a-z0-9_]*$`)
	unitPattern = regexp.MustCompile(`^(IRR|IRT)$`)
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

// RawMoney is raw decimal text with its EXPLICIT unit (IRR or IRT), as the
// connector proved it from source evidence. Never converted.
type RawMoney struct {
	Raw  *string
	Unit string
}

func (m RawMoney) valid() bool { return m.Raw == nil || unitPattern.MatchString(m.Unit) }

// Invoice state fields (each legacy status field stays separate).
type StateField string

const (
	FieldMain          StateField = "MAIN"
	FieldInvoice       StateField = "INVOICE"
	FieldPayment       StateField = "PAYMENT"
	FieldWorkflow      StateField = "WORKFLOW"
	FieldReturn        StateField = "RETURN"
	FieldLegacyDisplay StateField = "LEGACY_DISPLAY"
)

// Vocabulary membership of a raw invoice state value. There is no target
// mapping: the canonical invoice main-status resolver is deferred (TAD #20).
type Vocabulary string

const (
	KnownVocabulary Vocabulary = "KNOWN_VOCABULARY"
	UnknownValue    Vocabulary = "UNKNOWN"
	EmptyValue      Vocabulary = "EMPTY"
	LegacyDerived   Vocabulary = "LEGACY_DERIVED"
)

// StateEvidence is one raw invoice state field.
type StateEvidence struct {
	Field     StateField
	Namespace string
	Raw       *string
	Mapping   Vocabulary
}

// Link is an explicit reference to another module's aggregate.
type Link struct {
	ID    *string
	State string // RESOLVED | RELATED | UNKNOWN | CONFLICT | NOT_RECOVERABLE
	Basis string
}

var linkStates = map[string]bool{"RESOLVED": true, "RELATED": true, "UNKNOWN": true, "CONFLICT": true, "NOT_RECOVERABLE": true}

func (l Link) valid() bool {
	return linkStates[l.State] && l.Basis != "" && len(l.Basis) <= 300 && (l.ID == nil || uuidPattern.MatchString(*l.ID)) &&
		(l.State != "RESOLVED" || l.ID != nil)
}

// AmountKind names a declared invoice amount field.
type AmountKind string

// InvoiceRecord is one staged invoice row.
type InvoiceRecord struct {
	Source                               SourceRef
	CodeRaw, ProductRefRaw, PayMethodRaw *string
	PaymentPlanRaw, CurrentStageRaw      *string
	IssuedAtRaw, UpdatedAtRaw            *string
	Seller, IssuedBy                     *UserRef
	States                               []StateEvidence
	Declared                             map[AmountKind]RawMoney
	Customer, Case                       Link
}

// ItemRecord is one staged invoice line (issuance snapshot).
type ItemRecord struct {
	Source                                        SourceRef
	Invoice                                       RowRef
	ProductRefRaw, TitleSnapshot, KindRaw, QtyRaw *string
	UnitPrice, LineTotal                          RawMoney
	IsFreeRaw, CreatedAtRaw                       *string
}

// Stage states (explicitly mapped by the connector from verified writers).
var stageTargets = map[string]bool{"PENDING": true, "AWAITING_REVIEW": true, "ACCEPTED": true, "REJECTED": true, "CANCELLED": true}

// StageRecord is one staged payment stage.
type StageRecord struct {
	Source                                 SourceRef
	Invoice                                RowRef
	StageNoRaw, StageTypeRaw               *string
	Requested                              RawMoney
	StateNamespace                         string
	StateRaw                               *string
	StateTarget                            *string // only when explicitly mapped
	PayMethodRaw, PaymentSourceRaw         *string
	IssuedBy, ApprovedBy                   *UserRef
	ApprovedAtRaw, PaidAtRaw, CreatedAtRaw *string
}

// BillingImport is one batch. Complete names the specs whose staged entity
// is COMPLETE; only those may mark absence.
type BillingImport struct {
	BatchID, SnapshotID, SourceSystem string
	Complete                          map[string]bool
	// ItemsComplete / StagesComplete say whether the item and stage source
	// entities of this batch are COMPLETE; sums over an incomplete source
	// are INCOMPLETE, never final.
	ItemsComplete, StagesComplete bool
	Invoices                      []InvoiceRecord
	Items                         []ItemRecord
	Stages                        []StageRecord
}

// Validate checks the command contract.
func (in BillingImport) Validate() error {
	if !uuidPattern.MatchString(in.BatchID) || in.SnapshotID == "" || !sysPattern.MatchString(in.SourceSystem) {
		return fmt.Errorf("%w: batch/snapshot/source system", ErrInvalidImport)
	}
	for _, v := range in.Invoices {
		if err := v.Source.Validate(); err != nil {
			return err
		}
		if v.Source.BatchID != in.BatchID || !v.Seller.valid() || !v.IssuedBy.valid() || !v.Customer.valid() || !v.Case.valid() {
			return fmt.Errorf("%w: invoice %s", ErrInvalidImport, v.Source.Key)
		}
		seen := map[StateField]bool{}
		for _, s := range v.States {
			if seen[s.Field] || !nsPattern.MatchString(s.Namespace) {
				return fmt.Errorf("%w: state evidence of %s", ErrInvalidImport, v.Source.Key)
			}
			seen[s.Field] = true
			if (s.Mapping == LegacyDerived) != (s.Field == FieldLegacyDisplay) {
				return fmt.Errorf("%w: LEGACY_DERIVED is only for the legacy display field", ErrInvalidImport)
			}
			if (s.Raw == nil || *s.Raw == "") != (s.Mapping == EmptyValue) {
				return fmt.Errorf("%w: EMPTY iff raw is empty", ErrInvalidImport)
			}
		}
		for _, m := range v.Declared {
			if !m.valid() {
				return fmt.Errorf("%w: declared amount unit", ErrInvalidImport)
			}
		}
	}
	for _, it := range in.Items {
		if err := it.Source.Validate(); err != nil {
			return err
		}
		if !it.Invoice.valid() || !it.UnitPrice.valid() || !it.LineTotal.valid() {
			return fmt.Errorf("%w: item %s", ErrInvalidImport, it.Source.Key)
		}
	}
	for _, st := range in.Stages {
		if err := st.Source.Validate(); err != nil {
			return err
		}
		if !st.Invoice.valid() || !st.Requested.valid() || !nsPattern.MatchString(st.StateNamespace) || !st.IssuedBy.valid() || !st.ApprovedBy.valid() ||
			(st.StateTarget != nil && !stageTargets[*st.StateTarget]) {
			return fmt.Errorf("%w: stage %s", ErrInvalidImport, st.Source.Key)
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

// SourceInvoice is the stored provenance of one invoice source row.
type SourceInvoice struct {
	InvoiceID, BatchID string
	RowSHA256          []byte
	Present            bool
}

// RowProvenance is the stored provenance of an item or stage row.
type RowProvenance struct {
	BatchID   string
	RowSHA256 []byte
	Present   bool
}

// Reader is the published billing read surface.
type Reader interface {
	// ResolveInvoices maps invoice source keys of one spec to invoices.
	ResolveInvoices(ctx context.Context, system, spec string, keys []string) (map[string]SourceInvoice, error)
	// RowProvenance maps item or stage source keys of one spec to provenance.
	RowProvenance(ctx context.Context, system, spec string, keys []string) (map[string]RowProvenance, error)
	// Coverage is the billing coverage report.
	Coverage(ctx context.Context) (Coverage, error)
	// TotalClasses classifies, per invoice and unit, the total evidence
	// (declared vs item/stage/payment sums). payments is the accepted-attempt
	// evidence supplied by the payments published API; an invoice missing
	// from it has INCOMPLETE payment evidence.
	TotalClasses(ctx context.Context, invoiceIDs []string, payments map[string]PaymentEvidence) (map[string]map[string]string, error)
}

// PaymentUnitSum is one unit's accepted-attempt sum as payments reports it.
type PaymentUnitSum struct {
	Amount *int64
	State  string
	Rows   int
}

// PaymentEvidence is the accepted-attempt evidence of one invoice.
type PaymentEvidence struct {
	Units    map[string]PaymentUnitSum
	Complete bool
}

// Coverage is the machine-readable billing coverage report.
type Coverage struct {
	Invoices          int64                       `json:"invoices_present"`
	Items             int64                       `json:"items_present"`
	Stages            int64                       `json:"stages_present"`
	InvoicesNoItems   int64                       `json:"invoices_without_items"`
	InvoicesNoStages  int64                       `json:"invoices_without_stages"`
	StateMappings     map[string]map[string]int64 `json:"invoice_state_vocabulary_by_field"`
	UnknownRawStates  map[string]map[string]int64 `json:"unknown_raw_states"`
	StageStates       map[string]int64            `json:"stage_state_mapping"`
	UnknownStageRaw   map[string]int64            `json:"unknown_stage_raw_states"`
	CustomerLinks     map[string]int64            `json:"customer_link_states"`
	CaseLinks         map[string]int64            `json:"case_link_states"`
	Unrepresentable   int64                       `json:"unrepresentable_amounts"`
	PreInvoiceRawSeen int64                       `json:"invoices_with_raw_pre_invoice_value"`
}
