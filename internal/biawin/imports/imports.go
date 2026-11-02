// Package imports is biawin's OFFLINE shadow import (a job command, never
// reachable from a read path). One staged snapshot under one normalization
// version is one insert-only run: identical inputs replay the existing run
// and write nothing; a new snapshot is a new run; nothing is rewritten. It
// never activates, suspends, orders, charges or calls a provider.
package imports

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin/domain"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
)

// ErrInvalidImport reports a malformed import command.
var ErrInvalidImport = errors.New("biawin: invalid import")

// Run is the outcome of one import.
type Run struct {
	RunID    string
	Replayed bool
	Outcome  result.Status
	Reasons  []reasoncodes.Code
	Counts   map[string]int64
	InputSHA []byte
	FactsSHA []byte
	Cutoff   time.Time
	Parity   *ParityResult
}

// ParityResult is one recorded parity check.
type ParityResult struct {
	CheckID string
	domain.Parity
}

// Command is the import.
type Command struct {
	runner *db.Runner
	clock  timeutil.Clock
}

// New returns the command over a biawin-owned runner.
func New(runner *db.Runner, clock timeutil.Clock) (*Command, error) {
	if runner.Module() != biawin.Module {
		return nil, errors.New("biawin: import runner must be owned by biawin")
	}
	return &Command{runner: runner, clock: clock}, nil
}

func invalid(what string) error { return fmt.Errorf("%w: %s", ErrInvalidImport, what) }

func validate(in api.Import) error {
	if in.BatchID == "" || in.SnapshotID == "" || in.SourceSystem == "" || in.Cutoff.IsZero() {
		return invalid("batch, snapshot, source system and cutoff are required")
	}
	subs := map[string]bool{}
	for _, s := range in.Subscriptions {
		if s.SourceKey == "" || subs[s.SourceKey] {
			return invalid("subscription source keys must be present and unique")
		}
		subs[s.SourceKey] = true
		switch s.StateMapped {
		case api.StateActive, api.StateSuspended, api.StateEmpty:
		case api.StateUnknown:
			if s.StateRaw == nil {
				return invalid("an UNKNOWN state keeps its raw value")
			}
		default:
			return invalid("subscription state " + s.StateMapped)
		}
		if s.Customer.ID != nil && s.Customer.State != api.LinkResolved {
			return invalid("a customer id is only kept for a RESOLVED link")
		}
	}
	items := map[string]bool{}
	for _, it := range in.Items {
		if it.SourceKey == "" || items[it.SourceKey] || !subs[it.SubscriptionKey] {
			return invalid("items need a unique key and a staged subscription")
		}
		items[it.SourceKey] = true
	}
	for _, e := range in.Events {
		if e.SubscriptionKey == nil || !subs[*e.SubscriptionKey] || (e.ItemKey != nil && !items[*e.ItemKey]) {
			return invalid("events need a staged subscription (and item)")
		}
	}
	return nil
}

func sp(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}

type table struct {
	name string
	cols []string
	rows [][]any
}

func (t *table) add(r ...any) { t.rows = append(t.rows, r) }

