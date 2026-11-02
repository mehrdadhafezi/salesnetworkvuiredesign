// Package query is the biawin READ side (Slice 10). Every read runs in a
// READ ONLY repeatable-read transaction over the biawin schema and is pinned
// to ONE import run (one snapshot, cutoff and normalization version): list,
// detail, items and history of a request always come from the same run, and
// an id of another run is not found (no cross-run drill-down). Nothing is
// imported, activated, refreshed or recorded by a read; no provider is
// called.
package query

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
)

// FreshnessSLA is the age after which a snapshot is STALE (never "live").
const FreshnessSLA = 24 * time.Hour

// Errors.
var (
	ErrInvalid  = errors.New("biawin: invalid query")
	ErrNoRun    = errors.New("biawin: no import run")
	ErrNotFound = errors.New("biawin: not found in this run")
)

func invalid(f string) error { return fmt.Errorf("%w: %s", ErrInvalid, f) }

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// Service is the read service.
type Service struct {
	runner *db.Runner
	clock  timeutil.Clock
}

// New returns the read service over a biawin-owned runner.
func New(runner *db.Runner, clock timeutil.Clock) (*Service, error) {
	if runner.Module() != biawin.Module {
		return nil, errors.New("biawin: read runner must be owned by biawin")
	}
	return &Service{runner: runner, clock: clock}, nil
}

func (s *Service) read(ctx context.Context, fn func(ctx context.Context, tx db.Tx) error) error {
	return s.runner.InTx(ctx, db.TxOptions{ReadOnly: true, Isolation: db.RepeatableRead}, fn)
}

// Source is one source of a run.
type Source struct {
	Source, Status string
	Rows           int64
}

// Run is one import run.
type Run struct {
	RunID                            string
	Seq                              int64
	NormalizationVersion             int
	SnapshotID                       string
	SourceCutoff, Started, Completed time.Time
	Outcome                          string
	Reasons                          []string
	Counts                           map[string]int64
	Sources                          []Source
	Freshness                        string
}

func (s *Service) resolve(ctx context.Context, tx db.Tx, runID *string) (Run, error) {
	q := `SELECT run_id::text, run_seq, normalization_version, snapshot_id, source_cutoff, started_at, completed_at, outcome, reasons, counts
		FROM biawin.import_run`
	var row pgx.Row
	if runID != nil {
		if !uuidPattern.MatchString(*runID) {
			return Run{}, invalid("run_id")
		}
		row = tx.QueryRow(ctx, q+` WHERE run_id = $1`, *runID)
	} else {
		row = tx.QueryRow(ctx, q+` ORDER BY run_seq DESC LIMIT 1`)
	}
	var r Run
	var reasons, counts []byte
	err := row.Scan(&r.RunID, &r.Seq, &r.NormalizationVersion, &r.SnapshotID, &r.SourceCutoff, &r.Started, &r.Completed, &r.Outcome, &reasons, &counts)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, ErrNoRun
	}
	if err != nil {
		return r, err
	}
	_ = json.Unmarshal(reasons, &r.Reasons)
	_ = json.Unmarshal(counts, &r.Counts)
	r.Freshness = "FRESH"
	if s.clock.Now().Time().Sub(r.SourceCutoff) > FreshnessSLA {
		r.Freshness = "STALE"
	}
	rows, err := tx.Query(ctx, `SELECT source, status, rows_read FROM biawin.run_source WHERE run_id = $1 ORDER BY source`, r.RunID)
	if err != nil {
		return r, err
	}
	defer rows.Close()
	for rows.Next() {
		var src Source
		if err := rows.Scan(&src.Source, &src.Status, &src.Rows); err != nil {
			return r, err
		}
		r.Sources = append(r.Sources, src)
	}
	return r, rows.Err()
}

// Page is a cursor page request.
type Page struct {
	Cursor *string
	Limit  int
}

func (p Page) limit() (int, error) {
	switch {
	case p.Limit == 0:
		return 50, nil
	case p.Limit < 1 || p.Limit > 200:
		return 0, invalid("limit")
	}
	return p.Limit, nil
}

