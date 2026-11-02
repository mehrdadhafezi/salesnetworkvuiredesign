// Package imports is fulfillment's OFFLINE shadow import (a job command,
// never reachable from a read path). One staged snapshot under one
// normalization version is one insert-only run; identical inputs replay the
// existing run; nothing is rewritten. It never assigns, completes, ships,
// upgrades, uploads evidence or calls a provider.
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

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
)

// NormalizationVersion is the fulfillment shadow normalization version.
const NormalizationVersion = 1

// ErrInvalidImport reports a malformed import command.
var ErrInvalidImport = errors.New("fulfillment: invalid import")

var (
	kinds    = map[string]bool{api.KindServiceFollowup: true, api.KindExecution: true, api.KindUpgradeRequest: true, api.KindShipping: true}
	statuses = map[string]bool{api.StatusNotStarted: true, api.StatusInProgress: true, api.StatusAwaitingPayment: true, api.StatusAwaitingProvider: true,
		api.StatusProviderRecorded: true, api.StatusOutcomeRecorded: true, api.StatusShipped: true, api.StatusDelivered: true, api.StatusReturned: true,
		api.StatusNotExecuted: true, api.StatusCancelled: true, api.StatusCompleted: true, api.StatusFailedAtProvider: true, api.StatusUnknown: true,
		api.StatusEmpty: true}
)

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
}

// Command is the import.
type Command struct {
	runner *db.Runner
	clock  timeutil.Clock
}

// New returns the command over a fulfillment-owned runner.
func New(runner *db.Runner, clock timeutil.Clock) (*Command, error) {
	if runner.Module() != fulfillment.Module {
		return nil, errors.New("fulfillment: import runner must be owned by fulfillment")
	}
	return &Command{runner: runner, clock: clock}, nil
}

func invalid(what string) error { return fmt.Errorf("%w: %s", ErrInvalidImport, what) }

func caseKey(spec, key string) string { return spec + "\x00" + key }

