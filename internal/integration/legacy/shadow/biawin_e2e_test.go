package shadow_test

import (
	"context"
	"strings"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin"
	biawinapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin/api"
	biawinimports "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin/imports"
	biawinmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin/migrations"
	billingapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment"
	fulfillmentimports "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment/imports"
	fulfillmentmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/legacytest"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/shadow"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/snapshot"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
)

type biawinEnv struct {
	*billingEnv
	bIn *biawinimports.Command
	fIn *fulfillmentimports.Command
	bR  *db.Runner
	fR  *db.Runner
}

func setupBiawin(t *testing.T, extra ...migrations.Source) *biawinEnv {
	t.Helper()
	own := []migrations.Source{biawinmigrations.Source(), fulfillmentmigrations.Source()}
	e := setupBilling(t, append(own, extra...)...)
	for _, s := range own {
		if _, err := migrations.Up(context.Background(), e.DSN, s); err != nil {
			t.Fatalf("biawin / fulfillment migrations: %v", err)
		}
	}
	at, _ := timeutil.ParseRFC3339("2026-11-02T09:00:00Z")
	clock := timeutil.FixedClock{At: at}
	br, fr := e.Runner(t, biawin.Module), e.Runner(t, fulfillment.Module)
	bi, err := biawinimports.New(br, clock)
	if err != nil {
		t.Fatal(err)
	}
	fi, err := fulfillmentimports.New(fr, clock)
	if err != nil {
		t.Fatal(err)
	}
	return &biawinEnv{billingEnv: e, bIn: bi, fIn: fi, bR: br, fR: fr}
}

func biawinSource(c snapshot.Completeness) []snapshot.EntityRows {
	return append(billingSource(c), legacytest.BiawinScenario(c)...)
}

// resolved returns the identity and billing published resolutions of the
// batch's keys (never a schema join).
func (e *biawinEnv) resolved(t *testing.T, b *shadow.Batch) (shadow.Users, shadow.Invoices) {
	t.Helper()
	ctx := context.Background()
	users, err := e.ident.ResolveUsers(ctx, shadow.SourceSystem, shadow.BiawinUserKeys(b))
	if err != nil {
		t.Fatal(err)
	}
	inv := shadow.Invoices{}
	if err := e.billingQ.InvoiceFacts(ctx, func(f billingapi.InvoiceFact) error { inv[f.SourceKey] = f; return nil }); err != nil {
		t.Fatal(err)
	}
	return users, inv
}

func (e *biawinEnv) importBoth(t *testing.T, b *shadow.Batch) (biawinimports.Run, fulfillmentimports.Run) {
	t.Helper()
	ctx := context.Background()
	users, inv := e.resolved(t, b)
	in, ev := shadow.BiawinCommand(b, users, inv)
	br, err := e.bIn.Import(ctx, in, ev)
	if err != nil {
		t.Fatalf("biawin import: %v", err)
	}
	fr, err := e.fIn.Import(ctx, shadow.FulfillmentCommand(b, users, inv))
	if err != nil {
		t.Fatalf("fulfillment import: %v", err)
	}
	return br, fr
}

func (e *biawinEnv) loadedBiawin(t *testing.T, id string) *shadow.Batch {
	t.Helper()
	b := e.prepare(t, id, biawinSource(snapshot.Complete))
	e.importBillingOnly(t, b)
	return b
}

var biawinTables = []string{"biawin.import_run", "biawin.run_source", "biawin.subscription", "biawin.subscription_item", "biawin.item_invoice_link",
	"biawin.subscription_event", "biawin.parity_check", "biawin.parity_row"}
var fulfillmentTables = []string{"fulfillment.import_run", "fulfillment.run_source", "fulfillment.fulfillment_case", "fulfillment.assignment",
	"fulfillment.service_evidence", "fulfillment.case_event"}

func (e *biawinEnv) q1(t *testing.T, q string, args ...any) string {
	t.Helper()
	var v string
	if err := e.admin.QueryRow(context.Background(), q, args...).Scan(&v); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	return v
}

