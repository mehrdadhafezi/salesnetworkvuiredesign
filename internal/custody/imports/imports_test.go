package imports_test

import (
	"context"
	"crypto/sha256"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases"
	casesapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/api"
	casesimports "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/imports"
	casesmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/migrations"
	casesquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody/imports"
	custodymigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody/query"
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
	itemSpec = "synthetic.item.v1"
	logSpec  = "synthetic.item_log.v1"
)

type env struct {
	*pgtest.Env
	admin *pgx.Conn
	cases *casesimports.Command
	cmd   *imports.Command
	q     *query.Service
	r     *db.Runner
}

func setup(t *testing.T) *env {
	t.Helper()
	e := pgtest.New(t)
	ctx := context.Background()
	for _, s := range []migrations.Source{casesmigrations.Source(), custodymigrations.Source()} {
		if _, err := migrations.Up(ctx, e.DSN, s); err != nil {
			t.Fatal(err)
		}
	}
	admin, err := pgx.Connect(ctx, e.DSN)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Close(context.Background()) })
	at, _ := timeutil.ParseRFC3339("2026-10-31T09:00:00Z")
	clock := timeutil.FixedClock{At: at}
	cr, r := e.Runner(t, cases.Module), e.Runner(t, custody.Module)
	return &env{Env: e, admin: admin, cases: casesimports.New(cr, clock), cmd: imports.New(r, casesquery.New(cr), clock), q: query.New(r), r: r}
}

