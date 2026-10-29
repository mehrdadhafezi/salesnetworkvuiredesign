// Package result implements the frozen Slice 0 result contract (E0 §9A).
//
// Status is a closed set of seven outcomes. Aggregation is deterministic and
// order-independent:
//   - empty set → contract error (never SUCCESS)
//   - all items the same status → that status
//   - any other mix → PARTIAL, with per-item statuses preserved
//
// UNKNOWN is never coerced to FAILED or SUCCESS; PARTIAL is not SUCCESS;
// CONFLICT is not FAILED; UNAUTHORIZED is distinct from BLOCKED. Transport or
// HTTP success is not business SUCCESS.
package result

import (
	"errors"
	"fmt"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/refs"
)

var (
	// ErrInvalidStatus reports a status outside the closed set.
	ErrInvalidStatus = errors.New("result: invalid status")
	// ErrEmptyAggregate reports aggregation over zero items.
	ErrEmptyAggregate = errors.New("result: cannot aggregate an empty set")
	// ErrMissingRefReason reports an item without a ref and without a reason code.
	ErrMissingRefReason = errors.New("result: item without ref requires a reason code")
	// ErrAmbiguousItemRef reports an item ref that carries both a NativeRef and a CaseRef.
	ErrAmbiguousItemRef = errors.New("result: item ref must be exactly one of native or case")
	// ErrInvalidMetadata reports metadata outside the allowlist rules.
	ErrInvalidMetadata = errors.New("result: invalid metadata")
	// ErrInvalidCorrelation reports an empty or overlong correlation id.
	ErrInvalidCorrelation = errors.New("result: invalid correlation id")
)

// Status is one of the seven frozen outcomes. The zero value is invalid.
type Status uint8

const (
	statusInvalid Status = iota
	// Success: evaluation complete, or intent proven applied / idempotently unchanged.
	Success
	// Partial: a known mix of item outcomes, or incomplete coverage.
	Partial
	// Failed: a known failure; not a timeout after a possible commit.
	Failed
	// Unknown: outcome or commit effects cannot be determined; reconcile before retry.
	Unknown
	// Conflict: fresh facts, state or proof contradict the intent or each other.
	Conflict
	// Blocked: a known safety or policy barrier; nothing was executed.
	Blocked
	// Unauthorized: an observed route/module/action/row/field denial.
	Unauthorized
)

var statusNames = [...]string{
	statusInvalid: "INVALID",
	Success:       "SUCCESS",
	Partial:       "PARTIAL",
	Failed:        "FAILED",
	Unknown:       "UNKNOWN",
	Conflict:      "CONFLICT",
	Blocked:       "BLOCKED",
	Unauthorized:  "UNAUTHORIZED",
}

// Statuses returns the seven valid statuses in a stable order.
func Statuses() []Status {
	return []Status{Success, Partial, Failed, Unknown, Conflict, Blocked, Unauthorized}
}

// Valid reports whether s is one of the seven statuses.
func (s Status) Valid() bool { return s >= Success && s <= Unauthorized }

// String returns the stable name, or "INVALID".
func (s Status) String() string {
	if !s.Valid() {
		return statusNames[statusInvalid]
	}
	return statusNames[s]
}

// ParseStatus parses an exact status name.
func ParseStatus(name string) (Status, error) {
	for _, s := range Statuses() {
		if statusNames[s] == name {
			return s, nil
		}
	}
	return statusInvalid, fmt.Errorf("%w: %q", ErrInvalidStatus, name)
}

