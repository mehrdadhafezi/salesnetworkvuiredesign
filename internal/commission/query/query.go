// Package query is the commission READ side (Slice 13). Every read runs in a
// READ ONLY repeatable-read transaction over the commission schema and is
// pinned to ONE stored shadow run. A read never calculates, recalculates,
// imports, approves, posts or credits anything: it returns stored results
// only. The own-earnings read is filtered to the caller's user id.
package query

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/commission"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/commission/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/commission/domain"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
)

// Errors.
var (
	ErrInvalid  = errors.New("commission: invalid query")
	ErrNoRun    = errors.New("commission: no shadow run")
	ErrNotFound = errors.New("commission: not found")
)

func invalid(f string) error { return fmt.Errorf("%w: %s", ErrInvalid, f) }

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// Service is the read service.
type Service struct{ runner *db.Runner }

// New returns the read service over a commission-owned runner.
func New(runner *db.Runner) (*Service, error) {
	if runner.Module() != commission.Module {
		return nil, errors.New("commission: read runner must be owned by commission")
	}
	return &Service{runner: runner}, nil
}

func (s *Service) read(ctx context.Context, fn func(ctx context.Context, tx db.Tx) error) error {
	return s.runner.InTx(ctx, db.TxOptions{ReadOnly: true, Isolation: db.RepeatableRead}, fn)
}

// Page is a keyset page request.
type Page struct {
	Cursor *string
	Limit  int
}

func (p Page) limit() int {
	switch {
	case p.Limit <= 0:
		return 50
	case p.Limit > 200:
		return 200
	}
	return p.Limit
}

func (p Page) after() (string, error) {
	if p.Cursor == nil || *p.Cursor == "" {
		return "", nil
	}
	b, err := base64.RawURLEncoding.DecodeString(*p.Cursor)
	if err != nil || len(b) == 0 || len(b) > 300 {
		return "", invalid("cursor")
	}
	return string(b), nil
}

func cursorOf(k string) *string {
	c := base64.RawURLEncoding.EncodeToString([]byte(k))
	return &c
}

const runCols = `r.run_id::text, r.run_seq, r.state, r.normalization_version, r.rounding, r.snapshot_id, r.source_cutoff, r.completed_at, r.outcome,
	r.reasons, r.counts`

func scanRun(row pgx.Row) (api.Run, error) {
	var r api.Run
	var reasons, counts []byte
	if err := row.Scan(&r.RunID, &r.RunSeq, &r.State, &r.NormalizationVersion, &r.Rounding, &r.SnapshotID, &r.Cutoff, &r.CompletedAt, &r.Outcome,
		&reasons, &counts); err != nil {
		return r, err
	}
	r.RecordKind = "SHADOW"
	_ = json.Unmarshal(reasons, &r.Reasons)
	_ = json.Unmarshal(counts, &r.Counts)
	if r.Reasons == nil {
		r.Reasons = []string{}
	}
	return r, nil
}

func (s *Service) resolve(ctx context.Context, tx db.Tx, runID *string) (api.Run, error) {
	q := `SELECT ` + runCols + ` FROM commission.shadow_run r`
	var row pgx.Row
	if runID != nil {
		if !uuidPattern.MatchString(*runID) {
			return api.Run{}, invalid("run_id")
		}
		row = tx.QueryRow(ctx, q+` WHERE r.run_id = $1`, *runID)
	} else {
		row = tx.QueryRow(ctx, q+` ORDER BY r.run_seq DESC LIMIT 1`)
	}
	r, err := scanRun(row)
	if errors.Is(err, pgx.ErrNoRows) {
		if runID != nil {
			return r, ErrNotFound
		}
		return r, ErrNoRun
	}
	if err != nil {
		return r, err
	}
	r.Sources = map[string]string{}
	rows, err := tx.Query(ctx, `SELECT source, status FROM commission.run_source WHERE run_id = $1 ORDER BY source`, r.RunID)
	if err != nil {
		return r, err
	}
	defer rows.Close()
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return r, err
		}
		r.Sources[k] = v
	}
	return r, rows.Err()
}

// SourceRows returns rows read per source of a run.
func (s *Service) SourceRows(ctx context.Context, runID string) (map[string]int64, error) {
	out := map[string]int64{}
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		rows, err := tx.Query(ctx, `SELECT source, rows_read FROM commission.run_source WHERE run_id = $1`, runID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var k string
			var n int64
			if err := rows.Scan(&k, &n); err != nil {
				return err
			}
			out[k] = n
		}
		return rows.Err()
	})
	return out, err
}

