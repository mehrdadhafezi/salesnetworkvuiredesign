// Package shadow is the finance module's OFFLINE shadow ledger rebuild (a
// job command, never reachable from a read path: architecture lint
// read-path-import). It reads billing and payments through their PUBLISHED
// read APIs (each in the owner's own read-only snapshot) and the normalized
// legacy wallet evidence port, derives the SHADOW journal, and writes one
// immutable run with all its facts in one finance transaction.
//
// It never posts, approves, settles, refunds, credits or debits anything,
// never writes another schema, never calls a legacy code path and never
// runs from a GET, page render, session or report request.
package shadow

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	billingapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/finance"
	financeapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/finance/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/finance/domain"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	paymentsapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
)

// Source names.
const (
	SourceBilling   = "billing"
	SourcePayments  = "payments"
	SourceWallets   = "legacy_wallets"
	SourceWalletTxs = "legacy_wallet_transactions"
)

// AllSources lists every source a complete run needs.
func AllSources() []string {
	return []string{SourceBilling, SourcePayments, SourceWallets, SourceWalletTxs}
}

// Sources are the published inputs; a nil source is MISSING.
type Sources struct {
	Billing  billingapi.FactSource
	Amounts  billingapi.AmountEvidenceSource
	Payments paymentsapi.FactSource
	Wallets  financeapi.WalletSource
}

// SourceState is one source's contribution to a run.
type SourceState struct {
	Status     string     `json:"status"`
	Rows       *int64     `json:"rows_read"`
	SnapshotID *string    `json:"snapshot_id"`
	Cutoff     *time.Time `json:"source_cutoff"`
	ReadAt     time.Time  `json:"read_at"`
	Reason     string     `json:"reason"`
}

// Run is the outcome of one rebuild. Replayed reports that identical
// inputs were already derived under this version: the existing run is
// returned and nothing is written.
type Run struct {
	RunID     string
	Replayed  bool
	Outcome   result.Status
	Reasons   []reasoncodes.Code
	Sources   map[string]SourceState
	Counts    map[string]int64
	InputSHA  []byte
	FactsSHA  []byte
	Cutoff    time.Time
	Completed time.Time
}

// Command rebuilds the shadow ledger.
type Command struct {
	runner *db.Runner
	clock  timeutil.Clock
}

// New returns the command over a finance-owned runner.
func New(runner *db.Runner, clock timeutil.Clock) (*Command, error) {
	if runner.Module() != finance.Module {
		return nil, errors.New("finance: shadow rebuild runner must be owned by finance")
	}
	return &Command{runner: runner, clock: clock}, nil
}

func i64(n int) *int64 { v := int64(n); return &v }