func encodeCursor(v string) *string {
	c := base64.RawURLEncoding.EncodeToString([]byte(v))
	return &c
}

func decodeCursor(c *string) (*string, error) {
	if c == nil {
		return nil, nil
	}
	b, err := base64.RawURLEncoding.DecodeString(*c)
	if err != nil || len(*c) > 512 || len(b) == 0 {
		return nil, invalid("cursor")
	}
	v := string(b)
	return &v, nil
}

// Subscription is one subscription row.
type Subscription struct {
	SubscriptionID, SourceSpec, SourceKey, Provider, ProductVersionState string
	SourceRefRaw, SourceTypeRaw, ProductKey, ProductTitleSnapshot        *string
	CustomerLinkState, CustomerLinkBasis                                 string
	CustomerID, CustomerAccountKey                                       *string
	InvoiceLinkState                                                     string
	InvoiceID, InvoiceKey                                                *string
	FirstPaymentLinkState                                                string
	FirstPaymentInvoiceID, FirstPaymentInvoiceKey                        *string
	CaseLinkState, CaseLinkBasis                                         string
	CaseID, DotCaseKey, OriginUserKey, OriginUserID                      *string
	StateNamespace                                                       string
	StateRaw                                                             *string
	StateMapped, ActivationEvidence                                      string
	ActivatedAtRaw, CreatedAtRaw, UpdatedAtRaw                           *string
	ItemCount                                                            int
}

const subCols = `subscription_id::text, source_spec, source_key, provider, product_version_state, source_ref_raw, source_type_raw, product_key,
	product_title_snapshot, customer_link_state, customer_link_basis, customer_id::text, customer_account_key, invoice_link_state, invoice_id::text,
	invoice_key, first_payment_link_state, first_payment_invoice_id::text, first_payment_invoice_key, case_link_state, case_link_basis, case_id::text,
	dot_case_key, origin_user_key, origin_user_id::text, state_namespace, state_raw, state_mapped, activation_evidence, activated_at_raw,
	created_at_raw, updated_at_raw, item_count`

func scanSub(r interface{ Scan(...any) error }) (Subscription, error) {
	var x Subscription
	err := r.Scan(&x.SubscriptionID, &x.SourceSpec, &x.SourceKey, &x.Provider, &x.ProductVersionState, &x.SourceRefRaw, &x.SourceTypeRaw, &x.ProductKey,
		&x.ProductTitleSnapshot, &x.CustomerLinkState, &x.CustomerLinkBasis, &x.CustomerID, &x.CustomerAccountKey, &x.InvoiceLinkState, &x.InvoiceID,
		&x.InvoiceKey, &x.FirstPaymentLinkState, &x.FirstPaymentInvoiceID, &x.FirstPaymentInvoiceKey, &x.CaseLinkState, &x.CaseLinkBasis, &x.CaseID,
		&x.DotCaseKey, &x.OriginUserKey, &x.OriginUserID, &x.StateNamespace, &x.StateRaw, &x.StateMapped, &x.ActivationEvidence, &x.ActivatedAtRaw,
		&x.CreatedAtRaw, &x.UpdatedAtRaw, &x.ItemCount)
	return x, err
}

// Filter is the allowlisted subscription filter.
type Filter struct {
	RunID                                      *string
	State, CustomerLinkState, InvoiceLinkState *string
	ActivationEvidence, ProductKey             *string
}

var (
	states      = map[string]bool{"ACTIVE": true, "SUSPENDED": true, "UNKNOWN": true, "EMPTY": true}
	links       = map[string]bool{"RESOLVED": true, "RELATED": true, "UNKNOWN": true, "CONFLICT": true, "NOT_RECOVERABLE": true}
	invLinks    = map[string]bool{"RESOLVED": true, "UNKNOWN": true}
	activations = map[string]bool{"LEGACY_ACTIVATION_RECORDED": true, "NOT_EVIDENCED": true}
	keyPattern  = regexp.MustCompile(`^[0-9]{1,20}$`)
)

