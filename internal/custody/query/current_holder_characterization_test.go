package query_test

// RM-E04N — CurrentHolder security characterization (TEST ONLY, SYNTHETIC data, disposable PostgreSQL).
//
// Every test in this file is CURRENT_BEHAVIOR_CHARACTERIZATION: it pins what the code on main does today, through
// the real import commands and the real published read (custody/query CurrentHolder) and, where relevant, the real
// contact-outcome eligibility (cases/command Eligible / RecordContactOutcome). A passing assertion here is NOT an
// approved ownership policy and NOT a Seller S1 row-scope decision. Behaviour flagged "UNRESOLVED" or
// "POTENTIALLY_UNSAFE" is recorded so approvers can decide on evidence; it is not endorsed.
//
// If an authorized future change alters CurrentHolder (e.g. a parity allow-list or a freshness rule), the matching
// assertion here is expected to fail and must be updated in the same reviewed change, together with
// docs/implementation/rm-e04n-current-holder-characterization.md. Pending security requirements are listed in that
// document as non-gating specifications; none is encoded here as a failing test.
//
// No runtime code, migration, grant, gate or contract is changed by this file. The only schema statements are
// ALTER TABLE ... DROP CONSTRAINT inside the per-test disposable database (pgtest), used solely to simulate a
// hypothetical future parity / derivation value that the current schema rejects; the test first proves the current
// constraint rejects it.

import (
	"context"
	"crypto/sha256"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases"
	casesapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/api"
	casescommand "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/command"
	casesimports "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/imports"
	casesmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/migrations"
	casesquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody/imports"
	custodymigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pgtest"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/rollout"
)

// All values are SYNTHETIC.
const (
	nB1      = "11111111-1111-4111-8111-111111111111"
	nB2      = "22222222-2222-4222-8222-222222222222"
	nItem    = "synthetic.item.v1"
	nLog     = "synthetic.item_log.v1"
	nLead    = "synthetic.lead.v1"
	nUserA   = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	nUserB   = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	nUserC   = "cccccccc-cccc-4ccc-8ccc-cccccccccccc"
	nMissing = "dddddddd-dddd-4ddd-8ddd-dddddddddddd" // no case, no custody record
)

var (
	nT1, _ = timeutil.ParseRFC3339("2026-01-01T09:00:00Z")
	nT2, _ = timeutil.ParseRFC3339("2027-03-01T09:00:00Z") // > 1 year later; no freshness threshold is implied
)

type nEnv struct {
	*pgtest.Env
	admin    *pgx.Conn
	casesR   *db.Runner
	custodyR *db.Runner
	q        *query.Service
	cmd      *casescommand.Service
}

// holderAdapter mirrors cmd/crm-api/writes.go holderAdapter exactly (composition-root adapter; package main cannot
// be imported). It drops Basis, as the production adapter does.
type holderAdapter struct{ q *query.Service }

func (h holderAdapter) Holder(ctx context.Context, caseID string) (casescommand.Holder, error) {
	c, err := h.q.CurrentHolder(ctx, caseID)
	return casescommand.Holder{Known: c.Known, UserID: c.UserID}, err
}

func nSetup(t *testing.T) *nEnv {
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
	cr, r := e.Runner(t, cases.Module), e.Runner(t, custody.Module)
	q := query.New(r)
	cmd, err := casescommand.New(cr, holderAdapter{q}, timeutil.FixedClock{At: nT1})
	if err != nil {
		t.Fatal(err)
	}
	return &nEnv{Env: e, admin: admin, casesR: cr, custodyR: r, q: q, cmd: cmd}
}

func nSum(s string) []byte { h := sha256.Sum256([]byte(s)); return h[:] }

func nStr(s string) *string { return &s }

// nUser is a legacy user reference; id == "" leaves the target user id unresolved.
func nUser(k, id string) *api.UserRef {
	if k == "" {
		return nil
	}
	r := &api.UserRef{LegacyKey: k}
	if id != "" {
		r.UserID = nStr(id)
	}
	return r
}

func nItemCase(batch, key string) casesapi.CaseRecord {
	return casesapi.CaseRecord{Workflow: casesapi.DistributionItem, StateNamespace: "distribution_item.status", StateMapping: casesapi.StateUnknown,
		Source: casesapi.SourceRef{System: "wp", Spec: nItem, Key: key, BatchID: batch, RowSHA256: nSum("item/" + key)}}
}

