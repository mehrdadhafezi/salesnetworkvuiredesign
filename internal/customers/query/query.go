// Package query is the customers read service. Every call runs in a
// read-only, customers-owned transaction: a read can never mint, merge,
// repair or refresh anything. Filters are allowlisted.
package query

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers/store"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pagination"
)

// ErrInvalidQuery reports a filter, cursor or limit outside the allowlist.
var ErrInvalidQuery = errors.New("customers/query: invalid query")

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func invalid(format string, a ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrInvalidQuery}, a...)...)
}

// Service reads the customers model.
type Service struct{ runner *db.Runner }

var _ api.Reader = (*Service)(nil)

// New returns the read service over a customers-owned runner.
func New(runner *db.Runner) *Service { return &Service{runner: runner} }

func (s *Service) read(ctx context.Context, fn func(ctx context.Context, tx db.Tx) error) error {
	return s.runner.InTx(ctx, db.TxOptions{ReadOnly: true, Isolation: db.RepeatableRead}, fn)
}

func page(cursor *string, limitIn *int) (*string, int, error) {
	limit, err := pagination.Limit(limitIn)
	if err != nil {
		return nil, 0, invalid("%v", err)
	}
	if cursor == nil {
		return nil, limit, nil
	}
	c, err := pagination.Decode(*cursor, "id")
	if err != nil || !uuidPattern.MatchString(c.ID) {
		return nil, 0, invalid("cursor")
	}
	return &c.ID, limit, nil
}

func next(n, limit int, lastID func() string) (*string, error) {
	if n <= limit {
		return nil, nil
	}
	c, err := pagination.Encode(pagination.Cursor{Sort: "id", ID: lastID()})
	return &c, err
}

// CustomersPage is one page of customers.
type CustomersPage struct {
	Customers  []store.Customer
	NextCursor *string
}

// ListCustomers returns one page.
func (s *Service) ListCustomers(ctx context.Context, state *string, hasConflict *bool, presence, cursor *string, limitIn *int) (CustomersPage, error) {
	if state != nil && !store.ResolutionStates[*state] {
		return CustomersPage{}, invalid("state")
	}
	if presence != nil && *presence != "PRESENT" && *presence != "ABSENT_IN_LATEST" {
		return CustomersPage{}, invalid("presence")
	}
	after, limit, err := page(cursor, limitIn)
	if err != nil {
		return CustomersPage{}, err
	}
	var rows []store.Customer
	if err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		rows, err = store.ListCustomers(ctx, tx, state, hasConflict, presence, after, limit)
		return err
	}); err != nil {
		return CustomersPage{}, err
	}
	p := CustomersPage{Customers: rows}
	if len(rows) > limit {
		p.Customers = rows[:limit]
	}
	p.NextCursor, err = next(len(rows), limit, func() string { return p.Customers[limit-1].CustomerID })
	return p, err
}

// CustomerDetail is one customer with its identifiers and cases.
type CustomerDetail struct {
	Customer    store.Customer
	Identifiers []store.IdentifierView
	Cases       []store.Case
}

// GetCustomer returns one customer.
func (s *Service) GetCustomer(ctx context.Context, id string) (CustomerDetail, bool, error) {
	if !uuidPattern.MatchString(id) {
		return CustomerDetail{}, false, invalid("customer id")
	}
	var (
		d  CustomerDetail
		ok bool
	)
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		if d.Customer, ok, err = store.GetCustomer(ctx, tx, id); err != nil || !ok {
			return err
		}
		if d.Identifiers, err = store.IdentifiersOf(ctx, tx, id); err != nil {
			return err
		}
		d.Cases, err = store.ListCases(ctx, tx, store.CaseFilter{CustomerID: &id}, nil, pagination.MaxLimit)
		return err
	})
	return d, ok, err
}

// Contacts lists a customer's masked contact points (nil, false: unknown customer).
func (s *Service) Contacts(ctx context.Context, id string) ([]store.ContactView, bool, error) {
	if !uuidPattern.MatchString(id) {
		return nil, false, invalid("customer id")
	}
	var (
		out []store.ContactView
		ok  bool
	)
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		if _, ok, err = store.GetCustomer(ctx, tx, id); err != nil || !ok {
			return err
		}
		out, err = store.ContactsOf(ctx, tx, id)
		return err
	})
	return out, ok, err
}

// EvidencePage is one page of evidence.
type EvidencePage struct {
	Evidence   []store.EvidenceView
	NextCursor *string
}

// Evidence lists the evidence about a customer.
func (s *Service) Evidence(ctx context.Context, id string, strength, cursor *string, limitIn *int) (EvidencePage, bool, error) {
	if !uuidPattern.MatchString(id) {
		return EvidencePage{}, false, invalid("customer id")
	}
	if strength != nil && !store.Strengths[*strength] {
		return EvidencePage{}, false, invalid("strength")
	}
	after, limit, err := page(cursor, limitIn)
	if err != nil {
		return EvidencePage{}, false, err
	}
	var (
		rows []store.EvidenceView
		ok   bool
	)
	err = s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		if _, ok, err = store.GetCustomer(ctx, tx, id); err != nil || !ok {
			return err
		}
		rows, err = store.EvidenceOf(ctx, tx, id, strength, after, limit)
		return err
	})
	if err != nil || !ok {
		return EvidencePage{}, ok, err
	}
	p := EvidencePage{Evidence: rows}
	if len(rows) > limit {
		p.Evidence = rows[:limit]
	}
	p.NextCursor, err = next(len(rows), limit, func() string { return p.Evidence[limit-1].ID })
	return p, true, err
}

// CasesPage is one page of merge cases.
type CasesPage struct {
	Cases      []store.Case
	NextCursor *string
}

// ListCases returns one page of the read-only merge-case queue.
func (s *Service) ListCases(ctx context.Context, f store.CaseFilter, cursor *string, limitIn *int) (CasesPage, error) {
	if f.State != nil && !store.ResolutionStates[*f.State] {
		return CasesPage{}, invalid("state")
	}
	if f.Strength != nil && !store.Strengths[*f.Strength] {
		return CasesPage{}, invalid("evidence_strength")
	}
	if f.Reason != nil && !store.CaseReasons[*f.Reason] {
		return CasesPage{}, invalid("reason")
	}
	after, limit, err := page(cursor, limitIn)
	if err != nil {
		return CasesPage{}, err
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
	}
	p.NextCursor, err = next(len(rows), limit, func() string { return p.Cases[limit-1].ID })
	return p, err
}

// GetCase returns one merge case.
func (s *Service) GetCase(ctx context.Context, id string) (store.Case, bool, error) {
	if !uuidPattern.MatchString(id) {
		return store.Case{}, false, invalid("merge case id")
	}
	var (
		c  store.Case
		ok bool
	)
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		c, ok, err = store.GetCase(ctx, tx, id)
		return err
	})
	return c, ok, err
}

// ResolutionReport implements api.Reader.
func (s *Service) ResolutionReport(ctx context.Context) (api.Resolution, error) {
	var r api.Resolution
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		r, err = store.Report(ctx, tx)
		return err
	})
	return r, err
}
