// Package query is the fulfillment READ side (Slice 10). Every read runs in a
// READ ONLY repeatable-read transaction over the fulfillment schema, pinned
// to ONE import run: list, detail and history of a request always come from
// the same run, and an id of another run is not found. Nothing is imported,
// assigned, completed or recorded by a read.
package query

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
)

// FreshnessSLA is the age after which a snapshot is STALE (never "live").
const FreshnessSLA = 24 * time.Hour

// Errors.
var (
	ErrInvalid  = errors.New("fulfillment: invalid query")
	ErrNoRun    = errors.New("fulfillment: no import run")
	ErrNotFound = errors.New("fulfillment: not found in this run")
)

func invalid(f string) error { return fmt.Errorf("%w: %s", ErrInvalid, f) }

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// Service is the read service.
type Service struct {
	runner *db.Runner
	clock  timeutil.Clock
}

// New returns the read service over a fulfillment-owned runner.
func New(runner *db.Runner, clock timeutil.Clock) (*Service, error) {
	if runner.Module() != fulfillment.Module {
		return nil, errors.New("fulfillment: read runner must be owned by fulfillment")
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
		FROM fulfillment.import_run`
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
	rows, err := tx.Query(ctx, `SELECT source, status, rows_read FROM fulfillment.run_source WHERE run_id = $1 ORDER BY source`, r.RunID)
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

// Assignment is one current assignee.
type Assignment struct {
	Role, UserKey         string
	UserID, AssignedAtRaw *string
}

// Case is one fulfillment case.
type Case struct {
	CaseID, Kind, SourceSpec, SourceKey, InvoiceLinkState, StatusNamespace, StatusMapped string
	SubtypeRaw, SubscriptionID, ItemID, InvoiceID, InvoiceKey, StatusRaw                 *string
	ShippingStatusRaw, ProviderStatusRaw, ContactStatusRaw                               *string
	CompletedAtRaw, CompletedByKey, CompletedByUserID, CancelledAtRaw                    *string
	ShippedAtRaw, DeliveredAtRaw, StatusUpdatedAtRaw, CarrierRaw                         *string
	TrackingRef, ProviderRef, ProviderRefScheme                                          *string
	CreatedAtRaw, UpdatedAtRaw                                                           *string
	Assignments                                                                          []Assignment
}

const caseCols = `case_id::text, kind, source_spec, source_key, invoice_link_state, status_namespace, status_mapped, subtype_raw,
	subscription_id::text, item_id::text, invoice_id::text, invoice_key, status_raw, shipping_status_raw, provider_status_raw, contact_status_raw,
	completed_at_raw, completed_by_key, completed_by_user_id::text, cancelled_at_raw, shipped_at_raw, delivered_at_raw, status_updated_at_raw,
	carrier_raw, tracking_ref_sha256, provider_ref_sha256, provider_ref_scheme, created_at_raw, updated_at_raw`

func hexp(b []byte) *string {
	if b == nil {
		return nil
	}
	s := hex.EncodeToString(b)
	return &s
}

func scanCase(r interface{ Scan(...any) error }) (Case, error) {
	var c Case
	var tr, pr []byte
	err := r.Scan(&c.CaseID, &c.Kind, &c.SourceSpec, &c.SourceKey, &c.InvoiceLinkState, &c.StatusNamespace, &c.StatusMapped, &c.SubtypeRaw,
		&c.SubscriptionID, &c.ItemID, &c.InvoiceID, &c.InvoiceKey, &c.StatusRaw, &c.ShippingStatusRaw, &c.ProviderStatusRaw, &c.ContactStatusRaw,
		&c.CompletedAtRaw, &c.CompletedByKey, &c.CompletedByUserID, &c.CancelledAtRaw, &c.ShippedAtRaw, &c.DeliveredAtRaw, &c.StatusUpdatedAtRaw,
		&c.CarrierRaw, &tr, &pr, &c.ProviderRefScheme, &c.CreatedAtRaw, &c.UpdatedAtRaw)
	c.TrackingRef, c.ProviderRef = hexp(tr), hexp(pr)
	return c, err
}

func (s *Service) assignments(ctx context.Context, tx db.Tx, runID string, cs []Case) error {
	for i := range cs {
		rows, err := tx.Query(ctx, `SELECT role, user_key, user_id::text, assigned_at_raw FROM fulfillment.assignment WHERE run_id = $1 AND case_id = $2
			ORDER BY role`, runID, cs[i].CaseID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var a Assignment
			if err := rows.Scan(&a.Role, &a.UserKey, &a.UserID, &a.AssignedAtRaw); err != nil {
				rows.Close()
				return err
			}
			cs[i].Assignments = append(cs[i].Assignments, a)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
	}
	return nil
}

// Filter is the allowlisted case filter.
type Filter struct {
	RunID                                *string
	Kind, Status, SubscriptionID, ItemID *string
	AssigneeUserID                       *string
}

var (
	kinds    = map[string]bool{"SERVICE_FOLLOWUP": true, "EXECUTION": true, "UPGRADE_EXECUTION": true, "SHIPPING": true}
	statuses = map[string]bool{"NOT_STARTED": true, "IN_PROGRESS": true, "AWAITING_PAYMENT": true, "AWAITING_PROVIDER": true,
		"PROVIDER_ACTION_RECORDED": true, "OUTCOME_RECORDED": true, "SHIPPED": true, "DELIVERED": true, "RETURNED": true, "NOT_EXECUTED": true,
		"CANCELLED": true, "COMPLETED": true, "FAILED_AT_PROVIDER": true, "UNKNOWN": true, "EMPTY": true}
)

// Count is one grouped count.
type Count struct {
	Kind, Status string
	N            int64
}

// Cases lists the cases of one run with the full-filter counts per kind and
// mapped status (independent of the page).
func (s *Service) Cases(ctx context.Context, f Filter, p Page) (Run, []Case, []Count, *string, error) {
	var run Run
	var out []Case
	var sum []Count
	limit, err := p.limit()
	if err != nil {
		return run, nil, nil, nil, err
	}
	if f.Kind != nil && !kinds[*f.Kind] {
		return run, nil, nil, nil, invalid("kind")
	}
	if f.Status != nil && !statuses[*f.Status] {
		return run, nil, nil, nil, invalid("status")
	}
	for name, v := range map[string]*string{"subscription_id": f.SubscriptionID, "item_id": f.ItemID, "assignee": f.AssigneeUserID} {
		if v != nil && !uuidPattern.MatchString(*v) {
			return run, nil, nil, nil, invalid(name)
		}
	}
	after, err := decodeCursor(p.Cursor)
	if err != nil {
		return run, nil, nil, nil, err
	}
	const where = `c.run_id = $1 AND ($2::text IS NULL OR c.kind = $2) AND ($3::text IS NULL OR c.status_mapped = $3)
		AND ($4::uuid IS NULL OR c.subscription_id = $4) AND ($5::uuid IS NULL OR c.item_id = $5)
		AND ($6::uuid IS NULL OR EXISTS (SELECT 1 FROM fulfillment.assignment a WHERE a.run_id = c.run_id AND a.case_id = c.case_id AND a.user_id = $6))`
	err = s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		if run, err = s.resolve(ctx, tx, f.RunID); err != nil {
			return err
		}
		args := []any{run.RunID, f.Kind, f.Status, f.SubscriptionID, f.ItemID, f.AssigneeUserID}
		cr, err := tx.Query(ctx, `SELECT c.kind, c.status_mapped, count(*) FROM fulfillment.fulfillment_case c WHERE `+where+` GROUP BY 1, 2 ORDER BY 1, 2`, args...)
		if err != nil {
			return err
		}
		for cr.Next() {
			var c Count
			if err := cr.Scan(&c.Kind, &c.Status, &c.N); err != nil {
				cr.Close()
				return err
			}
			sum = append(sum, c)
		}
		cr.Close()
		rows, err := tx.Query(ctx, `SELECT `+caseCols+` FROM fulfillment.fulfillment_case c WHERE `+where+`
			AND ($7::uuid IS NULL OR c.case_id > $7) ORDER BY c.case_id LIMIT $8`, append(args, after, limit+1)...)
		if err != nil {
			return err
		}
		for rows.Next() {
			c, err := scanCase(rows)
			if err != nil {
				rows.Close()
				return err
			}
			out = append(out, c)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		if len(out) > limit {
			return s.assignments(ctx, tx, run.RunID, out[:limit])
		}
		return s.assignments(ctx, tx, run.RunID, out)
	})
	if err != nil {
		return run, nil, nil, nil, err
	}
	var next *string
	if len(out) > limit {
		out = out[:limit]
		next = encodeCursor(out[limit-1].CaseID)
	}
	if sum == nil {
		sum = []Count{}
	}
	return run, out, sum, next, nil
}

// Evidence is one service evidence metadata record.
type Evidence struct {
	SourceKey, OutcomeMapped                                             string
	FormDefinitionKey, SubmittedByKey, SubmittedByUserID, SubmittedAtRaw *string
	OutcomeRaw, DecidedByKey, DecidedAtRaw                               *string
}

// Detail is one case with its assignments and evidence.
type Detail struct {
	Case     Case
	Evidence []Evidence
}

func (s *Service) caseByID(ctx context.Context, tx db.Tx, runID, id string) (Case, error) {
	c, err := scanCase(tx.QueryRow(ctx, `SELECT `+caseCols+` FROM fulfillment.fulfillment_case c WHERE c.run_id = $1 AND c.case_id = $2`, runID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return c, ErrNotFound
	}
	return c, err
}

// CaseByID returns one case of one run.
func (s *Service) CaseByID(ctx context.Context, runID *string, id string) (Run, Detail, error) {
	var run Run
	var d Detail
	if !uuidPattern.MatchString(id) {
		return run, d, invalid("case_id")
	}
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		if run, err = s.resolve(ctx, tx, runID); err != nil {
			return err
		}
		if d.Case, err = s.caseByID(ctx, tx, run.RunID, id); err != nil {
			return err
		}
		cs := []Case{d.Case}
		if err := s.assignments(ctx, tx, run.RunID, cs); err != nil {
			return err
		}
		d.Case = cs[0]
		rows, err := tx.Query(ctx, `SELECT source_key, outcome_mapped, form_definition_key, submitted_by_key, submitted_by_user_id::text,
			submitted_at_raw, outcome_raw, decided_by_key, decided_at_raw FROM fulfillment.service_evidence WHERE run_id = $1 AND case_id = $2
			ORDER BY source_key`, run.RunID, id)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var e Evidence
			if err := rows.Scan(&e.SourceKey, &e.OutcomeMapped, &e.FormDefinitionKey, &e.SubmittedByKey, &e.SubmittedByUserID, &e.SubmittedAtRaw,
				&e.OutcomeRaw, &e.DecidedByKey, &e.DecidedAtRaw); err != nil {
				return err
			}
			d.Evidence = append(d.Evidence, e)
		}
		return rows.Err()
	})
	return run, d, err
}

// Event is one append-only history event.
type Event struct {
	SourceSpec, SourceKey, EventKeyRaw                               string
	ActorKey, ActorUserID, OldStatusRaw, NewStatusRaw, OccurredAtRaw *string
}

// History lists one case's events of one run (oldest first, by source key).
func (s *Service) History(ctx context.Context, runID *string, id string, p Page) (Run, []Event, *string, error) {
	var run Run
	var out []Event
	if !uuidPattern.MatchString(id) {
		return run, nil, nil, invalid("case_id")
	}
	limit, err := p.limit()
	if err != nil {
		return run, nil, nil, err
	}
	after, err := decodeCursor(p.Cursor)
	if err != nil {
		return run, nil, nil, err
	}
	err = s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		if run, err = s.resolve(ctx, tx, runID); err != nil {
			return err
		}
		if _, err := s.caseByID(ctx, tx, run.RunID, id); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `SELECT source_spec, source_key, event_key_raw, actor_key, actor_user_id::text, old_status_raw, new_status_raw,
			occurred_at_raw FROM fulfillment.case_event WHERE run_id = $1 AND case_id = $2 AND ($3::text IS NULL OR source_spec || ':' || source_key > $3)
			ORDER BY source_spec || ':' || source_key LIMIT $4`, run.RunID, id, after, limit+1)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var e Event
			if err := rows.Scan(&e.SourceSpec, &e.SourceKey, &e.EventKeyRaw, &e.ActorKey, &e.ActorUserID, &e.OldStatusRaw, &e.NewStatusRaw, &e.OccurredAtRaw); err != nil {
				return err
			}
			out = append(out, e)
		}
		return rows.Err()
	})
	if err != nil {
		return run, nil, nil, err
	}
	var next *string
	if len(out) > limit {
		out = out[:limit]
		last := out[limit-1]
		next = encodeCursor(last.SourceSpec + ":" + last.SourceKey)
	}
	return run, out, next, nil
}
