// Package api is the commission module's published contract: the neutral
// legacy commission EVIDENCE port (implemented by the integration connector,
// never by commission itself), the offline shadow job's command shape and the
// GET-only read DTOs. No type here approves, posts, credits or settles.
package api

import (
	"context"
	"time"
)

// Source states (same vocabulary as the other shadow modules).
const (
	SourceAvailable = "AVAILABLE"
	SourcePartial   = "PARTIAL"
	SourceMissing   = "MISSING"
)

// Evidence sources.
const (
	SourceCredits     = "wallet_commission_transactions"
	SourceMatrix      = "purpose_commission_matrix"
	SourceRules       = "commission_rules"
	SourceDryRuns     = "commission_dry_runs"
	SourceDryRunItems = "commission_dry_run_items"
	SourcePostingLogs = "commission_posting_logs"
)

// Directions.
const (
	DirCredit  = "CREDIT"
	DirDebit   = "DEBIT"
	DirUnknown = "UNKNOWN"
)

// LegacyCredit is one legacy commission wallet transaction (credit or its
// reversal debit) with ONLY the writer's event-time snapshot keys read from
// its meta. The free-text reason / note in meta is never read.
type LegacyCredit struct {
	TxKey            string   `json:"tx_key"`
	Engine           string   `json:"engine"`
	EngineBasis      string   `json:"engine_basis"`
	TypeRaw          string   `json:"type_raw"`
	Direction        string   `json:"direction"`
	StatusRaw        *string  `json:"status_raw"`
	RewrittenInPlace bool     `json:"rewritten_in_place"`
	WalletTypeRaw    *string  `json:"wallet_type_raw"`
	HolderKey        *string  `json:"holder_key"`
	HolderUserID     *string  `json:"holder_user_id"`
	InvoiceKey       *string  `json:"invoice_key"`
	InvoiceID        *string  `json:"invoice_id"`
	AmountRaw        *string  `json:"amount_raw"`
	Unit             string   `json:"unit"`
	StageRaw         *string  `json:"stage_raw"`
	PurposeRaw       *string  `json:"purpose_raw"`
	RoleRaw          *string  `json:"role_raw"`
	RateRaw          *string  `json:"rate_raw"`
	BaseRaw          *string  `json:"base_raw"`
	RecipientSnapKey *string  `json:"recipient_snapshot_key"`
	SnapshotVersion  *string  `json:"snapshot_version_raw"`
	ReversesTxKey    *string  `json:"reverses_tx_key"`
	DryRunKey        *string  `json:"dry_run_key"`
	DryRunItemKey    *string  `json:"dry_run_item_key"`
	MatchedRuleKeys  []string `json:"matched_rule_keys"`
	CreatedAtRaw     *string  `json:"created_at_raw"`
}

// MatrixRate is one OBSERVED purpose matrix cell (mutable in legacy; the
// value at any earlier date is unknown).
type MatrixRate struct {
	SourceKey    string  `json:"source_key"`
	PurposeRaw   string  `json:"purpose_raw"`
	RoleRaw      string  `json:"role_raw"`
	RateRaw      *string `json:"rate_raw"`
	UpdatedAtRaw *string `json:"updated_at_raw"`
}

// Rule is one OBSERVED legacy commission rule (E2 / E3).
type Rule struct {
	SourceKey          string  `json:"source_key"`
	RuleCode           *string `json:"rule_code"`
	CalculationTypeRaw *string `json:"calculation_type_raw"`
	PositionRaw        *string `json:"position_raw"`
	LevelRaw           *string `json:"level_raw"`
	RateRaw            *string `json:"rate_raw"`
	FixedRaw           *string `json:"fixed_raw"`
	EffectiveFromRaw   *string `json:"effective_from_raw"`
	EffectiveToRaw     *string `json:"effective_to_raw"`
	ActiveRaw          *string `json:"active_raw"`
	PriorityRaw        *string `json:"priority_raw"`
	UpdatedAtRaw       *string `json:"updated_at_raw"`
}

// DryRun is one legacy E3 run with its approval provenance (raw kept).
type DryRun struct {
	SourceKey     string  `json:"source_key"`
	RunCode       *string `json:"run_code"`
	DateFromRaw   *string `json:"date_from_raw"`
	DateToRaw     *string `json:"date_to_raw"`
	StatusRaw     *string `json:"status_raw"`
	ApprovalRaw   *string `json:"approval_raw"`
	CreatedByKey  *string `json:"created_by_key"`
	ApprovedByKey *string `json:"approved_by_key"`
	ApprovedAtRaw *string `json:"approved_at_raw"`
	RejectedByKey *string `json:"rejected_by_key"`
	RejectedAtRaw *string `json:"rejected_at_raw"`
	LockedAtRaw   *string `json:"locked_at_raw"`
	CreatedAtRaw  *string `json:"created_at_raw"`
}