func nLeadCase(batch, key string) casesapi.CaseRecord {
	return casesapi.CaseRecord{Workflow: casesapi.LegacyLead, StateNamespace: "legacy_lead.status", StateMapping: casesapi.StateUnknown,
		StateRaw: nStr("new"), Lead: &casesapi.LeadFacts{StatusRaw: nStr("new")},
		Source: casesapi.SourceRef{System: "wp", Spec: nLead, Key: key, BatchID: batch, RowSHA256: nSum("lead/" + key)}}
}

func (e *nEnv) importCases(t *testing.T, at timeutil.Instant, batch string, complete map[string]bool, recs ...casesapi.CaseRecord) {
	t.Helper()
	in := casesapi.CaseImport{BatchID: batch, SnapshotID: "synthetic-rm-e04n", SourceSystem: "wp", Complete: complete, Cases: recs}
	if _, err := casesimports.New(e.casesR, timeutil.FixedClock{At: at}).ImportCases(context.Background(), in); err != nil {
		t.Fatalf("cases import: %v", err)
	}
}

func (e *nEnv) importCustody(t *testing.T, at timeutil.Instant, in api.CustodyImport) api.ImportReport {
	t.Helper()
	in.SnapshotID, in.SourceSystem = "synthetic-rm-e04n", "wp"
	rep, err := imports.New(e.custodyR, casesquery.New(e.casesR), timeutil.FixedClock{At: at}).ImportCustody(context.Background(), in)
	if err != nil {
		t.Fatalf("custody import: %v", err)
	}
	return rep
}

func nEv(batch, key, item, kind string, from, to *api.UserRef, at string) api.EventRecord {
	var occurred *string
	if at != "" {
		occurred = &at
	}
	tag := kind
	if from != nil {
		tag += from.LegacyKey
	}
	if to != nil {
		tag += ">" + to.LegacyKey
	}
	return api.EventRecord{Source: api.SourceRef{System: "wp", Spec: nLog, Key: key, BatchID: batch, RowSHA256: nSum(key + tag)},
		Item: api.RowRef{Spec: nItem, Key: item}, Action: "synthetic_" + strings.ToLower(kind), Kind: kind, Context: "DISTRIBUTION",
		From: from, To: to, OccurredRaw: occurred}
}

func nHolder(spec, item string, h *api.UserRef) api.HolderRecord {
	return api.HolderRecord{Case: api.RowRef{Spec: spec, Key: item}, Holder: h}
}

func (e *nEnv) caseID(t *testing.T, spec, key string) string {
	t.Helper()
	var id string
	if err := e.admin.QueryRow(context.Background(), `SELECT case_id::text FROM cases.case_source WHERE source_spec = $1 AND source_key = $2`,
		spec, key).Scan(&id); err != nil {
		t.Fatalf("case %s/%s: %v", spec, key, err)
	}
	return id
}

type nRow struct {
	derivation, parity, presence string
	legacyUser, derivedUser      *string
	refreshedAt                  time.Time
}

func (e *nEnv) assignment(t *testing.T, caseID string) nRow {
	t.Helper()
	var r nRow
	if err := e.admin.QueryRow(context.Background(), `SELECT derivation, parity, presence, legacy_holder_user_id::text, derived_user_id::text,
		refreshed_at FROM custody.assignment WHERE case_id = $1`, caseID).Scan(&r.derivation, &r.parity, &r.presence, &r.legacyUser,
		&r.derivedUser, &r.refreshedAt); err != nil {
		t.Fatalf("assignment %s: %v", caseID, err)
	}
	return r
}

func (e *nEnv) holder(t *testing.T, caseID string) api.CurrentHolder {
	t.Helper()
	h, err := e.q.CurrentHolder(context.Background(), caseID)
	if err != nil {
		t.Fatalf("CurrentHolder(%s): %v", caseID, err)
	}
	return h
}

func deref(p *string) string {
	if p == nil {
		return "<NULL>"
	}
	return *p
}

// ---------------------------------------------------------------- A. parity handling (database-backed)

