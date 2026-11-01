package query

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pagination"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/reporting/domain"
)

// ReportFreshnessSLA is how old a run may be before report results are STALE.
const ReportFreshnessSLA = 24 * time.Hour

// Service reads the reporting projections.
type Service struct {
	runner *db.Runner
	clock  timeutil.Clock
}

// New returns the read service over a reporting-owned runner.
func New(runner *db.Runner, clock timeutil.Clock) *Service {
	return &Service{runner: runner, clock: clock}
}

func (s *Service) read(ctx context.Context, fn func(ctx context.Context, tx db.Tx) error) error {
	return s.runner.InTx(ctx, db.TxOptions{ReadOnly: true, Isolation: db.RepeatableRead}, fn)
}

// UnitAmount is a per-unit money aggregate (units never combined).
type UnitAmount struct {
	Unit     string
	Sum      *int64 // nil on int64 overflow (no value claimed)
	Rows     int64
	Overflow bool
}

// Summary is the aggregate of a spec's FULL membership (never a page).
type Summary struct {
	Total           int64
	Counts          map[string]map[string]int64
	Amounts         []UnitAmount
	Unrepresentable int64
	Absent          int64
	UnitClasses     map[string]map[string]int64
}

// Result is the metadata every report answer carries.
type Result struct {
	Spec           Spec
	Run            *Run
	Available      bool
	Freshness      string
	Coverage       string
	SourceCoverage map[string]string
	Summary        *Summary
}

func (s *Service) result(spec Spec, run *Run, sla time.Duration) Result {
	r := Result{Spec: spec, Run: run, Freshness: domain.Unknown, Coverage: domain.Unknown, SourceCoverage: map[string]string{}}
	if run == nil {
		for _, src := range RequiredSources(spec.Report) {
			r.SourceCoverage[src] = domain.Unknown
		}
		return r
	}
	r.Available = true
	r.Freshness = domain.FreshnessOf(&run.CompletedAt, s.clock.Now().Time(), sla)
	r.Coverage, r.SourceCoverage = run.Coverage(spec.Report)
	return r
}

// begin resolves the run of a spec inside the read transaction. A spec no
// run can answer yields a Result with Available=false (UNKNOWN), never zero.
func (s *Service) begin(ctx context.Context, tx db.Tx, spec *Spec, sla time.Duration) (Result, error) {
	if err := spec.Validate(); err != nil {
		return Result{}, err
	}
	if spec.TimeBasis == domain.EventRange && !EventRangeSupported(spec.Report) {
		return s.result(*spec, nil, sla), nil
	}
	run, err := resolveRun(ctx, tx, *spec)
	if errors.Is(err, ErrNotAvailable) {
		return s.result(*spec, nil, sla), nil
	}
	if err != nil {
		return Result{}, err
	}
	pinned := run.RunID
	spec.RunID = &pinned
	res := s.result(*spec, &run, sla)
	// The run exists but the report's own source contributed no facts
	// (MISSING / UNKNOWN / NOT_RECOVERABLE): no summary, no rows — UNKNOWN,
	// never a zero total. Coverage and the run stay visible.
	res.Available = run.PrimaryUsable(spec.Report)
	return res, nil
}

