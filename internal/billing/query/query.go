// Package query is the billing read service. Every read runs in a
// billing-owned READ ONLY transaction: it never imports, repairs, recomputes
// persisted state or records anything.
package query

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/domain"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/store"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pagination"
)

var (
	// ErrInvalidQuery reports a filter or cursor outside the contract.
	ErrInvalidQuery = errors.New("billing/query: invalid query")
	// ErrNotFound reports an unknown invoice.
	ErrNotFound = store.ErrNotFound
)

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func invalid(format string, a ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrInvalidQuery}, a...)...)
}

// Service reads the billing model.
type Service struct{ runner *db.Runner }

var _ api.Reader = (*Service)(nil)

// New returns the read service over a billing-owned runner.
func New(runner *db.Runner) *Service { return &Service{runner: runner} }

func (s *Service) read(ctx context.Context, fn func(ctx context.Context, tx db.Tx) error) error {
	return s.runner.InTx(ctx, db.TxOptions{ReadOnly: true, Isolation: db.RepeatableRead}, fn)
}

var links = map[string]bool{"RESOLVED": true, "RELATED": true, "UNKNOWN": true, "CONFLICT": true, "NOT_RECOVERABLE": true}

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

func nextCursor(n, limit int, last string) (*string, error) {
	if n <= limit {
		return nil, nil
	}
	c, err := pagination.Encode(pagination.Cursor{Sort: "id", ID: last})
	return &c, err
}

// InvoicesPage is one page of invoices.
type InvoicesPage struct {
	Invoices   []store.Invoice
	NextCursor *string
}

// ListInvoices lists invoices with allowlisted filters and a keyset cursor.
func (s *Service) ListInvoices(ctx context.Context, f store.InvoiceFilter, cursor *string, limitIn *int) (InvoicesPage, error) {
	if f.Presence != nil && *f.Presence != "PRESENT" && *f.Presence != "ABSENT_IN_LATEST" {
		return InvoicesPage{}, invalid("presence")
	}
	if (f.CustomerLink != nil && !links[*f.CustomerLink]) || (f.CaseLink != nil && !links[*f.CaseLink]) {
		return InvoicesPage{}, invalid("link state")
	}
	limit, err := pagination.Limit(limitIn)
	if err != nil {
		return InvoicesPage{}, invalid("%v", err)
	}
	after, err := cursorOf(cursor)
	if err != nil {
		return InvoicesPage{}, err
	}
	var rows []store.Invoice
	if err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		rows, err = store.ListInvoices(ctx, tx, f, after, limit)
		return err
	}); err != nil {
		return InvoicesPage{}, err
	}
	p := InvoicesPage{Invoices: rows}
	if len(rows) > limit {
		p.Invoices = rows[:limit]
	}
	if len(p.Invoices) > 0 {
		p.NextCursor, err = nextCursor(len(rows), limit, p.Invoices[len(p.Invoices)-1].InvoiceID)
	}
	return p, err
}

// GetInvoice returns one invoice with its state and declared evidence.
func (s *Service) GetInvoice(ctx context.Context, id string) (store.Invoice, error) {
	if !uuidPattern.MatchString(id) {
		return store.Invoice{}, invalid("invoice_id")
	}
	var v store.Invoice
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		v, err = store.GetInvoice(ctx, tx, id)
		return err
	})
	return v, err
}

// ItemsPage is one page of lines.
type ItemsPage struct {
	Items      []store.Item
	NextCursor *string
}

// Items pages an invoice's lines.
func (s *Service) Items(ctx context.Context, id string, cursor *string, limitIn *int) (ItemsPage, error) {
	if !uuidPattern.MatchString(id) {
		return ItemsPage{}, invalid("invoice_id")
	}
	limit, err := pagination.Limit(limitIn)
	if err != nil {
		return ItemsPage{}, invalid("%v", err)
	}
	after, err := cursorOf(cursor)
	if err != nil {
		return ItemsPage{}, err
	}
	var p ItemsPage
	err = s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		if _, err := store.GetInvoice(ctx, tx, id); err != nil {
			return err
		}
		rows, err := store.ItemsOf(ctx, tx, id, after, limit)
		if err != nil {
			return err
		}
		p.Items = rows
		if len(rows) > limit {
			p.Items = rows[:limit]
			p.NextCursor, err = nextCursor(len(rows), limit, p.Items[limit-1].ItemID)
		}
		return err
	})
	return p, err
}

// StagesPage is one page of stages.
type StagesPage struct {
	Stages     []store.Stage
	NextCursor *string
}