// TestRME04NParityMatrix — CURRENT_BEHAVIOR_CHARACTERIZATION.
// Every parity class the current schema permits, plus MULTIPLE / RELEASED derivations, unresolved legacy identity
// and key/user-id divergence, produced by the real custody import and read through the real CurrentHolder.
func TestRME04NParityMatrix(t *testing.T) {
	e := nSetup(t)
	items := []string{"701", "702", "703", "704", "705", "706", "707", "708", "709", "710", "711"}
	var recs []casesapi.CaseRecord
	for _, k := range items {
		recs = append(recs, nItemCase(nB1, k))
	}
	e.importCases(t, nT1, nB1, map[string]bool{nItem: true}, recs...)

	a10, b20, c40 := nUser("10", nUserA), nUser("20", nUserB), nUser("40", nUserC)
	rep := e.importCustody(t, nT1, api.CustodyImport{BatchID: nB1, Complete: map[string]bool{nItem: true},
		Holders: []api.HolderRecord{
			nHolder(nItem, "701", a10), nHolder(nItem, "702", b20), nHolder(nItem, "703", c40), nHolder(nItem, "704", a10),
			nHolder(nItem, "705", nil), nHolder(nItem, "706", nil), nHolder(nItem, "707", a10), nHolder(nItem, "708", a10),
			nHolder(nItem, "709", a10), nHolder(nItem, "710", nUser("60", "")), nHolder(nItem, "711", a10),
		},
		Events: []api.EventRecord{
			// 701 clean chain to 10 (user A); legacy 10.
			nEv(nB1, "1", "701", api.HolderAssigned, nil, a10, "2026-01-01 09:00:00"),
			// 702 gap: prior holder 15 never established; legacy 20 (user B).
			nEv(nB1, "2", "702", api.HolderAssigned, nil, nUser("10", nUserA), ""),
			nEv(nB1, "3", "702", api.HolderAssigned, nUser("15", ""), b20, ""),
			// 703 derived 50, legacy 40 (user C).
			nEv(nB1, "4", "703", api.HolderAssigned, nil, nUser("50", ""), ""),
			// 704 two holders at the same time; legacy 10 (user A).
			nEv(nB1, "5", "704", api.HolderAssigned, nil, nUser("10", nUserA), "2026-01-01 09:00:00"),
			nEv(nB1, "6", "704", api.HolderAssigned, nUser("10", nUserA), b20, "2026-01-02 09:00:00"),
			nEv(nB1, "7", "704", api.HolderAssigned, nUser("10", nUserA), nUser("30", nUserC), "2026-01-02 09:00:00"),
			// 705 nothing at all.
			// 706 derived 10 (user A), legacy holder absent.
			nEv(nB1, "8", "706", api.HolderAssigned, nil, a10, ""),
			// 707 no events; legacy 10 (user A).
			// 708 only an UNKNOWN-kind event; legacy 10 (user A).
			nEv(nB1, "9", "708", api.KindUnknown, nil, nil, ""),
			// 709 assigned to 10 then released; legacy still names 10 (user A).
			nEv(nB1, "10", "709", api.HolderAssigned, nil, a10, "2026-01-01 09:00:00"),
			nEv(nB1, "11", "709", api.HolderReleased, a10, nil, "2026-01-02 09:00:00"),
			// 710 chain to 60 with no resolved user id; legacy 60 (no user id).
			nEv(nB1, "12", "710", api.HolderAssigned, nil, nUser("60", ""), ""),
			// 711 same legacy KEY 10 on both sides, but the event resolves key 10 to user B while the holder column resolves it to user A.
			nEv(nB1, "13", "711", api.HolderAssigned, nil, nUser("10", nUserB), ""),
		},
	})
	if rep.Outcome != result.Success {
		t.Fatalf("import outcome %s: %+v", rep.Outcome, rep.Counts)
	}

	type want struct {
		derivation, parity string
		known              bool
		user, basis        string
		disposition        string // for the evidence report; never an approval
	}
	cases := map[string]want{
		"701": {"DERIVED", "MATCH", true, nUserA, "LEGACY_HOLDER_COLUMNS", "corroborated"},
		"702": {"DERIVED", "HISTORY_GAP", true, nUserB, "LEGACY_HOLDER_COLUMNS", "UNRESOLVED: admitted despite history gap"},
		"703": {"DERIVED", "CONFLICT", false, "", "CONTRADICTING_HOLDER_EVIDENCE", "fail closed"},
		"704": {"MULTIPLE", "MULTIPLE_CURRENT", false, "", "CONTRADICTING_HOLDER_EVIDENCE", "fail closed"},
		"705": {"NONE", "NOT_RECOVERABLE", false, "", "NO_RESOLVED_LEGACY_HOLDER", "fail closed"},
		"706": {"DERIVED", "LEGACY_UNKNOWN", false, "", "NO_RESOLVED_LEGACY_HOLDER", "fail closed (derived holder ignored)"},
		"707": {"NONE", "TARGET_UNKNOWN", true, nUserA, "LEGACY_HOLDER_COLUMNS", "UNRESOLVED: legacy column only, no event corroboration"},
		"708": {"UNKNOWN", "TARGET_UNKNOWN", true, nUserA, "LEGACY_HOLDER_COLUMNS", "UNRESOLVED: admitted although derivation is UNKNOWN"},
		"709": {"RELEASED", "CONFLICT", false, "", "CONTRADICTING_HOLDER_EVIDENCE", "fail closed"},
		"710": {"DERIVED", "MATCH", false, "", "NO_RESOLVED_LEGACY_HOLDER", "fail closed (unresolved identity)"},
		"711": {"DERIVED", "MATCH", true, nUserA, "LEGACY_HOLDER_COLUMNS", "POTENTIALLY_UNSAFE: MATCH by legacy key while user ids differ"},
	}
	for item, w := range cases {
		id := e.caseID(t, nItem, item)
		row := e.assignment(t, id)
		if row.derivation != w.derivation || row.parity != w.parity || row.presence != "PRESENT" {
			t.Errorf("%s: fixture produced derivation=%s parity=%s presence=%s, want %s/%s/PRESENT", item, row.derivation, row.parity, row.presence,
				w.derivation, w.parity)
		}
		h := e.holder(t, id)
		if h.Known != w.known || h.UserID != w.user || h.Basis != w.basis {
			t.Errorf("CURRENT_BEHAVIOR %s (%s/%s, %s): CurrentHolder = %+v, want known=%v user=%q basis=%s", item, w.derivation, w.parity,
				w.disposition, h, w.known, w.user, w.basis)
		}
	}
	// 711 identity detail: parity is computed on legacy KEYS; the stored user ids disagree and CurrentHolder returns the
	// holder-column user id without comparing it to the derived one.
	if r := e.assignment(t, e.caseID(t, nItem, "711")); deref(r.legacyUser) != nUserA || deref(r.derivedUser) != nUserB {
		t.Errorf("711: legacy user %s, derived user %s; want %s / %s", deref(r.legacyUser), deref(r.derivedUser), nUserA, nUserB)
	}
	// LEGACY_UNKNOWN, NOT_RECOVERABLE and an unresolved holder id never yield Known.
	// TARGET_UNKNOWN and HISTORY_GAP DO yield Known (current behaviour, NOT an approval).
}

