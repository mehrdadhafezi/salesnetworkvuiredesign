package query

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/finance/domain"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
)

// Account is one ledger account.
type Account struct {
	AccountID, Key, Category, NormalSide, SubjectDomain, SubjectRef string
	CustomerLink, CustomerID, HolderKey, HolderUserID               *string
	WalletTypeRaw, Provenance, Route                                *string
}

// Reconciled is one reconciliation row.
type Reconciled struct {
	No                        int64
	SubjectDomain, SubjectRef string
	AccountID                 *string
	Basis, Unit, Figure       string
	LegacyRaw                 *string
	Legacy                    *int64
	LegacyRepr                string
	Shadow, Difference        *int64
	State                     string
	Reasons                   []string
	Coverage                  string
	Snapshot                  *string
}

// Balance is one account balance in one basis and unit, with the legacy
// figure it is comparable to (if any) from the same run.
type Balance struct {
	Account              Account
	Basis, Unit          string
	Debits, Credits, Net Amount
	Lines                int64
	Legacy               *Reconciled
}

const accountCols = `a.account_id::text, a.account_key, a.category, a.normal_side, a.subject_domain, a.subject_ref, a.customer_link,
	a.customer_id::text, a.holder_key, a.holder_user_id::text, a.wallet_type_raw, a.provenance, a.route`

func scanAccount(dst *Account) []any {
	return []any{&dst.AccountID, &dst.Key, &dst.Category, &dst.NormalSide, &dst.SubjectDomain, &dst.SubjectRef, &dst.CustomerLink,
		&dst.CustomerID, &dst.HolderKey, &dst.HolderUserID, &dst.WalletTypeRaw, &dst.Provenance, &dst.Route}
}

const reconCols = `r.recon_no, r.subject_domain, r.subject_ref, r.account_id::text, r.basis, r.unit, r.legacy_figure, r.legacy_raw, r.legacy_amount,
	r.legacy_repr, r.shadow_amount, r.difference, r.state, r.reasons, r.coverage, r.source_snapshot`

func scanRecon(row interface{ Scan(...any) error }) (Reconciled, error) {
	var r Reconciled
	var reasons []byte
	err := row.Scan(&r.No, &r.SubjectDomain, &r.SubjectRef, &r.AccountID, &r.Basis, &r.Unit, &r.Figure, &r.LegacyRaw, &r.Legacy, &r.LegacyRepr,
		&r.Shadow, &r.Difference, &r.State, &reasons, &r.Coverage, &r.Snapshot)
	if err == nil {
		_ = json.Unmarshal(reasons, &r.Reasons)
	}
	return r, err
}

// BalanceFilter is the allowlisted balance filter.
type BalanceFilter struct {
	RunID                        *string
	Basis, Unit, Category, State *string
}

// BalanceSummary counts balance rows per basis and unit (no money total
// across units, accounts or bases is ever computed).
type BalanceSummary struct {
	Rows []Count // Basis, Unit, State (legacy comparison state or NOT_COMPARABLE), N
}

const balanceFrom = `FROM (
	SELECT l.run_id, l.account_id, e.basis, l.unit,
	       sum(l.amount) FILTER (WHERE l.side = 'DEBIT') AS debits,
	       sum(l.amount) FILTER (WHERE l.side = 'CREDIT') AS credits, count(*) AS lines
	  FROM finance.ledger_line l JOIN finance.ledger_entry e USING (run_id, entry_id)
	 WHERE l.run_id = $1 GROUP BY l.run_id, l.account_id, e.basis, l.unit) b
	JOIN finance.ledger_account a ON a.run_id = b.run_id AND a.account_id = b.account_id
	LEFT JOIN finance.reconciliation r ON r.run_id = b.run_id AND r.account_id = b.account_id AND r.basis = b.basis AND r.unit = b.unit
	WHERE ($2::text IS NULL OR b.basis = $2) AND ($3::text IS NULL OR b.unit = $3) AND ($4::text IS NULL OR a.category = $4)
	  AND ($5::text IS NULL OR coalesce(r.state, 'NOT_COMPARABLE') = $5)`