func check(v *string, ok map[string]bool, f string) error {
	if v != nil && !ok[*v] {
		return invalid(f)
	}
	return nil
}

// Count is one grouped count.
type Count struct {
	Key string
	N   int64
}

// Subscriptions lists subscriptions of one run with the full-filter state
// counts (independent of the page).
func (s *Service) Subscriptions(ctx context.Context, f Filter, p Page) (Run, []Subscription, []Count, *string, error) {
	var run Run
	var out []Subscription
	var sum []Count
	limit, err := p.limit()
	if err != nil {
		return run, nil, nil, nil, err
	}
	for _, c := range []error{check(f.State, states, "state"), check(f.CustomerLinkState, links, "customer_link_state"),
		check(f.InvoiceLinkState, invLinks, "invoice_link_state"), check(f.ActivationEvidence, activations, "activation_evidence")} {
		if c != nil {
			return run, nil, nil, nil, c
		}
	}
	if f.ProductKey != nil && !keyPattern.MatchString(*f.ProductKey) {
		return run, nil, nil, nil, invalid("product")
	}
	after, err := decodeCursor(p.Cursor)
	if err != nil {
		return run, nil, nil, nil, err
	}
	const where = `run_id = $1 AND ($2::text IS NULL OR state_mapped = $2) AND ($3::text IS NULL OR customer_link_state = $3)
		AND ($4::text IS NULL OR invoice_link_state = $4) AND ($5::text IS NULL OR activation_evidence = $5) AND ($6::text IS NULL OR product_key = $6)`
	err = s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		if run, err = s.resolve(ctx, tx, f.RunID); err != nil {
			return err
		}
		args := []any{run.RunID, f.State, f.CustomerLinkState, f.InvoiceLinkState, f.ActivationEvidence, f.ProductKey}
		cr, err := tx.Query(ctx, `SELECT state_mapped, count(*) FROM biawin.subscription WHERE `+where+` GROUP BY 1 ORDER BY 1`, args...)
		if err != nil {
			return err
		}
		for cr.Next() {
			var c Count
			if err := cr.Scan(&c.Key, &c.N); err != nil {
				cr.Close()
				return err
			}
			sum = append(sum, c)
		}
		cr.Close()
		rows, err := tx.Query(ctx, `SELECT `+subCols+` FROM biawin.subscription WHERE `+where+`
			AND ($7::uuid IS NULL OR subscription_id > $7) ORDER BY subscription_id LIMIT $8`, append(args, after, limit+1)...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			x, err := scanSub(rows)
			if err != nil {
				return err
			}
			out = append(out, x)
		}
		return rows.Err()
	})
	if err != nil {
		return run, nil, nil, nil, err
	}
	var next *string
	if len(out) > limit {
		out = out[:limit]
		next = encodeCursor(out[limit-1].SubscriptionID)
	}
	if sum == nil {
		sum = []Count{}
	}
	return run, out, sum, next, nil
}

// Item is one subscription item with its invoice links.
type Item struct {
	ItemID, SubscriptionID, SourceKey, StateMapped string
	ServiceKey, ServiceTitleSnapshot, CreatedAtRaw *string
	InvoiceLinks                                   []InvoiceLink
}

// InvoiceLink is one item invoice link.
type InvoiceLink struct {
	SourceKey, State                           string
	InvoiceID, InvoiceKey, StatusRaw, Verified *string
}

// Event is one history event.
type Event struct {
	SourceKey, EventClass, EventKeyRaw           string
	ItemID, ActorKey, ActorUserID, OccurredAtRaw *string
}

// Detail is one subscription with its history (items are paged apart).
type Detail struct {
	Subscription Subscription
	Events       []Event
}