// Overview returns the run, its parity summary and finding counts.
func (s *Service) Overview(ctx context.Context, runID *string) (api.Overview, error) {
	out := api.Overview{Findings: map[string]int64{}, Decisions: api.Decisions(), PostingState: api.PostingStateBlocked, Notice: api.Notice}
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		r, err := s.resolve(ctx, tx, runID)
		if err != nil {
			return err
		}
		out.Run = &r
		var p api.ParitySummary
		var classes, engines, lt, tt, reasons []byte
		err = tx.QueryRow(ctx, `SELECT cohort, window_from_raw, window_to_raw, legacy_rows, compared_rows, coverage, classes, engines, legacy_totals,
			target_totals, legacy_missing, target_missing, reasons FROM commission.parity_summary WHERE run_id = $1`, r.RunID).
			Scan(&p.Cohort, &p.WindowFrom, &p.WindowTo, &p.LegacyRows, &p.ComparedRows, &p.Coverage, &classes, &engines, &lt, &tt, &p.LegacyMissing,
				&p.TargetMissing, &reasons)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
		case err != nil:
			return err
		default:
			_ = json.Unmarshal(classes, &p.Classes)
			_ = json.Unmarshal(engines, &p.Engines)
			_ = json.Unmarshal(lt, &p.LegacyTotals)
			_ = json.Unmarshal(tt, &p.TargetTotals)
			_ = json.Unmarshal(reasons, &p.Reasons)
			out.Parity = &p
		}
		rows, err := tx.Query(ctx, `SELECT kind, count(*) FROM commission.reconciliation_finding WHERE run_id = $1 GROUP BY kind`, r.RunID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var k string
			var n int64
			if err := rows.Scan(&k, &n); err != nil {
				return err
			}
			out.Findings[k] = n
		}
		return rows.Err()
	})
	return out, err
}

// Runs lists the shadow runs, newest first.
func (s *Service) Runs(ctx context.Context, limit int) ([]api.Run, error) {
	out := []api.Run{}
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		rows, err := tx.Query(ctx, `SELECT `+runCols+` FROM commission.shadow_run r ORDER BY r.run_seq DESC LIMIT $1`, Page{Limit: limit}.limit())
		if err != nil {
			return err
		}
		for rows.Next() {
			r, err := scanRun(rows)
			if err != nil {
				rows.Close()
				return err
			}
			out = append(out, r)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		for i := range out {
			out[i].Sources = map[string]string{}
			srows, err := tx.Query(ctx, `SELECT source, status FROM commission.run_source WHERE run_id = $1`, out[i].RunID)
			if err != nil {
				return err
			}
			for srows.Next() {
				var k, v string
				if err := srows.Scan(&k, &v); err != nil {
					srows.Close()
					return err
				}
				out[i].Sources[k] = v
			}
			srows.Close()
		}
		return nil
	})
	return out, err
}

var parityClasses = func() map[string]bool {
	m := map[string]bool{}
	for _, c := range domain.ParityClasses() {
		m[string(c)] = true
	}
	return m
}()

var engines = func() map[string]bool {
	m := map[string]bool{}
	for _, e := range domain.Engines() {
		m[string(e)] = true
	}
	return m
}()

func hundredths(v *int64) *string {
	if v == nil {
		return nil
	}
	s := domain.FormatDecimal(*v, domain.AmountScale)
	return &s
}

func rateText(v *int64) *string {
	if v == nil {
		return nil
	}
	s := domain.FormatDecimal(*v, domain.RateScale)
	return &s
}

// Calculations lists the stored calculations of one run.
func (s *Service) Calculations(ctx context.Context, runID, class, engine *string, p Page) (api.Run, []api.Calculation, *string, error) {
	var run api.Run
	out := []api.Calculation{}
	var next *string
	if class != nil && !parityClasses[*class] {
		return run, nil, nil, invalid("parity_class")
	}
	if engine != nil && !engines[*engine] {
		return run, nil, nil, invalid("engine")
	}
	after, err := p.after()
	if err != nil {
		return run, nil, nil, err
	}
	err = s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		r, err := s.resolve(ctx, tx, runID)
		if err != nil {
			return err
		}
		run = r
		rows, err := tx.Query(ctx, `SELECT calc_id::text, source_key, entitlement_key, engine, function_id, policy_key, policy_version, event_ref,
			input_sha256, share_facet, recipient_state, recipient_key, recipient_user_id::text, base_hundredths, rate_e4, amount_hundredths,
			legacy_amount_hundredths, unit, eligibility, coverage, blocks, reasons, parity_class, parity_fields
			FROM commission.calculation WHERE run_id = $1 AND source_key > $2 AND ($3::text IS NULL OR parity_class = $3)
			AND ($4::text IS NULL OR engine = $4) ORDER BY source_key LIMIT $5`, r.RunID, after, class, engine, p.limit()+1)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var c api.Calculation
			var base, rate, amount, legacy *int64
			var blocks, reasons []byte
			if err := rows.Scan(&c.CalcID, &c.SourceKey, &c.EntitlementKey, &c.Engine, &c.FunctionID, &c.PolicyKey, &c.PolicyVersion, &c.EventRef,
				&c.InputSHA, &c.ShareFacet, &c.RecipientState, &c.RecipientKey, &c.RecipientUserID, &base, &rate, &amount, &legacy, &c.Unit,
				&c.Eligibility, &c.Coverage, &blocks, &reasons, &c.ParityClass, &c.ParityFields); err != nil {
				return err
			}
			c.Base, c.Rate, c.Amount, c.LegacyAmount = hundredths(base), rateText(rate), hundredths(amount), hundredths(legacy)
			_ = json.Unmarshal(blocks, &c.Blocks)
			_ = json.Unmarshal(reasons, &c.Reasons)
			out = append(out, c)
		}
		return rows.Err()
	})
	if len(out) > p.limit() {
		out = out[:p.limit()]
		next = cursorOf(out[len(out)-1].SourceKey)
	}
	return run, out, next, err
}

