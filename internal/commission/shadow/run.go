package shadow

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

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/commission"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/commission/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/commission/domain"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
)

// ErrInvalidRun reports a malformed shadow run request.
var ErrInvalidRun = errors.New("commission: invalid shadow run")

// Report is the outcome of one shadow run.
type Report struct {
	RunID    string
	Replayed bool
	Outcome  result.Status
	Reasons  []reasoncodes.Code
	Counts   map[string]int64
	InputSHA []byte
	FactsSHA []byte
	Cutoff   time.Time
	Rounding string
	Parity   api.ParitySummary
}

// Command is the offline shadow run.
type Command struct {
	runner *db.Runner
	clock  timeutil.Clock
}

// New returns the command over a commission-owned runner.
func New(runner *db.Runner, clock timeutil.Clock) (*Command, error) {
	if runner.Module() != commission.Module {
		return nil, errors.New("commission: shadow runner must be owned by commission")
	}
	return &Command{runner: runner, clock: clock}, nil
}

type table struct {
	name string
	cols []string
	rows [][]any
}

func (t *table) add(r ...any) { t.rows = append(t.rows, r) }

func sp(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}

func ex(e *domain.Exact) any {
	if e == nil {
		return nil
	}
	return e.Hundredths
}

func rt(r *domain.Rate) any {
	if r == nil {
		return nil
	}
	return r.E4
}

func normalized(ev api.Evidence) api.Evidence {
	out := ev
	out.BatchID = "" // the same snapshot staged twice is the same input
	out.Credits = append([]api.LegacyCredit(nil), ev.Credits...)
	sort.Slice(out.Credits, func(a, b int) bool { return out.Credits[a].TxKey < out.Credits[b].TxKey })
	out.Matrix = append([]api.MatrixRate(nil), ev.Matrix...)
	sort.Slice(out.Matrix, func(a, b int) bool { return out.Matrix[a].SourceKey < out.Matrix[b].SourceKey })
	out.Rules = append([]api.Rule(nil), ev.Rules...)
	sort.Slice(out.Rules, func(a, b int) bool { return out.Rules[a].SourceKey < out.Rules[b].SourceKey })
	out.DryRuns = append([]api.DryRun(nil), ev.DryRuns...)
	sort.Slice(out.DryRuns, func(a, b int) bool { return out.DryRuns[a].SourceKey < out.DryRuns[b].SourceKey })
	out.DryRunItems = append([]api.DryRunItem(nil), ev.DryRunItems...)
	sort.Slice(out.DryRunItems, func(a, b int) bool { return out.DryRunItems[a].SourceKey < out.DryRunItems[b].SourceKey })
	out.PostingLogs = append([]api.PostingLog(nil), ev.PostingLogs...)
	sort.Slice(out.PostingLogs, func(a, b int) bool { return out.PostingLogs[a].SourceKey < out.PostingLogs[b].SourceKey })
	return out
}

