// Package query is the cases read service. Every read runs in a cases-owned
// READ ONLY transaction: it never imports, backfills, repairs, recomputes or
// records anything (Architecture V1 §8, GATE-0 INV-026).
package query

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/store"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pagination"
)

var (
	// ErrInvalidQuery reports a filter or cursor outside the contract.
	ErrInvalidQuery = errors.New("cases/query: invalid query")
	// ErrNotFound reports an unknown Case.
	ErrNotFound = store.ErrNotFound
)

var (
	uuidPattern    = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	ordinalPattern = regexp.MustCompile(`^[1-9][0-9]{0,19}$`)
)

func invalid(format string, a ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrInvalidQuery}, a...)...)
}

// Service reads the cases model.
type Service struct{ runner *db.Runner }

var _ api.Reader = (*Service)(nil)

// New returns the read service over a cases-owned runner.
func New(runner *db.Runner) *Service { return &Service{runner: runner} }

func (s *Service) read(ctx context.Context, fn func(ctx context.Context, tx db.Tx) error) error {
	return s.runner.InTx(ctx, db.TxOptions{ReadOnly: true, Isolation: db.RepeatableRead}, fn)
}

var allowed = map[string]map[string]bool{
	"workflow_type":       {"LEGACY_LEAD": true, "DISTRIBUTION_ITEM": true},
	"state_mapping":       {"KNOWN": true, "UNKNOWN": true},
	"customer_link_state": {"RESOLVED": true, "RELATED": true, "UNKNOWN": true, "CONFLICT": true, "NOT_RECOVERABLE": true},
	"presence":            {"PRESENT": true, "ABSENT_IN_LATEST": true},
}

func check(name string, v *string) error {
	if v != nil && !allowed[name][*v] {
		return invalid("%s", name)
	}
	return nil
}

// CasesPage is one page of Cases.
type CasesPage struct {
	Cases      []store.Case
	NextCursor *string
}

// ListCases lists Cases with allowlisted filters and a keyset cursor bound
// to the case-id sort.
func (s *Service) ListCases(ctx context.Context, f store.CaseFilter, cursor *string, limitIn *int) (CasesPage, error) {
	for name, v := range map[string]*string{"workflow_type": f.Workflow, "state_mapping": f.StateMapping,
		"customer_link_state": f.CustomerLinkState, "presence": f.Presence} {
		if err := check(name, v); err != nil {
			return CasesPage{}, err
		}
	}
	limit, err := pagination.Limit(limitIn)
	if err != nil {
		return CasesPage{}, invalid("%v", err)
	}
	var after *string
	if cursor != nil {
		c, err := pagination.Decode(*cursor, "id")
		if err != nil || !uuidPattern.MatchString(c.ID) {
			return CasesPage{}, invalid("cursor")
		}
		after = &c.ID
	}
	var rows []store.Case
	if err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		rows, err = store.ListCases(ctx, tx, f, after, limit)
		return err
	}); err != nil {
		return CasesPage{}, err
	}
	p := CasesPage{Cases: rows}
	if len(rows) > limit {
		p.Cases = rows[:limit]
		c, err := pagination.Encode(pagination.Cursor{Sort: "id", ID: p.Cases[limit-1].CaseID})
		if err != nil {
			return CasesPage{}, err
		}
		p.NextCursor = &c
	}
	return p, nil
}

// CaseDetail is one Case with its lead facts and links.
type CaseDetail struct {
	Case    store.Case
	Lead    *store.Lead
	Aliases []store.Alias
}

// GetCase returns one Case.
func (s *Service) GetCase(ctx context.Context, id string) (CaseDetail, error) {
	if !uuidPattern.MatchString(id) {
		return CaseDetail{}, invalid("case_id")
	}
	var d CaseDetail
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		if d.Case, err = store.GetCase(ctx, tx, id); err != nil {
			return err
		}
		if d.Lead, err = store.LeadOf(ctx, tx, id); err != nil {
			return err
		}
		d.Aliases, err = store.AliasesOf(ctx, tx, id)
		return err
	})
	return d, err
}

// HistoryPage is one page of a Case's state history.
type HistoryPage struct {
	Rows       []store.History
	NextCursor *string
}

// History pages the append-only state history in source order; the cursor
// is bound to that order ("ordinal").
func (s *Service) History(ctx context.Context, id string, cursor *string, limitIn *int) (HistoryPage, error) {
	if !uuidPattern.MatchString(id) {
		return HistoryPage{}, invalid("case_id")
	}
	limit, err := pagination.Limit(limitIn)
	if err != nil {
		return HistoryPage{}, invalid("%v", err)
	}
	var after *string
	if cursor != nil {
		c, err := pagination.Decode(*cursor, "ordinal")
		if err != nil || !ordinalPattern.MatchString(c.ID) {
			return HistoryPage{}, invalid("cursor")
		}
		after = &c.ID
	}
	var p HistoryPage
	err = s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		if _, err := store.GetCase(ctx, tx, id); err != nil {
			return err
		}
		rows, ords, err := store.HistoryOf(ctx, tx, id, after, limit)
		if err != nil {
			return err
		}
		p.Rows = rows
		if len(rows) > limit {
			p.Rows = rows[:limit]
			c, err := pagination.Encode(pagination.Cursor{Sort: "ordinal", ID: ords[limit-1]})
			if err != nil {
				return err
			}
			p.NextCursor = &c
		}
		return nil
	})
	return p, err
}

// Attribution returns every immutable snapshot of a Case.
func (s *Service) Attribution(ctx context.Context, id string) ([]store.Snapshot, error) {
	if !uuidPattern.MatchString(id) {
		return nil, invalid("case_id")
	}
	var out []store.Snapshot
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		if _, err := store.GetCase(ctx, tx, id); err != nil {
			return err
		}
		var err error
		out, err = store.SnapshotsOf(ctx, tx, id)
		return err
	})
	return out, err
}

// ResolveSources implements api.Reader.
func (s *Service) ResolveSources(ctx context.Context, system, spec string, keys []string) (map[string]api.SourceCase, error) {
	var out map[string]api.SourceCase
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		out, err = store.ResolveSources(ctx, tx, system, spec, keys)
		return err
	})
	return out, err
}

// CoverageReport implements api.Reader.
func (s *Service) CoverageReport(ctx context.Context) (api.Coverage, error) {
	var r api.Coverage
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		r, err = store.Coverage(ctx, tx)
		return err
	})
	return r, err
}