var findingKinds = map[string]bool{"DUPLICATE_RISK": true, "REWRITTEN_IN_PLACE": true, "REVERSAL_LINK_MISSING": true, "ENGINE_ROUTING_UNKNOWN": true,
	"POSTING_LINK_MISMATCH": true, "RECIPIENT_MISMATCH": true, "REJECTED_THEN_APPROVED": true, "NON_INVOICE_BLOCKED": true, "COMPONENT_ALLOCATION_BLOCKED": true}

// Findings lists the reconciliation findings of one run.
func (s *Service) Findings(ctx context.Context, runID, kind *string, p Page) (api.Run, []api.Finding, *string, error) {
	var run api.Run
	out := []api.Finding{}
	if kind != nil && !findingKinds[*kind] {
		return run, nil, nil, invalid("kind")
	}
	after, err := p.after()
	if err != nil {
		return run, nil, nil, err
	}
	err = s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		r, err := s.resolve(ctx, tx, runID)
		if err != nil {
			return err
		}
		run = r
		rows, err := tx.Query(ctx, `SELECT finding_id::text, kind, severity, source_key, entitlement_key, detail FROM commission.reconciliation_finding
			WHERE run_id = $1 AND finding_id::text > $2 AND ($3::text IS NULL OR kind = $3) ORDER BY finding_id::text LIMIT $4`,
			r.RunID, after, kind, p.limit()+1)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var f api.Finding
			if err := rows.Scan(&f.FindingID, &f.Kind, &f.Severity, &f.SourceKey, &f.EntKey, &f.Detail); err != nil {
				return err
			}
			out = append(out, f)
		}
		return rows.Err()
	})
	var next *string
	if len(out) > p.limit() {
		out = out[:p.limit()]
		next = cursorOf(out[len(out)-1].FindingID)
	}
	return run, out, next, err
}

// Policies lists the observed policy versions (global, insert-only).
func (s *Service) Policies(ctx context.Context, p Page) ([]api.PolicyVersion, *string, error) {
	out := []api.PolicyVersion{}
	after, err := p.after()
	if err != nil {
		return nil, nil, err
	}
	err = s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		rows, err := tx.Query(ctx, `SELECT policy_key, version, purpose, share_facet, function_id, rate_e4, fixed_hundredths, unit,
			to_char(effective_from, 'YYYY-MM-DD'), to_char(effective_to, 'YYYY-MM-DD'), evidence, approval_state, provenance, source_ref, first_run_id::text
			FROM commission.policy_version WHERE policy_key || '#' || lpad(version::text, 6, '0') > $1
			ORDER BY policy_key, version LIMIT $2`, after, p.limit()+1)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var v api.PolicyVersion
			var rate, fixed *int64
			if err := rows.Scan(&v.PolicyKey, &v.Version, &v.Purpose, &v.ShareFacet, &v.FunctionID, &rate, &fixed, &v.Unit, &v.EffectiveFrom,
				&v.EffectiveTo, &v.Evidence, &v.ApprovalState, &v.Provenance, &v.SourceRef, &v.FirstRunID); err != nil {
				return err
			}
			v.Rate, v.Fixed = rateText(rate), hundredths(fixed)
			out = append(out, v)
		}
		return rows.Err()
	})
	var next *string
	if len(out) > p.limit() {
		out = out[:p.limit()]
		last := out[len(out)-1]
		next = cursorOf(fmt.Sprintf("%s#%06d", last.PolicyKey, last.Version))
	}
	return out, next, err
}