// MarshalText implements encoding.TextMarshaler.
func (s Status) MarshalText() ([]byte, error) {
	if !s.Valid() {
		return nil, ErrInvalidStatus
	}
	return []byte(s.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (s *Status) UnmarshalText(b []byte) error {
	parsed, err := ParseStatus(string(b))
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

// Aggregate combines item statuses per E0 §9A.
func Aggregate(statuses []Status) (Status, error) {
	if len(statuses) == 0 {
		return statusInvalid, ErrEmptyAggregate
	}
	first := statuses[0]
	if !first.Valid() {
		return statusInvalid, ErrInvalidStatus
	}
	uniform := true
	for _, s := range statuses[1:] {
		if !s.Valid() {
			return statusInvalid, ErrInvalidStatus
		}
		if s != first {
			uniform = false
		}
	}
	if uniform {
		return first, nil
	}
	return Partial, nil
}

// ItemRef identifies the subject of an item outcome: exactly one of Native or Case.
type ItemRef struct {
	Native *refs.NativeRef
	Case   *refs.CaseRef
}

// Validate requires exactly one valid ref.
func (r ItemRef) Validate() error {
	switch {
	case r.Native != nil && r.Case == nil:
		return r.Native.Validate()
	case r.Case != nil && r.Native == nil:
		return r.Case.Validate()
	default:
		return ErrAmbiguousItemRef
	}
}

// Disposition is the optional bulk-accounting disposition (Phase-0 spec §5).
// It is metadata, not an extra business status.
type Disposition uint8

const (
	// DispositionNone means no disposition was recorded.
	DispositionNone Disposition = iota
	DispositionApplied
	DispositionSkipped
	DispositionRejected
	DispositionFailed
	DispositionUnknown
	DispositionUnprocessed
)

// Valid reports whether d is a declared disposition (including none).
func (d Disposition) Valid() bool { return d <= DispositionUnprocessed }

// Coverage is the optional evaluation coverage.
type Coverage uint8

const (
	// CoverageNone means coverage was not stated.
	CoverageNone Coverage = iota
	CoverageComplete
	CoverageIncomplete
)

// Valid reports whether c is declared (including none).
func (c Coverage) Valid() bool { return c <= CoverageIncomplete }

// Metadata is the allowlisted diagnostic metadata of an item. It is a fixed
// struct so that arbitrary rows or secrets cannot be attached.
type Metadata struct {
	EvaluationType string
	Disposition    Disposition
	Coverage       Coverage
	EffectsKnown   *bool
}

const maxEvaluationTypeLen = 64

// Validate checks the allowlist rules.
func (m Metadata) Validate() error {
	if !m.Disposition.Valid() || !m.Coverage.Valid() {
		return ErrInvalidMetadata
	}
	if len(m.EvaluationType) > maxEvaluationTypeLen {
		return fmt.Errorf("%w: evaluation type too long", ErrInvalidMetadata)
	}
	for i := 0; i < len(m.EvaluationType); i++ {
		c := m.EvaluationType[i]
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_') {
			return fmt.Errorf("%w: evaluation type must be snake_case", ErrInvalidMetadata)
		}
	}
	return nil
}

const maxCorrelationLen = 128

func validCorrelation(c *string) error {
	if c == nil {
		return nil
	}
	if *c == "" || len(*c) > maxCorrelationLen {
		return ErrInvalidCorrelation
	}
	for i := 0; i < len(*c); i++ {
		if (*c)[i] <= ' ' || (*c)[i] > '~' {
			return ErrInvalidCorrelation
		}
	}
	return nil
}

// ItemOutcome is the per-item result shape (E0 §9A).
type ItemOutcome struct {
	ItemRef       *ItemRef
	Status        Status
	ReasonCodes   []reasoncodes.Code
	Message       *string
	CorrelationID *string
	BeforeRef     *refs.NativeRef
	AfterRef      *refs.NativeRef
	Metadata      Metadata
}

// Validate checks the item invariants.
func (o ItemOutcome) Validate() error {
	if !o.Status.Valid() {
		return ErrInvalidStatus
	}
	if _, err := reasoncodes.Normalize(o.ReasonCodes); err != nil {
		return err
	}
	if o.ItemRef == nil {
		if len(o.ReasonCodes) == 0 {
			return ErrMissingRefReason
		}
	} else if err := o.ItemRef.Validate(); err != nil {
		return err
	}
	for _, r := range []*refs.NativeRef{o.BeforeRef, o.AfterRef} {
		if r != nil {
			if err := r.Validate(); err != nil {
				return err
			}
		}
	}
	if err := validCorrelation(o.CorrelationID); err != nil {
		return err
	}
	if err := o.Metadata.Validate(); err != nil {
		return err
	}
	if o.Status == Success && o.Metadata.Disposition == DispositionSkipped && len(o.ReasonCodes) == 0 {
		return fmt.Errorf("%w: SKIPPED with SUCCESS requires a reason code", ErrInvalidMetadata)
	}
	return nil
}

// Result is an aggregated outcome over validated items.
type Result struct {
	Status        Status
	Items         []ItemOutcome
	CorrelationID *string
}

// New validates every item, aggregates their statuses and returns a Result.
// Items are kept in input order; aggregation does not depend on that order.
func New(items []ItemOutcome, correlation *string) (Result, error) {
	if len(items) == 0 {
		return Result{}, ErrEmptyAggregate
	}
	if err := validCorrelation(correlation); err != nil {
		return Result{}, err
	}
	statuses := make([]Status, len(items))
	for i, item := range items {
		if err := item.Validate(); err != nil {
			return Result{}, fmt.Errorf("item %d: %w", i, err)
		}
		statuses[i] = item.Status
	}
	status, err := Aggregate(statuses)
	if err != nil {
		return Result{}, err
	}
	return Result{Status: status, Items: append([]ItemOutcome(nil), items...), CorrelationID: correlation}, nil
}

// Counts returns the number of items per status, preserving per-item classes.
func (r Result) Counts() map[Status]int {
	out := make(map[Status]int, len(r.Items))
	for _, item := range r.Items {
		out[item.Status]++
	}
	return out
}