func (e *env) count(t *testing.T, q string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := e.admin.QueryRow(context.Background(), q, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	return n
}

func sum(s string) []byte  { h := sha256.Sum256([]byte(s)); return h[:] }
func str(s string) *string { return &s }
func user(k string) *api.UserRef {
	if k == "" {
		return nil
	}
	return &api.UserRef{LegacyKey: k}
}

// stageCases imports one Case per item key through the cases command.
func (e *env) stageCases(t *testing.T, batch string, keys ...string) {
	t.Helper()
	in := casesapi.CaseImport{BatchID: batch, SnapshotID: "synthetic-sales-01", SourceSystem: "wp", Complete: map[string]bool{itemSpec: true}}
	for _, k := range keys {
		in.Cases = append(in.Cases, casesapi.CaseRecord{Workflow: casesapi.DistributionItem, StateNamespace: "distribution_item.status",
			StateMapping: casesapi.StateUnknown, Source: casesapi.SourceRef{System: "wp", Spec: itemSpec, Key: k, BatchID: batch, RowSHA256: sum(k)}})
	}
	if _, err := e.cases.ImportCases(context.Background(), in); err != nil {
		t.Fatal(err)
	}
}

func ev(batch, key, item, kind, from, to, at string) api.EventRecord {
	var t *string
	if at != "" {
		t = &at
	}
	return api.EventRecord{Source: api.SourceRef{System: "wp", Spec: logSpec, Key: key, BatchID: batch, RowSHA256: sum(key + kind + from + to)},
		Item: api.RowRef{Spec: itemSpec, Key: item}, Action: "synthetic_" + strings.ToLower(kind), Kind: kind, Context: "DISTRIBUTION",
		From: user(from), To: user(to), OccurredRaw: t}
}

func holder(item, k string) api.HolderRecord {
	return api.HolderRecord{Case: api.RowRef{Spec: itemSpec, Key: item}, Holder: user(k)}
}

func (e *env) run(t *testing.T, in api.CustodyImport) api.ImportReport {
	t.Helper()
	in.SnapshotID, in.SourceSystem = "synthetic-sales-01", "wp"
	rep, err := e.cmd.ImportCustody(context.Background(), in)
	if err != nil {
		t.Fatalf("custody import: %v", err)
	}
	return rep
}

func (e *env) parity(t *testing.T, item string) string {
	t.Helper()
	var p string
	if err := e.admin.QueryRow(context.Background(), `SELECT a.parity FROM custody.assignment a JOIN cases.case_source s USING (case_id)
		WHERE s.source_key = $1`, item).Scan(&p); err != nil {
		t.Fatal(err)
	}
	return p
}

func scenario(batch string) api.CustodyImport {
	return api.CustodyImport{BatchID: batch, Complete: map[string]bool{itemSpec: true},
		Holders: []api.HolderRecord{holder("501", "30"), holder("502", "30"), holder("503", "20"), holder("504", "40"), holder("505", "")},
		Events: []api.EventRecord{
			// 501: clean chain 10 → 20 → 30, legacy says 30: MATCH.
			ev(batch, "1", "501", api.HolderAssigned, "", "10", "2026-01-01 09:00:00"),
			ev(batch, "2", "501", api.HolderAssigned, "10", "20", "2026-01-02 09:00:00"),
			ev(batch, "3", "501", api.HolderAssigned, "20", "30", "2026-01-03 09:00:00"),
			// 502: two holders at the same recorded time: MULTIPLE_CURRENT.
			ev(batch, "4", "502", api.HolderAssigned, "", "10", "2026-01-01 09:00:00"),
			ev(batch, "5", "502", api.HolderAssigned, "10", "20", "2026-01-02 09:00:00"),
			ev(batch, "6", "502", api.HolderAssigned, "10", "30", "2026-01-02 09:00:00"),
			// 503: missing transition (prior holder 15 never established): HISTORY_GAP.
			ev(batch, "7", "503", api.HolderAssigned, "", "10", ""),
			ev(batch, "8", "503", api.HolderAssigned, "15", "20", ""),
			// 504: derived 50, legacy 40: CONFLICT (never auto-picked).
			ev(batch, "9", "504", api.HolderAssigned, "", "50", ""),
			// 505: no events, no legacy holder: NOT_RECOVERABLE.
		},
	}
}

func TestCustodyParityClasses(t *testing.T) {
	e := setup(t)
	e.stageCases(t, b1, "501", "502", "503", "504", "505")
	rep := e.run(t, scenario(b1))
	if rep.Outcome != result.Success {
		t.Fatalf("import: %+v", rep)
	}
	for item, want := range map[string]string{"501": "MATCH", "502": "MULTIPLE_CURRENT", "503": "HISTORY_GAP", "504": "CONFLICT", "505": "NOT_RECOVERABLE"} {
		if got := e.parity(t, item); got != want {
			t.Errorf("%s parity = %s, want %s", item, got, want)
		}
	}
	// Multiple current holders are never resolved to one.
	if e.count(t, `SELECT count(*) FROM custody.assignment WHERE derivation = 'MULTIPLE' AND derived_legacy_key IS NULL`) != 1 {
		t.Fatal("MULTIPLE must not pick a holder")
	}
	// No event was synthesised for the gap.
	if e.count(t, `SELECT count(*) FROM custody.custody_event`) != 9 {
		t.Fatal("events synthesised")
	}
	p, err := e.q.ParityReport(context.Background())
	if err != nil || p.FullParity || p.ByParity["MATCH"] != 1 || p.FindingsByKind["SOURCE_CURRENT_MISMATCH"] != 1 ||
		p.FindingsByKind["MISSING_EVENT_HISTORY"] != 1 || p.FindingsByKind["MULTIPLE_CURRENT"] != 1 {
		t.Fatalf("parity report = %+v %v", p, err)
	}
}

// Events are append-only at the database and ownership level; replay
// creates nothing; a changed legacy row is a CONFLICT, not a rewrite.
func TestEventsAppendOnlyAndReplay(t *testing.T) {
	e := setup(t)
	e.stageCases(t, b1, "501", "502", "503", "504", "505")
	e.run(t, scenario(b1))
	ctx := context.Background()
	for _, q := range []string{`UPDATE custody.custody_event SET to_legacy_key = '99'`, `DELETE FROM custody.custody_event`, `TRUNCATE custody.custody_event CASCADE`} {
		if _, err := e.admin.Exec(ctx, q); err == nil || !strings.Contains(err.Error(), "append-only") {
			t.Fatalf("%s: err = %v", q, err)
		}
	}
	if err := e.r.InTx(ctx, db.TxOptions{}, func(ctx context.Context, tx db.Tx) error {
		_, err := tx.Exec(ctx, `DELETE FROM custody.custody_event`)
		return err
	}); err == nil {
		t.Fatal("owner role deleted an event")
	}
	r2 := e.run(t, scenario(b1))
	if !r2.Replay || e.count(t, `SELECT count(*) FROM custody.custody_event`) != 9 || e.count(t, `SELECT count(*) FROM custody.assignment`) != 5 {
		t.Fatalf("replay: %+v", r2)
	}
	in := scenario(b2)
	in.Events[0] = ev(b2, "1", "501", api.HolderAssigned, "", "11", "2026-01-01 09:00:00")
	if rep := e.run(t, in); rep.Outcome != result.Conflict || rep.Counts["events_changed_conflict"] != 1 {
		t.Fatalf("changed event: %+v", rep)
	}
	if e.count(t, `SELECT count(*) FROM custody.custody_event WHERE source_key = '1' AND to_legacy_key = '10'`) != 1 {
		t.Fatal("event rewritten")
	}
}

// COMPLETE empty holder source marks projections absent and keeps every
// event; an INCOMPLETE one marks nothing.
func TestAbsenceKeepsEvents(t *testing.T) {
	e := setup(t)
	e.stageCases(t, b1, "501", "502", "503", "504", "505")
	e.run(t, scenario(b1))
	e.run(t, api.CustodyImport{BatchID: b2, Complete: map[string]bool{itemSpec: false}})
	if e.count(t, `SELECT count(*) FROM custody.assignment WHERE presence = 'PRESENT'`) != 5 {
		t.Fatal("incomplete source marked absence")
	}
	e.run(t, api.CustodyImport{BatchID: b3, Complete: map[string]bool{itemSpec: true}})
	if e.count(t, `SELECT count(*) FROM custody.assignment WHERE presence = 'ABSENT_IN_LATEST'`) != 5 ||
		e.count(t, `SELECT count(*) FROM custody.custody_event`) != 9 {
		t.Fatal("complete empty source must mark absent and keep events")
	}
}
