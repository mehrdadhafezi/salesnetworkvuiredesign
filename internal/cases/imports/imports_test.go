package imports_test

import (
	"context"
	"crypto/sha256"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/imports"
	casesmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pgtest"
)

// All data is SYNTHETIC.
const (
	b1       = "11111111-1111-4111-8111-111111111111"
	b2       = "22222222-2222-4222-8222-222222222222"
	b3       = "33333333-3333-4333-8333-333333333333"
	leadSpec = "synthetic.lead.v1"
	itemSpec = "synthetic.item.v1"
	histSpec = "synthetic.lead_history.v1"
)

type env struct {
	*pgtest.Env
	admin *pgx.Conn
	cmd   *imports.Command
	q     *query.Service
	r     *db.Runner
}

func setup(t *testing.T) *env {
	t.Helper()
	e := pgtest.New(t)
	ctx := context.Background()
	if _, err := migrations.Up(ctx, e.DSN, casesmigrations.Source()); err != nil {
		t.Fatal(err)
	}
	admin, err := pgx.Connect(ctx, e.DSN)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Close(context.Background()) })
	at, _ := timeutil.ParseRFC3339("2026-10-31T09:00:00Z")
	r := e.Runner(t, cases.Module)
	return &env{Env: e, admin: admin, cmd: imports.New(r, timeutil.FixedClock{At: at}), q: query.New(r), r: r}
}

