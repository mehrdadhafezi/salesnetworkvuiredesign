package shadow_test

import (
	"context"
	"strings"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/commission"
	commissionapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/commission/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/commission/domain"
	commissionmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/commission/migrations"
	commissionshadow "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/commission/shadow"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/legacytest"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/shadow"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/snapshot"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
)

type commissionEnv struct {
	*billingEnv
	cmd *commissionshadow.Command
	cr  *db.Runner
}

func setupCommission(t *testing.T) *commissionEnv {
	t.Helper()
	e := setupBilling(t)
	if _, err := migrations.Up(context.Background(), e.DSN, commissionmigrations.Source()); err != nil {
		t.Fatalf("commission migrations: %v", err)
	}
	at, _ := timeutil.ParseRFC3339("2026-11-03T09:00:00Z")
	r := e.Runner(t, commission.Module)
	cmd, err := commissionshadow.New(r, timeutil.FixedClock{At: at})
	if err != nil {
		t.Fatal(err)
	}
	return &commissionEnv{billingEnv: e, cmd: cmd, cr: r}
}

func commissionSource(c snapshot.Completeness) []snapshot.EntityRows {
	return append(billingSource(c), legacytest.CommissionScenario(c)...)
}

// staged resolves the staged commission keys through the identity and
// billing published APIs (never a join into their schemas).
func (e *commissionEnv) staged(t *testing.T, b *shadow.Batch) shadow.StagedCommission {
	t.Helper()
	ctx := context.Background()
	users, err := e.ident.ResolveUsers(ctx, shadow.SourceSystem, shadow.CommissionUserKeys(b))
	if err != nil {
		t.Fatal(err)
	}
	src, err := e.billingQ.ResolveInvoices(ctx, shadow.SourceSystem, shadow.SpecInvoice, shadow.CommissionInvoiceKeys(b))
	if err != nil {
		t.Fatal(err)
	}
	inv := map[string]string{}
	for k, v := range src {
		inv[k] = v.InvoiceID
	}
	return shadow.StagedCommission{Batch: b, Users: users, Invoices: inv}
}

func (e *commissionEnv) loaded(t *testing.T, id string, ents []snapshot.EntityRows) *shadow.Batch {
	t.Helper()
	b := e.prepare(t, id, ents)
	e.importBillingOnly(t, b)
	return b
}

func (e *commissionEnv) class(t *testing.T, runID, source string) string {
	t.Helper()
	var c string
	if err := e.admin.QueryRow(context.Background(), `SELECT parity_class FROM commission.calculation WHERE run_id = $1 AND source_key = $2`, runID, source).
		Scan(&c); err != nil {
		t.Fatalf("%s: %v", source, err)
	}
	return c
}

var commissionTables = []string{"commission.shadow_run", "commission.run_source", "commission.policy_version", "commission.entitlement_identity",
	"commission.historical_entitlement", "commission.historical_run", "commission.historical_run_item", "commission.historical_posting_log",
	"commission.calculation", "commission.parity_summary", "commission.reconciliation_finding"}