// Stages pages an invoice's payment stages.
func (s *Service) Stages(ctx context.Context, id string, cursor *string, limitIn *int) (StagesPage, error) {
	if !uuidPattern.MatchString(id) {
		return StagesPage{}, invalid("invoice_id")
	}
	limit, err := pagination.Limit(limitIn)
	if err != nil {
		return StagesPage{}, invalid("%v", err)
	}
	after, err := cursorOf(cursor)
	if err != nil {
		return StagesPage{}, err
	}
	var p StagesPage
	err = s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		if _, err := store.GetInvoice(ctx, tx, id); err != nil {
			return err
		}
		rows, err := store.StagesOf(ctx, tx, id, after, limit)
		if err != nil {
			return err
		}
		p.Stages = rows
		if len(rows) > limit {
			p.Stages = rows[:limit]
			p.NextCursor, err = nextCursor(len(rows), limit, p.Stages[limit-1].StageID)
		}
		return err
	})
	return p, err
}

// ComparedDeclared are the declared amount kinds compared as invoice totals.
// They are compared side by side; none is selected as canonical (TAD #19).
var ComparedDeclared = map[string]bool{"FINAL_TOTAL": true, "PAYMENT_TOTAL_AMOUNT": true}

// UnattributedUnit labels the totals row of an invoice whose amounts cannot
// be attributed to any unit (all unrepresentable or absent). It is not a
// currency and is never summed.
const UnattributedUnit = "UNATTRIBUTED"

// UnitTotals is the total evidence of one invoice in one unit.
type UnitTotals struct {
	Unit       string
	Declared   map[string]*int64 // compared declared kinds present in this unit
	ItemSum    domain.Sum
	StageSum   domain.Sum
	PaymentSum domain.Sum
	Class      string
}

// PaymentSum is the accepted-attempt sum of one unit (from payments).
type PaymentSum = domain.Sum

// Totals computes the per-unit total evidence of an invoice. payments are
// the accepted-attempt sums per unit supplied by the payments read API and
// paymentsComplete says whether that evidence is complete (an incomplete
// payment source is INCOMPLETE in every unit, never zero). Units are never
// combined and no amount is selected as canonical.
func (s *Service) Totals(ctx context.Context, id string, payments map[string]PaymentSum, paymentsComplete bool) ([]UnitTotals, error) {
	if !uuidPattern.MatchString(id) {
		return nil, invalid("invoice_id")
	}
	var out []UnitTotals
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		out, err = totalsTx(ctx, tx, id, payments, paymentsComplete)
		return err
	})
	return out, err
}