// DryRunItem is one legacy E3 item.
type DryRunItem struct {
	SourceKey        string  `json:"source_key"`
	RunKey           *string `json:"run_key"`
	InvoiceKey       *string `json:"invoice_key"`
	InvoiceID        *string `json:"invoice_id"`
	SellerKey        *string `json:"seller_key"`
	SellerUserID     *string `json:"seller_user_id"`
	PositionRaw      *string `json:"position_raw"`
	LevelRaw         *string `json:"level_raw"`
	InvoiceAmountRaw *string `json:"invoice_amount_raw"`
	MatchedRuleKey   *string `json:"matched_rule_key"`
	CalculatedRaw    *string `json:"calculated_raw"`
	ItemStatusRaw    *string `json:"item_status_raw"`
	PostingStatusRaw *string `json:"posting_status_raw"`
	WalletTxKey      *string `json:"wallet_tx_key"`
	PostedAtRaw      *string `json:"posted_at_raw"`
	CompensationKey  *string `json:"compensation_key"`
	CalculationDate  *string `json:"calculation_date_raw"`
	Unit             string  `json:"unit"`
	NonInvoice       bool    `json:"non_invoice"`
}

// PostingLog is one legacy E3 posting / review audit row.
type PostingLog struct {
	SourceKey    string  `json:"source_key"`
	RunKey       *string `json:"run_key"`
	ItemKey      *string `json:"item_key"`
	WalletTxKey  *string `json:"wallet_tx_key"`
	ActorKey     *string `json:"actor_key"`
	ActionRaw    *string `json:"action_raw"`
	StatusRaw    *string `json:"status_raw"`
	CreatedAtRaw *string `json:"created_at_raw"`
}

// Evidence is one staged legacy commission snapshot.
type Evidence struct {
	SnapshotID  string            `json:"snapshot_id"`
	BatchID     string            `json:"batch_id"`
	Cutoff      time.Time         `json:"cutoff"`
	Sources     map[string]string `json:"sources"`
	SourceRows  map[string]int64  `json:"source_rows"`
	Credits     []LegacyCredit    `json:"credits"`
	Matrix      []MatrixRate      `json:"matrix"`
	Rules       []Rule            `json:"rules"`
	DryRuns     []DryRun          `json:"dry_runs"`
	DryRunItems []DryRunItem      `json:"dry_run_items"`
	PostingLogs []PostingLog      `json:"posting_logs"`
}

// EvidenceSource is the port the integration connector implements over a
// STAGED snapshot (never the live legacy system).
type EvidenceSource interface {
	CommissionEvidence(ctx context.Context) (Evidence, error)
}

// ---- read DTOs (GET only) ----

// Notice is attached to every read: nothing here is a financial effect.
const Notice = "SHADOW only: no commission was calculated, approved, posted or credited by this read; legacy remains the system of record"

// Blocked decisions surfaced on every overview.
type Decision struct {
	ID     string `json:"id"`
	State  string `json:"state"`
	Effect string `json:"effect"`
}

// Decisions is the fixed Slice 13 decision readiness list.
func Decisions() []Decision {
	return []Decision{
		{ID: "OPD-09", State: "REQUIRES_HUMAN_OWNER_DECISION", Effect: "engine routing, non-invoice keys and correction semantics unresolved: approval and posting blocked"},
		{ID: "D-36", State: "IMPLEMENTATION_BLOCKED", Effect: "component (pro-rata) allocation unsupported: Biawin project-card commission not comparable"},
		{ID: "D-41", State: "SAFE_DEFAULT_AVAILABLE", Effect: "shadow computes, publishes neither engine"},
		{ID: "OPD-06", State: "REQUIRES_HUMAN_OWNER_DECISION", Effect: "no approved target rounding: legacy half-away 2dp used for parity only"},
		{ID: "OPD-08", State: "IMPLEMENTATION_BLOCKED", Effect: "posting idempotency / unknown-outcome recovery with finance not built: no dispatch"},
		{ID: "HR-ATTRIBUTION", State: "EVIDENCE_INCOMPLETE", Effect: "legacy supervisor+ recipients use the CURRENT hierarchy: only writer meta snapshots are event-time evidence"},
	}
}

// Run is one commission shadow run (import + recalculation + parity).
type Run struct {
	RunID                string            `json:"run_id"`
	RunSeq               int64             `json:"run_seq"`
	RecordKind           string            `json:"record_kind"`
	State                string            `json:"state"`
	NormalizationVersion int               `json:"normalization_version"`
	SnapshotID           string            `json:"snapshot_id"`
	Cutoff               time.Time         `json:"source_cutoff"`
	CompletedAt          time.Time         `json:"completed_at"`
	Outcome              string            `json:"outcome"`
	Reasons              []string          `json:"reasons"`
	Rounding             string            `json:"rounding"`
	Counts               map[string]int64  `json:"counts"`
	Sources              map[string]string `json:"sources"`
}