// Balances lists account balances of one run (derived from lines).
func (s *Service) Balances(ctx context.Context, f BalanceFilter, p Page) (Run, []Balance, BalanceSummary, *string, error) {
	var run Run
	var out []Balance
	var sum BalanceSummary
	limit, err := p.limit()
	if err != nil {
		return run, nil, sum, nil, err
	}
	if err := check(f.Basis, bases, "basis"); err != nil {
		return run, nil, sum, nil, err
	}
	if err := check(f.Unit, units, "unit"); err != nil {
		return run, nil, sum, nil, err
	}
	if err := check(f.Category, categories, "account_category"); err != nil {
		return run, nil, sum, nil, err
	}
	if f.State != nil && !reconStates[*f.State] && *f.State != "NOT_COMPARABLE" {
		return run, nil, sum, nil, invalid("state")
	}
	cur, err := decodeCursor(p.Cursor, 3)
	if err != nil {
		return run, nil, sum, nil, err
	}
	var ak, ab, au *string
	if cur != nil {
		ak, ab, au = &cur[0], &cur[1], &cur[2]
	}
	err = s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		if run, err = s.resolve(ctx, tx, f.RunID); err != nil {
			return err
		}
		args := []any{run.RunID, f.Basis, f.Unit, f.Category, f.State}
		if sum.Rows, err = counts(ctx, tx, `SELECT b.basis, b.unit, coalesce(r.state, 'NOT_COMPARABLE'), '', count(*) `+balanceFrom+`
			GROUP BY 1, 2, 3 ORDER BY 1, 2, 3`, args...); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `SELECT `+accountCols+`, b.basis, b.unit, coalesce(b.debits, 0)::text, coalesce(b.credits, 0)::text, b.lines,
			r.recon_no IS NOT NULL, `+reconColsNullable+` `+balanceFrom+`
			AND ($6::text IS NULL OR (a.account_key, b.basis, b.unit) > ($6, $7, $8))
			ORDER BY a.account_key, b.basis, b.unit LIMIT $9`, append(args, ak, ab, au, limit+1)...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			b, err := scanBalance(rows)
			if err != nil {
				return err
			}
			out = append(out, b)
		}
		return rows.Err()
	})
	if err != nil {
		return run, nil, sum, nil, err
	}
	var next *string
	if len(out) > limit {
		out = out[:limit]
		l := out[limit-1]
		next = encodeCursor(l.Account.Key, l.Basis, l.Unit)
	}
	return run, out, sum, next, nil
}

const reconColsNullable = `r.recon_no, r.subject_domain, r.subject_ref, r.account_id::text, r.basis, r.unit, r.legacy_figure, r.legacy_raw,
	r.legacy_amount, r.legacy_repr, r.shadow_amount, r.difference, r.state, r.reasons, r.coverage, r.source_snapshot`

func scanBalance(rows pgx.Rows) (Balance, error) {
	var b Balance
	var d, c string
	var has bool
	var no *int64
	var sd, sr, rb, ru, fig, repr, st, cov *string
	var acc, lraw, snap *string
	var leg, sh, diff *int64
	var reasons []byte
	dst := append(scanAccount(&b.Account), &b.Basis, &b.Unit, &d, &c, &b.Lines, &has,
		&no, &sd, &sr, &acc, &rb, &ru, &fig, &lraw, &leg, &repr, &sh, &diff, &st, &reasons, &cov, &snap)
	if err := rows.Scan(dst...); err != nil {
		return b, err
	}
	b.Debits, b.Credits = parseSum(d), parseSum(c)
	b.Net = net(b.Debits, b.Credits, b.Account.NormalSide)
	if has {
		r := Reconciled{No: *no, SubjectDomain: *sd, SubjectRef: *sr, AccountID: acc, Basis: *rb, Unit: *ru, Figure: *fig, LegacyRaw: lraw,
			Legacy: leg, LegacyRepr: *repr, Shadow: sh, Difference: diff, State: *st, Coverage: *cov, Snapshot: snap}
		_ = json.Unmarshal(reasons, &r.Reasons)
		b.Legacy = &r
	}
	return b, nil
}

// AccountFilter is the allowlisted account filter.
type AccountFilter struct {
	RunID    *string
	Category *string
}

