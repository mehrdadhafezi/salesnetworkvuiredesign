package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/domain"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/money"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
)

// Provenance is the source lineage of one row.
type Provenance struct {
	System, Spec, Key, FirstBatch, LastBatch, Presence string
	FirstSeen, LastSeen                                time.Time
}

// State is one invoice state evidence row.
type State struct {
	Field, Namespace string
	Raw              *string
	Mapping          string
}

// DeclaredAmount is one declared amount evidence row.
type DeclaredAmount struct {
	Kind           string
	Raw            *string
	Amount         *int64
	Unit           *string
	Representation string
}

// Invoice is the read view of one invoice (no PII is stored at all).
type Invoice struct {
	InvoiceID                            string
	Source                               Provenance
	CodeRaw, ProductRefRaw, PayMethodRaw *string
	PaymentPlanRaw, CurrentStageRaw      *string
	IssuedAtRaw, UpdatedAtRaw            *string
	SellerKey, SellerUserID              *string
	IssuedByKey, IssuedByUserID          *string
	Lifecycle                            string
	CustomerID, CaseID                   *string
	CustomerLink, CustomerBasis          string
	CaseLink, CaseBasis                  string
	ItemsComplete, StagesComplete        bool
	ItemCount, StageCount, UnknownStates int64
	States                               []State
	Declared                             []DeclaredAmount
}

const invoiceSelect = `SELECT i.invoice_id::text, i.source_system, i.source_spec, i.source_key, i.first_batch_id::text, i.last_batch_id::text,
	i.presence, i.first_seen_at, i.last_seen_at, i.invoice_code_raw, i.product_ref_raw, i.pay_method_raw, i.payment_plan_raw,
	i.current_stage_raw, i.issued_at_raw, i.updated_at_raw, i.seller_legacy_key, i.seller_user_id::text, i.issued_by_legacy_key,
	i.issued_by_user_id::text, i.lifecycle_state, i.customer_id::text, i.case_id::text, i.customer_link_state, i.customer_link_basis,
	i.case_link_state, i.case_link_basis, i.items_source_complete, i.stages_source_complete,
	(SELECT count(*) FROM billing.invoice_item x WHERE x.invoice_id = i.invoice_id AND x.presence = 'PRESENT'),
	(SELECT count(*) FROM billing.payment_stage x WHERE x.invoice_id = i.invoice_id AND x.presence = 'PRESENT'),
	(SELECT count(*) FROM billing.invoice_state_evidence e WHERE e.invoice_id = i.invoice_id AND e.mapping = 'UNKNOWN')
	 + (SELECT count(*) FROM billing.payment_stage x WHERE x.invoice_id = i.invoice_id AND x.presence = 'PRESENT' AND x.state_mapping = 'UNKNOWN')
	FROM billing.invoice i`

func scanInvoice(r pgx.Row) (Invoice, error) {
	var v Invoice
	s := &v.Source
	err := r.Scan(&v.InvoiceID, &s.System, &s.Spec, &s.Key, &s.FirstBatch, &s.LastBatch, &s.Presence, &s.FirstSeen, &s.LastSeen,
		&v.CodeRaw, &v.ProductRefRaw, &v.PayMethodRaw, &v.PaymentPlanRaw, &v.CurrentStageRaw, &v.IssuedAtRaw, &v.UpdatedAtRaw,
		&v.SellerKey, &v.SellerUserID, &v.IssuedByKey, &v.IssuedByUserID, &v.Lifecycle, &v.CustomerID, &v.CaseID,
		&v.CustomerLink, &v.CustomerBasis, &v.CaseLink, &v.CaseBasis, &v.ItemsComplete, &v.StagesComplete,
		&v.ItemCount, &v.StageCount, &v.UnknownStates)
	return v, err
}

// InvoiceFilter is the allowlisted filter.
type InvoiceFilter struct {
	Presence, CustomerLink, CaseLink *string
	HasUnknownStatus                 *bool
}