func summarize(ctx context.Context, tx db.Tx, spec Spec, runID string) (*Summary, error) {
	def := reports[spec.Report]
	w, args := where(spec, runID)
	from := " FROM " + def.table + " f WHERE " + w
	sum := &Summary{Counts: map[string]map[string]int64{}}
	if err := tx.QueryRow(ctx, "SELECT count(*)"+from, args...).Scan(&sum.Total); err != nil {
		return nil, err
	}
	dims := make([]string, 0, len(def.dims))
	for d := range def.dims {
		dims = append(dims, d)
	}
	sort.Strings(dims)
	for _, d := range dims {
		rows, err := tx.Query(ctx, "SELECT "+def.dims[d]+", count(*)"+from+" GROUP BY 1", args...)
		if err != nil {
			return nil, err
		}
		m := map[string]int64{}
		for rows.Next() {
			var k string
			var n int64
			if err := rows.Scan(&k, &n); err != nil {
				rows.Close()
				return nil, err
			}
			m[k] = n
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
		sum.Counts[d] = m
	}
	switch spec.Report {
	case domain.ReportPayments:
		rows, err := tx.Query(ctx, "SELECT f.unit, sum(f.amount)::text, count(*)"+from+" AND f.amount_repr = 'REPRESENTABLE' GROUP BY 1 ORDER BY 1", args...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var u, total string
			var n int64
			if err := rows.Scan(&u, &total, &n); err != nil {
				rows.Close()
				return nil, err
			}
			a := UnitAmount{Unit: u, Rows: n}
			if v, err := strconv.ParseInt(total, 10, 64); err == nil {
				a.Sum = &v
			} else {
				a.Overflow = true
			}
			sum.Amounts = append(sum.Amounts, a)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
		if err := tx.QueryRow(ctx, "SELECT count(*) FILTER (WHERE f.amount_repr = 'UNREPRESENTABLE'), count(*) FILTER (WHERE f.amount_repr = 'ABSENT')"+from,
			args...).Scan(&sum.Unrepresentable, &sum.Absent); err != nil {
			return nil, err
		}
	case domain.ReportInvoices:
		sum.UnitClasses = map[string]map[string]int64{}
		rows, err := tx.Query(ctx, "SELECT u.unit, u.total_class, count(*) FROM "+def.table+
			" f JOIN reporting.invoice_unit_fact u ON u.run_id = f.run_id AND u.invoice_id = f.invoice_id WHERE "+w+" GROUP BY 1, 2", args...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var u, c string
			var n int64
			if err := rows.Scan(&u, &c, &n); err != nil {
				rows.Close()
				return nil, err
			}
			if sum.UnitClasses[u] == nil {
				sum.UnitClasses[u] = map[string]int64{}
			}
			sum.UnitClasses[u][c] = n
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	return sum, nil
}

// Page is one page of rows of a spec.
type Page struct {
	Rows       []any
	NextCursor *string
}

// Report answers a spec: the FULL-membership summary and one page of rows.
// The summary never depends on cursor or limit.
func (s *Service) Report(ctx context.Context, spec Spec, cursor *string, limitIn *int) (Result, Page, error) {
	limit, err := pagination.Limit(limitIn)
	if err != nil {
		return Result{}, Page{}, invalid("%v", err)
	}
	var after *string
	if cursor != nil {
		c, err := pagination.Decode(*cursor, "id")
		if err != nil || !uuidPattern.MatchString(c.ID) {
			return Result{}, Page{}, invalid("cursor")
		}
		after = &c.ID
	}
	var res Result
	var page Page
	err = s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		if res, err = s.begin(ctx, tx, &spec, ReportFreshnessSLA); err != nil || !res.Available {
			return err
		}
		if res.Summary, err = summarize(ctx, tx, spec, res.Run.RunID); err != nil {
			return err
		}
		page.Rows, err = rowsOf(ctx, tx, spec, res.Run.RunID, after, limit+1, nil)
		if err != nil {
			return err
		}
		if len(page.Rows) > limit {
			page.Rows = page.Rows[:limit]
			c, err := pagination.Encode(pagination.Cursor{Sort: "id", ID: rowID(page.Rows[limit-1])})
			if err != nil {
				return err
			}
			page.NextCursor = &c
		}
		return nil
	})
	return res, page, err
}

// Detail returns one row of the spec's run and whether it is a member of the
// spec (the same predicate as the card and the table).
func (s *Service) Detail(ctx context.Context, spec Spec, id string) (Result, any, bool, error) {
	if !uuidPattern.MatchString(id) {
		return Result{}, nil, false, invalid("id")
	}
	var res Result
	var row any
	var member bool
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		if res, err = s.begin(ctx, tx, &spec, ReportFreshnessSLA); err != nil || !res.Available {
			return err
		}
		def := reports[spec.Report]
		rows, err := rowsOf(ctx, tx, Spec{Report: spec.Report}, res.Run.RunID, nil, 1, &id)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return ErrNotFound
		}
		row = rows[0]
		w, args := where(spec, res.Run.RunID)
		args = append(args, id)
		return tx.QueryRow(ctx, fmt.Sprintf("SELECT EXISTS (SELECT 1 FROM %s f WHERE %s AND f.%s = $%d)", def.table, w, def.id, len(args)),
			args...).Scan(&member)
	})
	return res, row, member, err
}

// MetricValue is one metric evaluated through its contract's spec.
type MetricValue struct {
	Contract domain.Contract
	Status   string // FINAL / NOT_FINAL
	Reason   string
	// ValueState: VALUE, INCOMPLETE, UNKNOWN, NOT_EVALUATED.
	ValueState string
	Result     Result
}