// HistoricalRuns lists the legacy dry runs of one shadow run.
func (s *Service) HistoricalRuns(ctx context.Context, runID *string) (api.Run, []api.HistoricalRun, error) {
	var run api.Run
	out := []api.HistoricalRun{}
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		r, err := s.resolve(ctx, tx, runID)
		if err != nil {
			return err
		}
		run = r
		rows, err := tx.Query(ctx, `SELECT h.source_key, h.run_code, h.window_from_raw, h.window_to_raw, h.approval_raw, h.state, h.approved_by_key,
			h.approved_at_raw, h.rejected_by_key, h.rejected_at_raw, h.rejected_then_approved,
			(SELECT count(*) FROM commission.historical_run_item i WHERE i.run_id = h.run_id AND i.run_key = h.source_key),
			(SELECT count(*) FROM commission.historical_run_item i WHERE i.run_id = h.run_id AND i.run_key = h.source_key AND i.posting_state = 'POSTED'),
			(SELECT count(*) FROM commission.historical_run_item i WHERE i.run_id = h.run_id AND i.run_key = h.source_key AND i.posting_state = 'POSTING_FAILED')
			FROM commission.historical_run h WHERE h.run_id = $1 ORDER BY h.source_key LIMIT 200`, r.RunID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var h api.HistoricalRun
			if err := rows.Scan(&h.SourceKey, &h.RunCode, &h.WindowFrom, &h.WindowTo, &h.ApprovalRaw, &h.State, &h.ApprovedByKey, &h.ApprovedAtRaw,
				&h.RejectedByKey, &h.RejectedAtRaw, &h.RejectedThenApproved, &h.ItemCount, &h.PostedItems, &h.FailedItems); err != nil {
				return err
			}
			out = append(out, h)
		}
		return rows.Err()
	})
	return run, out, err
}

// Earnings lists the historical commission credits / reversals whose
// holder is userID (own scope only; never another recipient).
func (s *Service) Earnings(ctx context.Context, runID *string, userID string, p Page) (api.Run, []api.Earning, map[string]string, *string, error) {
	var run api.Run
	out := []api.Earning{}
	totals := map[string]string{}
	if !uuidPattern.MatchString(userID) {
		return run, nil, nil, nil, invalid("user")
	}
	after, err := p.after()
	if err != nil {
		return run, nil, nil, nil, err
	}
	err = s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		r, err := s.resolve(ctx, tx, runID)
		if err != nil {
			return err
		}
		run = r
		rows, err := tx.Query(ctx, `SELECT h.source_key, h.engine, h.direction, h.type_raw, h.invoice_id::text, h.stage_raw, h.purpose_raw,
			h.share_facet_raw, h.amount_hundredths, h.unit, (h.rewritten_in_place OR EXISTS (SELECT 1 FROM commission.historical_entitlement d
			WHERE d.run_id = h.run_id AND d.reverses_source_key = h.source_key)), h.occurred_at_raw, h.entitlement_key
			FROM commission.historical_entitlement h WHERE h.run_id = $1 AND h.holder_user_id = $2::uuid AND lpad(h.source_key, 20, '0') > $3
			ORDER BY lpad(h.source_key, 20, '0') LIMIT $4`, r.RunID, userID, after, p.limit()+1)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var e api.Earning
			var amount *int64
			if err := rows.Scan(&e.SourceKey, &e.Engine, &e.Direction, &e.TypeRaw, &e.InvoiceID, &e.Stage, &e.Purpose, &e.ShareFacet, &amount, &e.Unit,
				&e.Reversed, &e.OccurredAtRaw, &e.EntitlementKey); err != nil {
				return err
			}
			e.Amount = hundredths(amount)
			out = append(out, e)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		// Totals per unit over ALL own rows (credits minus debits); a row
		// without an exact amount is counted as missing, never zero.
		trows, err := tx.Query(ctx, `SELECT unit, coalesce(sum(CASE direction WHEN 'CREDIT' THEN amount_hundredths WHEN 'DEBIT' THEN -amount_hundredths END), 0),
			count(*) FILTER (WHERE amount_hundredths IS NULL OR direction = 'UNKNOWN')
			FROM commission.historical_entitlement WHERE run_id = $1 AND holder_user_id = $2::uuid GROUP BY unit`, r.RunID, userID)
		if err != nil {
			return err
		}
		defer trows.Close()
		for trows.Next() {
			var unit string
			var sum, missing int64
			if err := trows.Scan(&unit, &sum, &missing); err != nil {
				return err
			}
			totals[unit] = domain.FormatDecimal(sum, domain.AmountScale)
			if missing > 0 {
				totals[unit+"_rows_without_exact_amount"] = fmt.Sprint(missing)
			}
		}
		return trows.Err()
	})
	var next *string
	if len(out) > p.limit() {
		out = out[:p.limit()]
		k := out[len(out)-1].SourceKey
		next = cursorOf(strings.Repeat("0", max(0, 20-len(k))) + k)
	}
	return run, out, totals, next, err
}