// ListInvoices lists invoices by invoice id (keyset).
func ListInvoices(ctx context.Context, tx db.Tx, f InvoiceFilter, after *string, limit int) ([]Invoice, error) {
	if err := ReadTx(tx); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT * FROM (`+invoiceSelect+`) v (invoice_id, source_system, source_spec, source_key, first_batch_id,
		last_batch_id, presence, first_seen_at, last_seen_at, c1, c2, c3, c4, c5, c6, c7, c8, c9, c10, c11, lifecycle_state, customer_id,
		case_id, customer_link_state, c12, case_link_state, c13, c14, c15, item_count, stage_count, unknown_states)
		WHERE ($1::text IS NULL OR presence = $1) AND ($2::text IS NULL OR customer_link_state = $2) AND ($3::text IS NULL OR case_link_state = $3)
		  AND ($4::boolean IS NULL OR (unknown_states > 0) = $4) AND ($5::uuid IS NULL OR invoice_id::uuid > $5)
		ORDER BY invoice_id::uuid LIMIT $6`, f.Presence, f.CustomerLink, f.CaseLink, f.HasUnknownStatus, after, limit+1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Invoice
	for rows.Next() {
		v, err := scanInvoice(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// GetInvoice returns one invoice with its state and declared amount evidence.
func GetInvoice(ctx context.Context, tx db.Tx, id string) (Invoice, error) {
	if err := ReadTx(tx); err != nil {
		return Invoice{}, err
	}
	v, err := scanInvoice(tx.QueryRow(ctx, invoiceSelect+` WHERE i.invoice_id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Invoice{}, ErrNotFound
	}
	if err != nil {
		return v, err
	}
	rows, err := tx.Query(ctx, `SELECT field, namespace, raw_value, mapping FROM billing.invoice_state_evidence WHERE invoice_id = $1 ORDER BY field`, id)
	if err != nil {
		return v, err
	}
	for rows.Next() {
		var s State
		if err := rows.Scan(&s.Field, &s.Namespace, &s.Raw, &s.Mapping); err != nil {
			rows.Close()
			return v, err
		}
		v.States = append(v.States, s)
	}
	rows.Close()
	rows, err = tx.Query(ctx, `SELECT kind, raw_value, amount, unit, representation FROM billing.amount_evidence WHERE invoice_id = $1 ORDER BY kind`, id)
	if err != nil {
		return v, err
	}
	defer rows.Close()
	for rows.Next() {
		var d DeclaredAmount
		if err := rows.Scan(&d.Kind, &d.Raw, &d.Amount, &d.Unit, &d.Representation); err != nil {
			return v, err
		}
		v.Declared = append(v.Declared, d)
	}
	return v, rows.Err()
}

// Item is one invoice line.
type Item struct {
	ItemID                                        string
	Source                                        Provenance
	ProductRefRaw, TitleSnapshot, KindRaw, QtyRaw *string
	UnitPriceRaw, LineTotalRaw                    *string
	UnitPrice, LineTotal                          *int64
	Unit                                          *string
	LineRepr                                      string
	IsFreeRaw, CreatedAtRaw                       *string
}

