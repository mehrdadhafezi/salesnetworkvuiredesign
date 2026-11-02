package shadow_test

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/finance"
	financemigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/finance/migrations"
	financequery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/finance/query"
	financeshadow "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/finance/shadow"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/legacytest"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/shadow"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/snapshot"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
)

type financeEnv struct {
	*billingEnv
	rebuild *financeshadow.Command
	fin     *db.Runner
}

func setupFinance(t *testing.T) *financeEnv {
	t.Helper()
	e := setupBilling(t)
	if _, err := migrations.Up(context.Background(), e.DSN, financemigrations.Source()); err != nil {
		t.Fatalf("finance migrations: %v", err)
	}
	at, _ := timeutil.ParseRFC3339("2026-11-01T09:00:00Z")
	r := e.Runner(t, finance.Module)
	cmd, err := financeshadow.New(r, timeutil.FixedClock{At: at})
	if err != nil {
		t.Fatal(err)
	}
	return &financeEnv{billingEnv: e, rebuild: cmd, fin: r}
}

func financeSource(c snapshot.Completeness) []snapshot.EntityRows {
	return append(billingSource(c), legacytest.WalletScenario(c)...)
}

// wallets resolves the staged wallet keys through the identity and billing
// published APIs (never a join into their schemas).
func (e *financeEnv) wallets(t *testing.T, b *shadow.Batch) shadow.StagedWallets {
	t.Helper()
	ctx := context.Background()
	users, err := e.ident.ResolveUsers(ctx, shadow.SourceSystem, shadow.FinanceUserKeys(b))
	if err != nil {
		t.Fatal(err)
	}
	src, err := e.billingQ.ResolveInvoices(ctx, shadow.SourceSystem, shadow.SpecInvoice, shadow.FinanceInvoiceKeys(b))
	if err != nil {
		t.Fatal(err)
	}
	inv := map[string]string{}
	for k, v := range src {
		inv[k] = v.InvoiceID
	}
	return shadow.StagedWallets{Batch: b, Users: users, Invoices: inv}
}

func (e *financeEnv) sources(w shadow.StagedWallets) financeshadow.Sources {
	return financeshadow.Sources{Billing: e.billingQ, Amounts: e.billingQ, Payments: e.paymentsQ, Wallets: w}
}

func (e *financeEnv) loaded(t *testing.T, id string, c snapshot.Completeness) (*shadow.Batch, shadow.StagedWallets) {
	t.Helper()
	b := e.prepare(t, id, financeSource(c))
	e.importBillingOnly(t, b)
	e.importPaymentsOnly(t, b)
	return b, e.wallets(t, b)
}

func (e *financeEnv) one(t *testing.T, q string, args ...any) string {
	t.Helper()
	var v string
	if err := e.admin.QueryRow(context.Background(), q, args...).Scan(&v); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	return v
}

var financeTables = []string{"finance.shadow_run", "finance.run_source", "finance.ledger_account", "finance.ledger_entry", "finance.ledger_line",
	"finance.finding", "finance.legacy_balance", "finance.reconciliation"}