func totalsTx(ctx context.Context, tx db.Tx, id string, payments map[string]PaymentSum, paymentsComplete bool) ([]UnitTotals, error) {
	var out []UnitTotals
	err := func() error {
		v, err := store.GetInvoice(ctx, tx, id)
		if err != nil {
			return err
		}
		items, stages, err := store.Amounts(ctx, tx, id)
		if err != nil {
			return err
		}
		// A row whose amount is absent or unrepresentable cannot be attributed
		// to a unit: every unit of that evidence is then INCOMPLETE (never NONE,
		// never zero).
		itemsOK := v.ItemsComplete && allRepresentable(items)
		stagesOK := v.StagesComplete && allRepresentable(stages)
		itemSums := domain.SumByUnit(items, itemsOK)
		stageSums := domain.SumByUnit(stages, stagesOK)
		declared := map[string]map[string]*int64{}
		declaredUnrep := false
		for _, d := range v.Declared {
			if !ComparedDeclared[d.Kind] {
				continue
			}
			if d.Representation == domain.Unrepresentable {
				declaredUnrep = true
				continue
			}
			if d.Representation == domain.Representable && d.Unit != nil {
				if declared[*d.Unit] == nil {
					declared[*d.Unit] = map[string]*int64{}
				}
				declared[*d.Unit][d.Kind] = d.Amount
			}
		}
		units := map[string]bool{}
		for u := range declared {
			units[u] = true
		}
		for _, m := range []map[string]domain.Sum{itemSums, stageSums, payments} {
			for u := range m {
				units[u] = true
			}
		}
		sorted := make([]string, 0, len(units))
		for u := range units {
			sorted = append(sorted, u)
		}
		sort.Strings(sorted)
		if len(sorted) == 0 {
			// No amount could be attributed to any unit: report the invoice
			// explicitly instead of dropping it from the totals.
			state := func(ok bool) domain.Sum {
				if ok {
					return domain.Sum{Unit: UnattributedUnit, State: domain.SumNone}
				}
				return domain.Sum{Unit: UnattributedUnit, State: domain.SumIncomplete}
			}
			t := UnitTotals{Unit: UnattributedUnit, ItemSum: state(itemsOK), StageSum: state(stagesOK), PaymentSum: state(paymentsComplete),
				Class: domain.Unknown}
			if declaredUnrep || !itemsOK || !stagesOK || !paymentsComplete {
				t.Class = domain.Incomplete
			}
			out = append(out, t)
			return nil
		}
		for _, u := range sorted {
			t := UnitTotals{Unit: u, Declared: declared[u], ItemSum: sumOr(itemSums, u, itemsOK),
				StageSum: sumOr(stageSums, u, stagesOK), PaymentSum: sumOr(payments, u, paymentsComplete)}
			var ev []domain.Evidence
			kinds := make([]string, 0, len(t.Declared))
			for k := range t.Declared {
				kinds = append(kinds, k)
			}
			sort.Strings(kinds)
			for _, k := range kinds {
				ev = append(ev, domain.Evidence{Name: "declared:" + k, Amount: t.Declared[k], State: domain.Representable})
			}
			if declaredUnrep {
				ev = append(ev, domain.Evidence{Name: "declared:unrepresentable", State: domain.Unrepresentable})
			}
			ev = append(ev, domain.Evidence{Name: "item_sum", Amount: t.ItemSum.Amount, State: t.ItemSum.State},
				domain.Evidence{Name: "stage_sum", Amount: t.StageSum.Amount, State: t.StageSum.State},
				domain.Evidence{Name: "payment_sum", Amount: t.PaymentSum.Amount, State: t.PaymentSum.State})
			t.Class = domain.Compare(ev)
			out = append(out, t)
		}
		return nil
	}()
	return out, err
}

func allRepresentable(as []domain.Amount) bool {
	for _, a := range as {
		if a.Representation != domain.Representable {
			return false
		}
	}
	return true
}

// sumOr returns the unit's sum or NONE; an incomplete source with no rows
// in this unit is INCOMPLETE, never zero.
func sumOr(m map[string]domain.Sum, unit string, complete bool) domain.Sum {
	if s, ok := m[unit]; ok {
		if !complete && s.State == domain.SumComplete {
			s.State = domain.SumIncomplete
		}
		return s
	}
	if !complete {
		return domain.Sum{Unit: unit, State: domain.SumIncomplete}
	}
	return domain.Sum{Unit: unit, State: domain.SumNone}
}

// ResolveInvoices implements api.Reader.
func (s *Service) ResolveInvoices(ctx context.Context, system, spec string, keys []string) (map[string]api.SourceInvoice, error) {
	var out map[string]api.SourceInvoice
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		out, err = store.ResolveInvoices(ctx, tx, system, spec, keys)
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

// TotalClasses implements api.Reader: one read-only transaction over all
// requested invoices.
func (s *Service) TotalClasses(ctx context.Context, invoiceIDs []string, payments map[string]api.PaymentEvidence) (map[string]map[string]string, error) {
	out := map[string]map[string]string{}
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		for _, id := range invoiceIDs {
			if !uuidPattern.MatchString(id) {
				return invalid("invoice_id")
			}
			p, ok := payments[id]
			sums := map[string]PaymentSum{}
			for u, v := range p.Units {
				sums[u] = PaymentSum{Unit: u, Amount: v.Amount, State: v.State, Rows: v.Rows}
			}
			ts, err := totalsTx(ctx, tx, id, sums, ok && p.Complete)
			if err != nil {
				return err
			}
			out[id] = map[string]string{}
			for _, t := range ts {
				out[id][t.Unit] = t.Class
			}
		}
		return nil
	})
	return out, err
}

var _ api.FactSource = (*Service)(nil)

const factPage = 1000