// Metric evaluates one metric. NOT_FINAL contracts are never evaluated; a
// missing run or source yields UNKNOWN (no value), never zero.
func (s *Service) Metric(ctx context.Context, id string, tb domain.TimeBasis, from, to, asOf *time.Time, runID *string) (MetricValue, error) {
	c, ok := domain.Lookup(id)
	if !ok {
		return MetricValue{}, ErrNotFound
	}
	mv := MetricValue{Contract: c}
	mv.Status, mv.Reason = c.Status()
	if mv.Status != "FINAL" {
		mv.ValueState = "NOT_EVALUATED"
		return mv, nil
	}
	if tb == "" {
		tb = c.PrimaryBasis()
	}
	if !c.Allows(tb) {
		return MetricValue{}, invalid("time_basis %s is not allowed by %s v%d", tb, c.ID, c.Version)
	}
	filters := map[string]string{}
	for k, v := range c.Filters {
		filters[k] = v
	}
	spec := Spec{Report: c.Report, TimeBasis: tb, From: from, To: to, AsOf: asOf, RunID: runID, Filters: filters}
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		res, err := s.begin(ctx, tx, &spec, c.FreshnessSLA)
		if err != nil {
			return err
		}
		mv.Result = res
		if !res.Available {
			mv.ValueState = "UNKNOWN"
			return nil
		}
		if mv.Result.Summary, err = summarize(ctx, tx, spec, res.Run.RunID); err != nil {
			return err
		}
		mv.ValueState = "VALUE"
		// Incomplete coverage, or an amount metric with any unrepresentable
		// or ABSENT contributor, is never presented as a complete value.
		if res.Coverage != domain.Available || (c.Aggregation == domain.AggSumByUnit &&
			(mv.Result.Summary.Unrepresentable > 0 || mv.Result.Summary.Absent > 0)) {
			mv.ValueState = "INCOMPLETE"
		}
		return nil
	})
	return mv, err
}

// Runs pages the projection runs (newest first) for reconciliation.
func (s *Service) Runs(ctx context.Context, cursor *string, limitIn *int) ([]Run, *string, error) {
	limit, err := pagination.Limit(limitIn)
	if err != nil {
		return nil, nil, invalid("%v", err)
	}
	var before *int64
	if cursor != nil {
		c, err := pagination.Decode(*cursor, "run_seq")
		if err != nil {
			return nil, nil, invalid("cursor")
		}
		v, err := strconv.ParseInt(c.ID, 10, 64)
		if err != nil {
			return nil, nil, invalid("cursor")
		}
		before = &v
	}
	var out []Run
	var next *string
	err = s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		rows, err := tx.Query(ctx, `SELECT run_id::text, run_seq, projection_version, source_cutoff, completed_at, outcome FROM reporting.projection_run
			WHERE ($1::bigint IS NULL OR run_seq < $1) ORDER BY run_seq DESC LIMIT $2`, before, limit+1)
		if err != nil {
			return err
		}
		var seqs []int64
		for rows.Next() {
			var r Run
			var seq int64
			if err := rows.Scan(&r.RunID, &seq, &r.ProjectionVersion, &r.SourceCutoff, &r.CompletedAt, &r.Outcome); err != nil {
				rows.Close()
				return err
			}
			out, seqs = append(out, r), append(seqs, seq)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		if len(out) > limit {
			out, seqs = out[:limit], seqs[:limit]
			c, err := pagination.Encode(pagination.Cursor{Sort: "run_seq", ID: strconv.FormatInt(seqs[limit-1], 10)})
			if err != nil {
				return err
			}
			next = &c
		}
		for i := range out {
			if out[i].Sources, err = sourcesOf(ctx, tx, out[i].RunID); err != nil {
				return err
			}
		}
		return nil
	})
	return out, next, err
}

// Aggregates returns the owner aggregate evidence of a run for one source.
func (s *Service) Aggregates(ctx context.Context, runID, source string) (map[string]int64, error) {
	out := map[string]int64{}
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		rows, err := tx.Query(ctx, `SELECT key, value FROM reporting.source_aggregate WHERE run_id = $1 AND source = $2`, runID, source)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var k string
			var v int64
			if err := rows.Scan(&k, &v); err != nil {
				return err
			}
			out[k] = v
		}
		return rows.Err()
	})
	return out, err
}

// LatestRun returns the latest run of the current projection version.
func (s *Service) LatestRun(ctx context.Context) (*Run, error) {
	var run *Run
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		r, err := resolveRun(ctx, tx, Spec{TimeBasis: domain.CurrentSnapshot})
		if errors.Is(err, ErrNotAvailable) {
			return nil
		}
		if err == nil {
			run = &r
		}
		return err
	})
	return run, err
}

// Clock exposes the service clock (freshness evaluation).
func (s *Service) Clock() timeutil.Clock { return s.clock }