func (e *env) count(t *testing.T, q string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := e.admin.QueryRow(context.Background(), q, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	return n
}

func src(batch, spec, key, content string) api.SourceRef {
	sum := sha256.Sum256([]byte(spec + "/" + key + "/" + content))
	return api.SourceRef{System: "wp", Spec: spec, Key: key, BatchID: batch, RowSHA256: sum[:]}
}

func str(s string) *string { return &s }

func lead(batch, key, status string) api.CaseRecord {
	return api.CaseRecord{Workflow: api.LegacyLead, Source: src(batch, leadSpec, key, status), StateNamespace: "legacy_lead.status",
		StateRaw: str(status), StateMapping: api.StateUnknown, Lead: &api.LeadFacts{StatusRaw: str(status)}}
}

func item(batch, key string) api.CaseRecord {
	return api.CaseRecord{Workflow: api.DistributionItem, Source: src(batch, itemSpec, key, "x"), StateNamespace: "distribution_item.status",
		StateRaw: str("delivered_to_seller"), StateMapping: api.StateKnown}
}

func (e *env) run(t *testing.T, in api.CaseImport) api.ImportReport {
	t.Helper()
	in.SnapshotID, in.SourceSystem = "synthetic-sales-01", "wp"
	rep, err := e.cmd.ImportCases(context.Background(), in)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	return rep
}

func complete(specs ...string) map[string]bool {
	m := map[string]bool{}
	for _, s := range specs {
		m[s] = true
	}
	return m
}

func base(batch string) api.CaseImport {
	return api.CaseImport{BatchID: batch, Complete: complete(leadSpec, itemSpec, histSpec),
		Cases: []api.CaseRecord{lead(batch, "101", "assigned"), lead(batch, "102", "assigned"), item(batch, "501")},
		History: []api.HistoryRecord{
			{Source: src(batch, histSpec, "9001", "a"), Case: api.RowRef{Spec: leadSpec, Key: "101"}, NewRaw: str("assigned"), OccurredRaw: str("2026-01-01 10:00:00"),
				Actor: &api.UserRef{LegacyKey: "7"}},
			{Source: src(batch, histSpec, "9002", "b"), Case: api.RowRef{Spec: leadSpec, Key: "101"}, OldRaw: str("assigned"), NewRaw: str("callback")},
		},
		Aliases: []api.AliasRecord{
			{From: api.RowRef{Spec: itemSpec, Key: "501"}, Related: api.RowRef{Spec: leadSpec, Key: "102"}, Relation: api.ConversionLink, Basis: "item names the lead it was converted into"},
			{From: api.RowRef{Spec: itemSpec, Key: "501"}, Related: api.RowRef{Spec: leadSpec, Key: "999"}, Relation: api.ConversionLink, Basis: "item names a lead"},
		},
		Attributions: []api.AttributionRecord{
			{Case: api.RowRef{Spec: itemSpec, Key: "501"}, Facet: api.OriginalOwner, Resolution: api.Resolved, Subject: &api.UserRef{LegacyKey: "11"},
				PositionRaw: str("supervisor"), Basis: "first holder-assigning custody event", SourceTimeRaw: str("2026-01-01 09:00:00"),
				Evidence: &api.RowRef{Spec: "synthetic.item_log.v1", Key: "1"}},
			{Case: api.RowRef{Spec: leadSpec, Key: "101"}, Facet: api.OriginalOwner, Resolution: api.NotRecoverable, Basis: "no lead assignment log in the source"},
			{Case: api.RowRef{Spec: leadSpec, Key: "101"}, Facet: api.CreditOwner, Resolution: api.Unknown, Basis: "credit is outside Slice 6"},
		},
	}
}

// One source row = one Case; replay creates nothing; links are never merges.
func TestCasesIdempotentAndNeverMerged(t *testing.T) {
	e := setup(t)
	rep := e.run(t, base(b1))
	if rep.Outcome != result.Success {
		t.Fatalf("first import: %+v", rep)
	}
	if e.count(t, `SELECT count(*) FROM cases.work_case`) != 3 || e.count(t, `SELECT count(*) FROM cases.lead`) != 2 ||
		e.count(t, `SELECT count(*) FROM cases.case_source`) != 3 {
		t.Fatal("expected one Case and one source per staged row")
	}
	// Explicit link to a present lead: RELATED (never RESOLVED, OPD-01).
	// Link to an unstaged lead: UNKNOWN. Both Cases stay separate.
	if e.count(t, `SELECT count(*) FROM cases.case_alias WHERE resolution = 'RELATED' AND related_case_id IS NOT NULL AND basis LIKE '%OPD-01%'`) != 1 ||
		e.count(t, `SELECT count(*) FROM cases.case_alias WHERE resolution = 'UNKNOWN' AND related_case_id IS NULL`) != 1 ||
		e.count(t, `SELECT count(*) FROM cases.case_alias WHERE resolution = 'RESOLVED'`) != 0 {
		t.Fatal("alias resolution")
	}
	// No customer reference in the staged source: UNKNOWN, never derived.
	if e.count(t, `SELECT count(*) FROM cases.work_case WHERE customer_link_state = 'UNKNOWN' AND customer_id IS NULL`) != 3 {
		t.Fatal("customer linkage must stay UNKNOWN without an explicit reference")
	}
	fp := e.fingerprint(t)
	r2 := e.run(t, base(b1))
	if !r2.Replay || e.fingerprint(t) != fp {
		t.Fatalf("replay changed state: %+v", r2)
	}
	for tbl, n := range map[string]int64{"work_case": 3, "case_state_history": 2, "attribution_snapshot": 3, "case_alias": 2} {
		if got := e.count(t, `SELECT count(*) FROM cases.`+tbl); got != n {
			t.Fatalf("%s = %d after replay, want %d", tbl, got, n)
		}
	}
}

func (e *env) fingerprint(t *testing.T) string {
	t.Helper()
	var parts []string
	for _, tbl := range []string{"work_case", "case_source", "lead", "case_alias", "case_state_history", "attribution_snapshot"} {
		var s *string
		if err := e.admin.QueryRow(context.Background(), `SELECT md5(string_agg(row_to_json(x)::text, '|' ORDER BY row_to_json(x)::text)) FROM cases.`+tbl+` x`).Scan(&s); err != nil {
			t.Fatal(err)
		}
		if s != nil {
			parts = append(parts, *s)
		}
	}
	return strings.Join(parts, ",")
}

// History and snapshots are append-only: the database rejects UPDATE and
// DELETE; a changed history row is a CONFLICT and is not rewritten; a
// changed attribution is a NEW snapshot, the old one is untouched.
func TestHistoryAndSnapshotsAppendOnly(t *testing.T) {
	e := setup(t)
	e.run(t, base(b1))
	ctx := context.Background()
	for _, q := range []string{
		`UPDATE cases.case_state_history SET new_state_raw = 'x'`,
		`DELETE FROM cases.case_state_history`,
		`UPDATE cases.attribution_snapshot SET resolution = 'UNKNOWN'`,
		`DELETE FROM cases.attribution_snapshot`,
		`TRUNCATE cases.attribution_snapshot`,
	} {
		if _, err := e.admin.Exec(ctx, q); err == nil || !strings.Contains(err.Error(), "append-only") {
			t.Fatalf("%s: err = %v", q, err)
		}
	}
	// The owner role has no UPDATE/DELETE grant either.
	if err := e.r.InTx(ctx, db.TxOptions{}, func(ctx context.Context, tx db.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE cases.attribution_snapshot SET basis = 'x'`)
		return err
	}); err == nil {
		t.Fatal("owner role updated a snapshot")
	}
	before := e.count(t, `SELECT count(*) FROM cases.attribution_snapshot WHERE subject_legacy_key = '11'`)

	in := base(b2)
	in.History[0].Source = src(b2, histSpec, "9001", "changed")
	in.Attributions[0].Subject = &api.UserRef{LegacyKey: "12"}
	in.Attributions[0].Basis = "corrected evidence"
	rep := e.run(t, in)
	if rep.Outcome != result.Conflict || rep.Counts["history_changed_conflict"] != 1 {
		t.Fatalf("changed history row: %+v", rep)
	}
	if e.count(t, `SELECT count(*) FROM cases.case_state_history WHERE source_key = '9001' AND new_state_raw = 'assigned'`) != 1 {
		t.Fatal("history row was rewritten")
	}
	if e.count(t, `SELECT count(*) FROM cases.attribution_snapshot WHERE subject_legacy_key = '11'`) != before ||
		e.count(t, `SELECT count(*) FROM cases.attribution_snapshot WHERE subject_legacy_key = '12'`) != 1 {
		t.Fatal("old snapshot must stay; new evidence must append a new snapshot")
	}
}

// COMPLETE empty source marks absent (never deletes); INCOMPLETE marks nothing.
func TestAbsenceSemantics(t *testing.T) {
	e := setup(t)
	e.run(t, base(b1))
	e.run(t, api.CaseImport{BatchID: b2, Complete: map[string]bool{leadSpec: false, itemSpec: false}})
	if e.count(t, `SELECT count(*) FROM cases.work_case WHERE presence = 'PRESENT'`) != 3 {
		t.Fatal("incomplete source marked absence")
	}
	e.run(t, api.CaseImport{BatchID: b3, Complete: complete(leadSpec, itemSpec)})
	if e.count(t, `SELECT count(*) FROM cases.work_case WHERE presence = 'ABSENT_IN_LATEST'`) != 3 ||
		e.count(t, `SELECT count(*) FROM cases.work_case`) != 3 || e.count(t, `SELECT count(*) FROM cases.case_state_history`) != 2 ||
		e.count(t, `SELECT count(*) FROM cases.attribution_snapshot`) != 3 {
		t.Fatal("complete empty source must mark absent and keep history and snapshots")
	}
}

// An explicit customer reference whose identity is CONFLICT stays CONFLICT;
// it is never upgraded or replaced by another customer.
func TestCustomerLinkStaysUnresolved(t *testing.T) {
	e := setup(t)
	in := base(b1)
	in.Cases[0].Customer = &api.CustomerLink{CustomerID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", State: api.Conflict, Reason: "customer identity is CONFLICT"}
	e.run(t, in)
	if e.count(t, `SELECT count(*) FROM cases.work_case WHERE customer_link_state = 'CONFLICT'`) != 1 ||
		e.count(t, `SELECT count(*) FROM cases.work_case WHERE customer_link_state = 'RESOLVED'`) != 0 {
		t.Fatal("conflicting customer evidence must stay CONFLICT")
	}
	cov, err := e.q.CoverageReport(context.Background())
	if err != nil || cov.LegacyLeadsOnly != 2 || cov.CasesTotal != 3 || cov.SourcelessCases != 0 || cov.CustomerLinks["CONFLICT"] != 1 {
		t.Fatalf("coverage = %+v %v", cov, err)
	}
}