func TestFinanceShadowRebuild(t *testing.T) {
	e := setupFinance(t)
	ctx := context.Background()
	_, w := e.loaded(t, "syn-fin-001", snapshot.Complete)
	run, err := e.rebuild.Rebuild(ctx, e.sources(w))
	if err != nil {
		t.Fatal(err)
	}
	if run.Outcome != result.Success || run.Replayed || run.Counts["ledger_entry"] == 0 {
		t.Fatalf("rebuild: %+v", run)
	}
	// Every entry balances per unit; every line carries its entry's unit.
	if n := e.one(t, `SELECT count(*)::text FROM (SELECT e.entry_id FROM finance.ledger_entry e JOIN finance.ledger_line l USING (run_id, entry_id)
		GROUP BY e.entry_id HAVING sum(CASE WHEN l.side = 'DEBIT' THEN l.amount ELSE -l.amount END) <> 0) x`); n != "0" {
		t.Fatalf("%s unbalanced entries", n)
	}
	// Wallet reconciliation against the legacy balances.
	state := func(ref string) string {
		return e.one(t, `SELECT state || ':' || coalesce(difference::text, '-') FROM finance.reconciliation
			WHERE run_id = $1 AND subject_domain = 'legacy_wallet' AND subject_ref = $2`, run.RunID, ref)
	}
	for ref, want := range map[string]string{"1": "MATCH:0", "2": "INCOMPLETE:-", "3": "MATCH:0", "4": "MISMATCH:-974"} {
		if got := state(ref); got != want {
			t.Fatalf("wallet %s: %s, want %s", ref, got, want)
		}
	}
	if r := e.one(t, `SELECT reasons::text FROM finance.reconciliation WHERE run_id = $1 AND subject_ref = '4'`, run.RunID); !strings.Contains(r, "MISSING_LEGACY_TRANSACTION") {
		t.Fatalf("mismatch reasons: %s", r)
	}
	// The proven legacy reversal is linked; the original stays.
	if n := e.one(t, `SELECT count(*)::text FROM finance.ledger_entry r JOIN finance.ledger_entry o ON o.run_id = r.run_id AND o.entry_id = r.reverses_entry_id
		WHERE r.run_id = $1 AND r.business_key LIKE 'legacy_wallet/transaction/11/%' AND o.business_key LIKE 'legacy_wallet/transaction/10/%'`, run.RunID); n != "1" {
		t.Fatal("legacy reversal link")
	}
	// Gateway auto stays GATEWAY_AUTO; nothing is Finance-reviewed.
	if n := e.one(t, `SELECT count(*)::text FROM finance.ledger_entry WHERE run_id = $1 AND basis = 'CASH_SHADOW' AND provenance = 'GATEWAY_AUTO'`, run.RunID); n == "0" {
		t.Fatal("gateway-auto cash keeps its provenance")
	}
	if n := e.one(t, `SELECT count(*)::text FROM finance.ledger_entry WHERE finance_review <> 'NOT_FINANCE_REVIEWED' OR provenance LIKE '%REVIEWED%'`); n != "0" {
		t.Fatal("nothing is Finance-reviewed")
	}
	// Replay: identical inputs are the same run; nothing is duplicated.
	before := e.md5(t, financeTables...)
	again, err := e.rebuild.Rebuild(ctx, e.sources(w))
	if err != nil || !again.Replayed || again.RunID != run.RunID || string(again.FactsSHA) != string(run.FactsSHA) {
		t.Fatalf("replay: %+v %v", again, err)
	}
	if e.md5(t, financeTables...) != before {
		t.Fatal("a replay must write nothing")
	}
	// Historical credit ownership: a current-hierarchy change does not move
	// it (inputs unchanged → same run; holders come from the legacy rows).
	if _, err := e.admin.Exec(ctx, `UPDATE organization.org_assignment SET is_current = NOT COALESCE(is_current, false)`); err != nil {
		t.Fatal(err)
	}
	if h, err := e.rebuild.Rebuild(ctx, e.sources(w)); err != nil || !h.Replayed || e.md5(t, financeTables...) != before {
		t.Fatalf("historical owner must not move: %+v %v", h, err)
	}
}