func validate(in api.Import) error {
	if in.BatchID == "" || in.SnapshotID == "" || in.SourceSystem == "" || in.Cutoff.IsZero() {
		return invalid("batch, snapshot, source system and cutoff are required")
	}
	cases := map[string]bool{}
	for _, c := range in.Cases {
		k := caseKey(c.SourceSpec, c.SourceKey)
		if c.SourceKey == "" || cases[k] || !kinds[c.Kind] || !statuses[c.StatusMapped] {
			return invalid("cases need a unique source key, a known kind and a known mapped status")
		}
		if c.StatusMapped == api.StatusUnknown && c.StatusRaw == nil {
			return invalid("an UNKNOWN status keeps its raw value")
		}
		if c.ItemID != nil && c.SubscriptionID == nil {
			return invalid("an item reference needs its subscription reference")
		}
		cases[k] = true
	}
	for _, e := range in.Evidence {
		if !cases[caseKey(e.CaseSourceSpec, e.CaseSourceKey)] {
			return invalid("evidence needs a staged case")
		}
	}
	for _, e := range in.Events {
		if !cases[caseKey(e.CaseSourceSpec, e.CaseSourceKey)] {
			return invalid("events need a staged case")
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

func bp(b []byte) any {
	if b == nil {
		return nil
	}
	return b
}

func uref(u *api.UserRef) (any, any) {
	if u == nil {
		return nil, nil
	}
	return u.LegacyKey, sp(u.UserID)
}

type table struct {
	name string
	cols []string
	rows [][]any
}

func (t *table) add(r ...any) { t.rows = append(t.rows, r) }

// Import runs one shadow import.
func (c *Command) Import(ctx context.Context, in api.Import) (Run, error) {
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
	for _, n := range in.Skipped {
		if n > 0 {
			run.Outcome = result.Partial
			run.Reasons = append(run.Reasons, reasoncodes.LinkageConflict)
		}
	}
	run.Reasons, _ = reasoncodes.Normalize(run.Reasons)
	if run.Reasons == nil {
		run.Reasons = []reasoncodes.Code{}
	}
	norm := in
	norm.BatchID = ""
	norm.Cases = append([]api.CaseRecord(nil), in.Cases...)
	sort.Slice(norm.Cases, func(a, b int) bool {
		return caseKey(norm.Cases[a].SourceSpec, norm.Cases[a].SourceKey) < caseKey(norm.Cases[b].SourceSpec, norm.Cases[b].SourceKey)
	})
	norm.Evidence = append([]api.EvidenceRecord(nil), in.Evidence...)
	sort.Slice(norm.Evidence, func(a, b int) bool { return norm.Evidence[a].SourceKey < norm.Evidence[b].SourceKey })
	norm.Events = append([]api.EventRecord(nil), in.Events...)
	sort.Slice(norm.Events, func(a, b int) bool {
		return caseKey(norm.Events[a].SourceSpec, norm.Events[a].SourceKey) < caseKey(norm.Events[b].SourceSpec, norm.Events[b].SourceKey)
	})
	inputJSON, _ := json.Marshal(struct {
		V int
		I api.Import
	}{NormalizationVersion, norm})
	h := sha256.Sum256(inputJSON)
	run.InputSHA = h[:]

	var existing *Run
	if err := c.runner.InTx(ctx, db.TxOptions{ReadOnly: true}, func(ctx context.Context, tx db.Tx) error {
		var r Run
		var outcome string
		err := tx.QueryRow(ctx, `SELECT run_id::text, outcome, facts_sha256 FROM fulfillment.import_run WHERE normalization_version = $1 AND input_sha256 = $2`,
			NormalizationVersion, run.InputSHA).Scan(&r.RunID, &outcome, &r.FactsSHA)
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
	cs := &table{name: "fulfillment.fulfillment_case", cols: []string{"case_id", "kind", "source_system", "source_spec", "source_key", "subtype_raw",
		"subscription_id", "item_id", "invoice_link_state", "invoice_id", "invoice_key", "status_namespace", "status_raw", "status_mapped",
		"shipping_status_raw", "provider_status_raw", "contact_status_raw", "completed_at_raw", "completed_by_key", "completed_by_user_id",
		"cancelled_at_raw", "shipped_at_raw", "delivered_at_raw", "status_updated_at_raw", "carrier_raw", "tracking_ref_sha256", "provider_ref_sha256",
		"provider_ref_scheme", "created_at_raw", "updated_at_raw"}}
	as := &table{name: "fulfillment.assignment", cols: []string{"case_id", "role", "user_key", "user_id", "assigned_at_raw"}}
	evd := &table{name: "fulfillment.service_evidence", cols: []string{"source_key", "case_id", "form_definition_key", "submitted_by_key",
		"submitted_by_user_id", "submitted_at_raw", "outcome_raw", "outcome_mapped", "decided_by_key", "decided_at_raw"}}
	evt := &table{name: "fulfillment.case_event", cols: []string{"source_spec", "source_key", "case_id", "actor_key", "actor_user_id", "event_key_raw",
		"old_status_raw", "new_status_raw", "occurred_at_raw"}}
	fh := sha256.New()
	fp := func(parts ...any) {
		b, _ := json.Marshal(parts)
		fh.Write(b)
		fh.Write([]byte{'\n'})
	}
	for _, c := range norm.Cases {
		id := api.CaseID(in.SourceSystem, c.SourceSpec, c.SourceKey)
		inv := c.Invoice.State
		if inv != "RESOLVED" {
			inv = "UNKNOWN"
		}
		cbk, cbu := uref(c.CompletedBy)
		row := []any{id, c.Kind, in.SourceSystem, c.SourceSpec, c.SourceKey, sp(c.SubtypeRaw), sp(c.SubscriptionID), sp(c.ItemID), inv, sp(c.Invoice.ID),
			sp(c.Invoice.LegacyKey), c.StatusNamespace, sp(c.StatusRaw), c.StatusMapped, sp(c.ShippingStatusRaw), sp(c.ProviderStatusRaw),
			sp(c.ContactStatusRaw), sp(c.CompletedAtRaw), cbk, cbu, sp(c.CancelledAtRaw), sp(c.ShippedAtRaw), sp(c.DeliveredAtRaw),
			sp(c.StatusUpdatedAtRaw), sp(c.CarrierRaw), bp(c.TrackingRefSHA), bp(c.ProviderRefSHA), sp(c.ProviderRefScheme), sp(c.CreatedAtRaw),
			sp(c.UpdatedAtRaw)}
		cs.add(row...)
		fp(append([]any{"case"}, row...)...)
		seen := map[string]bool{}
		for _, a := range c.Assignments {
			if seen[a.Role] {
				continue
			}
			seen[a.Role] = true
			ar := []any{id, a.Role, a.User.LegacyKey, sp(a.User.UserID), sp(a.AssignedAtRaw)}
			as.add(ar...)
			fp(append([]any{"assignment"}, ar...)...)
		}
	}
	for _, e := range norm.Evidence {
		sk, su := uref(e.SubmittedBy)
		dk, _ := uref(e.DecidedBy)
		row := []any{e.SourceKey, api.CaseID(in.SourceSystem, e.CaseSourceSpec, e.CaseSourceKey), sp(e.FormDefinitionKey), sk, su, sp(e.SubmittedAtRaw),
			sp(e.OutcomeRaw), e.OutcomeMapped, dk, sp(e.DecidedAtRaw)}
		evd.add(row...)
		fp(append([]any{"evidence"}, row...)...)
	}
	for _, e := range norm.Events {
		ak, au := uref(e.Actor)
		row := []any{e.SourceSpec, e.SourceKey, api.CaseID(in.SourceSystem, e.CaseSourceSpec, e.CaseSourceKey), ak, au, e.EventKeyRaw, sp(e.OldStatusRaw),
			sp(e.NewStatusRaw), sp(e.OccurredAtRaw)}
		evt.add(row...)
		fp(append([]any{"event"}, row...)...)
	}
	run.FactsSHA = fh.Sum(nil)
	for _, t := range []*table{cs, as, evd, evt} {
		run.Counts[strings.TrimPrefix(t.name, "fulfillment.")] = int64(len(t.rows))
	}
	for k, n := range in.Skipped {
		run.Counts["skipped."+k] = n
	}
	completed := c.clock.Now().Time().UTC()
	if completed.Before(started) {
		completed = started
	}
	reasons, _ := json.Marshal(run.Reasons)
	counts, _ := json.Marshal(run.Counts)
	err = c.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO fulfillment.import_run (run_id, normalization_version, snapshot_id, batch_id, source_cutoff, started_at,
			completed_at, outcome, reasons, counts, input_sha256, facts_sha256) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			run.RunID, NormalizationVersion, in.SnapshotID, in.BatchID, run.Cutoff, started, completed, run.Outcome.String(), reasons, counts,
			run.InputSHA, run.FactsSHA); err != nil {
			return err
		}
		srcs := make([]string, 0, len(in.Sources))
		for s := range in.Sources {
			srcs = append(srcs, s)
		}
		sort.Strings(srcs)
		for _, s := range srcs {
			if _, err := tx.Exec(ctx, `INSERT INTO fulfillment.run_source (run_id, source, status, rows_read) VALUES ($1, $2, $3, $4)`,
				run.RunID, s, in.Sources[s], in.SourceRows[s]); err != nil {
				return err
			}
		}
		for _, t := range []*table{cs, as, evd, evt} {
			if err := insert(ctx, tx, run.RunID, t); err != nil {
				return err
			}
		}
		return nil
	})
	return run, err
}

func insert(ctx context.Context, tx db.Tx, runID string, t *table) error {
	const batch = 200
	cols := append([]string{"run_id"}, t.cols...)
	for start := 0; start < len(t.rows); start += batch {
		end := min(start+batch, len(t.rows))
		var ph []string
		args := []any{}
		for _, r := range t.rows[start:end] {
			row := append([]any{runID}, r...)
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
