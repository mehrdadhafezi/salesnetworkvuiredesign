// Package query is the payments read service. Every read runs in a
// payments-owned READ ONLY transaction: it never imports, repairs, recomputes
// persisted state or records anything.
package query

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/store"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pagination"
)

var (
	// ErrInvalidQuery reports a filter or cursor outside the contract.
	ErrInvalidQuery = errors.New("payments/query: invalid query")
	// ErrNotFound reports an unknown attempt or receipt.
	ErrNotFound = store.ErrNotFound
)

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func invalid(format string, a ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrInvalidQuery}, a...)...)
}

// Service reads the payments model.
type Service struct{ runner *db.Runner }

var _ api.Reader = (*Service)(nil)

// New returns the read service over a payments-owned runner.
func New(runner *db.Runner) *Service { return &Service{runner: runner} }

func (s *Service) read(ctx context.Context, fn func(ctx context.Context, tx db.Tx) error) error {
	return s.runner.InTx(ctx, db.TxOptions{ReadOnly: true, Isolation: db.RepeatableRead}, fn)
}

// Allowlisted filter vocabularies.
var (
	Channels     = map[string]bool{api.ChannelGateway: true, api.ChannelManualCard: true, api.ChannelUnknown: true}
	Mappings     = map[string]bool{"MAPPED": true, "UNKNOWN": true, "EMPTY": true}
	Targets      = map[string]bool{"PENDING": true, "ACCEPTED": true, "REJECTED": true, "CANCELLED": true, "RECONCILIATION_REQUIRED": true}
	Provenances  = map[string]bool{api.ProvGatewayAuto: true, api.ProvLegacyManualApproval: true, api.ProvNotApproved: true, api.ProvUnknown: true}
	Presences    = map[string]bool{"PRESENT": true, "ABSENT_IN_LATEST": true}
	InvoiceLinks = map[string]bool{"RESOLVED": true, "UNKNOWN": true}
)

func cursorOf(cursor *string) (*string, error) {
	if cursor == nil {
		return nil, nil
	}
	c, err := pagination.Decode(*cursor, "id")
	if err != nil || !uuidPattern.MatchString(c.ID) {
		return nil, invalid("cursor")
	}
	return &c.ID, nil
}

func page[T any](rows []T, limit int, id func(T) string) ([]T, *string, error) {
	if len(rows) <= limit {
		return rows, nil, nil
	}
	rows = rows[:limit]
	c, err := pagination.Encode(pagination.Cursor{Sort: "id", ID: id(rows[limit-1])})
	return rows, &c, err
}

func bounds(cursor *string, limitIn *int) (*string, int, error) {
	limit, err := pagination.Limit(limitIn)
	if err != nil {
		return nil, 0, invalid("%v", err)
	}
	after, err := cursorOf(cursor)
	return after, limit, err
}

func check(v *string, allowed map[string]bool, name string) error {
	if v != nil && !allowed[*v] {
		return invalid("%s", name)
	}
	return nil
}

// AttemptsPage is one page of attempts.
type AttemptsPage struct {
	Attempts   []store.Attempt
	NextCursor *string
}

// ListAttempts lists attempts with allowlisted filters and a keyset cursor.
func (s *Service) ListAttempts(ctx context.Context, f store.AttemptFilter, cursor *string, limitIn *int) (AttemptsPage, error) {
	if f.InvoiceID != nil && !uuidPattern.MatchString(*f.InvoiceID) {
		return AttemptsPage{}, invalid("invoice_id")
	}
	for _, c := range []error{check(f.Channel, Channels, "channel"), check(f.StateMapping, Mappings, "state_mapping"),
		check(f.StateTarget, Targets, "state"), check(f.Provenance, Provenances, "approval_provenance"),
		check(f.Presence, Presences, "presence"), check(f.InvoiceLink, InvoiceLinks, "invoice_link")} {
		if c != nil {
			return AttemptsPage{}, c
		}
	}
	after, limit, err := bounds(cursor, limitIn)
	if err != nil {
		return AttemptsPage{}, err
	}
	var p AttemptsPage
	err = s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		rows, err := store.ListAttempts(ctx, tx, f, after, limit)
		if err != nil {
			return err
		}
		p.Attempts, p.NextCursor, err = page(rows, limit, func(a store.Attempt) string { return a.AttemptID })
		return err
	})
	return p, err
}

// GetAttempt returns one attempt.
func (s *Service) GetAttempt(ctx context.Context, id string) (store.Attempt, error) {
	if !uuidPattern.MatchString(id) {
		return store.Attempt{}, invalid("attempt_id")
	}
	var a store.Attempt
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		a, err = store.GetAttempt(ctx, tx, id)
		return err
	})
	return a, err
}

// HistoryPage is one page of attempt state history.
type HistoryPage struct {
	Entries    []store.HistoryEntry
	NextCursor *string
}