func TestBiawinShadowImport(t *testing.T) {
	e := setupBiawin(t)
	b := e.loadedBiawin(t, "syn-biawin-001")
	br, fr := e.importBoth(t, b)
	// Complete staging: SUCCESS (a MESSAGING_ONLY event is classified, not lost).
	if br.Replayed || fr.Replayed || br.Outcome != result.Success || fr.Outcome != result.Success {
		t.Fatalf("runs: %+v %+v", br, fr)
	}
	if br.Counts["subscription"] != 5 || br.Counts["subscription_item"] != 14 || br.Counts["skipped.MESSAGING_ONLY"] != 1 ||
		br.Counts["subscription_event"] != 4 || br.Counts["item_invoice_link"] != 1 {
		t.Fatalf("biawin counts: %+v", br.Counts)
	}
	if fr.Counts["fulfillment_case"] != 14+2+4+2 || fr.Counts["service_evidence"] != 1 || fr.Counts["case_event"] != 3 {
		t.Fatalf("fulfillment counts: %+v", fr.Counts)
	}
	run := br.RunID
	state := func(key string) string {
		return e.q1(t, `SELECT state_mapped || ':' || coalesce(state_raw, '-') FROM biawin.subscription WHERE run_id = $1 AND source_key = $2`, run, key)
	}
	// Raw kept, mapped only where evidenced; unknown is never ACTIVE.
	for k, want := range map[string]string{"501": "ACTIVE:active", "502": "SUSPENDED:suspended", "503": "UNKNOWN:synthetic_paused", "504": "ACTIVE:active"} {
		if got := state(k); got != want {
			t.Fatalf("subscription %s: %s, want %s", k, got, want)
		}
	}
	// D-35: a COMPLETED execution on a suspended subscription's item, a paid
	// invoice and a Dot reference activate nothing.
	if got := e.q1(t, `SELECT status_mapped FROM fulfillment.fulfillment_case WHERE run_id = $1 AND source_key = '1004' AND kind = 'EXECUTION'`, fr.RunID); got != "COMPLETED" {
		t.Fatalf("execution 1004: %s", got)
	}
	if state("502") != "SUSPENDED:suspended" {
		t.Fatal("a fulfillment completion must not activate a subscription")
	}
	// Multi-item: 12 distinct items under 501, identical titles never merged.
	if n := e.q1(t, `SELECT count(*)::text FROM biawin.subscription_item i JOIN biawin.subscription s USING (run_id, subscription_id)
		WHERE i.run_id = $1 AND s.source_key = '501'`, run); n != "12" {
		t.Fatalf("items of 501: %s", n)
	}
	if n := e.q1(t, `SELECT count(*)::text FROM biawin.subscription_item WHERE run_id = $1 AND service_title_snapshot = 'Synthetic Service A'`, run); n != "2" {
		t.Fatalf("same-title items must stay distinct: %s", n)
	}
	// Customer links: through the source invoice only; unresolved stays explicit.
	if got := e.q1(t, `SELECT customer_link_state || ':' || (customer_id IS NULL)::text FROM biawin.subscription WHERE run_id = $1 AND source_key = '504'`, run); got != "UNKNOWN:true" {
		t.Fatalf("unresolved invoice keeps the customer UNKNOWN: %s", got)
	}
	if got := e.q1(t, `SELECT case_link_state || ':' || coalesce(dot_case_key, '-') FROM biawin.subscription WHERE run_id = $1 AND source_key = '504'`, run); got != "UNKNOWN:77" {
		t.Fatalf("Dot ref kept raw, no case manufactured: %s", got)
	}
	// Historical product title snapshot kept.
	if got := e.q1(t, `SELECT product_title_snapshot || ':' || product_version_state FROM biawin.subscription WHERE run_id = $1 AND source_key = '501'`, run); got != "Synthetic Bundle 501:UNKNOWN" {
		t.Fatalf("product evidence: %s", got)
	}
	// Active parity of the same snapshot: legacy {501, 504, 505} == target.
	p := br.Parity
	if p == nil || p.Outcome != "UNKNOWN" || p.Matched != 3 || p.LegacyCount != 3 || p.TargetCount != 3 || p.Unknown != 1 || p.MissingTarget != 0 || p.ExtraTarget != 0 {
		t.Fatalf("parity: %+v", p)
	}
	// External refs only hashed; no raw tracking / provider reference stored.
	if n := e.q1(t, `SELECT count(*)::text FROM fulfillment.fulfillment_case WHERE run_id = $1 AND (provider_ref_sha256 IS NOT NULL OR tracking_ref_sha256 IS NOT NULL)`, fr.RunID); n != "3" {
		t.Fatalf("hashed external refs: %s", n)
	}
	for _, tbl := range []string{"fulfillment.fulfillment_case", "biawin.subscription"} {
		if n := e.q1(t, `SELECT count(*)::text FROM `+tbl+` t WHERE t::text LIKE '%SYN-EXT%' OR t::text LIKE '%SYN-TRK%'`); n != "0" {
			t.Fatalf("raw external refs leaked into %s", tbl)
		}
	}
	// Replay: same snapshot → same runs, nothing written.
	before := e.md5(t, append(biawinTables, fulfillmentTables...)...)
	b2, f2 := e.importBoth(t, b)
	if !b2.Replayed || b2.RunID != run || !f2.Replayed || f2.RunID != fr.RunID || string(b2.FactsSHA) != string(br.FactsSHA) {
		t.Fatalf("replay: %+v %+v", b2, f2)
	}
	if e.md5(t, append(biawinTables, fulfillmentTables...)...) != before {
		t.Fatal("a replay must write nothing")
	}
	// Append-only even for the owner roles.
	for r, sql := range map[*db.Runner]string{e.bR: `UPDATE biawin.subscription SET state_mapped = 'ACTIVE'`, e.fR: `DELETE FROM fulfillment.case_event`} {
		if err := r.InTx(context.Background(), db.TxOptions{}, func(ctx context.Context, tx db.Tx) error { _, err := tx.Exec(ctx, sql); return err }); err == nil {
			t.Fatalf("%s must be refused", sql)
		}
	}
}