// TestRME04NHypotheticalParityAndDerivationAdmitted — CURRENT_BEHAVIOR_CHARACTERIZATION (POTENTIALLY_UNSAFE).
// The current schema rejects any parity / derivation outside its CHECK lists. CurrentHolder's own logic, however, is
// a deny-list: a value it does not name is admitted. The constraint is dropped ONLY inside this disposable test
// database to show what the read does if a future migration widens the enum without updating CurrentHolder.
func TestRME04NHypotheticalParityAndDerivationAdmitted(t *testing.T) {
	e := nSetup(t)
	ctx := context.Background()
	e.importCases(t, nT1, nB1, map[string]bool{nItem: true}, nItemCase(nB1, "720"), nItemCase(nB1, "721"))
	e.importCustody(t, nT1, api.CustodyImport{BatchID: nB1, Complete: map[string]bool{nItem: true},
		Holders: []api.HolderRecord{nHolder(nItem, "720", nUser("10", nUserA)), nHolder(nItem, "721", nUser("10", nUserA))},
		Events:  []api.EventRecord{nEv(nB1, "1", "720", api.HolderAssigned, nil, nUser("10", nUserA), ""), nEv(nB1, "2", "721", api.HolderAssigned, nil, nUser("10", nUserA), "")}})
	p, d := e.caseID(t, nItem, "720"), e.caseID(t, nItem, "721")

	// 1. What the current database permits: the CHECK constraints reject unknown values.
	if _, err := e.admin.Exec(ctx, `UPDATE custody.assignment SET parity = 'HYPOTHETICAL_FUTURE_PARITY' WHERE case_id = $1`, p); err == nil ||
		!strings.Contains(err.Error(), "assignment_parity") {
		t.Fatalf("current schema must reject an unknown parity; err = %v", err)
	}
	if _, err := e.admin.Exec(ctx, `UPDATE custody.assignment SET derivation = 'HYPOTHETICAL_FUTURE_DERIVATION' WHERE case_id = $1`, d); err == nil ||
		!strings.Contains(err.Error(), "assignment_derivation") {
		t.Fatalf("current schema must reject an unknown derivation; err = %v", err)
	}

	// 2. Simulated future schema (disposable DB only): the read logic admits values it does not know.
	for _, s := range []string{
		`ALTER TABLE custody.assignment DROP CONSTRAINT assignment_parity`,
		`ALTER TABLE custody.assignment DROP CONSTRAINT assignment_derivation`,
		`ALTER TABLE custody.assignment DROP CONSTRAINT assignment_derived_has_holder`,
	} {
		if _, err := e.admin.Exec(ctx, s); err != nil {
			t.Fatalf("%s: %v", s, err)
		}
	}
	if _, err := e.admin.Exec(ctx, `UPDATE custody.assignment SET parity = 'HYPOTHETICAL_FUTURE_PARITY' WHERE case_id = $1`, p); err != nil {
		t.Fatal(err)
	}
	if _, err := e.admin.Exec(ctx, `UPDATE custody.assignment SET derivation = 'HYPOTHETICAL_FUTURE_DERIVATION' WHERE case_id = $1`, d); err != nil {
		t.Fatal(err)
	}
	for name, id := range map[string]string{"unknown parity": p, "unknown derivation": d} {
		if h := e.holder(t, id); !h.Known || h.UserID != nUserA {
			t.Errorf("CURRENT_BEHAVIOR %s: CurrentHolder = %+v; the deny-list was expected to admit it (fail-open by construction)", name, h)
		}
	}
}