// History pages an attempt's observed raw state changes.
func (s *Service) History(ctx context.Context, id string, cursor *string, limitIn *int) (HistoryPage, error) {
	if !uuidPattern.MatchString(id) {
		return HistoryPage{}, invalid("attempt_id")
	}
	after, limit, err := bounds(cursor, limitIn)
	if err != nil {
		return HistoryPage{}, err
	}
	var p HistoryPage
	err = s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		if _, err := store.GetAttempt(ctx, tx, id); err != nil {
			return err
		}
		rows, err := store.History(ctx, tx, id, after, limit)
		if err != nil {
			return err
		}
		p.Entries, p.NextCursor, err = page(rows, limit, func(h store.HistoryEntry) string { return h.HistoryID })
		return err
	})
	return p, err
}

// ReceiptsPage is one page of receipt references.
type ReceiptsPage struct {
	Receipts   []store.Receipt
	NextCursor *string
}

// ReceiptsOf pages an invoice's receipt references. An invoice id unknown
// to payments yields an empty page (payments does not own invoices).
func (s *Service) ReceiptsOf(ctx context.Context, invoiceID string, cursor *string, limitIn *int) (ReceiptsPage, error) {
	if !uuidPattern.MatchString(invoiceID) {
		return ReceiptsPage{}, invalid("invoice_id")
	}
	after, limit, err := bounds(cursor, limitIn)
	if err != nil {
		return ReceiptsPage{}, err
	}
	var p ReceiptsPage
	err = s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		rows, err := store.ReceiptsOf(ctx, tx, invoiceID, after, limit)
		if err != nil {
			return err
		}
		p.Receipts, p.NextCursor, err = page(rows, limit, func(r store.Receipt) string { return r.ReceiptID })
		return err
	})
	return p, err
}

// GetReceipt returns one receipt reference.
func (s *Service) GetReceipt(ctx context.Context, id string) (store.Receipt, error) {
	if !uuidPattern.MatchString(id) {
		return store.Receipt{}, invalid("receipt_id")
	}
	var r store.Receipt
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		r, err = store.GetReceipt(ctx, tx, id)
		return err
	})
	return r, err
}

// AcceptedSums implements api.Reader.
func (s *Service) AcceptedSums(ctx context.Context, invoiceIDs []string) (map[string]api.InvoicePayments, error) {
	for _, id := range invoiceIDs {
		if !uuidPattern.MatchString(id) {
			return nil, invalid("invoice_id")
		}
	}
	var out map[string]api.InvoicePayments
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		out, err = store.AcceptedSums(ctx, tx, invoiceIDs)
		return err
	})
	return out, err
}

// RowProvenance implements api.Reader.
func (s *Service) RowProvenance(ctx context.Context, system, spec string, keys []string) (map[string]api.RowProvenance, error) {
	var out map[string]api.RowProvenance
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		out, err = store.RowProvenance(ctx, tx, system, spec, keys)
		return err
	})
	return out, err
}

// BatchImported implements api.Reader.
func (s *Service) BatchImported(ctx context.Context, batchID string) (bool, error) {
	var ok bool
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		ok, err = store.BatchImported(ctx, tx, batchID)
		return err
	})
	return ok, err
}

// Coverage implements api.Reader.
func (s *Service) Coverage(ctx context.Context) (api.Coverage, error) {
	var r api.Coverage
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		r, err = store.Coverage(ctx, tx)
		return err
	})
	return r, err
}

var _ api.FactSource = (*Service)(nil)

const factPage = 1000

// AttemptFacts implements api.FactSource over one read-only snapshot.
func (s *Service) AttemptFacts(ctx context.Context, fn func(api.AttemptFact) error) error {
	return s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var after *string
		n := 0
		for {
			rows, err := store.ListAttempts(ctx, tx, store.AttemptFilter{}, after, factPage)
			if err != nil {
				return err
			}
			more := len(rows) > factPage
			if more {
				rows = rows[:factPage]
			}
			for _, a := range rows {
				if n++; n > api.MaxFacts {
					return api.ErrTooManyFacts
				}
				f := api.AttemptFact{AttemptID: a.AttemptID, InvoiceID: a.InvoiceID, InvoiceLink: a.InvoiceLink, Channel: a.Channel,
					StateRaw: a.StateRaw, StateMapping: a.StateMapping, StateTarget: a.StateTarget, Provenance: a.ApprovalProvenance,
					AmountRaw: a.AmountRaw, Amount: a.Amount, Unit: a.Unit, AmountRepr: a.AmountRepr, Presence: a.Source.Presence,
					FirstSeenAt: a.Source.FirstSeen}
				if a.Gateway != nil {
					f.GatewayState = a.Gateway.StateTarget
				}
				if err := fn(f); err != nil {
					return err
				}
			}
			if !more {
				return nil
			}
			last := rows[len(rows)-1].AttemptID
			after = &last
		}
	})
}