// Accounts lists the accounts of one run.
func (s *Service) Accounts(ctx context.Context, f AccountFilter, p Page) (Run, []Account, map[string]int64, *string, error) {
	var run Run
	var out []Account
	byCat := map[string]int64{}
	limit, err := p.limit()
	if err != nil {
		return run, nil, nil, nil, err
	}
	if err := check(f.Category, categories, "account_category"); err != nil {
		return run, nil, nil, nil, err
	}
	cur, err := decodeCursor(p.Cursor, 1)
	if err != nil {
		return run, nil, nil, nil, err
	}
	var after *string
	if cur != nil {
		after = &cur[0]
	}
	err = s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		if run, err = s.resolve(ctx, tx, f.RunID); err != nil {
			return err
		}
		cs, err := counts(ctx, tx, `SELECT '', '', category, '', count(*) FROM finance.ledger_account WHERE run_id = $1
			AND ($2::text IS NULL OR category = $2) GROUP BY 3 ORDER BY 3`, run.RunID, f.Category)
		if err != nil {
			return err
		}
		for _, c := range cs {
			byCat[c.State] = c.N
		}
		rows, err := tx.Query(ctx, `SELECT `+accountCols+` FROM finance.ledger_account a WHERE a.run_id = $1
			AND ($2::text IS NULL OR a.category = $2) AND ($3::text IS NULL OR a.account_key > $3) ORDER BY a.account_key LIMIT $4`,
			run.RunID, f.Category, after, limit+1)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var a Account
			if err := rows.Scan(scanAccount(&a)...); err != nil {
				return err
			}
			out = append(out, a)
		}
		return rows.Err()
	})
	if err != nil {
		return run, nil, nil, nil, err
	}
	var next *string
	if len(out) > limit {
		out = out[:limit]
		next = encodeCursor(out[limit-1].Key)
	}
	return run, out, byCat, next, nil
}

// AccountDetail is one account with its balances (all bases and units of
// the same run) and its legacy comparisons.
type AccountDetail struct {
	Account  Account
	Balances []Balance
}

// AccountByID returns one account of one run.
func (s *Service) AccountByID(ctx context.Context, runID *string, accountID string) (Run, AccountDetail, error) {
	var run Run
	var d AccountDetail
	if !uuidPattern.MatchString(accountID) {
		return run, d, invalid("account_id")
	}
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		if run, err = s.resolve(ctx, tx, runID); err != nil {
			return err
		}
		err = tx.QueryRow(ctx, `SELECT `+accountCols+` FROM finance.ledger_account a WHERE a.run_id = $1 AND a.account_id = $2`,
			run.RunID, accountID).Scan(scanAccount(&d.Account)...)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `SELECT `+accountCols+`, b.basis, b.unit, coalesce(b.debits, 0)::text, coalesce(b.credits, 0)::text, b.lines,
			r.recon_no IS NOT NULL, `+reconColsNullable+` `+balanceFrom+` AND a.account_id = $6 ORDER BY b.basis, b.unit`,
			run.RunID, nil, nil, nil, nil, accountID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			b, err := scanBalance(rows)
			if err != nil {
				return err
			}
			d.Balances = append(d.Balances, b)
		}
		return rows.Err()
	})
	return run, d, err
}

// Line is one ledger line.
type Line struct {
	No                              int
	AccountID, AccountKey, Category string
	Side                            string
	Amount                          int64
	Unit                            string
	SourceRef                       *string
}

// Entry is one ledger entry with its lines.
type Entry struct {
	EntryID                        string
	Seq                            int64
	BusinessKey, Basis, Unit       string
	EffectKind, Provenance, Review string
	SourceRefs                     []domain.SourceRef
	EffectiveAt, PostedAt          *time.Time
	SourceTimeRaw                  *string
	TimeEvidence                   string
	ReversesEntryID, ReversedBy    *string
	Status                         string
	Lines                          []Line
	Debits, Credits                Amount
}

// LedgerFilter is the allowlisted ledger filter.
type LedgerFilter struct {
	RunID                                            *string
	Basis, Unit, Status, SourceType, AccountCategory *string
	EffectiveFrom, EffectiveTo                       *time.Time
}

// LedgerSummary counts the filtered entries per basis and unit, with the
// per-unit debit total (always equal to the credit total). Entries with an
// unknown effective time are counted, never dated.
type LedgerSummary struct {
	Rows []LedgerCount
}

// LedgerCount is one basis / unit group.
type LedgerCount struct {
	Basis, Unit        string
	Entries            int64
	Debits             Amount
	UnknownEffectiveAt int64
}

const entryCols = `e.entry_id::text, e.entry_seq, e.business_key, e.basis, e.unit, e.effect_kind, e.provenance, e.finance_review, e.source_refs,
	e.effective_at, e.posted_at, e.source_time_raw, e.time_evidence, e.reverses_entry_id::text,
	(SELECT x.entry_id::text FROM finance.ledger_entry x WHERE x.run_id = e.run_id AND x.reverses_entry_id = e.entry_id), e.reconciliation_status`

const ledgerWhere = `e.run_id = $1 AND ($2::text IS NULL OR e.basis = $2) AND ($3::text IS NULL OR e.unit = $3)
	AND ($4::text IS NULL OR e.reconciliation_status = $4)
	AND ($5::text IS NULL OR e.source_refs @> jsonb_build_array(jsonb_build_object('domain', $5::text, 'role', 'subject')))
	AND ($6::text IS NULL OR EXISTS (SELECT 1 FROM finance.ledger_line l JOIN finance.ledger_account a USING (run_id, account_id)
	     WHERE l.run_id = e.run_id AND l.entry_id = e.entry_id AND a.category = $6))
	AND ($7::timestamptz IS NULL OR e.effective_at >= $7) AND ($8::timestamptz IS NULL OR e.effective_at < $8)`