// ---------------------------------------------------------------- B. freshness

// TestRME04NFreshnessNotConsidered — CURRENT_BEHAVIOR_CHARACTERIZATION (UNRESOLVED; no threshold is implied).
//
// Three lead cases with the same holder (user A) after batch 1 (T1). Batch 2 (T2, more than a year later) is PARTIAL
// (the lead spec is not COMPLETE):
//   - 805 is not in batch 2 at all (never re-seen; legacy may have moved it);
//   - 806 is in batch 2 with identical evidence (re-confirmed);
//   - 808 is in batch 2 with the same holder but a changed position (projection changed).
//
// Observed: refreshed_at / refreshed_batch_id move only when the projection CHANGES (UpsertAssignment's
// IS DISTINCT FROM guard), so a re-confirmed row (806) and a never re-seen row (805) are indistinguishable; a
// partial batch never marks absence; and CurrentHolder gives the same Known answer for T1 and T2 rows because it
// never reads refreshed_at.
func TestRME04NFreshnessNotConsidered(t *testing.T) {
	e := nSetup(t)
	e.importCases(t, nT1, nB1, map[string]bool{nLead: true}, nLeadCase(nB1, "805"), nLeadCase(nB1, "806"), nLeadCase(nB1, "808"))
	pos := func(item, p string) api.HolderRecord {
		h := nHolder(nLead, item, nUser("10", nUserA))
		h.PositionRaw = nStr(p)
		return h
	}
	e.importCustody(t, nT1, api.CustodyImport{BatchID: nB1, Complete: map[string]bool{nLead: true},
		Holders: []api.HolderRecord{pos("805", "p1"), pos("806", "p1"), pos("808", "p1")}})
	rep := e.importCustody(t, nT2, api.CustodyImport{BatchID: nB2, Complete: map[string]bool{nLead: false},
		Holders: []api.HolderRecord{pos("806", "p1"), pos("808", "p2")}})
	if rep.Outcome != result.Partial {
		t.Fatalf("partial batch outcome = %s, want PARTIAL", rep.Outcome)
	}
	var batch = func(id string) string {
		var b string
		if err := e.admin.QueryRow(context.Background(), `SELECT refreshed_batch_id::text FROM custody.assignment WHERE case_id = $1`, id).Scan(&b); err != nil {
			t.Fatal(err)
		}
		return b
	}
	never, reconfirmed, changed := e.caseID(t, nLead, "805"), e.caseID(t, nLead, "806"), e.caseID(t, nLead, "808")
	rn, rr, rc := e.assignment(t, never), e.assignment(t, reconfirmed), e.assignment(t, changed)
	if !rn.refreshedAt.Equal(nT1.Time()) || !rr.refreshedAt.Equal(nT1.Time()) || !rc.refreshedAt.Equal(nT2.Time()) {
		t.Errorf("CURRENT_BEHAVIOR refreshed_at never=%s reconfirmed=%s changed=%s; want T1/T1/T2 (refreshed_at = last change, not last seen)",
			rn.refreshedAt, rr.refreshedAt, rc.refreshedAt)
	}
	if batch(never) != nB1 || batch(reconfirmed) != nB1 || batch(changed) != nB2 {
		t.Errorf("CURRENT_BEHAVIOR refreshed_batch_id never=%s reconfirmed=%s changed=%s; want b1/b1/b2", batch(never), batch(reconfirmed), batch(changed))
	}
	if rn.presence != "PRESENT" {
		t.Fatalf("a PARTIAL batch must not mark absence; presence = %s", rn.presence)
	}
	hn, hr, hc := e.holder(t, never), e.holder(t, reconfirmed), e.holder(t, changed)
	if hn != hr || hn != hc || !hn.Known || hn.UserID != nUserA {
		t.Errorf("CURRENT_BEHAVIOR: never=%+v reconfirmed=%+v changed=%+v; expected identical Known answers (refreshed_at ignored)", hn, hr, hc)
	}
}