func (s *Service) subscription(ctx context.Context, tx db.Tx, runID, id string) (Subscription, error) {
	x, err := scanSub(tx.QueryRow(ctx, `SELECT `+subCols+` FROM biawin.subscription WHERE run_id = $1 AND subscription_id = $2`, runID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return x, ErrNotFound
	}
	return x, err
}

// SubscriptionByID returns one subscription of one run with its history.
func (s *Service) SubscriptionByID(ctx context.Context, runID *string, id string) (Run, Detail, error) {
	var run Run
	var d Detail
	if !uuidPattern.MatchString(id) {
		return run, d, invalid("subscription_id")
	}
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		if run, err = s.resolve(ctx, tx, runID); err != nil {
			return err
		}
		if d.Subscription, err = s.subscription(ctx, tx, run.RunID, id); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `SELECT source_key, event_class, event_key_raw, item_id::text, actor_key, actor_user_id::text, occurred_at_raw
			FROM biawin.subscription_event WHERE run_id = $1 AND subscription_id = $2 ORDER BY source_key`, run.RunID, id)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var e Event
			if err := rows.Scan(&e.SourceKey, &e.EventClass, &e.EventKeyRaw, &e.ItemID, &e.ActorKey, &e.ActorUserID, &e.OccurredAtRaw); err != nil {
				return err
			}
			d.Events = append(d.Events, e)
		}
		return rows.Err()
	})
	return run, d, err
}

// Items lists the items of one subscription of one run (each distinct; the
// item count is the subscription's, independent of the page).
func (s *Service) Items(ctx context.Context, runID *string, id string, p Page) (Run, []Item, int, *string, error) {
	var run Run
	var out []Item
	total := 0
	if !uuidPattern.MatchString(id) {
		return run, nil, 0, nil, invalid("subscription_id")
	}
	limit, err := p.limit()
	if err != nil {
		return run, nil, 0, nil, err
	}
	after, err := decodeCursor(p.Cursor)
	if err != nil {
		return run, nil, 0, nil, err
	}
	err = s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		if run, err = s.resolve(ctx, tx, runID); err != nil {
			return err
		}
		sub, err := s.subscription(ctx, tx, run.RunID, id)
		if err != nil {
			return err
		}
		total = sub.ItemCount
		rows, err := tx.Query(ctx, `SELECT item_id::text, subscription_id::text, source_key, state_mapped, service_key, service_title_snapshot, created_at_raw
			FROM biawin.subscription_item WHERE run_id = $1 AND subscription_id = $2 AND ($3::uuid IS NULL OR item_id > $3)
			ORDER BY item_id LIMIT $4`, run.RunID, id, after, limit+1)
		if err != nil {
			return err
		}
		for rows.Next() {
			var it Item
			if err := rows.Scan(&it.ItemID, &it.SubscriptionID, &it.SourceKey, &it.StateMapped, &it.ServiceKey, &it.ServiceTitleSnapshot, &it.CreatedAtRaw); err != nil {
				rows.Close()
				return err
			}
			out = append(out, it)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		for i := range out {
			lr, err := tx.Query(ctx, `SELECT source_key, invoice_link_state, invoice_id::text, invoice_key, status_raw, verified_at_raw
				FROM biawin.item_invoice_link WHERE run_id = $1 AND item_id = $2 ORDER BY source_key`, run.RunID, out[i].ItemID)
			if err != nil {
				return err
			}
			for lr.Next() {
				var l InvoiceLink
				if err := lr.Scan(&l.SourceKey, &l.State, &l.InvoiceID, &l.InvoiceKey, &l.StatusRaw, &l.Verified); err != nil {
					lr.Close()
					return err
				}
				out[i].InvoiceLinks = append(out[i].InvoiceLinks, l)
			}
			lr.Close()
		}
		return nil
	})
	if err != nil {
		return run, nil, 0, nil, err
	}
	var next *string
	if len(out) > limit {
		out = out[:limit]
		next = encodeCursor(out[limit-1].ItemID)
	}
	return run, out, total, next, nil
}