// A later snapshot is a new run; the old run (and its history) is unchanged;
// legacy evidence of the new snapshot against the OLD run shows the drift.
func TestBiawinParityAcrossSnapshots(t *testing.T) {
	e := setupBiawin(t)
	b1 := e.loadedBiawin(t, "syn-biawin-p1")
	r1, _ := e.importBoth(t, b1)
	before := e.q1(t, `SELECT string_agg(source_key || ':' || event_key_raw, ',' ORDER BY source_key) FROM biawin.subscription_event WHERE run_id = $1`, r1.RunID)

	// New snapshot: 501 suspended, a new active 506.
	ents := biawinSource(snapshot.Complete)
	for i := range ents {
		if ents[i].SpecID == "legacy.project_membership.v1" {
			ents[i] = legacytest.BiawinEntity("legacy.project_membership.v1", []legacytest.Rec{
				legacytest.Membership("501", "2001", "suspended", ""), legacytest.Membership("502", "2002", "suspended", ""),
				legacytest.Membership("503", "2004", "synthetic_paused", ""), legacytest.Membership("504", "2999", "active", "77"),
				legacytest.Membership("505", "2003", "active", ""), legacytest.Membership("506", "2001", "active", ""),
			})
		}
	}
	b2 := e.stage(t, "syn-biawin-p2", ents)
	users, inv := e.resolved(t, b2)
	_, ev2 := shadow.BiawinCommand(b2, users, inv)
	// Evidence of the NEW snapshot vs the OLD run.
	p, err := e.bIn.Parity(context.Background(), r1.RunID, ev2)
	if err != nil {
		t.Fatal(err)
	}
	if p.Outcome != "MISMATCH" || p.MissingTarget != 1 || p.Conflict != 1 || p.Matched != 2 {
		t.Fatalf("drift vs the old run: %+v", p.Parity)
	}
	missing := ""
	for _, r := range p.Rows {
		if r.Result == "MISSING_TARGET" {
			missing = r.SourceKey
		}
	}
	if missing != "506" {
		t.Fatalf("missing target names its source key: %q", missing)
	}
	// Extra target: the old run's active row that the new evidence no longer has.
	_, evOnly := shadow.BiawinCommand(b2, users, inv)
	delete(evOnly.Classes, "505")
	p2, _ := e.bIn.Parity(context.Background(), r1.RunID, evOnly)
	if p2.ExtraTarget != 1 {
		t.Fatalf("extra target: %+v", p2.Parity)
	}
	// The new snapshot imports as a new run; the old run is unchanged.
	r2, _ := e.importBoth(t, b2)
	if r2.Replayed || r2.RunID == r1.RunID {
		t.Fatal("a new snapshot is a new run")
	}
	after := e.q1(t, `SELECT string_agg(source_key || ':' || event_key_raw, ',' ORDER BY source_key) FROM biawin.subscription_event WHERE run_id = $1`, r1.RunID)
	if after != before || !strings.Contains(e.q1(t, `SELECT state_mapped FROM biawin.subscription WHERE run_id = $1 AND source_key = '501'`, r1.RunID), "ACTIVE") {
		t.Fatal("the old run and its history must not change")
	}
	_ = biawinapi.StateActive
}
