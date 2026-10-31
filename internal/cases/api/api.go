// Package api is the published contract of the cases module: the neutral
// import command records the integration connector builds from staged legacy
// rows, and the read surface other modules (custody) and reconciliation use.
// It carries no legacy table or column names; legacy keys are provenance.
package api

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
)

// ErrInvalidImport reports a command that violates the contract.
var ErrInvalidImport = errors.New("cases: invalid import")

var (
	keyPattern  = regexp.MustCompile(`^[1-9][0-9]{0,19}$`)
	uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	specPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*\.[a-z][a-z0-9_]*\.v[1-9][0-9]*$`)
	sysPattern  = regexp.MustCompile(`^[a-z][a-z0-9_]{1,15}$`)
	nsPattern   = regexp.MustCompile(`^[a-z][a-z0-9_]*\.[a-z][a-z0-9_]*$`)
)

// SourceRef is the provenance of one staged source row.
type SourceRef struct {
	System    string
	Spec      string
	Key       string
	BatchID   string
	RowSHA256 []byte
}

// Validate checks a source reference.
func (s SourceRef) Validate() error {
	switch {
	case !sysPattern.MatchString(s.System):
		return fmt.Errorf("%w: source system %q", ErrInvalidImport, s.System)
	case !specPattern.MatchString(s.Spec):
		return fmt.Errorf("%w: source spec %q", ErrInvalidImport, s.Spec)
	case !keyPattern.MatchString(s.Key):
		return fmt.Errorf("%w: source key %q", ErrInvalidImport, s.Key)
	case !uuidPattern.MatchString(s.BatchID):
		return fmt.Errorf("%w: batch id %q", ErrInvalidImport, s.BatchID)
	case len(s.RowSHA256) != 32:
		return fmt.Errorf("%w: row checksum length %d", ErrInvalidImport, len(s.RowSHA256))
	}
	return nil
}

// RowRef names a staged source row by spec and key (no batch).
type RowRef struct {
	Spec string
	Key  string
}

func (r RowRef) valid() bool { return specPattern.MatchString(r.Spec) && keyPattern.MatchString(r.Key) }

// UserRef is a legacy user reference with its target user id when the
// identity module resolved it (never inferred from phone or name).
type UserRef struct {
	LegacyKey string
	UserID    *string
}

func (u *UserRef) valid() bool {
	return u == nil || (keyPattern.MatchString(u.LegacyKey) && (u.UserID == nil || uuidPattern.MatchString(*u.UserID)))
}

// WorkflowType is the kind of source work unit a Case stands for.
type WorkflowType string

const (
	LegacyLead       WorkflowType = "LEGACY_LEAD"
	DistributionItem WorkflowType = "DISTRIBUTION_ITEM"
)

// StateMapping says whether a raw state value is in the explicit verified
// mapping of its namespace. UNKNOWN keeps the raw value; nothing defaults.
type StateMapping string

const (
	StateKnown   StateMapping = "KNOWN"
	StateUnknown StateMapping = "UNKNOWN"
)

// Resolution states (kernel reference statuses).
type Resolution string

const (
	Resolved       Resolution = "RESOLVED"
	Related        Resolution = "RELATED"
	Unknown        Resolution = "UNKNOWN"
	Conflict       Resolution = "CONFLICT"
	NotRecoverable Resolution = "NOT_RECOVERABLE"
)

func (r Resolution) valid() bool {
	switch r {
	case Resolved, Related, Unknown, Conflict, NotRecoverable:
		return true
	}
	return false
}

// LeadFacts are the raw legacy lead facts (Legacy Leads Only grain).
type LeadFacts struct {
	StatusRaw, LeadStatusRaw, DestinationPanelRaw *string
	ImportedAtRaw, AssignedAtRaw, UpdatedAtRaw    *string
}

// CustomerLink is an explicit customers reference. Without one the Case's
// customer linkage is UNKNOWN; it is never derived from phone or name.
type CustomerLink struct {
	CustomerID string
	State      Resolution
	Reason     string
}

// CaseRecord is one staged source row that becomes one Case.
type CaseRecord struct {
	Workflow       WorkflowType
	Source         SourceRef
	ChannelRaw     *string
	OriginRefRaw   *string
	StateNamespace string
	StateRaw       *string
	StateMapping   StateMapping
	Lead           *LeadFacts
	Customer       *CustomerLink
}

// HistoryRecord is one append-only legacy state history row.
type HistoryRecord struct {
	Source      SourceRef
	Case        RowRef
	OldRaw      *string
	NewRaw      *string
	Actor       *UserRef
	OccurredRaw *string
}

// Relation is the kind of an explicit native link between representations.
type Relation string

// ConversionLink: a work item names the legacy lead it was converted into.
// Whether that is the same work unit (continuation) or a related distinct
// one is OPD-01 and stays unresolved.
const ConversionLink Relation = "CONVERSION_LINK"

// AliasRecord is one explicit native link evidenced by a staged row.
type AliasRecord struct {
	From     RowRef
	Related  RowRef
	Relation Relation
	Basis    string
}

// Facet is an attribution facet (Architecture V1 §7; GATE-0 §4).
type Facet string

const (
	OriginalOwner  Facet = "ORIGINAL_OWNER"
	OriginalSeller Facet = "ORIGINAL_SELLER"
	CreditOwner    Facet = "CREDIT_OWNER"
)

func (f Facet) valid() bool { return f == OriginalOwner || f == OriginalSeller || f == CreditOwner }

// AttributionRecord is attribution evidence frozen into a snapshot.
type AttributionRecord struct {
	Case          RowRef
	Facet         Facet
	Resolution    Resolution
	Subject       *UserRef
	PositionRaw   *string
	Basis         string
	SourceTimeRaw *string
	Evidence      *RowRef
}

// CaseImport is one batch for the cases module. Complete names the specs
// whose staged entity is COMPLETE; only those may mark absence.
type CaseImport struct {
	BatchID      string
	SnapshotID   string
	SourceSystem string
	Complete     map[string]bool
	Cases        []CaseRecord
	History      []HistoryRecord
	Aliases      []AliasRecord
	Attributions []AttributionRecord
}

// Validate checks the command contract.
func (in CaseImport) Validate() error {
	if !uuidPattern.MatchString(in.BatchID) || in.SnapshotID == "" || !sysPattern.MatchString(in.SourceSystem) {
		return fmt.Errorf("%w: batch/snapshot/source system", ErrInvalidImport)
	}
	seen := map[RowRef]bool{}
	for _, c := range in.Cases {
		if err := c.Source.Validate(); err != nil {
			return err
		}
		if c.Source.BatchID != in.BatchID || c.Source.System != in.SourceSystem {
			return fmt.Errorf("%w: case from another batch or system", ErrInvalidImport)
		}
		if c.Workflow != LegacyLead && c.Workflow != DistributionItem {
			return fmt.Errorf("%w: workflow %q", ErrInvalidImport, c.Workflow)
		}
		if !nsPattern.MatchString(c.StateNamespace) || (c.StateMapping != StateKnown && c.StateMapping != StateUnknown) {
			return fmt.Errorf("%w: state of %s", ErrInvalidImport, c.Source.Key)
		}
		if c.StateMapping == StateKnown && c.StateRaw == nil {
			return fmt.Errorf("%w: KNOWN state without a raw value", ErrInvalidImport)
		}
		if (c.Workflow == LegacyLead) != (c.Lead != nil) {
			return fmt.Errorf("%w: lead facts belong to legacy lead cases only", ErrInvalidImport)
		}
		if l := c.Customer; l != nil && (!uuidPattern.MatchString(l.CustomerID) || !l.State.valid() || l.Reason == "" || len(l.Reason) > 300) {
			return fmt.Errorf("%w: customer link of %s", ErrInvalidImport, c.Source.Key)
		}
		r := RowRef{Spec: c.Source.Spec, Key: c.Source.Key}
		if seen[r] {
			return fmt.Errorf("%w: duplicate case source %s/%s", ErrInvalidImport, r.Spec, r.Key)
		}
		seen[r] = true
	}
	for _, h := range in.History {
		if err := h.Source.Validate(); err != nil {
			return err
		}
		if !h.Case.valid() || !h.Actor.valid() {
			return fmt.Errorf("%w: history row %s", ErrInvalidImport, h.Source.Key)
		}
	}
	for _, a := range in.Aliases {
		if !a.From.valid() || !a.Related.valid() || a.Relation != ConversionLink || a.Basis == "" || len(a.Basis) > 300 {
			return fmt.Errorf("%w: alias", ErrInvalidImport)
		}
	}
	for _, a := range in.Attributions {
		if !a.Case.valid() || !a.Facet.valid() || !a.Resolution.valid() || !a.Subject.valid() || a.Basis == "" || len(a.Basis) > 300 ||
			(a.Evidence != nil && !a.Evidence.valid()) {
			return fmt.Errorf("%w: attribution", ErrInvalidImport)
		}
		if a.Resolution == Resolved && a.Subject == nil {
			return fmt.Errorf("%w: RESOLVED attribution without a subject", ErrInvalidImport)
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

// Importer is the cases import command.
type Importer interface {
	ImportCases(ctx context.Context, in CaseImport) (ImportReport, error)
}

// SourceCase is the stored provenance of the Case of one source row.
type SourceCase struct {
	CaseID    string
	BatchID   string
	RowSHA256 []byte
	Present   bool
}

// Reader is the published read surface used by custody and reconciliation.
type Reader interface {
	// ResolveSources maps source keys of one spec to their Case.
	ResolveSources(ctx context.Context, system, spec string, keys []string) (map[string]SourceCase, error)
	// CoverageReport is the machine-readable cases coverage report.
	CoverageReport(ctx context.Context) (Coverage, error)
}

// Coverage is the cases coverage and source-honesty report (F02/F03).
type Coverage struct {
	CasesTotal          int64                       `json:"cases_total"`
	CasesPresent        int64                       `json:"cases_present"`
	CasesBySource       map[string]int64            `json:"cases_by_source_spec"`
	CasesByWorkflow     map[string]int64            `json:"cases_by_workflow_type"`
	UnknownStates       int64                       `json:"cases_with_unknown_state_mapping"`
	SourcelessCases     int64                       `json:"sourceless_cases"`
	LegacyLeadsOnly     int64                       `json:"legacy_leads_only"`
	AliasesByResolution map[string]int64            `json:"aliases_by_resolution"`
	StatusHistoryRows   int64                       `json:"status_history_rows"`
	CasesWithHistory    int64                       `json:"cases_with_status_history"`
	AttributionByFacet  map[string]map[string]int64 `json:"attribution_by_facet_and_resolution"`
	CustomerLinks       map[string]int64            `json:"customer_link_states"`
}
