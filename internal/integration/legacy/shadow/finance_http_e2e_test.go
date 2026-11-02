package shadow_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	financehttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/finance/http"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/legacytest"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/snapshot"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/apiserver"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/authstub"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/featureflags"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
)

const financeFlag = "finance.shadow_ledger_read"

func (e *financeEnv) server(t *testing.T) http.Handler {
	t.Helper()
	flags := e.Runner(t, owner.FeatureFlags)
	guard, err := authorization.NewGuard(func(ctx context.Context) (bool, error) {
		var on bool
		err := flags.InTx(ctx, db.TxOptions{ReadOnly: true}, func(ctx context.Context, tx db.Tx) error {
			var err error
			on, err = featureflags.Enabled(ctx, tx, financeFlag)
			return err
		})
		return on, err
	}, authorization.DevSubjects{})
	if err != nil {
		t.Fatal(err)
	}
	cfg := apiserver.DefaultConfig()
	cfg.Environment, cfg.DevAuth = authstub.Development, true
	at, _ := timeutil.ParseRFC3339("2026-11-01T10:00:00Z")
	h, err := apiserver.NewHandlerWithSurfaces(cfg, timeutil.FixedClock{At: at}, slog.New(slog.NewTextHandler(io.Discard, nil)),
		apiserver.Surfaces{Finance: financehttp.New(guard, e.query(t))})
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func (e *financeEnv) setFlag(t *testing.T, on bool) {
	t.Helper()
	at, _ := timeutil.ParseRFC3339("2026-11-01T09:30:00Z")
	if err := e.Runner(t, owner.FeatureFlags).InTx(context.Background(), db.TxOptions{}, func(ctx context.Context, tx db.Tx) error {
		return featureflags.Set(ctx, tx, featureflags.Flag{Key: financeFlag, Enabled: on, UpdatedAt: at})
	}); err != nil {
		t.Fatal(err)
	}
}

var financeGets = []string{"/api/v1/finance/shadow-runs", "/api/v1/finance/accounts", "/api/v1/finance/ledger", "/api/v1/finance/balances",
	"/api/v1/finance/reconciliation"}

// F06 / F07 / no-write proof over HTTP against the real owners.
func TestFinanceHTTPReadOnly(t *testing.T) {
	e := setupFinance(t)
	ctx := context.Background()
	_, w := e.loaded(t, "syn-fin-http", snapshot.Complete)
	h := e.server(t)

	// Flag off (default): BLOCKED / FEATURE_DISABLED, no data.
	for _, p := range financeGets {
		if r := get(t, h, p, "dev-user-001"); r.code != 200 || r.json["outcome"] != "BLOCKED" || r.json["items"] != nil {
			t.Fatalf("%s with the flag off: %d %s", p, r.code, r.body)
		}
	}
	e.setFlag(t, true)
	// No run yet: UNKNOWN, never an empty zero answer.
	if r := get(t, h, "/api/v1/finance/balances", "dev-user-001"); r.json["outcome"] != "UNKNOWN" || r.json["items"] != nil {
		t.Fatalf("no run: %s", r.body)
	}
	run, err := e.rebuild.Rebuild(ctx, e.sources(w))
	if err != nil {
		t.Fatal(err)
	}

	fp := e.md5(t, append(financeTables, ownerTables...)...)
	staged := e.md5(t, "integration.stg_wallet_v1", "integration.stg_wallet_transaction_v1", "integration.import_batch", "integration.import_entity")
	audit := e.md5(t, "audit.audit_event")

	// Every GET answers SHADOW, pinned to the run, repeatedly (F07).
	for i := 0; i < 3; i++ {
		for _, p := range append(financeGets, "/api/v1/finance/shadow-runs/"+run.RunID) {
			r := get(t, h, p, "dev-user-001")
			if r.code != 200 || (r.json["outcome"] != "SUCCESS" && r.json["outcome"] != "PARTIAL") {
				t.Fatalf("%s: %d %s", p, r.code, r.body)
			}
			if !strings.Contains(r.body, `"SHADOW"`) || !strings.Contains(r.body, run.RunID) {
				t.Fatalf("%s must be labelled SHADOW and pinned to the run: %s", p, r.body[:min(300, len(r.body))])
			}
		}
	}
	// Detail parity: an entry of the list is found in the same run.
	led := get(t, h, "/api/v1/finance/ledger?limit=1&run_id="+run.RunID, "dev-user-001")
	id := items(led)[0]["entry_id"].(string)
	if r := get(t, h, "/api/v1/finance/ledger/"+id+"?run_id="+run.RunID, "dev-user-001"); r.code != 200 || !strings.Contains(r.body, id) {
		t.Fatalf("entry detail: %d %s", r.code, r.body)
	}
	if r := get(t, h, "/api/v1/finance/ledger/"+id+"?run_id=00000000-0000-4000-8000-0000000000ff", "dev-user-001"); r.code != 404 {
		t.Fatalf("cross-run drilldown must be not found: %d", r.code)
	}
	// Accrual and cash are separate rows; never a combined total.
	bal := get(t, h, "/api/v1/finance/balances?limit=200", "dev-user-001")
	bases := map[string]bool{}
	for _, it := range items(bal) {
		bases[it["basis"].(string)] = true
	}
	if !bases["ACCRUAL_SHADOW"] || !bases["CASH_SHADOW"] || !bases["WALLET_MIRROR_SHADOW"] {
		t.Fatalf("bases: %v", bases)
	}
	// No combined total / final / official balance key; nothing Finance-reviewed.
	for _, bad := range []string{`"total":`, `"total_balance":`, `"final_balance":`, `"official_balance":`, `"FINANCE_REVIEWED"`} {
		if i := strings.Index(bal.body, bad); i >= 0 {
			t.Fatalf("balances must not carry %q: %s", bad, bal.body[max(0, i-80):i+20])
		}
	}
	// Unknown / phone / invalid filters are 400; no arbitrary query.
	for _, p := range []string{"/api/v1/finance/balances?phone=09000000000", "/api/v1/finance/ledger?sql=select", "/api/v1/finance/ledger?basis=FINAL",
		"/api/v1/finance/reconciliation?unit=TOMAN", "/api/v1/finance/balances?mobile=1", "/api/v1/finance/ledger?cursor=%%%", "/api/v1/finance/accounts?customer_name=x"} {
		if r := get(t, h, p, "dev-user-001"); r.code != 400 {
			t.Fatalf("%s: %d", p, r.code)
		}
	}
	// Mutating methods: 405. Guessed write routes: never 200.
	for _, p := range append(financeGets, "/api/v1/finance/ledger/"+id) {
		for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
			req := httptest.NewRequest(m, p, strings.NewReader(`{}`))
			req.Header.Set(authstub.Header, "dev-user-001")
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != http.StatusMethodNotAllowed {
				t.Fatalf("%s %s: %d", m, p, rec.Code)
			}
		}
	}
	for _, p := range []string{"/api/v1/finance/post", "/api/v1/finance/approve", "/api/v1/finance/reverse", "/api/v1/finance/settle",
		"/api/v1/finance/refund", "/api/v1/finance/credit", "/api/v1/finance/debit", "/api/v1/finance/ledger/" + id + "/reverse",
		"/api/v1/finance/shadow-runs/rebuild", "/api/v1/finance/wallets/1/credit"} {
		for _, m := range []string{http.MethodGet, http.MethodPost} {
			req := httptest.NewRequest(m, p, strings.NewReader(`{}`))
			req.Header.Set(authstub.Header, "dev-user-001")
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code == 200 || rec.Code == 201 || rec.Code == 202 || rec.Code == 204 {
				t.Fatalf("hidden write route %s %s answered %d", m, p, rec.Code)
			}
		}
	}
	// A principal without the read capabilities is denied (F06: view is
	// never granted implicitly; no write capability exists at all).
	for _, p := range financeGets {
		if r := get(t, h, p, "dev-user-002"); r.code != 403 {
			t.Fatalf("%s for a viewer: %d", p, r.code)
		}
	}
	// No GET wrote anything: finance, owners, staging, audit all unchanged;
	// no new run, no wallet transaction, no posting (F07).
	if e.md5(t, append(financeTables, ownerTables...)...) != fp {
		t.Fatal("finance / owner state changed on read")
	}
	if e.md5(t, "integration.stg_wallet_v1", "integration.stg_wallet_transaction_v1", "integration.import_batch", "integration.import_entity") != staged {
		t.Fatal("staged legacy wallet evidence changed on read")
	}
	if e.md5(t, "audit.audit_event") != audit {
		t.Fatal("audit changed on read")
	}
	if e.one(t, `SELECT count(*)::text FROM finance.shadow_run`) != "1" {
		t.Fatal("a GET must never derive a run")
	}
}