// The database itself enforces the ledger invariants, even against the
// owner role: append-only, balanced at commit, unique business key,
// single-unit lines, reversal of an earlier entry only.
func TestFinanceLedgerDatabaseInvariants(t *testing.T) {
	e := setupFinance(t)
	ctx := context.Background()
	_, w := e.loaded(t, "syn-fin-002", snapshot.Complete)
	run, err := e.rebuild.Rebuild(ctx, e.sources(w))
	if err != nil {
		t.Fatal(err)
	}
	owner := func(sql string, args ...any) error {
		return e.fin.InTx(ctx, db.TxOptions{}, func(ctx context.Context, tx db.Tx) error {
			_, err := tx.Exec(ctx, sql, args...)
			return err
		})
	}
	for _, sql := range []string{
		`UPDATE finance.ledger_line SET amount = amount + 1`,
		`DELETE FROM finance.ledger_entry`,
		`UPDATE finance.ledger_entry SET business_key = business_key`,
		`TRUNCATE finance.reconciliation`,
		`UPDATE finance.reconciliation SET state = 'MATCH'`,
	} {
		if err := owner(sql); err == nil {
			t.Fatalf("%s must be refused", sql)
		}
	}
	acc := e.one(t, `SELECT account_id::text FROM finance.ledger_account WHERE run_id = $1 ORDER BY account_key LIMIT 1`, run.RunID)
	acc2 := e.one(t, `SELECT account_id::text FROM finance.ledger_account WHERE run_id = $1 ORDER BY account_key DESC LIMIT 1`, run.RunID)
	entry := func(id, key, unit string, rev any) string {
		return `INSERT INTO finance.ledger_entry (run_id, entry_id, entry_seq, business_key, basis, unit, effect_kind, provenance, source_refs,
			time_evidence, reverses_entry_id, reconciliation_status, derivation_version) VALUES ('` + run.RunID + `', '` + id + `', ` +
			"(SELECT max(entry_seq) + 1 FROM finance.ledger_entry WHERE run_id = '" + run.RunID + "'), '" + key + `', 'CASH_SHADOW', '` + unit +
			`', 'TEST', 'TEST', '[{"domain":"t","ref":"1","role":"subject"}]', 'BUSINESS_TIME_ABSENT', ` + rev.(string) + `, 'CONSISTENT', 1)`
	}
	line := func(id string, no int, accID, side string, amount int, unit string) string {
		return `INSERT INTO finance.ledger_line (run_id, entry_id, line_no, account_id, side, amount, unit) VALUES ('` + run.RunID + `', '` + id +
			`', ` + strconv.Itoa(no) + `, '` + accID + `', '` + side + `', ` + strconv.Itoa(amount) + `, '` + unit + `')`
	}
	tx := func(stmts ...string) error {
		return e.fin.InTx(ctx, db.TxOptions{}, func(ctx context.Context, tx db.Tx) error {
			for _, s := range stmts {
				if _, err := tx.Exec(ctx, s); err != nil {
					return err
				}
			}
			return nil
		})
	}
	id1, id2 := "00000000-0000-4000-8000-00000000f001", "00000000-0000-4000-8000-00000000f002"
	k1 := "test/entry/x1/effect/IRT/CASH_SHADOW/v1"
	if err := tx(entry(id1, k1, "IRT", "NULL"), line(id1, 1, acc, "DEBIT", 10, "IRT"), line(id1, 2, acc2, "CREDIT", 9, "IRT")); err == nil {
		t.Fatal("an unbalanced entry must be rejected at commit")
	}
	if err := tx(entry(id1, k1, "IRT", "NULL"), line(id1, 1, acc, "DEBIT", 10, "IRT")); err == nil {
		t.Fatal("a one-sided entry must be rejected")
	}
	if err := tx(entry(id1, k1, "IRT", "NULL"), line(id1, 1, acc, "DEBIT", 10, "IRT"), line(id1, 2, acc2, "CREDIT", 10, "IRR")); err == nil {
		t.Fatal("a cross-unit line must be rejected")
	}
	dupKey := e.one(t, `SELECT business_key FROM finance.ledger_entry WHERE run_id = $1 LIMIT 1`, run.RunID)
	if err := tx(entry(id1, dupKey, "IRT", "NULL"), line(id1, 1, acc, "DEBIT", 10, "IRT"), line(id1, 2, acc2, "CREDIT", 10, "IRT")); err == nil {
		t.Fatal("a duplicate business key must be rejected")
	}
	if err := tx(entry(id1, k1, "IRT", "'"+id1+"'"), line(id1, 1, acc, "DEBIT", 10, "IRT"), line(id1, 2, acc2, "CREDIT", 10, "IRT")); err == nil {
		t.Fatal("self reversal must be rejected")
	}
	if err := tx(entry(id1, k1, "IRT", "'"+id2+"'"), line(id1, 1, acc, "DEBIT", 10, "IRT"), line(id1, 2, acc2, "CREDIT", 10, "IRT"),
		entry(id2, "test/entry/x2/effect/IRT/CASH_SHADOW/v1", "IRT", "'"+id1+"'"), line(id2, 1, acc2, "DEBIT", 10, "IRT"), line(id2, 2, acc, "CREDIT", 10, "IRT")); err == nil {
		t.Fatal("a reversal cycle must be impossible (the reversed entry must already exist)")
	}
	reversed := e.one(t, `SELECT reverses_entry_id::text FROM finance.ledger_entry WHERE run_id = $1 AND reverses_entry_id IS NOT NULL LIMIT 1`, run.RunID)
	if err := tx(entry(id1, k1, "IRT", "'"+reversed+"'"), line(id1, 1, acc, "DEBIT", 100, "IRT"), line(id1, 2, acc2, "CREDIT", 100, "IRT")); err == nil {
		t.Fatal("an entry may be reversed only once")
	}
	if err := tx(entry(id1, k1, "IRT", "NULL"), line(id1, 1, acc, "DEBIT", 10, "IRT"), line(id1, 2, acc2, "CREDIT", 10, "IRT")); err != nil {
		t.Fatalf("a balanced single-unit entry is accepted by the database: %v", err)
	}
}