// ---------------------------------------------------------------- C. presence

// TestRME04NPresence — CURRENT_BEHAVIOR_CHARACTERIZATION.
// Custody presence is honoured; a missing custody record is reported as such. Reassignment by a COMPLETE batch
// moves the holder on the next read.
func TestRME04NPresence(t *testing.T) {
	e := nSetup(t)
	e.importCases(t, nT1, nB1, map[string]bool{nItem: true}, nItemCase(nB1, "730"), nItemCase(nB1, "731"), nItemCase(nB1, "732"))
	e.importCustody(t, nT1, api.CustodyImport{BatchID: nB1, Complete: map[string]bool{nItem: true},
		Holders: []api.HolderRecord{nHolder(nItem, "730", nUser("10", nUserA)), nHolder(nItem, "731", nUser("10", nUserA))}})
	// 732 is a case with no custody record at all.
	gone, moved, none := e.caseID(t, nItem, "730"), e.caseID(t, nItem, "731"), e.caseID(t, nItem, "732")

	// Batch 2 is COMPLETE for the item spec: 730 is no longer listed, 731 moved to user B.
	e.importCustody(t, nT2, api.CustodyImport{BatchID: nB2, Complete: map[string]bool{nItem: true},
		Holders: []api.HolderRecord{nHolder(nItem, "731", nUser("20", nUserB))}})

	if r := e.assignment(t, gone); r.presence != "ABSENT_IN_LATEST" {
		t.Fatalf("730 presence = %s", r.presence)
	}
	if h := e.holder(t, gone); h.Known || h.Basis != "CASE_ABSENT_IN_LATEST" {
		t.Errorf("CURRENT_BEHAVIOR ABSENT_IN_LATEST: %+v", h)
	}
	if h := e.holder(t, moved); !h.Known || h.UserID != nUserB {
		t.Errorf("CURRENT_BEHAVIOR reassignment by complete batch: %+v, want user B", h)
	}
	if h := e.holder(t, none); h.Known || h.Basis != "NO_CUSTODY_RECORD" {
		t.Errorf("CURRENT_BEHAVIOR case without custody record: %+v", h)
	}
	if h := e.holder(t, nMissing); h.Known || h.Basis != "NO_CUSTODY_RECORD" {
		t.Errorf("CURRENT_BEHAVIOR nonexistent case id: %+v", h)
	}
	// Missing case id and case-without-custody are indistinguishable at this layer (same Basis).
	if _, err := e.q.CurrentHolder(context.Background(), "not-a-uuid"); err == nil {
		t.Error("malformed case id must be rejected as invalid input")
	}
}

// ---------------------------------------------------------------- D. cross-module consistency

