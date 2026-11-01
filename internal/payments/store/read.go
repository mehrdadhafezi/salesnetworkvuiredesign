package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/money"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/domain"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
)

// Provenance is the source lineage of one row.
type Provenance struct {
	System, Spec, Key, FirstBatch, LastBatch, Presence string
	FirstSeen, LastSeen                                time.Time
}

// Gateway is the gateway evidence of an attempt (hash presence only).
type Gateway struct {
	ProviderRaw                *string
	HasAuthority, HasReference bool
	StateRaw                   *string
	StateMapping               string
	StateTarget                *string
}

// Attempt is the read view of one payment attempt.
type Attempt struct {
	AttemptID                      string
	InvoiceID                      *string
	InvoiceLink                    string
	Source                         Provenance
	StageNoRaw                     *string
	Channel                        string
	PayMethodRaw, PaymentSourceRaw *string
	AmountRaw                      *string
	Amount                         *int64
	Unit                           *string
	AmountRepr, StateNamespace     string
	StateRaw                       *string
	StateMapping                   string
	StateTarget                    *string
	ApprovalProvenance             string
	UploadedByTypeRaw              *string
	UploadedByKey, UploadedByUser  *string
	CreatedAtRaw, UpdatedAtRaw     *string
	Version                        int64
	Gateway                        *Gateway
}

const attemptSelect = `SELECT a.attempt_id::text, a.invoice_id::text, a.invoice_link_state, a.source_system, a.source_spec, a.source_key,
	a.first_batch_id::text, a.last_batch_id::text, a.presence, a.first_seen_at, a.last_seen_at, a.stage_no_raw, a.channel, a.pay_method_raw,
	a.payment_source_raw, a.amount_raw, a.amount, a.unit, a.amount_repr, a.state_namespace, a.state_raw, a.state_mapping, a.state_target,
	a.approval_provenance, a.uploaded_by_type_raw, a.uploaded_by_legacy_key, a.uploaded_by_user_id::text, a.created_at_raw, a.updated_at_raw,
	a.version, g.attempt_id IS NOT NULL, g.provider_raw, g.authority_sha256 IS NOT NULL, g.ref_sha256 IS NOT NULL, g.state_raw,
	COALESCE(g.state_mapping, ''), g.state_target
	FROM payments.payment_attempt a LEFT JOIN payments.gateway_attempt g ON g.attempt_id = a.attempt_id`

func scanAttempt(r pgx.Row) (Attempt, error) {
	var a Attempt
	var g Gateway
	var hasG bool
	s := &a.Source
	err := r.Scan(&a.AttemptID, &a.InvoiceID, &a.InvoiceLink, &s.System, &s.Spec, &s.Key, &s.FirstBatch, &s.LastBatch, &s.Presence,
		&s.FirstSeen, &s.LastSeen, &a.StageNoRaw, &a.Channel, &a.PayMethodRaw, &a.PaymentSourceRaw, &a.AmountRaw, &a.Amount, &a.Unit,
		&a.AmountRepr, &a.StateNamespace, &a.StateRaw, &a.StateMapping, &a.StateTarget, &a.ApprovalProvenance, &a.UploadedByTypeRaw,
		&a.UploadedByKey, &a.UploadedByUser, &a.CreatedAtRaw, &a.UpdatedAtRaw, &a.Version, &hasG, &g.ProviderRaw, &g.HasAuthority,
		&g.HasReference, &g.StateRaw, &g.StateMapping, &g.StateTarget)
	if hasG {
		a.Gateway = &g
	}
	return a, err
}

// AttemptFilter is the allowlisted attempt filter (no phone, no amount).
type AttemptFilter struct {
	InvoiceID, Channel, StateMapping, StateTarget, Provenance, Presence, InvoiceLink *string
}

