// Package api is the published contract of the custody module: neutral
// import records built by the integration connector from staged legacy
// rows, and the read surface used by reconciliation. Case references are
// resolved through the published cases API; custody never reads cases
// tables. Legacy keys are provenance only.
package api

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
)

// ErrInvalidImport reports a command that violates the contract.
var ErrInvalidImport = errors.New("custody: invalid import")

var (
	keyPattern  = regexp.MustCompile(`^[1-9][0-9]{0,19}$`)
	uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	specPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*\.[a-z][a-z0-9_]*\.v[1-9][0-9]*$`)
	sysPattern  = regexp.MustCompile(`^[a-z][a-z0-9_]{1,15}$`)
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
	if !sysPattern.MatchString(s.System) || !specPattern.MatchString(s.Spec) || !keyPattern.MatchString(s.Key) ||
		!uuidPattern.MatchString(s.BatchID) || len(s.RowSHA256) != 32 {
		return fmt.Errorf("%w: source ref %s/%s", ErrInvalidImport, s.Spec, s.Key)
	}
	return nil
}

// RowRef names the source row of the Case an event or holder belongs to.
type RowRef struct {
	Spec string
	Key  string
}

func (r RowRef) valid() bool { return specPattern.MatchString(r.Spec) && keyPattern.MatchString(r.Key) }

// UserRef is a legacy user reference with its target user id when the
// identity module resolved it.
type UserRef struct {
	LegacyKey string
	UserID    *string
}

func (u *UserRef) valid() bool {
	return u == nil || (keyPattern.MatchString(u.LegacyKey) && (u.UserID == nil || uuidPattern.MatchString(*u.UserID)))
}

// Event kinds and contexts (assigned by the integration's explicit mapping).
const (
	HolderAssigned = "HOLDER_ASSIGNED"
	HolderReleased = "HOLDER_RELEASED"
	NoHolderChange = "NO_HOLDER_CHANGE"
	KindUnknown    = "UNKNOWN"
)

var (
	kinds    = map[string]bool{HolderAssigned: true, HolderReleased: true, NoHolderChange: true, KindUnknown: true}
	contexts = map[string]bool{"DISTRIBUTION": true, "MIS": true, "PAYMENT_WORKFLOW": true, "REFERRAL": true, "CONVERSION": true,
		"SELLER_UPDATE": true, "UNKNOWN": true}
)

// EventRecord is one append-only legacy custody log row.
type EventRecord struct {
	Source          SourceRef
	Item            RowRef
	Action          string
	Kind            string
	Context         string
	Actor           *UserRef
	From            *UserRef
	To              *UserRef
	FromPositionRaw *string
	ToPositionRaw   *string
	StepOrderRaw    *string
	OccurredRaw     *string
}

// HolderRecord is the legacy current-holder evidence of one Case (source
// state at extraction). Holder nil means the legacy row names no holder.
type HolderRecord struct {
	Case        RowRef
	Holder      *UserRef
	PositionRaw *string
}

// CustodyImport is one batch for the custody module. Complete names the
// holder-source specs whose staged entity is COMPLETE.
type CustodyImport struct {
	BatchID      string
	SnapshotID   string
	SourceSystem string
	Complete     map[string]bool
	Holders      []HolderRecord
	Events       []EventRecord
}

// Validate checks the command contract.
func (in CustodyImport) Validate() error {
	if !uuidPattern.MatchString(in.BatchID) || in.SnapshotID == "" || !sysPattern.MatchString(in.SourceSystem) {
		return fmt.Errorf("%w: batch/snapshot/source system", ErrInvalidImport)
	}
	seen := map[RowRef]bool{}
	for _, h := range in.Holders {
		if !h.Case.valid() || !h.Holder.valid() {
			return fmt.Errorf("%w: holder record", ErrInvalidImport)
		}
		if seen[h.Case] {
			return fmt.Errorf("%w: duplicate holder record %s/%s", ErrInvalidImport, h.Case.Spec, h.Case.Key)
		}
		seen[h.Case] = true
	}
	for _, e := range in.Events {
		if err := e.Source.Validate(); err != nil {
			return err
		}
		if e.Source.BatchID != in.BatchID || e.Source.System != in.SourceSystem || !e.Item.valid() || !kinds[e.Kind] || !contexts[e.Context] ||
			e.Action == "" || len(e.Action) > 100 || !e.Actor.valid() || !e.From.valid() || !e.To.valid() {
			return fmt.Errorf("%w: event %s", ErrInvalidImport, e.Source.Key)
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

// Importer is the custody import command.
type Importer interface {
	ImportCustody(ctx context.Context, in CustodyImport) (ImportReport, error)
}

// Reader is the published custody read surface for reconciliation.
type Reader interface {
	ParityReport(ctx context.Context) (Parity, error)
	// CaseCoverage reports, per case id, the parity of its PRESENT custody
	// projection (absent from the map = no present projection).
	CaseCoverage(ctx context.Context, caseIDs []string) (map[string]string, error)
	// BatchImported reports whether a custody import run exists for a batch.
	BatchImported(ctx context.Context, batchID string) (bool, error)
}

// Parity is the machine-readable custody parity report.
type Parity struct {
	Assignments     int64            `json:"assignments_present"`
	ByParity        map[string]int64 `json:"current_holder_parity"`
	ByDerivation    map[string]int64 `json:"derivation_states"`
	FindingsByKind  map[string]int64 `json:"findings_by_kind"`
	Events          int64            `json:"custody_events"`
	EventsByKind    map[string]int64 `json:"custody_events_by_kind"`
	CasesWithEvents int64            `json:"cases_with_custody_events"`
	UnknownActions  map[string]int64 `json:"unknown_legacy_actions"`
	FullParity      bool             `json:"full_parity"`
}