// InvoiceFacts implements api.FactSource over one read-only snapshot.
func (s *Service) InvoiceFacts(ctx context.Context, fn func(api.InvoiceFact) error) error {
	return s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		states, err := store.AllStates(ctx, tx)
		if err != nil {
			return err
		}
		var after *string
		n := 0
		for {
			rows, err := store.ListInvoices(ctx, tx, store.InvoiceFilter{}, after, factPage)
			if err != nil {
				return err
			}
			more := len(rows) > factPage
			if more {
				rows = rows[:factPage]
			}
			for _, v := range rows {
				if n++; n > api.MaxFacts {
					return api.ErrTooManyFacts
				}
				if err := fn(api.InvoiceFact{InvoiceID: v.InvoiceID, SourceSpec: v.Source.Spec, SourceKey: v.Source.Key, CodeRaw: v.CodeRaw,
					Presence: v.Source.Presence, CustomerLink: v.CustomerLink, CustomerID: resolvedCustomer(v), CaseLink: v.CaseLink, SellerKey: v.SellerKey,
					States: states[v.InvoiceID], ItemCount: v.ItemCount, StageCount: v.StageCount, UnknownStates: v.UnknownStates,
					ItemsComplete: v.ItemsComplete, StagesComplete: v.StagesComplete, FirstSeenAt: v.Source.FirstSeen}); err != nil {
					return err
				}
			}
			if !more {
				return nil
			}
			last := rows[len(rows)-1].InvoiceID
			after = &last
		}
	})
}

func resolvedCustomer(v store.Invoice) *string {
	if v.CustomerLink != "RESOLVED" {
		return nil
	}
	return v.CustomerID
}

// InvoiceAmountEvidence implements api.AmountEvidenceSource: per invoice and
// unit, the declared totals compared with item and stage sums (payment
// evidence not compared) plus the legacy declared paid / remaining figures,
// in one read-only snapshot.
func (s *Service) InvoiceAmountEvidence(ctx context.Context, invoiceIDs []string) (map[string][]api.AmountEvidence, error) {
	out := map[string][]api.AmountEvidence{}
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		for _, id := range invoiceIDs {
			if !uuidPattern.MatchString(id) {
				return invalid("invoice_id")
			}
			// No payment evidence: payment_sum is NONE and never compared.
			ts, err := totalsTx(ctx, tx, id, nil, true)
			if err != nil {
				return err
			}
			v, err := store.GetInvoice(ctx, tx, id)
			if err != nil {
				return err
			}
			legacy := func(kind, unit string) (*int64, string) {
				for _, d := range v.Declared {
					if d.Kind != kind {
						continue
					}
					if d.Representation == domain.Representable && d.Unit != nil && *d.Unit == unit {
						return d.Amount, d.Representation
					}
					if d.Representation == domain.Representable {
						// declared in another unit: never compared across units
						return nil, api.OtherUnit
					}
					return nil, d.Representation
				}
				return nil, domain.Absent
			}
			for _, t := range ts {
				conv := func(x domain.Sum) api.PaymentUnitSum {
					return api.PaymentUnitSum{Amount: x.Amount, State: x.State, Rows: x.Rows}
				}
				e := api.AmountEvidence{Unit: t.Unit, Declared: t.Declared, ItemSum: conv(t.ItemSum), StageSum: conv(t.StageSum), Class: t.Class}
				e.LegacyPaid, e.LegacyPaidRepr = legacy("PAID_TOTAL_AMOUNT", t.Unit)
				e.LegacyRemaining, e.LegacyRemainingRepr = legacy("REMAINING_AMOUNT", t.Unit)
				out[id] = append(out[id], e)
			}
		}
		return nil
	})
	return out, err
}

// InvoiceUnitTotals implements api.FactSource: the same per-unit evidence
// as Totals, for many invoices, in one read-only snapshot.
func (s *Service) InvoiceUnitTotals(ctx context.Context, invoiceIDs []string, payments map[string]api.PaymentEvidence) (map[string][]api.UnitTotal, error) {
	out := map[string][]api.UnitTotal{}
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		for _, id := range invoiceIDs {
			if !uuidPattern.MatchString(id) {
				return invalid("invoice_id")
			}
			p, ok := payments[id]
			sums := map[string]PaymentSum{}
			for u, v := range p.Units {
				sums[u] = PaymentSum{Unit: u, Amount: v.Amount, State: v.State, Rows: v.Rows}
			}
			ts, err := totalsTx(ctx, tx, id, sums, ok && p.Complete)
			if err != nil {
				return err
			}
			for _, t := range ts {
				conv := func(x domain.Sum) api.PaymentUnitSum {
					return api.PaymentUnitSum{Amount: x.Amount, State: x.State, Rows: x.Rows}
				}
				out[id] = append(out[id], api.UnitTotal{Unit: t.Unit, Declared: t.Declared, ItemSum: conv(t.ItemSum), StageSum: conv(t.StageSum),
					PaySum: conv(t.PaymentSum), Class: t.Class})
			}
		}
		return nil
	})
	return out, err
}