// TestRME04NCaseAbsentButCustodyPresent — CURRENT_BEHAVIOR_CHARACTERIZATION (read/write discrepancy).
// The cases import marks a lead ABSENT_IN_LATEST (work_case and case_source together); the custody projection was not
// re-run. CurrentHolder still names the holder because it never reads cases presence. The contact-outcome write guard
// and its eligibility read do read cases presence and refuse. A Seller S1 read built on CurrentHolder alone would
// show a case the write path treats as stale. The write guard is not changed here.
func TestRME04NCaseAbsentButCustodyPresent(t *testing.T) {
	e := nSetup(t)
	ctx := context.Background()
	e.importCases(t, nT1, nB1, map[string]bool{nLead: true}, nLeadCase(nB1, "803"), nLeadCase(nB1, "809"))
	e.importCustody(t, nT1, api.CustodyImport{BatchID: nB1, Complete: map[string]bool{nLead: true},
		Holders: []api.HolderRecord{nHolder(nLead, "803", nUser("10", nUserA)), nHolder(nLead, "809", nUser("10", nUserA))}})
	// Cases batch 2 (COMPLETE) omits 803. No custody batch follows.
	e.importCases(t, nT2, nB2, map[string]bool{nLead: true}, nLeadCase(nB2, "809"))
	id := e.caseID(t, nLead, "803")

	var wc, src string
	if err := e.admin.QueryRow(ctx, `SELECT c.presence, s.presence FROM cases.work_case c JOIN cases.case_source s USING (case_id) WHERE c.case_id = $1`,
		id).Scan(&wc, &src); err != nil {
		t.Fatal(err)
	}
	if wc != "ABSENT_IN_LATEST" || src != "ABSENT_IN_LATEST" {
		t.Fatalf("fixture: work_case=%s case_source=%s", wc, src)
	}
	if r := e.assignment(t, id); r.presence != "PRESENT" {
		t.Fatalf("fixture: custody presence = %s", r.presence)
	}
	if h := e.holder(t, id); !h.Known || h.UserID != nUserA {
		t.Errorf("CURRENT_BEHAVIOR read helper: %+v (expected Known: CurrentHolder does not consult cases presence)", h)
	}
	ok, code, err := e.cmd.Eligible(ctx, id, casescommand.Actor{Principal: nUserA, UserID: nUserA})
	if err != nil || ok || code != reasoncodes.StaleState {
		t.Errorf("CURRENT_BEHAVIOR Eligible: ok=%v code=%s err=%v, want false/STALE_STATE", ok, code, err)
	}
	res, err := e.cmd.RecordContactOutcome(ctx, rollout.Shadow, casescommand.Command{CaseID: id, Outcome: casesapi.OutcomeNoAnswer, ExpectedVersion: 1,
		IdempotencyKey: "rm-e04n-803", CorrelationID: "rm-e04n", Actor: casescommand.Actor{Principal: nUserA, UserID: nUserA}})
	if err != nil || res.Status != result.Blocked || len(res.Reasons) != 1 || res.Reasons[0] != reasoncodes.StaleState {
		t.Errorf("CURRENT_BEHAVIOR write guard: %+v err=%v, want BLOCKED/STALE_STATE", res, err)
	}
}

// TestRME04NEligibilityDistinguishesRefusals — CURRENT_BEHAVIOR_CHARACTERIZATION (404-parity evidence).
// The contact-outcome eligibility read (the source of allowed_actions) returns a different signal for a missing case,
// another holder's case, a case with no known holder, an absent case and a non-lead case. Reusing it unchanged behind
// a Seller own-read would let a caller distinguish "exists, not mine" from "does not exist". Nothing is changed here.
func TestRME04NEligibilityDistinguishesRefusals(t *testing.T) {
	e := nSetup(t)
	ctx := context.Background()
	e.importCases(t, nT1, nB1, map[string]bool{nLead: true, nItem: true}, nLeadCase(nB1, "801"), nLeadCase(nB1, "802"), nLeadCase(nB1, "803"),
		nLeadCase(nB1, "804"), nItemCase(nB1, "740"))
	e.importCustody(t, nT1, api.CustodyImport{BatchID: nB1, Complete: map[string]bool{nLead: true, nItem: true},
		Holders: []api.HolderRecord{nHolder(nLead, "801", nUser("10", nUserA)), nHolder(nLead, "802", nil), nHolder(nLead, "803", nUser("10", nUserA)),
			nHolder(nItem, "740", nUser("10", nUserA))}})
	e.importCases(t, nT2, nB2, map[string]bool{nLead: true, nItem: true}, nLeadCase(nB2, "801"), nLeadCase(nB2, "802"), nLeadCase(nB2, "804"),
		nItemCase(nB2, "740")) // 803 absent in cases
	a := casescommand.Actor{Principal: nUserA, UserID: nUserA}
	b := casescommand.Actor{Principal: nUserB, UserID: nUserB}

	// 801 is a lead with no custody events: its parity is TARGET_UNKNOWN, yet it is the actor's own, eligible case.
	if r := e.assignment(t, e.caseID(t, nLead, "801")); r.derivation != "NONE" || r.parity != "TARGET_UNKNOWN" {
		t.Fatalf("lead fixture: %s/%s", r.derivation, r.parity)
	}
	type obs struct {
		ok       bool
		code     reasoncodes.Code
		notFound bool
	}
	check := func(name, id string, actor casescommand.Actor, want obs) {
		ok, code, err := e.cmd.Eligible(ctx, id, actor)
		got := obs{ok: ok, code: code, notFound: errors.Is(err, casescommand.ErrNotFound)}
		if err != nil && !got.notFound {
			t.Fatalf("%s: %v", name, err)
		}
		if got != want {
			t.Errorf("CURRENT_BEHAVIOR %s: %+v, want %+v", name, got, want)
		}
	}
	check("own lead (TARGET_UNKNOWN)", e.caseID(t, nLead, "801"), a, obs{ok: true})
	check("other holder's lead", e.caseID(t, nLead, "801"), b, obs{code: reasoncodes.RowScopeDenied})
	check("lead without legacy holder", e.caseID(t, nLead, "802"), a, obs{code: reasoncodes.DependencyUnknown})
	check("lead absent in cases", e.caseID(t, nLead, "803"), a, obs{code: reasoncodes.StaleState})
	check("lead without custody record", e.caseID(t, nLead, "804"), a, obs{code: reasoncodes.DependencyUnknown})
	check("non-lead case held by actor", e.caseID(t, nItem, "740"), a, obs{code: reasoncodes.PolicyUnresolved})
	check("nonexistent case id", nMissing, a, obs{code: reasoncodes.ContractInvalid, notFound: true})
	check("actor without resolved user id", e.caseID(t, nLead, "801"), casescommand.Actor{Principal: nUserA}, obs{code: reasoncodes.RowScopeDenied})
}