func scanEntry(row interface{ Scan(...any) error }) (Entry, error) {
	var e Entry
	var refs []byte
	err := row.Scan(&e.EntryID, &e.Seq, &e.BusinessKey, &e.Basis, &e.Unit, &e.EffectKind, &e.Provenance, &e.Review, &refs, &e.EffectiveAt,
		&e.PostedAt, &e.SourceTimeRaw, &e.TimeEvidence, &e.ReversesEntryID, &e.ReversedBy, &e.Status)
	if err == nil {
		_ = json.Unmarshal(refs, &e.SourceRefs)
	}
	return e, err
}

func (s *Service) lines(ctx context.Context, tx db.Tx, runID string, es []Entry) error {
	for i := range es {
		rows, err := tx.Query(ctx, `SELECT l.line_no, l.account_id::text, a.account_key, a.category, l.side, l.amount, l.unit, l.source_ref
			FROM finance.ledger_line l JOIN finance.ledger_account a USING (run_id, account_id)
			WHERE l.run_id = $1 AND l.entry_id = $2 ORDER BY l.line_no`, runID, es[i].EntryID)
		if err != nil {
			return err
		}
		var dr, cr domain.Balance
		for rows.Next() {
			var l Line
			if err := rows.Scan(&l.No, &l.AccountID, &l.AccountKey, &l.Category, &l.Side, &l.Amount, &l.Unit, &l.SourceRef); err != nil {
				rows.Close()
				return err
			}
			if l.Side == domain.DebitSide {
				dr.Debits += l.Amount
			} else {
				cr.Credits += l.Amount
			}
			es[i].Lines = append(es[i].Lines, l)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		d, c := dr.Debits, cr.Credits
		es[i].Debits, es[i].Credits = Amount{Value: &d}, Amount{Value: &c}
	}
	return nil
}

func (f LedgerFilter) validate() error {
	if err := check(f.Basis, bases, "basis"); err != nil {
		return err
	}
	if err := check(f.Unit, units, "unit"); err != nil {
		return err
	}
	if err := check(f.Status, entryStates, "state"); err != nil {
		return err
	}
	if err := check(f.SourceType, sourceDomain, "source_type"); err != nil {
		return err
	}
	if err := check(f.AccountCategory, categories, "account_category"); err != nil {
		return err
	}
	if f.EffectiveFrom != nil && f.EffectiveTo != nil && !f.EffectiveFrom.Before(*f.EffectiveTo) {
		return invalid("effective_from")
	}
	return nil
}

// Ledger lists entries (with lines) of one run.
func (s *Service) Ledger(ctx context.Context, f LedgerFilter, p Page) (Run, []Entry, LedgerSummary, *string, error) {
	var run Run
	var out []Entry
	var sum LedgerSummary
	more := false
	limit, err := p.limit()
	if err != nil {
		return run, nil, sum, nil, err
	}
	if err := f.validate(); err != nil {
		return run, nil, sum, nil, err
	}
	cur, err := decodeCursor(p.Cursor, 1)
	if err != nil {
		return run, nil, sum, nil, err
	}
	var after *int64
	if cur != nil {
		v, err := strconv.ParseInt(cur[0], 10, 64)
		if err != nil {
			return run, nil, sum, nil, invalid("cursor")
		}
		after = &v
	}
	err = s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		if run, err = s.resolve(ctx, tx, f.RunID); err != nil {
			return err
		}
		args := []any{run.RunID, f.Basis, f.Unit, f.Status, f.SourceType, f.AccountCategory, f.EffectiveFrom, f.EffectiveTo}
		srows, err := tx.Query(ctx, `SELECT e.basis, e.unit, count(DISTINCT e.entry_id), coalesce(sum(l.amount) FILTER (WHERE l.side = 'DEBIT'), 0)::text,
			count(DISTINCT e.entry_id) FILTER (WHERE e.effective_at IS NULL)
			FROM finance.ledger_entry e JOIN finance.ledger_line l USING (run_id, entry_id) WHERE `+ledgerWhere+`
			GROUP BY 1, 2 ORDER BY 1, 2`, args...)
		if err != nil {
			return err
		}
		for srows.Next() {
			var c LedgerCount
			var d string
			if err := srows.Scan(&c.Basis, &c.Unit, &c.Entries, &d, &c.UnknownEffectiveAt); err != nil {
				srows.Close()
				return err
			}
			c.Debits = parseSum(d)
			sum.Rows = append(sum.Rows, c)
		}
		srows.Close()
		if err := srows.Err(); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `SELECT `+entryCols+` FROM finance.ledger_entry e WHERE `+ledgerWhere+`
			AND ($9::bigint IS NULL OR e.entry_seq > $9) ORDER BY e.entry_seq LIMIT $10`, append(args, after, limit+1)...)
		if err != nil {
			return err
		}
		for rows.Next() {
			e, err := scanEntry(rows)
			if err != nil {
				rows.Close()
				return err
			}
			out = append(out, e)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		if more = len(out) > limit; more {
			out = out[:limit]
		}
		return s.lines(ctx, tx, run.RunID, out)
	})
	if err != nil {
		return run, nil, sum, nil, err
	}
	var next *string
	if more {
		next = encodeCursor(strconv.FormatInt(out[limit-1].Seq, 10))
	}
	return run, out, sum, next, nil
}

