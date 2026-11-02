// Package query is the finance SHADOW ledger read side (Slice 9). Every
// read runs in a READ ONLY repeatable-read transaction over the finance
// schema: it never derives, refreshes, repairs, posts or records anything.
// Every answer is pinned to ONE shadow run (its source cutoff and snapshot):
// list, detail and summary of a request always come from the same run, and a
// detail id of another run is not found (no cross-run drilldown).
//
// Balances are derived from ledger lines at read time, per account, basis
// and unit; units are never combined and no basis is preferred (D-41).
package query

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/finance"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/finance/domain"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
)

// FreshnessSLA is the age after which a run or a source snapshot is STALE.
const FreshnessSLA = 24 * time.Hour

// Errors.
var (
	ErrInvalid  = errors.New("finance: invalid query")
	ErrNoRun    = errors.New("finance: no shadow run")
	ErrNotFound = errors.New("finance: not found in this run")
)

func invalid(field string) error { return fmt.Errorf("%w: %s", ErrInvalid, field) }

// Service is the read service over a finance-owned runner.
type Service struct {
	runner *db.Runner
	clock  timeutil.Clock
}

// New returns the read service.
func New(runner *db.Runner, clock timeutil.Clock) (*Service, error) {
	if runner.Module() != finance.Module {
		return nil, errors.New("finance: read runner must be owned by finance")
	}
	return &Service{runner: runner, clock: clock}, nil
}

func (s *Service) read(ctx context.Context, fn func(ctx context.Context, tx db.Tx) error) error {
	return s.runner.InTx(ctx, db.TxOptions{ReadOnly: true, Isolation: db.RepeatableRead}, fn)
}

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// Source is one source of a run.
type Source struct {
	Source, Status, Reason string
	Rows                   *int64
	SnapshotID             *string
	Cutoff                 *time.Time
	ReadAt                 time.Time
	Freshness              string
}

// Run is one shadow run as read back.
type Run struct {
	RunID                            string
	Seq                              int64
	DerivationVersion                int
	Bases                            []string
	SourceCutoff, Started, Completed time.Time
	Outcome                          string
	Reasons                          []string
	Counts                           map[string]int64
	FactsSHA                         string
	Sources                          []Source
	Freshness                        string
}

const runSelect = `SELECT run_id::text, run_seq, derivation_version, bases, source_cutoff, started_at, completed_at, outcome, reasons, fact_counts,
	facts_sha256 FROM finance.shadow_run`

func (s *Service) scanRun(r pgx.Row) (Run, error) {
	var v Run
	var reasons, counts []byte
	var sha []byte
	if err := r.Scan(&v.RunID, &v.Seq, &v.DerivationVersion, &v.Bases, &v.SourceCutoff, &v.Started, &v.Completed, &v.Outcome, &reasons, &counts, &sha); err != nil {
		return v, err
	}
	_ = json.Unmarshal(reasons, &v.Reasons)
	_ = json.Unmarshal(counts, &v.Counts)
	v.FactsSHA = hex.EncodeToString(sha)
	v.Freshness = s.freshness(&v.Completed)
	return v, nil
}

func (s *Service) freshness(t *time.Time) string {
	if t == nil {
		return "UNKNOWN"
	}
	if s.clock.Now().Time().Sub(*t) > FreshnessSLA {
		return "STALE"
	}
	return "FRESH"
}

func (s *Service) sources(ctx context.Context, tx db.Tx, run *Run) error {
	rows, err := tx.Query(ctx, `SELECT source, status, rows_read, snapshot_id, source_cutoff, read_at, reason FROM finance.run_source
		WHERE run_id = $1 ORDER BY source`, run.RunID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var src Source
		if err := rows.Scan(&src.Source, &src.Status, &src.Rows, &src.SnapshotID, &src.Cutoff, &src.ReadAt, &src.Reason); err != nil {
			return err
		}
		src.Freshness = s.freshness(src.Cutoff)
		run.Sources = append(run.Sources, src)
	}
	return rows.Err()
}