// ---------------------------------------------------------------- E. identity space and organization boundary

// TestRME04NCustodyStoresUnverifiedUserIDs — CURRENT_BEHAVIOR_CHARACTERIZATION.
// custody.assignment.legacy_holder_user_id has no foreign key to identity (cross-schema references are forbidden) and
// the import stores any well-formed uuid supplied by the integration. Whether that uuid names an existing, active
// identity user is not verified by custody; CurrentHolder returns it as-is. The schema carries no organization key on
// cases or custody tables.
func TestRME04NCustodyStoresUnverifiedUserIDs(t *testing.T) {
	e := nSetup(t)
	ctx := context.Background()
	unknownUser := "eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee" // no identity schema exists in this database at all
	e.importCases(t, nT1, nB1, map[string]bool{nLead: true}, nLeadCase(nB1, "850"))
	e.importCustody(t, nT1, api.CustodyImport{BatchID: nB1, Complete: map[string]bool{nLead: true},
		Holders: []api.HolderRecord{nHolder(nLead, "850", nUser("90", unknownUser))}})
	if h := e.holder(t, e.caseID(t, nLead, "850")); !h.Known || h.UserID != unknownUser {
		t.Errorf("CURRENT_BEHAVIOR: %+v", h)
	}
	var fks int64
	if err := e.admin.QueryRow(ctx, `SELECT count(*) FROM information_schema.table_constraints WHERE table_schema = 'custody'
		AND table_name = 'assignment' AND constraint_type = 'FOREIGN KEY'`).Scan(&fks); err != nil || fks != 0 {
		t.Errorf("custody.assignment foreign keys = %d (%v); characterization expected none", fks, err)
	}
	var orgCols int64
	if err := e.admin.QueryRow(ctx, `SELECT count(*) FROM information_schema.columns WHERE table_schema IN ('custody', 'cases')
		AND (column_name LIKE '%organization%' OR column_name LIKE '%org_id%' OR column_name LIKE '%tenant%')`).Scan(&orgCols); err != nil || orgCols != 0 {
		t.Errorf("organization-like columns in cases/custody = %d (%v); characterization expected none", orgCols, err)
	}
	var holderIdx int64
	if err := e.admin.QueryRow(ctx, `SELECT count(*) FROM pg_indexes WHERE schemaname = 'custody' AND tablename = 'assignment'
		AND indexdef LIKE '%legacy_holder_user_id%'`).Scan(&holderIdx); err != nil || holderIdx != 0 {
		t.Errorf("indexes on legacy_holder_user_id = %d (%v); characterization expected none (no set-based held-by read exists)", holderIdx, err)
	}
}