// ParityCheck is one recorded parity check.
type ParityCheck struct {
	CheckID, RunID, EvidenceSnapshotID, EvidenceStatus, Outcome                                      string
	EvidenceCutoff, CheckedAt                                                                        time.Time
	LegacyCount, TargetCount, Matched, MissingTarget, ExtraTarget, Unknown, Conflict, NotRecoverable int64
	Reasons                                                                                          []string
	// CutoffMismatch: the evidence snapshot is not the target run's snapshot.
	CutoffMismatch bool
	Rows           []ParityRow
}

// ParityRow is one compared source key.
type ParityRow struct {
	SourceKey, Result        string
	LegacyClass, TargetState *string
}

var results = map[string]bool{"MATCHED": true, "MISSING_TARGET": true, "EXTRA_TARGET": true, "UNKNOWN": true, "CONFLICT": true, "NOT_RECOVERABLE": true,
	"INACTIVE_BOTH": true}

// Parity returns the latest parity check of one run (or a named check of
// it), with its rows paged; the counts never depend on the page.
func (s *Service) Parity(ctx context.Context, runID *string, checkID *string, result *string, p Page) (Run, *ParityCheck, *string, error) {
	var run Run
	var pc *ParityCheck
	if checkID != nil && !uuidPattern.MatchString(*checkID) {
		return run, nil, nil, invalid("check_id")
	}
	if err := check(result, results, "result"); err != nil {
		return run, nil, nil, err
	}
	limit, err := p.limit()
	if err != nil {
		return run, nil, nil, err
	}
	after, err := decodeCursor(p.Cursor)
	if err != nil {
		return run, nil, nil, err
	}
	var more bool
	err = s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		if run, err = s.resolve(ctx, tx, runID); err != nil {
			return err
		}
		var c ParityCheck
		var reasons []byte
		err = tx.QueryRow(ctx, `SELECT check_id::text, run_id::text, evidence_snapshot_id, evidence_status, outcome, evidence_cutoff, checked_at,
			legacy_count, target_count, matched, missing_target, extra_target, unknown, conflict, not_recoverable, reasons
			FROM biawin.parity_check WHERE run_id = $1 AND ($2::uuid IS NULL OR check_id = $2) ORDER BY check_seq DESC LIMIT 1`, run.RunID, checkID).
			Scan(&c.CheckID, &c.RunID, &c.EvidenceSnapshotID, &c.EvidenceStatus, &c.Outcome, &c.EvidenceCutoff, &c.CheckedAt, &c.LegacyCount,
				&c.TargetCount, &c.Matched, &c.MissingTarget, &c.ExtraTarget, &c.Unknown, &c.Conflict, &c.NotRecoverable, &reasons)
		if errors.Is(err, pgx.ErrNoRows) {
			if checkID != nil {
				return ErrNotFound
			}
			return nil
		}
		if err != nil {
			return err
		}
		_ = json.Unmarshal(reasons, &c.Reasons)
		c.CutoffMismatch = c.EvidenceSnapshotID != run.SnapshotID
		rows, err := tx.Query(ctx, `SELECT source_key, result, legacy_class, target_state FROM biawin.parity_row WHERE check_id = $1
			AND ($2::text IS NULL OR result = $2) AND ($3::text IS NULL OR source_key > $3) ORDER BY source_key LIMIT $4`, c.CheckID, result, after, limit+1)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var r ParityRow
			if err := rows.Scan(&r.SourceKey, &r.Result, &r.LegacyClass, &r.TargetState); err != nil {
				return err
			}
			c.Rows = append(c.Rows, r)
		}
		if more = len(c.Rows) > limit; more {
			c.Rows = c.Rows[:limit]
		}
		pc = &c
		return rows.Err()
	})
	if err != nil {
		return run, nil, nil, err
	}
	var next *string
	if pc != nil && more {
		next = encodeCursor(pc.Rows[limit-1].SourceKey)
	}
	return run, pc, next, nil
}

// RunByID returns one run (the latest when runID is nil) with its manifest.
func (s *Service) RunByID(ctx context.Context, runID *string) (Run, error) {
	var run Run
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		run, err = s.resolve(ctx, tx, runID)
		return err
	})
	return run, err
}
