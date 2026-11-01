// Package store holds the billing module's SQL. It is private to billing
// (architecture lint: foreign-store). Write functions require a writable
// billing-owned transaction (import path only); read functions require a
// read-only one. There is no delete and no write of any payments state.
package store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/domain"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
)

var (
	// ErrWrongOwner reports a transaction not owned by billing.
	ErrWrongOwner = errors.New("billing/store: transaction is not owned by billing")
	// ErrNotReadOnly reports a read path given a writable transaction.
	ErrNotReadOnly = errors.New("billing/store: read path requires a read-only transaction")
	// ErrReadOnly reports a write path given a read-only transaction.
	ErrReadOnly = errors.New("billing/store: write path requires a writable transaction")
	// ErrNotFound reports an unknown invoice.
	ErrNotFound = errors.New("billing/store: not found")
)

func writable(tx db.Tx) error {
	if tx.Module() != billing.Module {
		return ErrWrongOwner
	}
	if tx.ReadOnly() {
		return ErrReadOnly
	}
	return nil
}

// ReadTx checks a read path transaction.
func ReadTx(tx db.Tx) error {
	if tx.Module() != billing.Module {
		return ErrWrongOwner
	}
	if !tx.ReadOnly() {
		return ErrNotReadOnly
	}
	return nil
}