func TestCommissionShadowRun(t *testing.T) {
	e := setupCommission(t)
	ctx := context.Background()
	b := e.loaded(t, "syn-commission-001", commissionSource(snapshot.Complete))
	src := e.staged(t, b)
	walletBefore := e.count(t, `SELECT count(*) FROM integration.stg_wallet_transaction_v1`)
	rep, err := e.cmd.Run(ctx, src, domain.RoundingLegacyHalfAwayTwoDP)
	if err != nil {
		t.Fatal(err)
	}
	// Blocked rows (E4, unproven engine, non-invoice) keep the run PARTIAL.
	if rep.Replayed || rep.Outcome != result.Partial || rep.Rounding != string(domain.RoundingLegacyHalfAwayTwoDP) {
		t.Fatalf("run: %+v", rep)
	}
	want := map[string]string{
		"tx:100": "MATCH", "tx:101": "POLICY_MISMATCH", "tx:102": "AMOUNT_MISMATCH", "tx:105": "MATCH", "tx:106": "ATTRIBUTION_UNKNOWN",
		"tx:108": "DUPLICATE_RISK", "tx:110": "MATCH", "tx:111": "ENGINE_ROUTING_UNKNOWN", "tx:113": "NOT_COMPARABLE",
		"item:1": "MATCH", "item:2": "NOT_COMPARABLE",
	}
	for k, v := range want {
		if got := e.class(t, rep.RunID, k); got != v {
			t.Errorf("%s: class %s, want %s", k, got, v)
		}
	}
	// 1 / 6: the reversed-then-recredited entitlement is ONE calculation and
	// one identity; its first recipient snapshot is kept as an attribute.
	if n := e.count(t, `SELECT count(*) FROM commission.calculation WHERE entitlement_key = 'inv:2002|stage:1|purpose:assessment_fee|facet:seller|ctx:PURPOSE_MATRIX'`); n != 1 {
		t.Fatalf("one entitlement, got %d calculations", n)
	}
	if n := e.count(t, `SELECT count(*) FROM commission.calculation WHERE source_key IN ('tx:103', 'tx:107')`); n != 0 {
		t.Fatal("superseded credits are historical evidence, not separate calculations")
	}
	if n := e.count(t, `SELECT count(*) FROM commission.reconciliation_finding WHERE kind = 'RECIPIENT_MISMATCH'`); n != 1 {
		t.Fatalf("recipient change finding: %d", n)
	}
	// Cross-engine double credit (E1 + E2 on invoice 2002) and the E1 duplicate.
	if n := e.count(t, `SELECT count(*) FROM commission.reconciliation_finding WHERE kind = 'DUPLICATE_RISK' AND severity = 'BLOCKING'`); n < 2 {
		t.Fatalf("duplicate findings: %d", n)
	}
	// 19: the reversal debit references the original; the rewritten-in-place
	// original is preserved raw and reported.
	if n := e.count(t, `SELECT count(*) FROM commission.historical_entitlement WHERE source_key = '104' AND reverses_source_key = '103' AND direction = 'DEBIT'
		AND engine = 'PURPOSE_MATRIX'`); n != 1 {
		t.Fatal("reversal must reference its original")
	}
	if n := e.count(t, `SELECT count(*) FROM commission.reconciliation_finding WHERE kind = 'REWRITTEN_IN_PLACE' AND source_key = '103'`); n != 1 {
		t.Fatal("rewritten-in-place history must be reported")
	}
	// Rejected-then-approved legacy run: both kept.
	if n := e.count(t, `SELECT count(*) FROM commission.historical_run WHERE state = 'APPROVED' AND rejected_then_approved AND rejected_at_raw IS NOT NULL`); n != 1 {
		t.Fatal("rejected-then-approved history must be preserved")
	}
	// 15 / 16: generated / approved / posted stay distinct per item.
	if n := e.count(t, `SELECT count(*) FROM commission.historical_run_item WHERE (source_key = '1' AND posting_state = 'POSTED')
		OR (source_key = '2' AND posting_state = 'GENERATED' AND non_invoice)`); n != 2 {
		t.Fatal("item posting states")
	}
	// 9 / 12 / 8: blocked rows carry no amount and an explicit block.
	for src, block := range map[string]string{"item:2": "NON_INVOICE_KEY_UNAPPROVED", "tx:113": "COMPONENT_ALLOCATION_UNSUPPORTED", "tx:111": "ENGINE_ROUTING_UNKNOWN"} {
		if n := e.count(t, `SELECT count(*) FROM commission.calculation WHERE source_key = $1 AND eligibility = 'BLOCKED' AND amount_hundredths IS NULL
			AND blocks ? $2`, src, block); n != 1 {
			t.Errorf("%s must be BLOCKED by %s", src, block)
		}
	}
	// Exact amounts in hundredths; never float.
	if n := e.count(t, `SELECT count(*) FROM commission.calculation WHERE source_key = 'tx:100' AND amount_hundredths = 2500 AND legacy_amount_hundredths = 2500
		AND base_hundredths = 100000 AND rate_e4 = 25000 AND unit = 'IRT' AND recipient_state = 'SNAPSHOT' AND recipient_key = '5'`); n != 1 {
		t.Fatal("exact amount / snapshot of tx 100")
	}
	// 7: missing attribution snapshot: no recipient filled in.
	if n := e.count(t, `SELECT count(*) FROM commission.calculation WHERE source_key = 'tx:106' AND recipient_state = 'UNKNOWN' AND recipient_key IS NULL
		AND amount_hundredths IS NULL`); n != 1 {
		t.Fatal("unknown attribution must stay unknown")
	}
	// Policy versions: the observed (unversioned) matrix value and the
	// event-time snapshot of the same key are distinct versions; none approved.
	if n := e.count(t, `SELECT count(*) FROM commission.policy_version WHERE policy_key = 'purpose_matrix:product_sale:supervisor'`); n != 2 {
		t.Fatalf("supervisor policy versions: %d", n)
	}
	if n := e.count(t, `SELECT count(*) FROM commission.policy_version WHERE approval_state <> 'NOT_APPROVED'`); n != 0 {
		t.Fatal("no policy may be approved")
	}
	// Parity summary is honest: coverage PARTIAL, totals by unit, missing counted.
	var cov string
	var legacyRows, compared int64
	if err := e.admin.QueryRow(ctx, `SELECT coverage, legacy_rows, compared_rows FROM commission.parity_summary WHERE run_id = $1`, rep.RunID).
		Scan(&cov, &legacyRows, &compared); err != nil || cov != "PARTIAL" || compared >= legacyRows || compared == 0 {
		t.Fatalf("summary: %s %d %d %v", cov, legacyRows, compared, err)
	}
	// 20 / 21: nothing else was written: staging unchanged, no finance or
	// wallet object reachable by the commission owner.
	if e.count(t, `SELECT count(*) FROM integration.stg_wallet_transaction_v1`) != walletBefore {
		t.Fatal("the shadow run must not write staging")
	}
	if n := e.count(t, `SELECT count(*) FROM information_schema.role_table_grants WHERE grantee = 'crm_owner_commission' AND table_schema <> 'commission'`); n != 0 {
		t.Fatalf("commission owner holds %d grants outside its schema", n)
	}
	if n := e.count(t, `SELECT count(*) FROM commission.outbox_event`); n != 0 {
		t.Fatal("no posting request may be appended to the outbox")
	}
	// Append-only: an approved historical snapshot cannot be rewritten.
	if _, err := e.admin.Exec(ctx, `UPDATE commission.historical_run SET approval_raw = 'rejected'`); err == nil || !strings.Contains(err.Error(), "append-only") {
		t.Fatalf("historical run must be immutable: %v", err)
	}

	// 18: replaying the same snapshot is the same run; nothing is written twice.
	before := map[string]int64{}
	for _, tb := range commissionTables {
		before[tb] = e.count(t, `SELECT count(*) FROM `+tb)
	}
	again, err := e.cmd.Run(ctx, src, domain.RoundingLegacyHalfAwayTwoDP)
	if err != nil || !again.Replayed || again.RunID != rep.RunID {
		t.Fatalf("replay: %+v %v", again, err)
	}
	for _, tb := range commissionTables {
		if e.count(t, `SELECT count(*) FROM `+tb) != before[tb] {
			t.Fatalf("%s changed on replay", tb)
		}
	}

	// 14: no rounding policy: a distinct run whose every calculation is BLOCKED.
	none, err := e.cmd.Run(ctx, src, domain.RoundingNone)
	if err != nil || none.Replayed || none.RunID == rep.RunID || none.Rounding != "NONE" {
		t.Fatalf("no-rounding run: %+v %v", none, err)
	}
	if n := e.count(t, `SELECT count(*) FROM commission.calculation WHERE run_id = $1 AND (amount_hundredths IS NOT NULL OR NOT blocks ? 'ROUNDING_POLICY_MISSING')`,
		none.RunID); n != 0 {
		t.Fatalf("%d calculations escaped the missing rounding block", n)
	}
	// 11: policy versions are not duplicated by a later run.
	if n := e.count(t, `SELECT count(*) FROM commission.policy_version WHERE policy_key = 'purpose_matrix:product_sale:supervisor'`); n != 2 {
		t.Fatal("a later run must reuse existing policy versions")
	}
	if _, err := e.cmd.Run(ctx, src, "BANKERS"); err == nil {
		t.Fatal("an unregistered rounding must be refused")
	}
}

// 24: a source missing from the snapshot is explicit uncertainty, never zero.
func TestCommissionShadowMissingSource(t *testing.T) {
	e := setupCommission(t)
	ctx := context.Background()
	ents := legacytest.Without(commissionSource(snapshot.Complete), "legacy.commission_dry_run_item.v1")
	b := e.loaded(t, "syn-commission-002", ents)
	rep, err := e.cmd.Run(ctx, e.staged(t, b), domain.RoundingLegacyHalfAwayTwoDP)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Outcome != result.Partial {
		t.Fatalf("outcome %s", rep.Outcome)
	}
	if n := e.count(t, `SELECT count(*) FROM commission.run_source WHERE run_id = $1 AND source = $2 AND status = 'MISSING' AND rows_read = 0`,
		rep.RunID, commissionapi.SourceDryRunItems); n != 1 {
		t.Fatal("the missing source must be recorded MISSING")
	}
	// The posted E3 credit now names an item that is not staged.
	if n := e.count(t, `SELECT count(*) FROM commission.reconciliation_finding WHERE run_id = $1 AND kind = 'POSTING_LINK_MISMATCH' AND source_key = '112'`,
		rep.RunID); n != 1 {
		t.Fatal("posting link must be reported")
	}
}