// ItemsOf pages an invoice's lines (keyset on item id).
func ItemsOf(ctx context.Context, tx db.Tx, invoiceID string, after *string, limit int) ([]Item, error) {
	if err := ReadTx(tx); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT item_id::text, source_system, source_spec, source_key, first_batch_id::text, last_batch_id::text, presence,
		first_seen_at, last_seen_at, product_ref_raw, title_snapshot, kind_raw, qty_raw, unit_price_raw, line_total_raw, unit_price, line_total,
		unit, line_total_repr, is_free_raw, created_at_raw FROM billing.invoice_item
		WHERE invoice_id = $1 AND ($2::uuid IS NULL OR item_id > $2) ORDER BY item_id LIMIT $3`, invoiceID, after, limit+1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Item
	for rows.Next() {
		var it Item
		s := &it.Source
		if err := rows.Scan(&it.ItemID, &s.System, &s.Spec, &s.Key, &s.FirstBatch, &s.LastBatch, &s.Presence, &s.FirstSeen, &s.LastSeen,
			&it.ProductRefRaw, &it.TitleSnapshot, &it.KindRaw, &it.QtyRaw, &it.UnitPriceRaw, &it.LineTotalRaw, &it.UnitPrice, &it.LineTotal,
			&it.Unit, &it.LineRepr, &it.IsFreeRaw, &it.CreatedAtRaw); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// Stage is one payment stage.
type Stage struct {
	StageID                                string
	Source                                 Provenance
	StageNoRaw, StageTypeRaw, RequestedRaw *string
	Requested                              *int64
	Unit                                   *string
	RequestedRepr, StateNamespace          string
	StateRaw                               *string
	StateMapping                           string
	StateTarget                            *string
	PayMethodRaw, PaymentSourceRaw         *string
	IssuedByKey, IssuedByUserID            *string
	ApprovedByKey, ApprovedByUserID        *string
	ApprovedAtRaw, PaidAtRaw, CreatedAtRaw *string
}

// StagesOf pages an invoice's stages (keyset on stage id).
func StagesOf(ctx context.Context, tx db.Tx, invoiceID string, after *string, limit int) ([]Stage, error) {
	if err := ReadTx(tx); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT stage_id::text, source_system, source_spec, source_key, first_batch_id::text, last_batch_id::text, presence,
		first_seen_at, last_seen_at, stage_no_raw, stage_type_raw, requested_raw, requested_amount, unit, requested_repr, state_namespace,
		state_raw, state_mapping, state_target, pay_method_raw, payment_source_raw, issued_by_legacy_key, issued_by_user_id::text,
		approved_by_legacy_key, approved_by_user_id::text, approved_at_raw, paid_at_raw, created_at_raw FROM billing.payment_stage
		WHERE invoice_id = $1 AND ($2::uuid IS NULL OR stage_id > $2) ORDER BY stage_id LIMIT $3`, invoiceID, after, limit+1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Stage
	for rows.Next() {
		var st Stage
		s := &st.Source
		if err := rows.Scan(&st.StageID, &s.System, &s.Spec, &s.Key, &s.FirstBatch, &s.LastBatch, &s.Presence, &s.FirstSeen, &s.LastSeen,
			&st.StageNoRaw, &st.StageTypeRaw, &st.RequestedRaw, &st.Requested, &st.Unit, &st.RequestedRepr, &st.StateNamespace, &st.StateRaw,
			&st.StateMapping, &st.StateTarget, &st.PayMethodRaw, &st.PaymentSourceRaw, &st.IssuedByKey, &st.IssuedByUserID, &st.ApprovedByKey,
			&st.ApprovedByUserID, &st.ApprovedAtRaw, &st.PaidAtRaw, &st.CreatedAtRaw); err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	return out, rows.Err()
}

// Amounts loads present line totals and stage requests of an invoice as
// domain amounts (raw kept; unrepresentable rows stay unrepresentable).
func Amounts(ctx context.Context, tx db.Tx, invoiceID string) (items, stages []domain.Amount, err error) {
	if err := ReadTx(tx); err != nil {
		return nil, nil, err
	}
	load := func(q string) ([]domain.Amount, error) {
		rows, err := tx.Query(ctx, q, invoiceID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var out []domain.Amount
		for rows.Next() {
			var raw *string
			var amount *int64
			var unit *string
			var repr string
			if err := rows.Scan(&raw, &amount, &unit, &repr); err != nil {
				return nil, err
			}
			a := domain.Amount{Raw: raw, Representation: repr}
			if repr == domain.Representable && amount != nil && unit != nil {
				u, err := money.ParseUnit(*unit)
				if err != nil {
					return nil, err
				}
				a.Money = &money.Money{Amount: *amount, Unit: u}
			}
			out = append(out, a)
		}
		return out, rows.Err()
	}
	if items, err = load(`SELECT line_total_raw, line_total, unit, line_total_repr FROM billing.invoice_item WHERE invoice_id = $1 AND presence = 'PRESENT'`); err != nil {
		return nil, nil, err
	}
	stages, err = load(`SELECT requested_raw, requested_amount, unit, requested_repr FROM billing.payment_stage WHERE invoice_id = $1 AND presence = 'PRESENT'`)
	return items, stages, err
}

// ResolveInvoices maps invoice source keys of one spec to invoices.
func ResolveInvoices(ctx context.Context, tx db.Tx, system, spec string, keys []string) (map[string]api.SourceInvoice, error) {
	if err := ReadTx(tx); err != nil {
		return nil, err
	}
	if keys == nil {
		keys = []string{}
	}
	rows, err := tx.Query(ctx, `SELECT source_key, invoice_id::text, last_batch_id::text, row_sha256, presence = 'PRESENT' FROM billing.invoice
		WHERE source_system = $1 AND source_spec = $2 AND source_key = ANY($3::text[])`, system, spec, keys)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]api.SourceInvoice{}
	for rows.Next() {
		var k string
		var s api.SourceInvoice
		if err := rows.Scan(&k, &s.InvoiceID, &s.BatchID, &s.RowSHA256, &s.Present); err != nil {
			return nil, err
		}
		out[k] = s
	}
	return out, rows.Err()
}

// RowProvenance maps item or stage source keys of one spec to provenance.
func RowProvenance(ctx context.Context, tx db.Tx, system, spec string, keys []string) (map[string]api.RowProvenance, error) {
	if err := ReadTx(tx); err != nil {
		return nil, err
	}
	if keys == nil {
		keys = []string{}
	}
	rows, err := tx.Query(ctx, `SELECT source_key, last_batch_id::text, row_sha256, presence = 'PRESENT' FROM billing.invoice_item
		WHERE source_system = $1 AND source_spec = $2 AND source_key = ANY($3::text[])
		UNION ALL SELECT source_key, last_batch_id::text, row_sha256, presence = 'PRESENT' FROM billing.payment_stage
		WHERE source_system = $1 AND source_spec = $2 AND source_key = ANY($3::text[])`, system, spec, keys)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]api.RowProvenance{}
	for rows.Next() {
		var k string
		var p api.RowProvenance
		if err := rows.Scan(&k, &p.BatchID, &p.RowSHA256, &p.Present); err != nil {
			return nil, err
		}
		out[k] = p
	}
	return out, rows.Err()
}