// A legacy balance change after a run never mutates that run: a new
// snapshot is a new run, and the old run stays queryable and unchanged.
func TestFinanceHistoricalRunsArePreserved(t *testing.T) {
	e := setupFinance(t)
	ctx := context.Background()
	_, w1 := e.loaded(t, "syn-fin-h1", snapshot.Complete)
	r1, err := e.rebuild.Rebuild(ctx, e.sources(w1))
	if err != nil {
		t.Fatal(err)
	}
	before := e.one(t, `SELECT string_agg(subject_ref || ':' || state || ':' || coalesce(difference::text, '-'), ',' ORDER BY recon_no)
		FROM finance.reconciliation WHERE run_id = $1`, r1.RunID)
	// The legacy wallet 4 balance is corrected in a NEW snapshot.
	ents := financeSource(snapshot.Complete)
	for i := range ents {
		if ents[i].SpecID == "legacy.wallet.v1" {
			ents[i] = walletRows(t)[0]
		}
	}
	b2 := e.stage(t, "syn-fin-h2", ents)
	r2, err := e.rebuild.Rebuild(ctx, e.sources(e.wallets(t, b2)))
	if err != nil || r2.Replayed || r2.RunID == r1.RunID {
		t.Fatalf("a new snapshot is a new run: %+v %v", r2, err)
	}
	if got := e.one(t, `SELECT state FROM finance.reconciliation WHERE run_id = $1 AND subject_domain = 'legacy_wallet' AND subject_ref = '4'`, r2.RunID); got != "MATCH" {
		t.Fatalf("new run: %s", got)
	}
	after := e.one(t, `SELECT string_agg(subject_ref || ':' || state || ':' || coalesce(difference::text, '-'), ',' ORDER BY recon_no)
		FROM finance.reconciliation WHERE run_id = $1`, r1.RunID)
	if after != before || !strings.Contains(before, "4:MISMATCH:-974") {
		t.Fatalf("the prior run must not change: %s vs %s", before, after)
	}
}

// walletRows is the wallet scenario with wallet 4's legacy balance at 25.
func walletRows(t *testing.T) []snapshot.EntityRows {
	t.Helper()
	ws := legacytest.WalletScenario(snapshot.Complete)
	for i := range ws {
		if ws[i].SpecID == "legacy.wallet.v1" {
			ws[i] = legacytest.WalletEntity([]legacytest.Rec{
				legacytest.Wallet("1", "5", "seller", "50.00", "150.00", "100.00"),
				legacytest.Wallet("2", "4", "supervisor", "52.50", "52.50", "0.00"),
				legacytest.Wallet("3", "6", "synthetic_bonus", "10.00", "10.00", "0.00"),
				legacytest.Wallet("4", "5", "biavin", "25.00", "25.00", "0.00"),
			})
		}
	}
	return ws[:1]
}