// resolve returns the pinned run, or the latest run when runID is nil.
func (s *Service) resolve(ctx context.Context, tx db.Tx, runID *string) (Run, error) {
	var row pgx.Row
	if runID != nil {
		if !uuidPattern.MatchString(*runID) {
			return Run{}, invalid("run_id")
		}
		row = tx.QueryRow(ctx, runSelect+` WHERE run_id = $1`, *runID)
	} else {
		row = tx.QueryRow(ctx, runSelect+` ORDER BY run_seq DESC LIMIT 1`)
	}
	run, err := s.scanRun(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Run{}, ErrNoRun
	}
	if err != nil {
		return Run{}, err
	}
	return run, s.sources(ctx, tx, &run)
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

func encodeCursor(parts ...string) *string {
	c := base64.RawURLEncoding.EncodeToString([]byte(strings.Join(parts, "\x1f")))
	return &c
}

func decodeCursor(c *string, n int) ([]string, error) {
	if c == nil {
		return nil, nil
	}
	b, err := base64.RawURLEncoding.DecodeString(*c)
	if err != nil || len(*c) > 512 {
		return nil, invalid("cursor")
	}
	parts := strings.Split(string(b), "\x1f")
	if len(parts) != n {
		return nil, invalid("cursor")
	}
	return parts, nil
}

// Runs lists shadow runs, newest first.
func (s *Service) Runs(ctx context.Context, p Page) ([]Run, *string, error) {
	limit, err := p.limit()
	if err != nil {
		return nil, nil, err
	}
	cur, err := decodeCursor(p.Cursor, 1)
	if err != nil {
		return nil, nil, err
	}
	var after *int64
	if cur != nil {
		v, err := strconv.ParseInt(cur[0], 10, 64)
		if err != nil {
			return nil, nil, invalid("cursor")
		}
		after = &v
	}
	var out []Run
	err = s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		rows, err := tx.Query(ctx, runSelect+` WHERE ($1::bigint IS NULL OR run_seq < $1) ORDER BY run_seq DESC LIMIT $2`, after, limit+1)
		if err != nil {
			return err
		}
		for rows.Next() {
			r, err := s.scanRun(rows)
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
			if err := s.sources(ctx, tx, &out[i]); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	var next *string
	if len(out) > limit {
		out = out[:limit]
		next = encodeCursor(strconv.FormatInt(out[limit-1].Seq, 10))
	}
	return out, next, nil
}

// Count is one grouped count.
type Count struct {
	Basis, Unit, State, Reason string
	N                          int64
}

// RunDetail is one run with its reconciliation and finding summaries.
type RunDetail struct {
	Run
	Reconciliation []Count
	Findings       []Count
}

// RunByID returns one run (latest when runID is nil) with its summaries.
func (s *Service) RunByID(ctx context.Context, runID *string) (RunDetail, error) {
	var d RunDetail
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		if d.Run, err = s.resolve(ctx, tx, runID); err != nil {
			return err
		}
		if d.Reconciliation, err = counts(ctx, tx, `SELECT basis, unit, state, '', count(*) FROM finance.reconciliation WHERE run_id = $1
			GROUP BY 1, 2, 3 ORDER BY 1, 2, 3`, d.RunID); err != nil {
			return err
		}
		d.Findings, err = counts(ctx, tx, `SELECT coalesce(basis, ''), '', state, reason, count(*) FROM finance.finding WHERE run_id = $1
			GROUP BY 1, 3, 4 ORDER BY 1, 3, 4`, d.RunID)
		return err
	})
	return d, err
}

func counts(ctx context.Context, tx db.Tx, q string, args ...any) ([]Count, error) {
	rows, err := tx.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Count{}
	for rows.Next() {
		var c Count
		if err := rows.Scan(&c.Basis, &c.Unit, &c.State, &c.Reason, &c.N); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Amount is an exact sum; nil Value means the int64 sum overflowed (no
// value is claimed).
type Amount struct {
	Value *int64
}

func parseSum(text string) Amount {
	v, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return Amount{}
	}
	return Amount{Value: &v}
}

// net returns the normal-side net of debits and credits (nil on overflow).
func net(d, c Amount, normal string) Amount {
	if d.Value == nil || c.Value == nil {
		return Amount{}
	}
	b := domain.Balance{Debits: *d.Value, Credits: *c.Value}
	if v, ok := b.Net(normal); ok {
		return Amount{Value: &v}
	}
	return Amount{}
}

var (
	bases      = map[string]bool{domain.AccrualShadow: true, domain.CashShadow: true, domain.WalletMirrorShadow: true}
	units      = map[string]bool{"IRR": true, "IRT": true}
	categories = func() map[string]bool {
		m := map[string]bool{}
		for c := range domain.Categories {
			m[c] = true
		}
		return m
	}()
	reconStates  = map[string]bool{"MATCH": true, "MISMATCH": true, "UNKNOWN": true, "INCOMPLETE": true, "CONFLICT": true, "NOT_RECOVERABLE": true}
	entryStates  = map[string]bool{domain.EntryConsistent: true, domain.EntryConflict: true, domain.EntryUnresolvedCorrection: true}
	sourceDomain = map[string]bool{"billing.invoice": true, "payments.attempt": true, "legacy.wallet_transaction": true}
	subjects     = map[string]bool{"legacy_wallet": true, "invoice": true}
)

func check(v *string, allowed map[string]bool, field string) error {
	if v != nil && !allowed[*v] {
		return invalid(field)
	}
	return nil
}