// Run executes one shadow run. A missing rounding is accepted and recorded
// as NONE: every calculation is then BLOCKED (OPD-06), never defaulted.
func (c *Command) Run(ctx context.Context, src api.EvidenceSource, rounding domain.Rounding) (Report, error) {
	if src == nil {
		return Report{}, fmt.Errorf("%w: an evidence source is required", ErrInvalidRun)
	}
	ev, err := src.CommissionEvidence(ctx)
	if err != nil {
		return Report{}, err
	}
	if ev.SnapshotID == "" || ev.Cutoff.IsZero() {
		return Report{}, fmt.Errorf("%w: snapshot and cutoff are required", ErrInvalidRun)
	}
	roundingCol := string(rounding)
	if rounding == domain.RoundingNone {
		roundingCol = "NONE"
	} else if !rounding.Known() {
		return Report{}, fmt.Errorf("%w: rounding %q is not registered", ErrInvalidRun, rounding)
	}
	started := c.clock.Now().Time().UTC()
	rep := Report{Counts: map[string]int64{}, Cutoff: ev.Cutoff.UTC(), Outcome: result.Success, Reasons: []reasoncodes.Code{}, Rounding: roundingCol}
	inputJSON, _ := json.Marshal(struct {
		V int
		R string
		E api.Evidence
	}{domain.NormalizationVersion, roundingCol, normalized(ev)})
	h := sha256.Sum256(inputJSON)
	rep.InputSHA = h[:]

	var existing *Report
	if err := c.runner.InTx(ctx, db.TxOptions{ReadOnly: true}, func(ctx context.Context, tx db.Tx) error {
		var r Report
		var outcome string
		err := tx.QueryRow(ctx, `SELECT run_id::text, outcome, facts_sha256 FROM commission.shadow_run
			WHERE normalization_version = $1 AND rounding = $2 AND input_sha256 = $3`, domain.NormalizationVersion, roundingCol, rep.InputSHA).
			Scan(&r.RunID, &outcome, &r.FactsSHA)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		r.Replayed, r.InputSHA, r.Cutoff, r.Counts, r.Reasons, r.Rounding = true, rep.InputSHA, rep.Cutoff, map[string]int64{}, []reasoncodes.Code{}, roundingCol
		r.Outcome, err = result.ParseStatus(outcome)
		existing = &r
		return err
	}); err != nil {
		return rep, err
	}
	if existing != nil {
		return *existing, nil
	}
	if rep.RunID, err = ids.NewUUID(); err != nil {
		return rep, err
	}
	d := Derive(ev, rounding)
	srcs := make([]string, 0, len(ev.Sources))
	for s := range ev.Sources {
		srcs = append(srcs, s)
	}
	sort.Strings(srcs)
	for _, s := range srcs {
		if ev.Sources[s] != api.SourceAvailable {
			rep.Outcome = result.Partial
			rep.Reasons = append(rep.Reasons, reasoncodes.CoverageIncomplete)
		}
	}
	baseOutcome, baseReasons := rep.Outcome, append([]reasoncodes.Code(nil), rep.Reasons...)
	err = c.runner.InTx(ctx, db.TxOptions{Isolation: db.Serializable}, func(ctx context.Context, tx db.Tx) error {
		// A retried transaction starts from the same pre-transaction state.
		rep.Outcome, rep.Reasons, rep.Counts = baseOutcome, append([]reasoncodes.Code(nil), baseReasons...), map[string]int64{}
		// Resolve policy versions (global, insert-only): an existing content
		// keeps its version; a new content is the next version.
		versions := map[string]int{}
		newPolicies := []PolicyObs{}
		for _, p := range d.Policies {
			var v int
			err := tx.QueryRow(ctx, `SELECT version FROM commission.policy_version WHERE policy_key = $1 AND content_sha256 = $2`,
				p.Policy.PolicyKey, p.SHA).Scan(&v)
			if errors.Is(err, pgx.ErrNoRows) {
				if err := tx.QueryRow(ctx, `SELECT coalesce(max(version), 0) FROM commission.policy_version WHERE policy_key = $1`,
					p.Policy.PolicyKey).Scan(&v); err != nil {
					return err
				}
				for _, q := range newPolicies {
					if q.Policy.PolicyKey == p.Policy.PolicyKey {
						v++
					}
				}
				v++
				newPolicies = append(newPolicies, p)
			} else if err != nil {
				return err
			}
			versions[p.Policy.PolicyKey+"\x1f"+string(p.SHA)] = v
		}
		d.Finish(versions)
		for _, cr := range d.Calcs {
			if cr.Output.Eligibility == domain.Blocked {
				rep.Outcome = result.Partial
				for _, b := range cr.Output.Blocks {
					rep.Reasons = append(rep.Reasons, b.Code())
				}
			}
		}
		if d.Summary.Coverage != "FULL" {
			rep.Outcome = result.Partial
			rep.Reasons = append(rep.Reasons, reasoncodes.CoverageIncomplete)
		}
		rep.Reasons, _ = reasoncodes.Normalize(rep.Reasons)
		if rep.Reasons == nil {
			rep.Reasons = []reasoncodes.Code{}
		}
		rep.Parity = d.Summary
		rep.Parity.WindowFrom, rep.Parity.WindowTo = Window(ev)
		facts, err := c.tables(rep.RunID, ev, d)
		if err != nil {
			return err
		}
		fh := sha256.New()
		for _, t := range facts {
			for _, r := range t.rows {
				b, _ := json.Marshal(append([]any{t.name}, r...))
				fh.Write(b)
				fh.Write([]byte{'\n'})
			}
			rep.Counts[strings.TrimPrefix(t.name, "commission.")] = int64(len(t.rows))
		}
		rep.FactsSHA = fh.Sum(nil)
		completed := c.clock.Now().Time().UTC()
		if completed.Before(started) {
			completed = started
		}
		reasons, _ := json.Marshal(rep.Reasons)
		counts, _ := json.Marshal(rep.Counts)
		if _, err := tx.Exec(ctx, `INSERT INTO commission.shadow_run (run_id, normalization_version, rounding, snapshot_id, batch_id, source_cutoff,
			started_at, completed_at, outcome, reasons, counts, input_sha256, facts_sha256) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
			rep.RunID, domain.NormalizationVersion, roundingCol, ev.SnapshotID, ev.BatchID, rep.Cutoff, started, completed, rep.Outcome.String(),
			reasons, counts, rep.InputSHA, rep.FactsSHA); err != nil {
			return err
		}
		for _, s := range srcs {
			if _, err := tx.Exec(ctx, `INSERT INTO commission.run_source (run_id, source, status, rows_read) VALUES ($1, $2, $3, $4)`,
				rep.RunID, s, ev.Sources[s], ev.SourceRows[s]); err != nil {
				return err
			}
		}
		for _, p := range newPolicies {
			pp := p.Policy
			var from, to any
			if pp.EffectiveFrom != nil {
				from = pp.EffectiveFrom.Format("2006-01-02")
			}
			if pp.EffectiveTo != nil {
				to = pp.EffectiveTo.Format("2006-01-02")
			}
			if _, err := tx.Exec(ctx, `INSERT INTO commission.policy_version (policy_key, version, content_sha256, purpose, share_facet, eligibility_ref,
				function_id, base_kind, rate_e4, fixed_hundredths, unit, effective_from, effective_to, attribution_facet, scope, provenance, evidence,
				source_ref, first_run_id) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12::date,$13::date,$14,$15,$16,$17,$18,$19)`,
				pp.PolicyKey, versions[pp.PolicyKey+"\x1f"+string(p.SHA)], p.SHA, pp.Purpose, pp.ShareFacet, pp.EligibilityRef, string(pp.FunctionID),
				string(pp.Base), rt(pp.Rate), ex(pp.Fixed), pp.Unit.String(), from, to, string(pp.AttributionFacet), pp.Scope, pp.Provenance,
				string(pp.Evidence), pp.SourceRef, rep.RunID); err != nil {
				return fmt.Errorf("policy_version: %w", err)
			}
		}
		for _, cr := range d.Calcs {
			if cr.Output.EntitlementKey == nil {
				continue
			}
			id := cr.Identity
			if _, err := tx.Exec(ctx, `INSERT INTO commission.entitlement_identity (entitlement_key, kind, invoice_key, stage, purpose, share_facet,
				generation_context, first_run_id, first_recipient_key) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT (entitlement_key) DO NOTHING`,
				*cr.Output.EntitlementKey, string(id.Kind), id.InvoiceKey, id.Stage, id.Purpose, id.ShareFacet, id.Context, rep.RunID,
				sp(firstRecipient(cr))); err != nil {
				return err
			}
		}
		for _, t := range facts {
			if err := insert(ctx, tx, rep.RunID, t); err != nil {
				return err
			}
		}
		s := rep.Parity
		js := func(v any) []byte { b, _ := json.Marshal(v); return b }
		_, err = tx.Exec(ctx, `INSERT INTO commission.parity_summary (run_id, cohort, window_from_raw, window_to_raw, legacy_rows, compared_rows,
			coverage, classes, engines, legacy_totals, target_totals, legacy_missing, target_missing, reasons)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`, rep.RunID, s.Cohort, sp(s.WindowFrom), sp(s.WindowTo), s.LegacyRows,
			s.ComparedRows, s.Coverage, js(s.Classes), js(s.Engines), js(s.LegacyTotals), js(s.TargetTotals), s.LegacyMissing, s.TargetMissing, js(s.Reasons))
		return err
	})
	return rep, err
}

func (c *Command) tables(runID string, ev api.Evidence, d Derived) ([]*table, error) {
	hist := &table{name: "commission.historical_entitlement", cols: []string{"source_key", "engine", "engine_basis", "type_raw", "direction", "status_raw",
		"rewritten_in_place", "holder_key", "holder_user_id", "invoice_key", "invoice_id", "amount_raw", "amount_hundredths", "unit", "stage_raw",
		"purpose_raw", "share_facet_raw", "rate_raw", "base_raw", "recipient_snapshot_key", "snapshot_version_raw", "reverses_source_key", "dry_run_key",
		"dry_run_item_key", "entitlement_key", "occurred_at_raw"}}
	entOf := map[string]*string{}
	for _, cr := range d.Calcs {
		if strings.HasPrefix(cr.SourceKey, "tx:") {
			entOf[strings.TrimPrefix(cr.SourceKey, "tx:")] = cr.Output.EntitlementKey
		}
	}
	credits := normalized(ev).Credits
	for _, cr := range credits {
		ent := entOf[cr.TxKey]
		if ent == nil {
			if id, ok := identityOf(cr); ok {
				if k, err := id.Key(); err == nil {
					ent = &k
				}
			}
		}
		hist.add(cr.TxKey, cr.Engine, cr.EngineBasis, cr.TypeRaw, cr.Direction, sp(cr.StatusRaw), cr.RewrittenInPlace, sp(cr.HolderKey),
			sp(cr.HolderUserID), sp(cr.InvoiceKey), sp(cr.InvoiceID), sp(cr.AmountRaw), ex(exact(cr.AmountRaw)), cr.Unit, sp(cr.StageRaw),
			sp(cr.PurposeRaw), sp(cr.RoleRaw), sp(cr.RateRaw), sp(cr.BaseRaw), sp(cr.RecipientSnapKey), sp(cr.SnapshotVersion), sp(cr.ReversesTxKey),
			sp(cr.DryRunKey), sp(cr.DryRunItemKey), sp(ent), sp(cr.CreatedAtRaw))
	}
	runs := &table{name: "commission.historical_run", cols: []string{"source_key", "run_code", "window_from_raw", "window_to_raw", "status_raw",
		"approval_raw", "state", "created_by_key", "approved_by_key", "approved_at_raw", "rejected_by_key", "rejected_at_raw", "locked_at_raw",
		"rejected_then_approved", "created_at_raw"}}
	for _, r := range normalized(ev).DryRuns {
		rta := r.ApprovalRaw != nil && *r.ApprovalRaw == "approved" && r.RejectedAtRaw != nil && *r.RejectedAtRaw != ""
		runs.add(r.SourceKey, sp(r.RunCode), sp(r.DateFromRaw), sp(r.DateToRaw), sp(r.StatusRaw), sp(r.ApprovalRaw), string(domain.LegacyRunState(r.ApprovalRaw)),
			sp(r.CreatedByKey), sp(r.ApprovedByKey), sp(r.ApprovedAtRaw), sp(r.RejectedByKey), sp(r.RejectedAtRaw), sp(r.LockedAtRaw), rta, sp(r.CreatedAtRaw))
	}
	items := &table{name: "commission.historical_run_item", cols: []string{"source_key", "run_key", "invoice_key", "invoice_id", "seller_key",
		"seller_user_id", "position_raw", "level_raw", "invoice_amount_raw", "matched_rule_key", "calculated_raw", "item_status_raw", "posting_status_raw",
		"posting_state", "wallet_tx_key", "posted_at_raw", "compensation_key", "calculation_date_raw", "non_invoice"}}
	for _, it := range normalized(ev).DryRunItems {
		items.add(it.SourceKey, sp(it.RunKey), sp(it.InvoiceKey), sp(it.InvoiceID), sp(it.SellerKey), sp(it.SellerUserID), sp(it.PositionRaw),
			sp(it.LevelRaw), sp(it.InvoiceAmountRaw), sp(it.MatchedRuleKey), sp(it.CalculatedRaw), sp(it.ItemStatusRaw), sp(it.PostingStatusRaw),
			string(domain.LegacyItemPosting(it.PostingStatusRaw)), sp(it.WalletTxKey), sp(it.PostedAtRaw), sp(it.CompensationKey), sp(it.CalculationDate),
			it.NonInvoice)
	}
	logs := &table{name: "commission.historical_posting_log", cols: []string{"source_key", "run_key", "item_key", "wallet_tx_key", "actor_key",
		"action_raw", "status_raw", "created_at_raw"}}
	for _, l := range normalized(ev).PostingLogs {
		logs.add(l.SourceKey, sp(l.RunKey), sp(l.ItemKey), sp(l.WalletTxKey), sp(l.ActorKey), sp(l.ActionRaw), sp(l.StatusRaw), sp(l.CreatedAtRaw))
	}
	calc := &table{name: "commission.calculation", cols: []string{"calc_id", "source_key", "entitlement_key", "engine", "function_id", "policy_key",
		"policy_version", "event_ref", "input_sha256", "attribution", "share_facet", "recipient_state", "recipient_key", "recipient_user_id",
		"base_hundredths", "rate_e4", "amount_hundredths", "legacy_amount_hundredths", "unit", "eligibility", "coverage", "blocks", "reasons",
		"parity_class", "parity_fields"}}
	seenEnt := map[string]bool{}
	for _, cr := range d.Calcs {
		o := cr.Output
		ent := o.EntitlementKey
		if ent != nil && seenEnt[*ent] {
			return nil, fmt.Errorf("commission: entitlement %s derived twice in one run", *ent)
		}
		if ent != nil {
			seenEnt[*ent] = true
		}
		var pv any
		if cr.PolicyKey != nil && o.PolicyVer > 0 {
			pv = o.PolicyVer
		}
		var pk any
		if pv != nil {
			pk = *cr.PolicyKey
		}
		blocks := []string{}
		for _, b := range o.Blocks {
			blocks = append(blocks, string(b))
		}
		reasons := o.Reasons
		if reasons == nil {
			reasons = []string{}
		}
		attr, _ := json.Marshal(cr.Attribution)
		bj, _ := json.Marshal(blocks)
		rj, _ := json.Marshal(cr.Parity.Reasons)
		fj, _ := json.Marshal(cr.Parity.Fields)
		_ = reasons
		var recipientUser any
		if o.Recipient.State == domain.AttrSnapshot && cr.RecipientUser != nil {
			recipientUser = *cr.RecipientUser
		}
		calcID := ids.Stable("commission.calculation", runID, cr.SourceKey)
		calc.add(calcID, cr.SourceKey, sp(ent), string(cr.Engine), string(o.FunctionID), pk, pv, o.EventRef, o.InputSHA, attr, o.ShareFacet,
			string(o.Recipient.State), sp(o.Recipient.UserKey), recipientUser, ex(o.Base), rt(o.Rate), ex(o.Amount), ex(cr.LegacyAmount), "IRT",
			string(o.Eligibility), o.Coverage, bj, rj, string(cr.Parity.Class), fj)
	}
	find := &table{name: "commission.reconciliation_finding", cols: []string{"finding_id", "kind", "severity", "source_key", "entitlement_key", "detail"}}
	for i, f := range d.Findings {
		find.add(ids.Stable("commission.finding", runID, f.Kind, f.SourceKey, fmt.Sprint(i)), f.Kind, f.Severity, f.SourceKey, sp(f.EntitlementKey), f.Detail)
	}
	return []*table{hist, runs, items, logs, calc, find}, nil
}

// firstRecipient is the original recipient evidence of an entitlement: the
// holder of its earliest legacy credit when known, else the calculation's
// own event-time recipient snapshot.
func firstRecipient(cr CalcRow) *string {
	if cr.FirstRecipient != nil {
		return cr.FirstRecipient
	}
	return cr.Output.Recipient.UserKey
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