// Coverage computes the billing coverage report.
func Coverage(ctx context.Context, tx db.Tx) (api.Coverage, error) {
	if err := ReadTx(tx); err != nil {
		return api.Coverage{}, err
	}
	r := api.Coverage{StateMappings: map[string]map[string]int64{}, UnknownRawStates: map[string]map[string]int64{},
		StageStates: map[string]int64{}, UnknownStageRaw: map[string]int64{}, CustomerLinks: map[string]int64{}, CaseLinks: map[string]int64{}}
	for q, dst := range map[string]*int64{
		`SELECT count(*) FROM billing.invoice WHERE presence = 'PRESENT'`:       &r.Invoices,
		`SELECT count(*) FROM billing.invoice_item WHERE presence = 'PRESENT'`:  &r.Items,
		`SELECT count(*) FROM billing.payment_stage WHERE presence = 'PRESENT'`: &r.Stages,
		`SELECT count(*) FROM billing.invoice i WHERE presence = 'PRESENT' AND NOT EXISTS (SELECT 1 FROM billing.invoice_item x WHERE x.invoice_id = i.invoice_id AND x.presence = 'PRESENT')`:                                                                                           &r.InvoicesNoItems,
		`SELECT count(*) FROM billing.invoice i WHERE presence = 'PRESENT' AND NOT EXISTS (SELECT 1 FROM billing.payment_stage x WHERE x.invoice_id = i.invoice_id AND x.presence = 'PRESENT')`:                                                                                          &r.InvoicesNoStages,
		`SELECT (SELECT count(*) FROM billing.amount_evidence WHERE representation = 'UNREPRESENTABLE') + (SELECT count(*) FROM billing.invoice_item WHERE line_total_repr = 'UNREPRESENTABLE') + (SELECT count(*) FROM billing.payment_stage WHERE requested_repr = 'UNREPRESENTABLE')`: &r.Unrepresentable,
		`SELECT count(DISTINCT e.invoice_id) FROM billing.invoice_state_evidence e JOIN billing.invoice i USING (invoice_id) WHERE i.presence = 'PRESENT' AND e.raw_value = 'pre_invoice' AND e.field <> 'LEGACY_DISPLAY'`:                                                               &r.PreInvoiceRawSeen,
	} {
		if err := tx.QueryRow(ctx, q).Scan(dst); err != nil {
			return r, err
		}
	}
	two := func(q string, into map[string]map[string]int64) error {
		rows, err := tx.Query(ctx, q)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var a, b string
			var n int64
			if err := rows.Scan(&a, &b, &n); err != nil {
				return err
			}
			if into[a] == nil {
				into[a] = map[string]int64{}
			}
			into[a][b] = n
		}
		return rows.Err()
	}
	one := func(q string, into map[string]int64) error {
		rows, err := tx.Query(ctx, q)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var a string
			var n int64
			if err := rows.Scan(&a, &n); err != nil {
				return err
			}
			into[a] = n
		}
		return rows.Err()
	}
	pres := ` JOIN billing.invoice i USING (invoice_id) WHERE i.presence = 'PRESENT'`
	if err := two(`SELECT e.field, e.mapping, count(*) FROM billing.invoice_state_evidence e`+pres+` GROUP BY 1, 2`, r.StateMappings); err != nil {
		return r, err
	}
	if err := two(`SELECT e.field, COALESCE(e.raw_value, ''), count(*) FROM billing.invoice_state_evidence e`+pres+` AND e.mapping = 'UNKNOWN' GROUP BY 1, 2`, r.UnknownRawStates); err != nil {
		return r, err
	}
	for q, into := range map[string]map[string]int64{
		`SELECT state_mapping, count(*) FROM billing.payment_stage WHERE presence = 'PRESENT' GROUP BY 1`:                                         r.StageStates,
		`SELECT COALESCE(state_raw, ''), count(*) FROM billing.payment_stage WHERE presence = 'PRESENT' AND state_mapping = 'UNKNOWN' GROUP BY 1`: r.UnknownStageRaw,
		`SELECT customer_link_state, count(*) FROM billing.invoice WHERE presence = 'PRESENT' GROUP BY 1`:                                         r.CustomerLinks,
		`SELECT case_link_state, count(*) FROM billing.invoice WHERE presence = 'PRESENT' GROUP BY 1`:                                             r.CaseLinks,
	} {
		if err := one(q, into); err != nil {
			return r, err
		}
	}
	return r, nil
}