// EntryByID returns one entry of one run with its lines.
func (s *Service) EntryByID(ctx context.Context, runID *string, entryID string) (Run, Entry, error) {
	var run Run
	var e Entry
	if !uuidPattern.MatchString(entryID) {
		return run, e, invalid("entry_id")
	}
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		if run, err = s.resolve(ctx, tx, runID); err != nil {
			return err
		}
		e, err = scanEntry(tx.QueryRow(ctx, `SELECT `+entryCols+` FROM finance.ledger_entry e WHERE e.run_id = $1 AND e.entry_id = $2`,
			run.RunID, entryID))
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		es := []Entry{e}
		if err := s.lines(ctx, tx, run.RunID, es); err != nil {
			return err
		}
		e = es[0]
		return nil
	})
	return run, e, err
}

// ReconFilter is the allowlisted reconciliation filter.
type ReconFilter struct {
	RunID                             *string
	Basis, Unit, State, SubjectDomain *string
}

// Reconciliation lists reconciliation rows of one run with the full-filter
// state counts (independent of the page).
func (s *Service) Reconciliation(ctx context.Context, f ReconFilter, p Page) (Run, []Reconciled, []Count, *string, error) {
	var run Run
	var out []Reconciled
	var sum []Count
	limit, err := p.limit()
	if err != nil {
		return run, nil, nil, nil, err
	}
	if err := check(f.Basis, bases, "basis"); err != nil {
		return run, nil, nil, nil, err
	}
	if f.Unit != nil && !units[*f.Unit] && *f.Unit != "UNKNOWN" {
		return run, nil, nil, nil, invalid("unit")
	}
	if err := check(f.State, reconStates, "state"); err != nil {
		return run, nil, nil, nil, err
	}
	if err := check(f.SubjectDomain, subjects, "subject_type"); err != nil {
		return run, nil, nil, nil, err
	}
	cur, err := decodeCursor(p.Cursor, 1)
	if err != nil {
		return run, nil, nil, nil, err
	}
	var after *int64
	if cur != nil {
		v, err := strconv.ParseInt(cur[0], 10, 64)
		if err != nil {
			return run, nil, nil, nil, invalid("cursor")
		}
		after = &v
	}
	const where = `r.run_id = $1 AND ($2::text IS NULL OR r.basis = $2) AND ($3::text IS NULL OR r.unit = $3)
		AND ($4::text IS NULL OR r.state = $4) AND ($5::text IS NULL OR r.subject_domain = $5)`
	err = s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		if run, err = s.resolve(ctx, tx, f.RunID); err != nil {
			return err
		}
		args := []any{run.RunID, f.Basis, f.Unit, f.State, f.SubjectDomain}
		if sum, err = counts(ctx, tx, `SELECT r.basis, r.unit, r.state, '', count(*) FROM finance.reconciliation r WHERE `+where+`
			GROUP BY 1, 2, 3 ORDER BY 1, 2, 3`, args...); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `SELECT `+reconCols+` FROM finance.reconciliation r WHERE `+where+`
			AND ($6::bigint IS NULL OR r.recon_no > $6) ORDER BY r.recon_no LIMIT $7`, append(args, after, limit+1)...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			r, err := scanRecon(rows)
			if err != nil {
				return err
			}
			out = append(out, r)
		}
		return rows.Err()
	})
	if err != nil {
		return run, nil, nil, nil, err
	}
	var next *string
	if len(out) > limit {
		out = out[:limit]
		next = encodeCursor(strconv.FormatInt(out[limit-1].No, 10))
	}
	return run, out, sum, next, nil
}