// Import runs one shadow import and records the active parity of the
// snapshot's independent legacy evidence against the resulting run.
func (c *Command) Import(ctx context.Context, in api.Import, ev api.LegacyEvidence) (Run, error) {
	if err := validate(in); err != nil {
		return Run{}, err
	}
	started := c.clock.Now().Time().UTC()
	run := Run{Counts: map[string]int64{}, Cutoff: in.Cutoff.UTC(), Outcome: result.Success, Reasons: []reasoncodes.Code{}}
	for _, st := range in.Sources {
		if st != api.SourceAvailable {
			run.Outcome = result.Partial
			run.Reasons = append(run.Reasons, reasoncodes.CoverageIncomplete)
		}
	}
	for reason, n := range in.Skipped {
		if n > 0 && reason != "MESSAGING_ONLY" {
			run.Outcome = result.Partial
			run.Reasons = append(run.Reasons, reasoncodes.LinkageConflict)
		}
	}
	run.Reasons, _ = reasoncodes.Normalize(run.Reasons)
	if run.Reasons == nil {
		run.Reasons = []reasoncodes.Code{}
	}
	inputJSON, _ := json.Marshal(struct {
		V int
		I api.Import
	}{domain.NormalizationVersion, normalized(in)})
	h := sha256.Sum256(inputJSON)
	run.InputSHA = h[:]

	var existing *Run
	if err := c.runner.InTx(ctx, db.TxOptions{ReadOnly: true}, func(ctx context.Context, tx db.Tx) error {
		var r Run
		var outcome string
		err := tx.QueryRow(ctx, `SELECT run_id::text, outcome, facts_sha256 FROM biawin.import_run WHERE normalization_version = $1 AND input_sha256 = $2`,
			domain.NormalizationVersion, run.InputSHA).Scan(&r.RunID, &outcome, &r.FactsSHA)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		r.Replayed, r.InputSHA, r.Cutoff, r.Counts, r.Reasons = true, run.InputSHA, run.Cutoff, map[string]int64{}, []reasoncodes.Code{}
		r.Outcome, err = result.ParseStatus(outcome)
		existing = &r
		return err
	}); err != nil {
		return run, err
	}
	if existing != nil {
		return *existing, nil
	}

	var err error
	if run.RunID, err = ids.NewUUID(); err != nil {
		return run, err
	}
	sub := &table{name: "biawin.subscription", cols: []string{"subscription_id", "source_system", "source_spec", "source_key", "source_ref_raw",
		"source_type_raw", "product_key", "product_title_snapshot", "customer_link_state", "customer_link_basis", "customer_id", "customer_account_key",
		"invoice_link_state", "invoice_id", "invoice_key", "first_payment_link_state", "first_payment_invoice_id", "first_payment_invoice_key",
		"case_link_state", "case_link_basis", "case_id", "dot_case_key", "origin_user_key", "origin_user_id", "state_namespace", "state_raw",
		"state_mapped", "activation_evidence", "activated_at_raw", "created_at_raw", "updated_at_raw", "item_count"}}
	item := &table{name: "biawin.subscription_item", cols: []string{"item_id", "subscription_id", "source_key", "service_key", "service_title_snapshot",
		"created_at_raw"}}
	link := &table{name: "biawin.item_invoice_link", cols: []string{"item_id", "source_key", "invoice_link_state", "invoice_id", "invoice_key", "status_raw",
		"verified_at_raw"}}
	evt := &table{name: "biawin.subscription_event", cols: []string{"source_key", "subscription_id", "item_id", "event_class", "event_key_raw",
		"actor_key", "actor_user_id", "occurred_at_raw"}}
	itemCount := map[string]int{}
	for _, it := range in.Items {
		itemCount[it.SubscriptionKey]++
	}
	fh := sha256.New()
	fp := func(parts ...any) {
		b, _ := json.Marshal(parts)
		fh.Write(b)
		fh.Write([]byte{'\n'})
	}
	subs := append([]api.SubscriptionRecord(nil), in.Subscriptions...)
	sort.Slice(subs, func(a, b int) bool { return subs[a].SourceKey < subs[b].SourceKey })
	target := map[string]string{}
	for _, s := range subs {
		id := api.SubscriptionID(in.SourceSystem, s.SourceKey)
		invState, fpState := s.Invoice.State, s.FirstPaymentInvoice.State
		if invState != api.LinkResolved {
			invState = api.LinkUnknown
		}
		if fpState != api.LinkResolved {
			fpState = api.LinkUnknown
		}
		caseState := api.LinkUnknown
		if s.Case.State == api.LinkResolved && s.Case.ID != nil {
			caseState = api.LinkResolved
		}
		row := []any{id, in.SourceSystem, "legacy.project_membership.v1", s.SourceKey, sp(s.SourceRefRaw), sp(s.SourceTypeRaw), sp(s.ProductKey),
			sp(s.ProductTitleSnapshot), s.Customer.State, s.Customer.Basis, sp(s.Customer.ID), sp(s.CustomerAccountKey), invState, sp(s.Invoice.ID),
			sp(s.Invoice.LegacyKey), fpState, sp(s.FirstPaymentInvoice.ID), sp(s.FirstPaymentInvoice.LegacyKey), caseState, s.Case.Basis, sp(s.Case.ID),
			sp(s.Case.LegacyKey), sp(s.OriginUserKey), sp(s.OriginUserID), s.StateNamespace, sp(s.StateRaw), s.StateMapped, s.ActivationEvidence,
			sp(s.ActivatedAtRaw), sp(s.CreatedAtRaw), sp(s.UpdatedAtRaw), itemCount[s.SourceKey]}
		sub.add(row...)
		fp(append([]any{"subscription"}, row...)...)
		target[s.SourceKey] = s.StateMapped
	}
	items := append([]api.ItemRecord(nil), in.Items...)
	sort.Slice(items, func(a, b int) bool { return items[a].SourceKey < items[b].SourceKey })
	for _, it := range items {
		iid := api.ItemID(in.SourceSystem, it.SourceKey)
		row := []any{iid, api.SubscriptionID(in.SourceSystem, it.SubscriptionKey), it.SourceKey, sp(it.ServiceKey), sp(it.ServiceTitleSnapshot), sp(it.CreatedAtRaw)}
		item.add(row...)
		fp(append([]any{"item"}, row...)...)
		for _, l := range it.InvoiceLinks {
			st := l.Invoice.State
			if st != api.LinkResolved {
				st = api.LinkUnknown
			}
			lr := []any{iid, l.SourceKey, st, sp(l.Invoice.ID), sp(l.Invoice.LegacyKey), sp(l.StatusRaw), sp(l.VerifiedAtRaw)}
			link.add(lr...)
			fp(append([]any{"link"}, lr...)...)
		}
	}
	events := append([]api.EventRecord(nil), in.Events...)
	sort.Slice(events, func(a, b int) bool { return events[a].SourceKey < events[b].SourceKey })
	for _, e := range events {
		var iid any
		if e.ItemKey != nil {
			iid = api.ItemID(in.SourceSystem, *e.ItemKey)
		}
		row := []any{e.SourceKey, api.SubscriptionID(in.SourceSystem, *e.SubscriptionKey), iid, e.Class, e.EventKeyRaw, sp(e.ActorKey), sp(e.ActorUserID),
			sp(e.OccurredAtRaw)}
		evt.add(row...)
		fp(append([]any{"event"}, row...)...)
	}
	run.FactsSHA = fh.Sum(nil)
	for _, t := range []*table{sub, item, link, evt} {
		run.Counts[strings.TrimPrefix(t.name, "biawin.")] = int64(len(t.rows))
	}
	for k, n := range in.Skipped {
		run.Counts["skipped."+k] = n
	}
	par := domain.Compare(ev.Classes, target, ev.Status)
	completed := c.clock.Now().Time().UTC()
	if completed.Before(started) {
		completed = started
	}
	reasons, _ := json.Marshal(run.Reasons)
	counts, _ := json.Marshal(run.Counts)
	err = c.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO biawin.import_run (run_id, normalization_version, snapshot_id, batch_id, source_cutoff, started_at,
			completed_at, outcome, reasons, counts, input_sha256, facts_sha256) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			run.RunID, domain.NormalizationVersion, in.SnapshotID, in.BatchID, run.Cutoff, started, completed, run.Outcome.String(), reasons, counts,
			run.InputSHA, run.FactsSHA); err != nil {
			return err
		}
		srcs := make([]string, 0, len(in.Sources))
		for s := range in.Sources {
			srcs = append(srcs, s)
		}
		sort.Strings(srcs)
		for _, s := range srcs {
			if _, err := tx.Exec(ctx, `INSERT INTO biawin.run_source (run_id, source, status, rows_read) VALUES ($1, $2, $3, $4)`,
				run.RunID, s, in.Sources[s], in.SourceRows[s]); err != nil {
				return err
			}
		}
		for _, t := range []*table{sub, item, link, evt} {
			if err := insert(ctx, tx, run.RunID, t); err != nil {
				return err
			}
		}
		pr, err := recordParity(ctx, tx, run.RunID, ev, par, completed)
		run.Parity = pr
		return err
	})
	return run, err
}