func (e *financeEnv) query(t *testing.T) *financequery.Service {
	t.Helper()
	at, _ := timeutil.ParseRFC3339("2026-11-01T10:00:00Z")
	q, err := financequery.New(e.fin, timeutil.FixedClock{At: at})
	if err != nil {
		t.Fatal(err)
	}
	return q
}

// Reads are pinned to one run; pages never change totals; detail derives
// from the same run; another run's id is not found; reads write nothing.
func TestFinanceReadParity(t *testing.T) {
	e := setupFinance(t)
	ctx := context.Background()
	_, w := e.loaded(t, "syn-fin-003", snapshot.Complete)
	run, err := e.rebuild.Rebuild(ctx, e.sources(w))
	if err != nil {
		t.Fatal(err)
	}
	q := e.query(t)
	fp := e.md5(t, append(financeTables, ownerTables...)...)
	staged := e.md5(t, "integration.stg_wallet_v1", "integration.stg_wallet_transaction_v1", "integration.import_batch")

	// Ledger: union of pages (limit 1, 2, 200) = full set; summary constant.
	var all []string
	for _, limit := range []int{1, 2, 200} {
		var got []string
		var cursor *string
		var first financequery.LedgerSummary
		for i := 0; ; i++ {
			r, es, sum, next, err := q.Ledger(ctx, financequery.LedgerFilter{}, financequery.Page{Cursor: cursor, Limit: limit})
			if err != nil || r.RunID != run.RunID {
				t.Fatalf("ledger: %v", err)
			}
			if i == 0 {
				first = sum
			} else if !sameLedgerSummary(first, sum) {
				t.Fatal("paging must never change the summary")
			}
			for _, x := range es {
				if *x.Debits.Value != *x.Credits.Value {
					t.Fatal("every listed entry balances")
				}
				got = append(got, x.EntryID)
			}
			if next == nil {
				break
			}
			cursor = next
		}
		if all == nil {
			all = got
		} else if strings.Join(all, ",") != strings.Join(got, ",") {
			t.Fatalf("limit %d: pages differ", limit)
		}
	}
	if int64(len(all)) != run.Counts["ledger_entry"] {
		t.Fatalf("pages = %d, entries = %d", len(all), run.Counts["ledger_entry"])
	}
	seen := map[string]bool{}
	for _, id := range all {
		if seen[id] {
			t.Fatal("duplicate across pages")
		}
		seen[id] = true
	}
	// Detail parity: same run, same lines; another run's id is not found.
	_, x, err := q.EntryByID(ctx, &run.RunID, all[0])
	if err != nil || len(x.Lines) < 2 {
		t.Fatalf("entry detail: %v", err)
	}
	other := "00000000-0000-4000-8000-0000000000ff"
	if _, _, err := q.EntryByID(ctx, &other, all[0]); !errors.Is(err, financequery.ErrNoRun) {
		t.Fatalf("an unknown run: %v", err)
	}
	// Reconciliation: page union = all rows; state counts constant.
	_, rows, sum, _, err := q.Reconciliation(ctx, financequery.ReconFilter{}, financequery.Page{Limit: 200})
	if err != nil {
		t.Fatal(err)
	}
	var n int64
	for _, c := range sum {
		n += c.N
	}
	if n != int64(len(rows)) || n != run.Counts["reconciliation"] {
		t.Fatalf("reconciliation summary %d vs rows %d", n, len(rows))
	}
	var paged int
	var cursor *string
	for {
		_, rs, s2, next, err := q.Reconciliation(ctx, financequery.ReconFilter{}, financequery.Page{Cursor: cursor, Limit: 3})
		if err != nil || len(s2) != len(sum) {
			t.Fatalf("paging changed the summary: %v", err)
		}
		paged += len(rs)
		if next == nil {
			break
		}
		cursor = next
	}
	if paged != len(rows) {
		t.Fatal("paged reconciliation differs")
	}
	// Balances: every wallet row carries its legacy comparison of the run.
	_, bals, _, _, err := q.Balances(ctx, financequery.BalanceFilter{Basis: ptr("WALLET_MIRROR_SHADOW"), Category: ptr("commission_payable")}, financequery.Page{Limit: 200})
	if err != nil || len(bals) == 0 {
		t.Fatalf("balances: %v", err)
	}
	for _, b := range bals {
		if b.Legacy == nil || b.Unit != "IRT" {
			t.Fatalf("wallet balance without its legacy comparison: %+v", b)
		}
	}
	// Accounts: list and detail of the same run; the detail balances are
	// per basis and unit and carry the legacy comparison of that run.
	_, accts, byCat, _, err := q.Accounts(ctx, financequery.AccountFilter{Category: ptr("commission_payable")}, financequery.Page{Limit: 200})
	if err != nil || len(accts) == 0 || byCat["commission_payable"] != int64(len(accts)) {
		t.Fatalf("accounts: %v %v", err, byCat)
	}
	for _, a := range accts {
		if a.HolderKey == nil || a.CustomerID != nil {
			t.Fatalf("wallet account keyed by its historical holder only: %+v", a)
		}
	}
	_, ad, err := q.AccountByID(ctx, &run.RunID, accts[0].AccountID)
	if err != nil || ad.Account.Key != accts[0].Key || len(ad.Balances) == 0 || ad.Balances[0].Legacy == nil {
		t.Fatalf("account detail: %+v %v", ad, err)
	}
	if _, _, err := q.AccountByID(ctx, &run.RunID, "00000000-0000-4000-8000-0000000000aa"); !errors.Is(err, financequery.ErrNotFound) {
		t.Fatalf("unknown account: %v", err)
	}
	// Runs: the run lists itself with its sources; its summary counts every row.
	runs, _, err := q.Runs(ctx, financequery.Page{})
	if err != nil || len(runs) != 1 || runs[0].RunID != run.RunID || len(runs[0].Sources) != 4 {
		t.Fatalf("runs: %+v %v", runs, err)
	}
	rd, err := q.RunByID(ctx, &run.RunID)
	if err != nil || len(rd.Reconciliation) == 0 || len(rd.Findings) == 0 {
		t.Fatalf("run detail: %v", err)
	}
	// Invalid filters are refused.
	for _, f := range []financequery.ReconFilter{{Basis: ptr("FINAL")}, {Unit: ptr("TOMAN")}, {State: ptr("OK")}, {SubjectDomain: ptr("phone")}} {
		if _, _, _, _, err := q.Reconciliation(ctx, f, financequery.Page{}); !errors.Is(err, financequery.ErrInvalid) {
			t.Fatalf("filter %+v: %v", f, err)
		}
	}
	if e.md5(t, append(financeTables, ownerTables...)...) != fp || e.md5(t, "integration.stg_wallet_v1", "integration.stg_wallet_transaction_v1", "integration.import_batch") != staged {
		t.Fatal("reads must not write anything")
	}
}

func ptr(s string) *string { return &s }

func sameLedgerSummary(a, b financequery.LedgerSummary) bool {
	if len(a.Rows) != len(b.Rows) {
		return false
	}
	for i := range a.Rows {
		x, y := a.Rows[i], b.Rows[i]
		if x.Basis != y.Basis || x.Unit != y.Unit || x.Entries != y.Entries || *x.Debits.Value != *y.Debits.Value {
			return false
		}
	}
	return true
}