// ListAttempts lists attempts by attempt id (keyset).
func ListAttempts(ctx context.Context, tx db.Tx, f AttemptFilter, after *string, limit int) ([]Attempt, error) {
	if err := ReadTx(tx); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, attemptSelect+` WHERE ($1::uuid IS NULL OR a.invoice_id = $1) AND ($2::text IS NULL OR a.channel = $2)
		AND ($3::text IS NULL OR a.state_mapping = $3) AND ($4::text IS NULL OR a.state_target = $4) AND ($5::text IS NULL OR a.approval_provenance = $5)
		AND ($6::text IS NULL OR a.presence = $6) AND ($7::text IS NULL OR a.invoice_link_state = $7) AND ($8::uuid IS NULL OR a.attempt_id > $8)
		ORDER BY a.attempt_id LIMIT $9`, f.InvoiceID, f.Channel, f.StateMapping, f.StateTarget, f.Provenance, f.Presence, f.InvoiceLink, after, limit+1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Attempt
	for rows.Next() {
		a, err := scanAttempt(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// GetAttempt returns one attempt.
func GetAttempt(ctx context.Context, tx db.Tx, id string) (Attempt, error) {
	if err := ReadTx(tx); err != nil {
		return Attempt{}, err
	}
	a, err := scanAttempt(tx.QueryRow(ctx, attemptSelect+` WHERE a.attempt_id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Attempt{}, ErrNotFound
	}
	return a, err
}

// HistoryEntry is one observed raw state change.
type HistoryEntry struct {
	HistoryID           string
	OldStateRaw, NewRaw *string
	Regression          bool
	BatchID             string
	RecordedAt          time.Time
}

// History pages an attempt's state history (keyset on recorded_at, id).
func History(ctx context.Context, tx db.Tx, attemptID string, after *string, limit int) ([]HistoryEntry, error) {
	if err := ReadTx(tx); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT history_id::text, old_state_raw, new_state_raw, regression, batch_id::text, recorded_at
		FROM payments.attempt_state_history WHERE attempt_id = $1 AND ($2::uuid IS NULL OR history_id > $2) ORDER BY history_id LIMIT $3`,
		attemptID, after, limit+1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []HistoryEntry
	for rows.Next() {
		var h HistoryEntry
		if err := rows.Scan(&h.HistoryID, &h.OldStateRaw, &h.NewRaw, &h.Regression, &h.BatchID, &h.RecordedAt); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// Receipt is one receipt reference (metadata only).
type Receipt struct {
	ReceiptID       string
	InvoiceID       *string
	StageNoRaw      *string
	OwnerKind       string
	Source          Provenance
	RefIndex        int
	HasReference    bool
	Host, Extension *string
	State           string
}

const receiptSelect = `SELECT receipt_id::text, invoice_id::text, stage_no_raw, owner_kind, source_system, source_spec, source_key,
	first_batch_id::text, last_batch_id::text, presence, first_seen_at, last_seen_at, ref_index, ref_sha256 IS NOT NULL, host, extension, state
	FROM payments.receipt`

func scanReceipt(r pgx.Row) (Receipt, error) {
	var v Receipt
	s := &v.Source
	err := r.Scan(&v.ReceiptID, &v.InvoiceID, &v.StageNoRaw, &v.OwnerKind, &s.System, &s.Spec, &s.Key, &s.FirstBatch, &s.LastBatch,
		&s.Presence, &s.FirstSeen, &s.LastSeen, &v.RefIndex, &v.HasReference, &v.Host, &v.Extension, &v.State)
	return v, err
}

// ReceiptsOf pages an invoice's receipt references (keyset on receipt id).
func ReceiptsOf(ctx context.Context, tx db.Tx, invoiceID string, after *string, limit int) ([]Receipt, error) {
	if err := ReadTx(tx); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, receiptSelect+` WHERE invoice_id = $1 AND ($2::uuid IS NULL OR receipt_id > $2) ORDER BY receipt_id LIMIT $3`,
		invoiceID, after, limit+1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Receipt
	for rows.Next() {
		v, err := scanReceipt(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// GetReceipt returns one receipt reference.
func GetReceipt(ctx context.Context, tx db.Tx, id string) (Receipt, error) {
	if err := ReadTx(tx); err != nil {
		return Receipt{}, err
	}
	v, err := scanReceipt(tx.QueryRow(ctx, receiptSelect+` WHERE receipt_id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Receipt{}, ErrNotFound
	}
	return v, err
}

// AttemptsSourceComplete reports whether the latest import run carried a
// COMPLETE attempt source. No run at all is not complete.
func AttemptsSourceComplete(ctx context.Context, tx db.Tx) (bool, error) {
	var counts []byte
	err := tx.QueryRow(ctx, `SELECT counts FROM payments.import_run ORDER BY run_seq DESC LIMIT 1`).Scan(&counts)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var m map[string]int64
	if err := json.Unmarshal(counts, &m); err != nil {
		return false, err
	}
	return m["attempts_source_complete"] == 1, nil
}

// AcceptedSums returns per invoice the per-unit sum of PRESENT attempts
// explicitly mapped ACCEPTED. Nothing else is counted; anything that could
// hide a contribution makes the invoice INCOMPLETE.
func AcceptedSums(ctx context.Context, tx db.Tx, invoiceIDs []string) (map[string]api.InvoicePayments, error) {
	if err := ReadTx(tx); err != nil {
		return nil, err
	}
	if invoiceIDs == nil {
		invoiceIDs = []string{}
	}
	sourceComplete, err := AttemptsSourceComplete(ctx, tx)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT invoice_id::text, state_target, amount_raw, amount, unit, amount_repr FROM payments.payment_attempt
		WHERE invoice_id = ANY($1::uuid[]) AND presence = 'PRESENT' AND (state_target = 'ACCEPTED' OR state_mapping <> 'MAPPED')`, invoiceIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	accepted := map[string][]domain.Amount{}
	unclassified := map[string]int64{}
	for rows.Next() {
		var inv string
		var target, raw, unit *string
		var amount *int64
		var repr string
		if err := rows.Scan(&inv, &target, &raw, &amount, &unit, &repr); err != nil {
			return nil, err
		}
		if target == nil {
			unclassified[inv]++
			continue
		}
		a := domain.Amount{Raw: raw, Representation: repr}
		if repr == domain.Representable && amount != nil && unit != nil {
			u, err := money.ParseUnit(*unit)
			if err != nil {
				return nil, err
			}
			a.Money = &money.Money{Amount: *amount, Unit: u}
		}
		accepted[inv] = append(accepted[inv], a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := map[string]api.InvoicePayments{}
	for _, inv := range invoiceIDs {
		complete := sourceComplete && unclassified[inv] == 0
		for _, a := range accepted[inv] {
			if a.Representation != domain.Representable {
				complete = false
			}
		}
		p := api.InvoicePayments{Units: map[string]api.UnitSum{}, Unclassified: unclassified[inv], Complete: complete}
		for u, s := range domain.SumByUnit(accepted[inv], complete) {
			p.Units[u] = api.UnitSum{Unit: u, Amount: s.Amount, State: s.State, Accepted: s.Rows}
		}
		out[inv] = p
	}
	return out, nil
}

// RowProvenance maps attempt source keys of one spec to provenance.
func RowProvenance(ctx context.Context, tx db.Tx, system, spec string, keys []string) (map[string]api.RowProvenance, error) {
	if err := ReadTx(tx); err != nil {
		return nil, err
	}
	if keys == nil {
		keys = []string{}
	}
	rows, err := tx.Query(ctx, `SELECT source_key, last_batch_id::text, row_sha256, presence = 'PRESENT' FROM payments.payment_attempt
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

// BatchImported reports whether a payments import run exists for a batch.
func BatchImported(ctx context.Context, tx db.Tx, batchID string) (bool, error) {
	if err := ReadTx(tx); err != nil {
		return false, err
	}
	return PriorRun(ctx, tx, batchID)
}

// Coverage computes the payments coverage report.
func Coverage(ctx context.Context, tx db.Tx) (api.Coverage, error) {
	if err := ReadTx(tx); err != nil {
		return api.Coverage{}, err
	}
	r := api.Coverage{ByChannel: map[string]int64{}, ByStateMapping: map[string]int64{}, ByTarget: map[string]int64{},
		UnknownRaw: map[string]int64{}, ByProvenance: map[string]int64{}, GatewayStates: map[string]int64{}}
	pres := `FROM payments.payment_attempt WHERE presence = 'PRESENT'`
	for q, dst := range map[string]*int64{
		`SELECT count(*) ` + pres:                                                                                                         &r.Attempts,
		`SELECT count(*) ` + pres + ` AND state_raw = 'approved'`:                                                                         &r.ApprovedRaw,
		`SELECT count(*) ` + pres + ` AND state_raw = 'reconcile_error'`:                                                                  &r.ReconcileErrorRaw,
		`SELECT count(*) ` + pres + ` AND invoice_link_state = 'UNKNOWN'`:                                                                 &r.WithoutInvoice,
		`SELECT count(*) ` + pres + ` AND amount_repr = 'UNREPRESENTABLE'`:                                                                &r.Unrepresentable,
		`SELECT count(*) FROM payments.gateway_attempt g JOIN payments.payment_attempt a USING (attempt_id) WHERE a.presence = 'PRESENT'`: &r.GatewayAttempts,
		`SELECT count(*) FROM payments.attempt_state_history WHERE regression`:                                                            &r.StateRegressions,
		`SELECT count(*) FROM payments.receipt WHERE presence = 'PRESENT' AND state = 'REFERENCE_ONLY'`:                                   &r.ReceiptsReferenced,
		`SELECT count(*) FROM payments.receipt WHERE presence = 'PRESENT' AND state = 'UNKNOWN'`:                                          &r.ReceiptsUnknown,
		`SELECT count(*) FROM payments.receipt WHERE presence = 'PRESENT' AND state = 'MISSING'`:                                          &r.ReceiptsMissing,
	} {
		if err := tx.QueryRow(ctx, q).Scan(dst); err != nil {
			return r, err
		}
	}
	for q, into := range map[string]map[string]int64{
		`SELECT channel, count(*) ` + pres + ` GROUP BY 1`:                                               r.ByChannel,
		`SELECT state_mapping, count(*) ` + pres + ` GROUP BY 1`:                                         r.ByStateMapping,
		`SELECT state_target, count(*) ` + pres + ` AND state_target IS NOT NULL GROUP BY 1`:             r.ByTarget,
		`SELECT COALESCE(state_raw, ''), count(*) ` + pres + ` AND state_mapping = 'UNKNOWN' GROUP BY 1`: r.UnknownRaw,
		`SELECT approval_provenance, count(*) ` + pres + ` GROUP BY 1`:                                   r.ByProvenance,
		`SELECT COALESCE(g.state_target, g.state_mapping), count(*) FROM payments.gateway_attempt g JOIN payments.payment_attempt a USING (attempt_id)
			WHERE a.presence = 'PRESENT' GROUP BY 1`: r.GatewayStates,
	} {
		rows, err := tx.Query(ctx, q)
		if err != nil {
			return r, err
		}
		for rows.Next() {
			var k string
			var n int64
			if err := rows.Scan(&k, &n); err != nil {
				rows.Close()
				return r, err
			}
			into[k] = n
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return r, err
		}
	}
	return r, nil
}