// Parity records the parity of a (possibly newer) legacy evidence against
// an EXISTING run — e.g. to show a stale target. Offline only.
func (c *Command) Parity(ctx context.Context, runID string, ev api.LegacyEvidence) (*ParityResult, error) {
	var out *ParityResult
	err := c.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		rows, err := tx.Query(ctx, `SELECT source_key, state_mapped FROM biawin.subscription WHERE run_id = $1`, runID)
		if err != nil {
			return err
		}
		target := map[string]string{}
		for rows.Next() {
			var k, s string
			if err := rows.Scan(&k, &s); err != nil {
				rows.Close()
				return err
			}
			target[k] = s
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		var n int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM biawin.import_run WHERE run_id = $1`, runID).Scan(&n); err != nil || n == 0 {
			return fmt.Errorf("%w: unknown run", ErrInvalidImport)
		}
		out, err = recordParity(ctx, tx, runID, ev, domain.Compare(ev.Classes, target, ev.Status), c.clock.Now().Time().UTC())
		return err
	})
	return out, err
}

func recordParity(ctx context.Context, tx db.Tx, runID string, ev api.LegacyEvidence, p domain.Parity, at time.Time) (*ParityResult, error) {
	id, err := ids.NewUUID()
	if err != nil {
		return nil, err
	}
	reasons, _ := json.Marshal(p.Reasons)
	if _, err := tx.Exec(ctx, `INSERT INTO biawin.parity_check (check_id, run_id, evidence_snapshot_id, evidence_cutoff, evidence_status, checked_at,
		legacy_count, target_count, matched, missing_target, extra_target, unknown, conflict, not_recoverable, outcome, reasons)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`,
		id, runID, ev.SnapshotID, ev.Cutoff.UTC(), ev.Status, at, p.LegacyCount, p.TargetCount, p.Matched, p.MissingTarget, p.ExtraTarget, p.Unknown,
		p.Conflict, p.NotRec, p.Outcome, reasons); err != nil {
		return nil, err
	}
	t := &table{name: "biawin.parity_row", cols: []string{"source_key", "legacy_class", "target_state", "result"}}
	for _, r := range p.Rows {
		t.add(r.SourceKey, sp(r.LegacyClass), sp(r.TargetState), r.Result)
	}
	if err := insertKeyed(ctx, tx, "check_id", id, t); err != nil {
		return nil, err
	}
	return &ParityResult{CheckID: id, Parity: p}, nil
}

// normalized drops nothing but orders the inputs deterministically.
func normalized(in api.Import) api.Import {
	out := in
	out.BatchID = "" // the same snapshot staged twice is the same input
	out.Subscriptions = append([]api.SubscriptionRecord(nil), in.Subscriptions...)
	sort.Slice(out.Subscriptions, func(a, b int) bool { return out.Subscriptions[a].SourceKey < out.Subscriptions[b].SourceKey })
	out.Items = append([]api.ItemRecord(nil), in.Items...)
	sort.Slice(out.Items, func(a, b int) bool { return out.Items[a].SourceKey < out.Items[b].SourceKey })
	out.Events = append([]api.EventRecord(nil), in.Events...)
	sort.Slice(out.Events, func(a, b int) bool { return out.Events[a].SourceKey < out.Events[b].SourceKey })
	return out
}

func insert(ctx context.Context, tx db.Tx, runID string, t *table) error {
	return insertKeyed(ctx, tx, "run_id", runID, t)
}

func insertKeyed(ctx context.Context, tx db.Tx, keyCol, keyVal string, t *table) error {
	const batch = 200
	cols := append([]string{keyCol}, t.cols...)
	for start := 0; start < len(t.rows); start += batch {
		end := min(start+batch, len(t.rows))
		var ph []string
		args := []any{}
		for _, r := range t.rows[start:end] {
			row := append([]any{keyVal}, r...)
			p := make([]string, len(row))
			for i, v := range row {
				args = append(args, v)
				p[i] = fmt.Sprintf("$%d", len(args))
			}
			ph = append(ph, "("+strings.Join(p, ", ")+")")
		}
		if _, err := tx.Exec(ctx, `INSERT INTO `+t.name+` (`+strings.Join(cols, ", ")+`) VALUES `+strings.Join(ph, ", "), args...); err != nil {
			return fmt.Errorf("%s: %w", t.name, err)
		}
	}
	return nil
}