// ParitySummary is the honest parity report of one run.
type ParitySummary struct {
	Cohort        string            `json:"cohort"`
	WindowFrom    *string           `json:"window_from_raw"`
	WindowTo      *string           `json:"window_to_raw"`
	LegacyRows    int64             `json:"legacy_rows"`
	ComparedRows  int64             `json:"compared_rows"`
	Coverage      string            `json:"coverage"`
	Classes       map[string]int64  `json:"classes"`
	Engines       map[string]int64  `json:"engines"`
	LegacyTotals  map[string]string `json:"legacy_totals"`
	TargetTotals  map[string]string `json:"target_totals"`
	LegacyMissing int64             `json:"legacy_amount_missing"`
	TargetMissing int64             `json:"target_amount_missing"`
	Reasons       map[string]int64  `json:"reasons"`
}

// Overview is the commission landing read.
type Overview struct {
	Run          *Run             `json:"run"`
	Parity       *ParitySummary   `json:"parity"`
	Findings     map[string]int64 `json:"findings"`
	Decisions    []Decision       `json:"decisions"`
	PostingState string           `json:"posting_state"`
	Notice       string           `json:"notice"`
}

// PostingStateBlocked is the only posting state Slice 13 can report.
const PostingStateBlocked = "BLOCKED_NOT_AUTHORIZED"

// Calculation is one stored shadow calculation with its parity.
type Calculation struct {
	CalcID          string   `json:"calc_id"`
	SourceKey       string   `json:"source_key"`
	EntitlementKey  *string  `json:"entitlement_key"`
	Engine          string   `json:"engine"`
	FunctionID      string   `json:"function_id"`
	PolicyKey       *string  `json:"policy_key"`
	PolicyVersion   *int     `json:"policy_version"`
	EventRef        string   `json:"event_ref"`
	InputSHA        string   `json:"input_sha256"`
	ShareFacet      *string  `json:"share_facet"`
	RecipientState  string   `json:"recipient_state"`
	RecipientKey    *string  `json:"recipient_key"`
	RecipientUserID *string  `json:"recipient_user_id"`
	Base            *string  `json:"base"`
	Rate            *string  `json:"rate_percent"`
	Amount          *string  `json:"amount"`
	LegacyAmount    *string  `json:"legacy_amount"`
	Unit            string   `json:"unit"`
	Eligibility     string   `json:"eligibility"`
	Coverage        string   `json:"coverage"`
	Blocks          []string `json:"blocks"`
	ParityClass     string   `json:"parity_class"`
	ParityFields    []byte   `json:"-"`
	Reasons         []string `json:"reasons"`
}

// Finding is one reconciliation finding.
type Finding struct {
	FindingID string  `json:"finding_id"`
	Kind      string  `json:"kind"`
	SourceKey string  `json:"source_key"`
	Detail    string  `json:"detail"`
	Severity  string  `json:"severity"`
	EntKey    *string `json:"entitlement_key"`
}

// PolicyVersion is one observed policy version.
type PolicyVersion struct {
	PolicyKey     string  `json:"policy_key"`
	Version       int     `json:"version"`
	Purpose       string  `json:"purpose"`
	ShareFacet    string  `json:"share_facet"`
	FunctionID    string  `json:"function_id"`
	Rate          *string `json:"rate_percent"`
	Fixed         *string `json:"fixed"`
	Unit          string  `json:"unit"`
	EffectiveFrom *string `json:"effective_from"`
	EffectiveTo   *string `json:"effective_to"`
	Evidence      string  `json:"evidence"`
	ApprovalState string  `json:"approval_state"`
	Provenance    string  `json:"provenance"`
	SourceRef     string  `json:"source_ref"`
	FirstRunID    string  `json:"first_run_id"`
}

// HistoricalRun is one legacy E3 run with approval provenance.
type HistoricalRun struct {
	SourceKey            string  `json:"source_key"`
	RunCode              *string `json:"run_code"`
	WindowFrom           *string `json:"window_from_raw"`
	WindowTo             *string `json:"window_to_raw"`
	ApprovalRaw          *string `json:"approval_raw"`
	State                string  `json:"state"`
	ApprovedByKey        *string `json:"approved_by_key"`
	ApprovedAtRaw        *string `json:"approved_at_raw"`
	RejectedByKey        *string `json:"rejected_by_key"`
	RejectedAtRaw        *string `json:"rejected_at_raw"`
	RejectedThenApproved bool    `json:"rejected_then_approved"`
	ItemCount            int64   `json:"item_count"`
	PostedItems          int64   `json:"posted_items"`
	FailedItems          int64   `json:"failed_items"`
}

// Earning is one historical entitlement of the caller (own scope only).
type Earning struct {
	SourceKey      string  `json:"source_key"`
	Engine         string  `json:"engine"`
	Direction      string  `json:"direction"`
	TypeRaw        string  `json:"type_raw"`
	InvoiceID      *string `json:"invoice_id"`
	Stage          *string `json:"stage_raw"`
	Purpose        *string `json:"purpose_raw"`
	ShareFacet     *string `json:"share_facet_raw"`
	Amount         *string `json:"amount"`
	Unit           string  `json:"unit"`
	Reversed       bool    `json:"reversed"`
	OccurredAtRaw  *string `json:"occurred_at_raw"`
	EntitlementKey *string `json:"entitlement_key"`
}