// LockImport serialises billing import commands.
func LockImport(ctx context.Context, tx db.Tx) error {
	if err := writable(tx); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('crm.billing.import'))`)
	return err
}

// Change classifies what an upsert did.
type Change string

const (
	Inserted   Change = "inserted"
	Refreshed  Change = "refreshed"
	Reappeared Change = "reappeared"
	Rebatched  Change = "rebatched"
	Unchanged  Change = "unchanged"
)

// table describes one provenance-carrying billing table (whitelisted).
type table struct{ name, id string }

var (
	invoiceTable = table{"billing.invoice", "invoice_id"}
	itemTable    = table{"billing.invoice_item", "item_id"}
	stageTable   = table{"billing.payment_stage", "stage_id"}
)

// upsert mints (once) or refreshes one row keyed by its source triple. A
// changed checksum refreshes the mutable projection (never forks the row);
// an unchanged one only records batch and presence.
func upsert(ctx context.Context, tx db.Tx, t table, src api.SourceRef, cols map[string]any, now time.Time) (string, Change, error) {
	names := make([]string, 0, len(cols))
	for k := range cols {
		names = append(names, k)
	}
	sort.Strings(names)
	var id, batch, presence string
	var sha []byte
	err := tx.QueryRow(ctx, `SELECT `+t.id+`::text, last_batch_id::text, row_sha256, presence FROM `+t.name+`
		WHERE source_system = $1 AND source_spec = $2 AND source_key = $3 FOR UPDATE`, src.System, src.Spec, src.Key).Scan(&id, &batch, &sha, &presence)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		if id, err = ids.NewUUID(); err != nil {
			return "", "", err
		}
		all := append([]string{t.id, "source_system", "source_spec", "source_key", "first_batch_id", "last_batch_id", "row_sha256",
			"first_seen_at", "last_seen_at", "presence"}, names...)
		args := []any{id, src.System, src.Spec, src.Key, src.BatchID, src.BatchID, src.RowSHA256, now, now, "PRESENT"}
		for _, n := range names {
			args = append(args, cols[n])
		}
		ph := make([]string, len(all))
		for i := range all {
			ph[i] = fmt.Sprintf("$%d", i+1)
		}
		_, err = tx.Exec(ctx, `INSERT INTO `+t.name+` (`+strings.Join(all, ", ")+`) VALUES (`+strings.Join(ph, ", ")+`)`, args...)
		return id, Inserted, err
	case err != nil:
		return "", "", err
	}
	if !bytes.Equal(sha, src.RowSHA256) {
		sets := []string{"last_batch_id = $2", "row_sha256 = $3", "last_seen_at = $4", "presence = 'PRESENT'"}
		args := []any{id, src.BatchID, src.RowSHA256, now}
		for _, n := range names {
			args = append(args, cols[n])
			sets = append(sets, fmt.Sprintf("%s = $%d", n, len(args)))
		}
		_, err = tx.Exec(ctx, `UPDATE `+t.name+` SET `+strings.Join(sets, ", ")+` WHERE `+t.id+` = $1`, args...)
		return id, Refreshed, err
	}
	if _, err := tx.Exec(ctx, `UPDATE `+t.name+` SET last_batch_id = $2, last_seen_at = $3, presence = 'PRESENT' WHERE `+t.id+` = $1
		AND (last_batch_id, presence) IS DISTINCT FROM ($2::uuid, 'PRESENT')`, id, src.BatchID, now); err != nil {
		return "", "", err
	}
	switch {
	case presence != "PRESENT":
		return id, Reappeared, nil
	case batch != src.BatchID:
		return id, Rebatched, nil
	}
	return id, Unchanged, nil
}

func str(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}

func userCols(prefix string, u *api.UserRef, cols map[string]any) {
	cols[prefix+"_legacy_key"], cols[prefix+"_user_id"] = nil, nil
	if u != nil {
		cols[prefix+"_legacy_key"], cols[prefix+"_user_id"] = u.LegacyKey, str(u.UserID)
	}
}

// moneyCols renders one raw amount into (raw, amount, unit, representation).
func moneyCols(m api.RawMoney) (raw, amount, unit any, repr string) {
	a := domain.ParseAmount(m.Raw, m.Unit)
	raw = str(m.Raw)
	if a.Representation == domain.Representable {
		return raw, a.Money.Amount, a.Money.Unit.String(), a.Representation
	}
	return raw, nil, nil, a.Representation
}

// UpsertInvoice writes one invoice, its separate state evidence rows and
// its declared amount evidence. Version increments on a refresh.
func UpsertInvoice(ctx context.Context, tx db.Tx, v api.InvoiceRecord, now time.Time) (string, Change, error) {
	if err := writable(tx); err != nil {
		return "", "", err
	}
	cols := map[string]any{"invoice_code_raw": str(v.CodeRaw), "product_ref_raw": str(v.ProductRefRaw), "pay_method_raw": str(v.PayMethodRaw),
		"payment_plan_raw": str(v.PaymentPlanRaw), "current_stage_raw": str(v.CurrentStageRaw), "issued_at_raw": str(v.IssuedAtRaw),
		"updated_at_raw": str(v.UpdatedAtRaw), "lifecycle_state": "UNKNOWN",
		"customer_id": str(v.Customer.ID), "customer_link_state": v.Customer.State, "customer_link_basis": v.Customer.Basis,
		"case_id": str(v.Case.ID), "case_link_state": v.Case.State, "case_link_basis": v.Case.Basis}
	userCols("seller", v.Seller, cols)
	userCols("issued_by", v.IssuedBy, cols)
	id, ch, err := upsert(ctx, tx, invoiceTable, v.Source, cols, now)
	if err != nil {
		return "", "", err
	}
	if ch == Refreshed {
		if _, err := tx.Exec(ctx, `UPDATE billing.invoice SET version = version + 1 WHERE invoice_id = $1`, id); err != nil {
			return "", "", err
		}
	}
	if ch == Inserted || ch == Refreshed {
		for _, s := range v.States {
			if _, err := tx.Exec(ctx, `INSERT INTO billing.invoice_state_evidence (invoice_id, field, namespace, raw_value, mapping)
				VALUES ($1, $2, $3, $4, $5) ON CONFLICT (invoice_id, field) DO UPDATE SET namespace = EXCLUDED.namespace,
				raw_value = EXCLUDED.raw_value, mapping = EXCLUDED.mapping`, id, string(s.Field), s.Namespace, str(s.Raw), string(s.Mapping)); err != nil {
				return "", "", err
			}
		}
		kinds := make([]string, 0, len(v.Declared))
		for k := range v.Declared {
			kinds = append(kinds, string(k))
		}
		sort.Strings(kinds)
		for _, k := range kinds {
			raw, amount, unit, repr := moneyCols(v.Declared[api.AmountKind(k)])
			if _, err := tx.Exec(ctx, `INSERT INTO billing.amount_evidence (invoice_id, kind, raw_value, amount, unit, representation)
				VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (invoice_id, kind) DO UPDATE SET raw_value = EXCLUDED.raw_value,
				amount = EXCLUDED.amount, unit = EXCLUDED.unit, representation = EXCLUDED.representation`, id, k, raw, amount, unit, repr); err != nil {
				return "", "", err
			}
		}
	}
	return id, ch, nil
}

// UpsertItem writes one invoice line (issuance snapshot).
func UpsertItem(ctx context.Context, tx db.Tx, invoiceID string, it api.ItemRecord, now time.Time) (Change, error) {
	if err := writable(tx); err != nil {
		return "", err
	}
	praw, pamt, punit, _ := moneyCols(it.UnitPrice)
	lraw, lamt, lunit, lrepr := moneyCols(it.LineTotal)
	// The line total is the per-line amount; a unit price is kept as an
	// amount only when it is representable in the SAME unit (raw always kept).
	if lrepr != domain.Representable || punit != lunit {
		pamt = nil
	}
	cols := map[string]any{"invoice_id": invoiceID, "product_ref_raw": str(it.ProductRefRaw), "title_snapshot": str(it.TitleSnapshot),
		"kind_raw": str(it.KindRaw), "qty_raw": str(it.QtyRaw), "unit_price_raw": praw, "unit_price": pamt, "line_total_raw": lraw,
		"line_total": lamt, "unit": lunit, "line_total_repr": lrepr, "is_free_raw": str(it.IsFreeRaw), "created_at_raw": str(it.CreatedAtRaw)}
	_, ch, err := upsert(ctx, tx, itemTable, it.Source, cols, now)
	return ch, err
}

// UpsertStage writes one payment stage with its own state.
func UpsertStage(ctx context.Context, tx db.Tx, invoiceID string, st api.StageRecord, now time.Time) (Change, error) {
	if err := writable(tx); err != nil {
		return "", err
	}
	raw, amount, unit, repr := moneyCols(st.Requested)
	mapping := "UNKNOWN"
	switch {
	case st.StateRaw == nil || *st.StateRaw == "":
		mapping = "EMPTY"
	case st.StateTarget != nil:
		mapping = "MAPPED"
	}
	var target any
	if mapping == "MAPPED" {
		target = *st.StateTarget
	}
	cols := map[string]any{"invoice_id": invoiceID, "stage_no_raw": str(st.StageNoRaw), "stage_type_raw": str(st.StageTypeRaw),
		"requested_raw": raw, "requested_amount": amount, "unit": unit, "requested_repr": repr, "state_namespace": st.StateNamespace,
		"state_raw": str(st.StateRaw), "state_mapping": mapping, "state_target": target, "pay_method_raw": str(st.PayMethodRaw),
		"payment_source_raw": str(st.PaymentSourceRaw), "approved_at_raw": str(st.ApprovedAtRaw), "paid_at_raw": str(st.PaidAtRaw),
		"created_at_raw": str(st.CreatedAtRaw)}
	userCols("issued_by", st.IssuedBy, cols)
	userCols("approved_by", st.ApprovedBy, cols)
	_, ch, err := upsert(ctx, tx, stageTable, st.Source, cols, now)
	return ch, err
}

// MarkAbsent flags rows of a COMPLETE spec whose source row is gone. Nothing
// is deleted.
func MarkAbsent(ctx context.Context, tx db.Tx, which string, system, spec string, present []string, now time.Time) (int64, error) {
	if err := writable(tx); err != nil {
		return 0, err
	}
	t, ok := map[string]table{"invoice": invoiceTable, "item": itemTable, "stage": stageTable}[which]
	if !ok {
		return 0, fmt.Errorf("billing/store: unknown table %q", which)
	}
	if present == nil {
		present = []string{}
	}
	tag, err := tx.Exec(ctx, `UPDATE `+t.name+` SET presence = 'ABSENT_IN_LATEST', last_seen_at = $4
		WHERE source_system = $1 AND source_spec = $2 AND presence = 'PRESENT' AND NOT (source_key = ANY($3::text[]))`, system, spec, present, now)
	return tag.RowsAffected(), err
}

// InvoiceOf returns the invoice id of one source row (any presence).
func InvoiceOf(ctx context.Context, tx db.Tx, system string, r api.RowRef) (string, bool, error) {
	var id string
	err := tx.QueryRow(ctx, `SELECT invoice_id::text FROM billing.invoice WHERE source_system = $1 AND source_spec = $2 AND source_key = $3`,
		system, r.Spec, r.Key).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	return id, err == nil, err
}

// PriorRun reports whether a batch was imported before.
func PriorRun(ctx context.Context, tx db.Tx, batchID string) (bool, error) {
	var n int64
	err := tx.QueryRow(ctx, `SELECT count(*) FROM billing.import_run WHERE batch_id = $1`, batchID).Scan(&n)
	return n > 0, err
}

// InsertRun records one import run.
func InsertRun(ctx context.Context, tx db.Tx, runID, batchID, snapshotID, outcome string, replay bool, counts map[string]int64, now time.Time) error {
	if err := writable(tx); err != nil {
		return err
	}
	b, err := json.Marshal(counts)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO billing.import_run (run_id, batch_id, snapshot_id, outcome, replay, counts, recorded_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`, runID, batchID, snapshotID, outcome, replay, b, now)
	return err
}