// Rebuild runs one shadow derivation. Source reads happen first (each in its
// owner's own read-only snapshot); the finance write is one transaction.
func (c *Command) Rebuild(ctx context.Context, src Sources) (Run, error) {
	started := c.clock.Now().Time().UTC()
	run := Run{Sources: map[string]SourceState{}, Counts: map[string]int64{}}
	in := domain.Sources{BillingStatus: financeapi.SourceMissing, PaymentsStatus: financeapi.SourceMissing}
	now := func() time.Time { return c.clock.Now().Time().UTC() }
	failed := func(err error) string { return "source read failed; nothing derived from it: " + err.Error() }

	// billing: invoice facts + amount evidence (one owner snapshot each).
	switch {
	case src.Billing == nil || src.Amounts == nil:
		run.Sources[SourceBilling] = SourceState{Status: financeapi.SourceMissing, ReadAt: now(), Reason: "billing published API not configured"}
	default:
		var facts []billingapi.InvoiceFact
		err := src.Billing.InvoiceFacts(ctx, func(f billingapi.InvoiceFact) error { facts = append(facts, f); return nil })
		var ev map[string][]billingapi.AmountEvidence
		if err == nil {
			ids := make([]string, 0, len(facts))
			for _, f := range facts {
				if f.Presence == "PRESENT" {
					ids = append(ids, f.InvoiceID)
				}
			}
			ev, err = src.Amounts.InvoiceAmountEvidence(ctx, ids)
		}
		if err != nil {
			run.Sources[SourceBilling] = SourceState{Status: financeapi.SourceUnknown, ReadAt: now(), Reason: failed(err)}
			in.BillingStatus = financeapi.SourceUnknown
			break
		}
		st := SourceState{Status: financeapi.SourceAvailable, Rows: i64(len(facts)), ReadAt: now(),
			Reason: "billing published invoice facts and amount evidence (payments not compared)"}
		for _, f := range facts {
			if f.Presence == "PRESENT" && (!f.ItemsComplete || !f.StagesComplete) {
				st.Status, st.Reason = financeapi.SourcePartial, "invoice item / stage sources incomplete for some invoices"
				break
			}
		}
		st.Cutoff = &st.ReadAt
		run.Sources[SourceBilling] = st
		in.Invoices, in.Evidence, in.BillingStatus = facts, ev, st.Status
	}

	// payments: attempt facts.
	if src.Payments == nil {
		run.Sources[SourcePayments] = SourceState{Status: financeapi.SourceMissing, ReadAt: now(), Reason: "payments published API not configured"}
	} else {
		var atts []paymentsapi.AttemptFact
		if err := src.Payments.AttemptFacts(ctx, func(a paymentsapi.AttemptFact) error { atts = append(atts, a); return nil }); err != nil {
			run.Sources[SourcePayments] = SourceState{Status: financeapi.SourceUnknown, ReadAt: now(), Reason: failed(err)}
			in.PaymentsStatus = financeapi.SourceUnknown
		} else {
			st := SourceState{Status: financeapi.SourceAvailable, Rows: i64(len(atts)), ReadAt: now(),
				Reason: "payments published attempt facts; only ACCEPTED attempts are cash evidence; provenance kept"}
			st.Cutoff = &st.ReadAt
			run.Sources[SourcePayments] = st
			in.Attempts, in.PaymentsStatus = atts, financeapi.SourceAvailable
		}
	}

	// legacy wallets: normalized evidence from staging (never live legacy).
	if src.Wallets == nil {
		for _, s := range []string{SourceWallets, SourceWalletTxs} {
			run.Sources[s] = SourceState{Status: financeapi.SourceMissing, ReadAt: now(), Reason: "no staged legacy wallet snapshot given"}
		}
	} else if ev, err := src.Wallets.WalletEvidence(ctx); err != nil {
		for _, s := range []string{SourceWallets, SourceWalletTxs} {
			run.Sources[s] = SourceState{Status: financeapi.SourceUnknown, ReadAt: now(), Reason: failed(err)}
		}
	} else {
		cut := ev.Cutoff.UTC()
		snap := ev.SnapshotID
		run.Sources[SourceWallets] = SourceState{Status: ev.WalletStatus, Rows: i64(len(ev.Wallets)), SnapshotID: &snap, Cutoff: &cut, ReadAt: now(), Reason: ev.Reason}
		run.Sources[SourceWalletTxs] = SourceState{Status: ev.TransactionStatus, Rows: i64(len(ev.Transactions)), SnapshotID: &snap, Cutoff: &cut,
			ReadAt: now(), Reason: ev.Reason}
		in.Wallets = &ev
	}

	d, err := domain.Derive(in)
	if err != nil {
		return run, fmt.Errorf("finance: derivation refused an entry: %w", err)
	}

	run.Outcome, run.Reasons = result.Success, []reasoncodes.Code{}
	run.Cutoff = started
	for _, name := range AllSources() {
		s := run.Sources[name]
		if s.Cutoff != nil && s.Cutoff.Before(run.Cutoff) {
			run.Cutoff = *s.Cutoff
		}
		switch s.Status {
		case financeapi.SourceAvailable:
		case financeapi.SourcePartial, financeapi.SourceStale:
			run.Outcome = result.Partial
			run.Reasons = append(run.Reasons, reasoncodes.CoverageIncomplete)
		default:
			run.Outcome = result.Partial
			run.Reasons = append(run.Reasons, reasoncodes.SourceUnavailable)
		}
	}
	run.Reasons, _ = reasoncodes.Normalize(run.Reasons)

	run.InputSHA = inputSHA(in, run.Sources)
	// Replay: identical inputs under this derivation version are one run.
	var existing *Run
	if err := c.runner.InTx(ctx, db.TxOptions{ReadOnly: true}, func(ctx context.Context, tx db.Tx) error {
		var r Run
		var outcome string
		var facts []byte
		err := tx.QueryRow(ctx, `SELECT run_id::text, outcome, facts_sha256, source_cutoff, completed_at FROM finance.shadow_run
			WHERE derivation_version = $1 AND input_sha256 = $2`, domain.DerivationVersion, run.InputSHA).Scan(&r.RunID, &outcome, &facts, &r.Cutoff, &r.Completed)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		r.Replayed, r.FactsSHA, r.InputSHA, r.Sources, r.Counts = true, facts, run.InputSHA, run.Sources, map[string]int64{}
		if r.Outcome, err = result.ParseStatus(outcome); err != nil {
			return err
		}
		existing = &r
		return nil
	}); err != nil {
		return run, err
	}
	if existing != nil {
		return *existing, nil
	}

	if run.RunID, err = ids.NewUUID(); err != nil {
		return run, err
	}
	rows := build(run.RunID, d)
	run.FactsSHA = rows.sha
	for _, t := range rows.tables {
		run.Counts[t.name[len("finance."):]] = int64(len(t.rows))
	}
	run.Completed = c.clock.Now().Time().UTC()
	if run.Completed.Before(started) {
		run.Completed = started
	}
	reasons, _ := json.Marshal(run.Reasons)
	counts, _ := json.Marshal(run.Counts)
	err = c.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO finance.shadow_run (run_id, derivation_version, bases, source_cutoff, started_at, completed_at,
			outcome, reasons, fact_counts, input_sha256, facts_sha256) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
			run.RunID, domain.DerivationVersion, domain.Bases(), run.Cutoff, started, run.Completed, run.Outcome.String(), reasons, counts,
			run.InputSHA, run.FactsSHA); err != nil {
			return err
		}
		for _, name := range AllSources() {
			s := run.Sources[name]
			if _, err := tx.Exec(ctx, `INSERT INTO finance.run_source (run_id, source, status, rows_read, snapshot_id, source_cutoff, read_at, reason)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, run.RunID, name, s.Status, s.Rows, s.SnapshotID, s.Cutoff, s.ReadAt, s.Reason); err != nil {
				return err
			}
		}
		for _, t := range rows.tables {
			if err := insert(ctx, tx, run.RunID, t); err != nil {
				return err
			}
		}
		return nil
	})
	return run, err
}

type table struct {
	name string
	cols []string
	rows [][]any
}

func (t *table) add(row ...any) { t.rows = append(t.rows, row) }

type built struct {
	tables []*table
	sha    []byte
}

// derivedID is a deterministic id inside a run.
func derivedID(runID, kind, key string) string {
	h := sha256.Sum256([]byte(runID + "\x00" + kind + "\x00" + key))
	h[6] = (h[6] & 0x0f) | 0x50 // version nibble
	h[8] = (h[8] & 0x3f) | 0x80 // RFC 4122 variant
	x := hex.EncodeToString(h[:16])
	return x[0:8] + "-" + x[8:12] + "-" + x[12:16] + "-" + x[16:20] + "-" + x[20:32]
}

func sp(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}

func ip(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}

// build renders the derivation as insert rows (in insertion order: a
// reversed entry always precedes its reversal) and the facts fingerprint,
// which excludes run-local ids so identical facts hash identically.
func build(runID string, d *domain.Derivation) built {
	h := sha256.New()
	fp := func(parts ...any) {
		b, _ := json.Marshal(parts)
		h.Write(b)
		h.Write([]byte{'\n'})
	}
	acct := &table{name: "finance.ledger_account", cols: []string{"account_id", "account_key", "category", "normal_side", "subject_domain",
		"subject_ref", "customer_link", "customer_id", "holder_key", "holder_user_id", "wallet_type_raw", "provenance", "route"}}
	keys := make([]string, 0, len(d.Journal.Accounts))
	for k := range d.Journal.Accounts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	accID := map[string]string{}
	for _, k := range keys {
		a := d.Journal.Accounts[k]
		accID[k] = derivedID(runID, "account", k)
		acct.add(accID[k], k, a.Category, a.NormalSide(), a.SubjectDomain, a.SubjectRef, sp(a.CustomerLink), sp(a.CustomerID), sp(a.HolderKey),
			sp(a.HolderUserID), sp(a.WalletTypeRaw), sp(a.Provenance), sp(a.Route))
		fp("account", k, a.Category, a.CustomerLink, a.CustomerID, a.HolderKey, a.HolderUserID, a.WalletTypeRaw, a.Provenance, a.Route)
	}
	entry := &table{name: "finance.ledger_entry", cols: []string{"entry_id", "entry_seq", "business_key", "basis", "unit", "effect_kind",
		"provenance", "source_refs", "source_time_raw", "time_evidence", "reverses_entry_id", "reconciliation_status", "derivation_version"}}
	line := &table{name: "finance.ledger_line", cols: []string{"entry_id", "line_no", "account_id", "side", "amount", "unit", "source_ref"}}
	for i, e := range d.Journal.Entries {
		id := derivedID(runID, "entry", e.BusinessKey)
		var rev any
		if e.ReversesKey != nil {
			rev = derivedID(runID, "entry", *e.ReversesKey)
		}
		refs, _ := json.Marshal(e.SourceRefs)
		entry.add(id, i+1, e.BusinessKey, e.Basis, e.Unit.String(), e.EffectKind, e.Provenance, refs, sp(e.SourceTimeRaw), e.TimeEvidence, rev,
			e.Status, domain.DerivationVersion)
		fp("entry", e.BusinessKey, e.Basis, e.Unit.String(), e.EffectKind, e.Provenance, e.SourceRefs, e.SourceTimeRaw, e.TimeEvidence, e.ReversesKey, e.Status)
		for n, l := range e.Lines {
			line.add(id, n+1, accID[l.AccountKey], l.Side, l.Amount, e.Unit.String(), sp(l.SourceRef))
			fp("line", e.BusinessKey, n+1, l.AccountKey, l.Side, l.Amount, l.SourceRef)
		}
	}
	find := &table{name: "finance.finding", cols: []string{"finding_no", "subject_domain", "subject_ref", "basis", "reason", "state", "detail"}}
	for i, f := range d.Findings {
		find.add(i+1, f.SubjectDomain, f.SubjectRef, sp(f.Basis), f.Reason, f.State, f.Detail)
		fp("finding", f.SubjectDomain, f.SubjectRef, f.Basis, f.Reason, f.State, f.Detail)
	}
	lb := &table{name: "finance.legacy_balance", cols: []string{"wallet_key", "holder_key", "holder_user_id", "wallet_type_raw", "category", "unit",
		"balance_raw", "balance", "balance_repr", "total_credit", "total_credit_repr", "total_debit", "total_debit_repr", "snapshot_id"}}
	for _, b := range d.LegacyBalances {
		var unit any
		if b.Unit != nil {
			unit = b.Unit.String()
		}
		lb.add(b.WalletKey, b.HolderKey, sp(b.HolderUserID), b.WalletTypeRaw, b.Category, unit, sp(b.BalanceRaw), ip(b.Balance.Amount), b.Balance.Repr,
			ip(b.TotalCredit.Amount), b.TotalCredit.Repr, ip(b.TotalDebit.Amount), b.TotalDebit.Repr, b.SnapshotID)
		fp("legacy_balance", b.WalletKey, b.HolderKey, b.HolderUserID, b.WalletTypeRaw, b.Category, unit, b.BalanceRaw, b.Balance.Amount,
			b.TotalCredit.Amount, b.TotalDebit.Amount, b.SnapshotID)
	}
	rec := &table{name: "finance.reconciliation", cols: []string{"recon_no", "subject_domain", "subject_ref", "account_id", "basis", "unit",
		"legacy_figure", "legacy_raw", "legacy_amount", "legacy_repr", "shadow_amount", "difference", "state", "reasons", "coverage", "source_snapshot"}}
	for i, r := range d.Reconciliations {
		var acc any
		if r.AccountKey != nil {
			if id, ok := accID[*r.AccountKey]; ok {
				acc = id
			}
		}
		reasons, _ := json.Marshal(r.Reasons)
		repr := r.Legacy.Repr
		if repr == "" {
			repr = "ABSENT"
		}
		rec.add(i+1, r.SubjectDomain, r.SubjectRef, acc, r.Basis, r.Unit, r.Figure, sp(r.LegacyRaw), ip(r.Legacy.Amount), repr, ip(r.Shadow),
			ip(r.Difference), r.State, reasons, r.Coverage, sp(r.Snapshot))
		fp("reconciliation", r.SubjectDomain, r.SubjectRef, r.AccountKey, r.Basis, r.Unit, r.Figure, r.Legacy.Amount, repr, r.Shadow, r.Difference,
			r.State, r.Reasons, r.Coverage, r.Snapshot)
	}
	return built{tables: []*table{acct, entry, line, find, lb, rec}, sha: h.Sum(nil)}
}

// inputSHA fingerprints the normalized inputs that the derivation reads.
func inputSHA(in domain.Sources, sources map[string]SourceState) []byte {
	h := sha256.New()
	enc := json.NewEncoder(h)
	_ = enc.Encode([]any{"derivation", domain.DerivationVersion})
	for _, name := range AllSources() {
		s := sources[name]
		_ = enc.Encode([]any{name, s.Status, s.Rows, s.SnapshotID})
	}
	invs := append([]billingapi.InvoiceFact(nil), in.Invoices...)
	sort.Slice(invs, func(a, b int) bool { return invs[a].InvoiceID < invs[b].InvoiceID })
	for _, f := range invs {
		_ = enc.Encode([]any{"invoice", f.InvoiceID, f.SourceSpec, f.SourceKey, f.Presence, f.CustomerLink, f.CustomerID, f.ItemsComplete,
			f.StagesComplete, in.Evidence[f.InvoiceID]})
	}
	atts := append([]paymentsapi.AttemptFact(nil), in.Attempts...)
	sort.Slice(atts, func(a, b int) bool { return atts[a].AttemptID < atts[b].AttemptID })
	for _, a := range atts {
		_ = enc.Encode([]any{"attempt", a.AttemptID, a.InvoiceID, a.InvoiceLink, a.StateTarget, a.Provenance, a.Amount, a.Unit, a.AmountRepr, a.Presence})
	}
	if in.Wallets != nil {
		_ = enc.Encode([]any{"wallets", in.Wallets.SnapshotID, in.Wallets.WalletStatus, in.Wallets.TransactionStatus})
		for _, w := range in.Wallets.Wallets {
			_ = enc.Encode([]any{"wallet", w})
		}
		for _, t := range in.Wallets.Transactions {
			_ = enc.Encode([]any{"tx", t})
		}
	}
	return h.Sum(nil)
}

// insert writes a table's rows in bounded multi-row statements.
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
