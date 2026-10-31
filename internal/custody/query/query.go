// Package query is the custody read service. Every read runs in a
// custody-owned READ ONLY transaction: it never imports, re-derives,
// repairs or records anything.
package query

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody/store"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pagination"
)

var (
	// ErrInvalidQuery reports a filter or cursor outside the contract.
	ErrInvalidQuery = errors.New("custody/query: invalid query")
	// ErrNotFound reports a case without custody evidence.
	ErrNotFound = store.ErrNotFound
)

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func invalid(format string, a ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrInvalidQuery}, a...)...)
}

// Service reads the custody model.
type Service struct{ runner *db.Runner }

var _ api.Reader = (*Service)(nil)

// New returns the read service over a custody-owned runner.
func New(runner *db.Runner) *Service { return &Service{runner: runner} }

func (s *Service) read(ctx context.Context, fn func(ctx context.Context, tx db.Tx) error) error {
	return s.runner.InTx(ctx, db.TxOptions{ReadOnly: true, Isolation: db.RepeatableRead}, fn)
}

var allowed = map[string]map[string]bool{
	"parity": {"MATCH": true, "TARGET_UNKNOWN": true, "LEGACY_UNKNOWN": true, "CONFLICT": true, "MULTIPLE_CURRENT": true,
		"HISTORY_GAP": true, "NOT_RECOVERABLE": true},
	"derivation": {"DERIVED": true, "RELEASED": true, "UNKNOWN": true, "MULTIPLE": true, "NONE": true},
	"presence":   {"PRESENT": true, "ABSENT_IN_LATEST": true},
}

// AssignmentsPage is one page of projections.
type AssignmentsPage struct {
	Assignments []store.Assignment
	NextCursor  *string
}

// ListAssignments lists derived projections with allowlisted filters.
func (s *Service) ListAssignments(ctx context.Context, f store.AssignmentFilter, cursor *string, limitIn *int) (AssignmentsPage, error) {
	for name, v := range map[string]*string{"parity": f.Parity, "derivation": f.Derivation, "presence": f.Presence} {
		if v != nil && !allowed[name][*v] {
			return AssignmentsPage{}, invalid("%s", name)
		}
	}
	limit, err := pagination.Limit(limitIn)
	if err != nil {
		return AssignmentsPage{}, invalid("%v", err)
	}
	var after *string
	if cursor != nil {
		c, err := pagination.Decode(*cursor, "id")
		if err != nil || !uuidPattern.MatchString(c.ID) {
			return AssignmentsPage{}, invalid("cursor")
		}
		after = &c.ID
	}
	var rows []store.Assignment
	if err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		rows, err = store.ListAssignments(ctx, tx, f, after, limit)
		return err
	}); err != nil {
		return AssignmentsPage{}, err
	}
	p := AssignmentsPage{Assignments: rows}
	if len(rows) > limit {
		p.Assignments = rows[:limit]
		c, err := pagination.Encode(pagination.Cursor{Sort: "id", ID: p.Assignments[limit-1].CaseID})
		if err != nil {
			return AssignmentsPage{}, err
		}
		p.NextCursor = &c
	}
	return p, nil
}

// CaseCustody is the custody record of one case.
type CaseCustody struct {
	Assignment store.Assignment
	Events     []store.Event
	Findings   []store.Finding
}

// CaseCustody returns the projection, events and findings of one case.
func (s *Service) CaseCustody(ctx context.Context, caseID string) (CaseCustody, error) {
	if !uuidPattern.MatchString(caseID) {
		return CaseCustody{}, invalid("case_id")
	}
	var c CaseCustody
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		if c.Assignment, err = store.AssignmentOf(ctx, tx, caseID); err != nil {
			return err
		}
		if c.Events, err = store.EventView(ctx, tx, caseID); err != nil {
			return err
		}
		c.Findings, err = store.FindingsOf(ctx, tx, caseID)
		return err
	})
	return c, err
}

// ParityReport implements api.Reader.
func (s *Service) ParityReport(ctx context.Context) (api.Parity, error) {
	var r api.Parity
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		r, err = store.ParityReport(ctx, tx)
		return err
	})
	return r, err
}

// CaseCoverage implements api.Reader.
func (s *Service) CaseCoverage(ctx context.Context, caseIDs []string) (map[string]string, error) {
	var out map[string]string
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		out, err = store.CaseCoverage(ctx, tx, caseIDs)
		return err
	})
	return out, err
}

// BatchImported implements api.Reader.
func (s *Service) BatchImported(ctx context.Context, batchID string) (bool, error) {
	if !uuidPattern.MatchString(batchID) {
		return false, invalid("batch_id")
	}
	var ok bool
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		ok, err = store.BatchImported(ctx, tx, batchID)
		return err
	})
	return ok, err
}
