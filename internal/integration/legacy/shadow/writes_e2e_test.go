package shadow_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	biawinmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin/migrations"
	billingmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases"
	casescommand "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/command"
	financemigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/finance/migrations"
	fulfillmentmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/legacytest"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/shadow"
	lwb "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/writeback"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/writeback"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/hrrequest"
	paymentsmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/rollout"
)

// writesEnv composes the Slice 11 command services, the write-back worker and
// a synthetic legacy master over the real module owners.
type writesEnv struct {
	*salesEnv
	clock   timeutil.FixedClock
	cases   *casescommand.Service
	hr      *hrrequest.Service
	worker  *writeback.Worker
	legacy  *legacytest.SyntheticLegacy
	users   map[string]string // legacy key → target user id
	caseIDs map[string]string // legacy lead id → case id
}

// holders adapts the custody published read (composition root role).
type holders struct{ e *salesEnv }

func (h holders) Holder(ctx context.Context, caseID string) (casescommand.Holder, error) {
	c, err := h.e.custodyQ.CurrentHolder(ctx, caseID)
	return casescommand.Holder{Known: c.Known, UserID: c.UserID}, err
}

func setupWrites(t *testing.T) *writesEnv {
	t.Helper()
	ctx := context.Background()
	// Financial and Biawin owners exist (empty) so non-mutation is provable.
	own := []migrations.Source{billingmigrations.Source(), paymentsmigrations.Source(), financemigrations.Source(),
		biawinmigrations.Source(), fulfillmentmigrations.Source()}
	se := setupSales(t, own...)
	for _, src := range own {
		if _, err := migrations.Up(ctx, se.DSN, src); err != nil {
			t.Fatal(err)
		}
	}
	b := se.stage(t, "syn-writes", legacytest.WritesScenario())
	se.importAll(t, b)
	if rc, rs, _ := se.importSales(t, b); rc == 0 || rs == 0 {
		t.Fatal("sales import")
	}
	at, _ := timeutil.ParseRFC3339("2026-11-02T09:00:00Z")
	clock := timeutil.FixedClock{At: at}
	cs, err := casescommand.New(se.Runner(t, cases.Module), holders{se}, clock)
	if err != nil {
		t.Fatal(err)
	}
	hr, err := hrrequest.New(se.Runner(t, organization.Module), clock)
	if err != nil {
		t.Fatal(err)
	}
	legacy := legacytest.NewSyntheticLegacy()
	legacy.Leads["101"] = &legacytest.SyntheticLead{SellerLegacyUserID: "6"}
	legacy.Leads["102"] = &legacytest.SyntheticLead{SellerLegacyUserID: "7", Archived: true}
	conn, err := lwb.NewConnector(legacy, se.ident)
	if err != nil {
		t.Fatal(err)
	}
	w, err := writeback.New(se.Runner(t, integration.Module), clock, conn)
	if err != nil {
		t.Fatal(err)
	}
	keys := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}
	users, err := se.ident.ResolveUsers(ctx, shadow.SourceSystem, keys)
	if err != nil || len(users) != 10 {
		t.Fatalf("users: %v %v", users, err)
	}
	srcs, err := se.casesQ.ResolveSources(ctx, shadow.SourceSystem, shadow.SpecLead, []string{"101", "102", "103"})
	if err != nil || len(srcs) != 3 {
		t.Fatalf("cases: %v %v", srcs, err)
	}
	ids := map[string]string{}
	for k, v := range srcs {
		ids[k] = v.CaseID
	}
	return &writesEnv{salesEnv: se, clock: clock, cases: cs, hr: hr, worker: w, legacy: legacy, users: users, caseIDs: ids}
}

func (e *writesEnv) actor(key string) casescommand.Actor {
	return casescommand.Actor{Principal: "synthetic:" + key, UserID: e.users[key]}
}

func (e *writesEnv) hrActor(key string) hrrequest.Actor {
	return hrrequest.Actor{Principal: "synthetic:" + key, UserID: e.users[key]}
}

func (e *writesEnv) caseVersion(t *testing.T, lead string) int64 {
	t.Helper()
	return e.count(t, `SELECT version FROM cases.work_case WHERE case_id = $1`, e.caseIDs[lead])
}

func (e *writesEnv) drain(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	for _, m := range []string{"cases", "organization"} {
		r := e.Runner(t, cases.Module)
		if m == "organization" {
			r = e.Runner(t, organization.Module)
		}
		if _, err := e.worker.Drain(ctx, r, 50); err != nil {
			t.Fatalf("drain %s: %v", m, err)
		}
	}
}

func (e *writesEnv) step(t *testing.T) writeback.Report {
	t.Helper()
	rep, err := e.worker.Step(context.Background(), 50)
	if err != nil {
		t.Fatal(err)
	}
	return rep
}

func (e *writesEnv) state(t *testing.T, commandID string) string {
	t.Helper()
	return e.q(t, `SELECT state || ':' || coalesce(parity, '-') FROM integration.writeback_command WHERE command_id = $1`, commandID)
}

func (e *writesEnv) q(t *testing.T, sql string, args ...any) string {
	t.Helper()
	var v string
	if err := e.admin.QueryRow(context.Background(), sql, args...).Scan(&v); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return v
}

func outcome(key, caseID, out string, version int64, a casescommand.Actor) casescommand.Command {
	return casescommand.Command{CaseID: caseID, Outcome: out, ExpectedVersion: version, IdempotencyKey: key, CorrelationID: "corr-" + key, Actor: a}
}

func reason(s string) *string { return &s }

// Contact outcome: authority matrix, idempotency, concurrency, history, the
// write-back to the legacy master, parity and financial non-mutation.
func TestContactOutcomeCommand(t *testing.T) {
	e := setupWrites(t)
	ctx := context.Background()
	c101 := e.caseIDs["101"]
	v := e.caseVersion(t, "101")
	finTables := []string{"billing.invoice", "billing.payment_stage", "payments.payment_attempt", "finance.ledger_entry", "biawin.subscription",
		"fulfillment.fulfillment_case"}
	finFP := e.md5(t, finTables...)
	finRows := e.count(t, `SELECT (SELECT count(*) FROM billing.invoice) + (SELECT count(*) FROM payments.payment_attempt) +
		(SELECT count(*) FROM finance.ledger_entry) + (SELECT count(*) FROM biawin.subscription) + (SELECT count(*) FROM fulfillment.fulfillment_case)`)
	custodyFP := e.md5(t, "custody.assignment", "custody.custody_event", "cases.attribution_snapshot")

	// Disabled mode (flag off): BLOCKED / FEATURE_DISABLED, nothing written.
	r, err := e.cases.RecordContactOutcome(ctx, rollout.Disabled, outcome("key-disabled-1", c101, "NO_ANSWER", v, e.actor("6")))
	if err != nil || r.Status != result.Blocked || r.Reasons[0] != reasoncodes.FeatureDisabled {
		t.Fatalf("disabled: %+v %v", r, err)
	}
	// Authority matrix (CONTROLLED_WRITEBACK).
	matrix := []struct {
		name, actor, lead, out string
		want                   result.Status
		code                   reasoncodes.Code
	}{
		{"holder seller", "6", "101", "NO_ANSWER", result.Success, ""},
		{"other seller", "7", "101", "NO_ANSWER", result.Unauthorized, reasoncodes.RowScopeDenied},
		{"supervisor of holder (higher rank, no row scope)", "4", "101", "NO_ANSWER", result.Unauthorized, reasoncodes.RowScopeDenied},
		{"manager (higher rank)", "2", "101", "NO_ANSWER", result.Unauthorized, reasoncodes.RowScopeDenied},
		{"HR", "1", "101", "NO_ANSWER", result.Unauthorized, reasoncodes.RowScopeDenied},
		{"no holder case", "4", "103", "NO_ANSWER", result.Blocked, reasoncodes.DependencyUnknown},
		{"duplicate outcome (D-29/D-31)", "6", "101", "DUPLICATE", result.Blocked, reasoncodes.PolicyUnresolved},
		{"pre-invoice (ADR-01)", "6", "101", "PRE_INVOICE", result.Blocked, reasoncodes.PolicyUnresolved},
	}
	for i, m := range matrix {
		cmd := outcome("key-matrix-"+string(rune('a'+i))+"-0", e.caseIDs[m.lead], m.out, e.caseVersion(t, m.lead), e.actor(m.actor))
		r, err := e.cases.RecordContactOutcome(ctx, rollout.Controlled, cmd)
		if err != nil || r.Status != m.want || (m.code != "" && r.Reasons[0] != m.code) {
			t.Fatalf("%s: %+v %v", m.name, r, err)
		}
	}
	if r := matrix; len(r) == 0 {
		t.Fatal()
	}
	v = e.caseVersion(t, "101")
	// Field rules: NOT_PURCHASED needs a reason; CALLBACK needs a time.
	if _, err := e.cases.RecordContactOutcome(ctx, rollout.Controlled, outcome("key-field-1", c101, "NOT_PURCHASED", v, e.actor("6"))); err == nil {
		t.Fatal("NOT_PURCHASED without reason must be invalid")
	}
	// Stale version: CONFLICT.
	if r, _ := e.cases.RecordContactOutcome(ctx, rollout.Controlled, outcome("key-stale-1", c101, "NO_ANSWER", v-1, e.actor("6"))); r.Status != result.Conflict {
		t.Fatalf("stale: %+v", r)
	}
	// Idempotency: same key + same body replays; same key + other body conflicts.
	np := outcome("key-np-0001", c101, "NOT_PURCHASED", v, e.actor("6"))
	np.Reason = reason("synthetic reason")
	r1, _ := e.cases.RecordContactOutcome(ctx, rollout.Controlled, np)
	r2, _ := e.cases.RecordContactOutcome(ctx, rollout.Controlled, np)
	if r1.Status != result.Success || !r2.Replayed || r2.CommandID != r1.CommandID || r1.WritebackState != casescommand.WritebackPending {
		t.Fatalf("idempotency: %+v %+v", r1, r2)
	}
	other := np
	other.Reason = reason("a different synthetic reason")
	if r, _ := e.cases.RecordContactOutcome(ctx, rollout.Controlled, other); r.Status != result.Conflict {
		t.Fatalf("same key, different body: %+v", r)
	}
	stolen := np
	stolen.Actor = e.actor("7")
	if r, _ := e.cases.RecordContactOutcome(ctx, rollout.Controlled, stolen); r.Status != result.Conflict || r.Replayed {
		t.Fatalf("another actor reusing a key never replays someone else's result: %+v", r)
	}
	if n := e.count(t, `SELECT count(*) FROM cases.contact_outcome_event WHERE case_id = $1`, c101); n != 2 {
		t.Fatalf("one effect per intent: %d events", n)
	}
	// Concurrency: two commands on the same version — exactly one applies.
	v = e.caseVersion(t, "101")
	var wg sync.WaitGroup
	results := make([]casescommand.Result, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i], _ = e.cases.RecordContactOutcome(ctx, rollout.Controlled, outcome("key-race-000"+string(rune('a'+i)), c101, "NO_ANSWER", v, e.actor("6")))
		}(i)
	}
	wg.Wait()
	ok := 0
	for _, r := range results {
		if r.Status == result.Success {
			ok++
		} else if r.Status != result.Conflict {
			t.Fatalf("race loser: %+v", r)
		}
	}
	if ok != 1 {
		t.Fatalf("exactly one concurrent command applies: %+v", results)
	}
	// History is append-only and the custody / attribution / finance facts are untouched.
	if _, err := e.admin.Exec(ctx, `SET ROLE crm_owner_cases`); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{`UPDATE cases.contact_outcome_event SET outcome = 'NO_ANSWER'`, `DELETE FROM cases.command_record`} {
		if _, err := e.admin.Exec(ctx, q); err == nil {
			t.Fatalf("history mutation allowed: %s", q)
		}
	}
	if _, err := e.admin.Exec(ctx, `RESET ROLE`); err != nil {
		t.Fatal(err)
	}
	if e.md5(t, "custody.assignment", "custody.custody_event", "cases.attribution_snapshot") != custodyFP ||
		e.md5(t, finTables...) != finFP || finRows != 0 {
		t.Fatal("a contact outcome changed custody, attribution or financial facts")
	}

	// Write-back: drain the outbox, deliver, observe parity.
	e.drain(t)
	e.drain(t) // a redelivery is a no-op (inbox)
	if n := e.count(t, `SELECT count(*) FROM integration.writeback_command`); n != 3 {
		t.Fatalf("one write-back command per accepted command: %d", n)
	}
	rep := e.step(t)
	if rep.Applied != 3 || e.legacy.FlowEvents() != 3 {
		t.Fatalf("delivery: %+v %d", rep, e.legacy.FlowEvents())
	}
	if got := e.state(t, r1.CommandID); got != "APPLIED:MATCH" {
		t.Fatalf("parity: %s", got)
	}
	if l := e.legacy.Lead("101"); l.StatusLabel != "جواب نداده" {
		t.Fatalf("legacy lead: %+v", l)
	}
	// Attempts carry the service actor, never the human.
	if got := e.q(t, `SELECT DISTINCT service_actor FROM integration.writeback_attempt`); got != writeback.ServiceActor {
		t.Fatal(got)
	}
	if e.step(t) != (writeback.Report{}) || e.legacy.Calls["lead"] != 3 {
		t.Fatal("an applied command is never delivered again")
	}
}

// UNKNOWN is reconciled before any retry: a timeout after the legacy commit
// is discovered by observation (no duplicate effect); a timeout before the
// commit is retried only after observation proves the effect absent; a known
// refusal is REJECTED with a reconciliation record and target history kept.
func TestWritebackUnknownAndRejection(t *testing.T) {
	e := setupWrites(t)
	ctx := context.Background()
	c101, c102 := e.caseIDs["101"], e.caseIDs["102"]

	e.legacy.Inject("lead", legacytest.TimeoutAfterCommit)
	r, _ := e.cases.RecordContactOutcome(ctx, rollout.Controlled, outcome("key-unk-0001", c101, "NO_ANSWER", e.caseVersion(t, "101"), e.actor("6")))
	e.drain(t)
	if rep := e.step(t); rep.Unknown != 1 || e.state(t, r.CommandID) != "UNKNOWN:-" {
		t.Fatalf("timeout after commit is UNKNOWN, never FAILED: %+v %s", rep, e.state(t, r.CommandID))
	}
	if n := e.count(t, `SELECT count(*) FROM integration.writeback_reconciliation WHERE command_id = $1 AND reason = 'WRITEBACK_UNKNOWN'
		AND next_safe_action = 'OBSERVE_LEGACY_BEFORE_RETRY'`, r.CommandID); n != 1 {
		t.Fatal("UNKNOWN opens a reconcile-first record")
	}
	if rep := e.step(t); rep.Applied != 1 || e.state(t, r.CommandID) != "APPLIED:MATCH" || e.legacy.Calls["lead"] != 1 || e.legacy.FlowEvents() != 1 {
		t.Fatalf("reconcile discovered the commit without a second apply: %+v %s calls=%d", rep, e.state(t, r.CommandID), e.legacy.Calls["lead"])
	}

	e.legacy.Inject("lead", legacytest.TimeoutBeforeCommit)
	cb := outcome("key-unk-0002", c101, "CALLBACK", e.caseVersion(t, "101"), e.actor("6"))
	at := time.Date(2026, 11, 3, 10, 30, 0, 0, time.UTC)
	cb.CallbackAt = &at
	r2, _ := e.cases.RecordContactOutcome(ctx, rollout.Controlled, cb)
	e.drain(t)
	e.step(t) // UNKNOWN
	e.legacy.Inject("lead_observe", legacytest.ObserveDown)
	if rep := e.step(t); rep.Unknown != 1 || e.legacy.Calls["lead"] != 2 {
		t.Fatalf("legacy unreadable: stays UNKNOWN, no retry: %+v", rep)
	}
	if rep := e.step(t); rep.Retried != 1 || e.legacy.Calls["lead"] != 2 {
		t.Fatalf("observation proved absence first: %+v", rep)
	}
	if rep := e.step(t); rep.Applied != 1 || e.state(t, r2.CommandID) != "APPLIED:MATCH" || e.legacy.Calls["lead"] != 3 {
		t.Fatalf("safe retry after proven absence: %+v %s", rep, e.state(t, r2.CommandID))
	}
	if l := e.legacy.Lead("101"); l.CallbackJalali != "1405/08/12 14:00" {
		t.Fatalf("callback mapped to legacy Jalali Tehran time: %q", l.CallbackJalali)
	}

	// Known refusal: legacy has lead 102 archived. Target history stays.
	r3, _ := e.cases.RecordContactOutcome(ctx, rollout.Controlled, outcome("key-rej-0001", c102, "NO_ANSWER", e.caseVersion(t, "102"), e.actor("7")))
	e.drain(t)
	if rep := e.step(t); rep.Rejected != 1 || e.state(t, r3.CommandID) != "REJECTED:-" {
		t.Fatalf("rejection: %+v %s", rep, e.state(t, r3.CommandID))
	}
	if n := e.count(t, `SELECT count(*) FROM cases.contact_outcome_event WHERE command_id = $1`, r3.CommandID); n != 1 {
		t.Fatal("a rejected write-back never deletes target history")
	}
	if got := e.q(t, `SELECT reason || ':' || next_safe_action FROM integration.writeback_reconciliation WHERE command_id = $1`, r3.CommandID); got !=
		"WRITEBACK_REJECTED:DOMAIN_OWNER_REVIEW" {
		t.Fatal(got)
	}
	if got := e.q(t, `SELECT outcome FROM integration.writeback_attempt WHERE command_id = $1`, r3.CommandID); got != "CONFLICT" {
		t.Fatalf("archived refusal recorded as a known non-commit: %s", got)
	}
}

// SHADOW_COMPARE evaluates without changing the aggregate or legacy, and
// parity compares the expected effect with what legacy actually holds.
func TestShadowCompareParity(t *testing.T) {
	e := setupWrites(t)
	ctx := context.Background()
	c101 := e.caseIDs["101"]
	v := e.caseVersion(t, "101")
	r, _ := e.cases.RecordContactOutcome(ctx, rollout.Shadow, outcome("key-shadow-01", c101, "NO_ANSWER", v, e.actor("6")))
	if r.Status != result.Success || r.Applied || e.caseVersion(t, "101") != v || r.WritebackState != casescommand.WritebackNotRequested {
		t.Fatalf("shadow: %+v", r)
	}
	if n := e.count(t, `SELECT count(*) FROM cases.contact_outcome_event`); n != 0 {
		t.Fatal("shadow never changes history")
	}
	// Legacy users act in legacy: the same intent applied there (MATCH)…
	e.legacy.Leads["101"].StatusLabel = "جواب نداده"
	e.drain(t)
	if rep := e.step(t); rep.Compared != 1 || e.legacy.Calls["lead"] != 0 {
		t.Fatalf("shadow never writes legacy: %+v", rep)
	}
	if got := e.state(t, r.CommandID); got != "COMPARED:INCOMPLETE" {
		t.Fatalf("legacy holds no effect of this intent: %s", got)
	}
	// MATCH / MISMATCH are decided only against an observation of the same intent.
	exp := map[string]string{"outcome": "NO_ANSWER"}
	if writeback.ParityOf(exp, writeback.Observed{Observation: writeback.Committed, State: map[string]string{"outcome": "NO_ANSWER"}}) != "MATCH" ||
		writeback.ParityOf(exp, writeback.Observed{Observation: writeback.Committed, State: map[string]string{"outcome": "CALLBACK"}}) != "MISMATCH" ||
		writeback.ParityOf(exp, writeback.Observed{Observation: writeback.Ambiguous}) != "CONFLICT" ||
		writeback.ParityOf(exp, writeback.Observed{Observation: writeback.Unseen}) != "UNKNOWN" {
		t.Fatal("parity classes")
	}
}

func hrCreate(key, typ, subject string, target *string, a hrrequest.Actor) hrrequest.Create {
	return hrrequest.Create{RequestType: typ, SubjectUserID: subject, TargetParentUserID: target, Reason: "synthetic reason " + key,
		IdempotencyKey: key, CorrelationID: "corr-" + key, Actor: a}
}

// HR requests: creation authority and scope (SD-04), step review, OPD-05
// final apply blocked, D-42 (no capability leakage), write-back.
func TestHRRequestCommands(t *testing.T) {
	e := setupWrites(t)
	ctx := context.Background()
	u := e.users
	ptr := func(s string) *string { return &s }
	idFP := e.md5(t, "identity.user_account", "identity.role_grant", "organization.org_assignment", "organization.employment")

	cases := []struct {
		name, actor, typ, subject string
		target                    *string
		want                      result.Status
		code                      reasoncodes.Code
	}{
		{"seller cannot create", "6", "terminate_seller", u["7"], nil, result.Unauthorized, reasoncodes.ActionDenied},
		{"HR creator blocked (runtime not verified)", "1", "terminate_seller", u["6"], nil, result.Blocked, reasoncodes.PolicyUnresolved},
		{"supervisor C: seller outside scope", "8", "terminate_seller", u["6"], nil, result.Unauthorized, reasoncodes.RowScopeDenied},
		{"senior: cross-scope destination (SD-04)", "3", "transfer_seller", u["6"], ptr(u["8"]), result.Blocked, reasoncodes.PolicyUnresolved},
		{"senior: no-op destination (SD-04)", "3", "transfer_seller", u["6"], ptr(u["4"]), result.Blocked, reasoncodes.PolicyUnresolved},
		{"senior: destination not a supervisor", "3", "transfer_seller", u["6"], ptr(u["7"]), result.Blocked, reasoncodes.RecipientIneligible},
		{"unknown employment status of subject", "4", "terminate_seller", u["10"], nil, result.Blocked, reasoncodes.LinkageConflict},
	}
	for i, c := range cases {
		r, err := e.hr.CreateRequest(ctx, rollout.Controlled, hrCreate("key-hr-neg-"+string(rune('a'+i)), c.typ, c.subject, c.target, e.hrActor(c.actor)))
		if err != nil || r.Status != c.want || r.Reasons[0] != c.code {
			t.Fatalf("%s: %+v %v", c.name, r, err)
		}
	}
	// Senior transfers seller X (under A) to supervisor B (in scope): pending_review → manager (2).
	tr, err := e.hr.CreateRequest(ctx, rollout.Controlled, hrCreate("key-hr-tr-001", "transfer_seller", u["6"], ptr(u["5"]), e.hrActor("3")))
	if err != nil || tr.Status != result.Success || tr.RequestStatus != "pending_review" {
		t.Fatalf("transfer create: %+v %v", tr, err)
	}
	if got := e.q(t, `SELECT current_reviewer_user_id::text || ':' || subject_parent_user_id::text FROM organization.hr_request WHERE request_id = $1`,
		tr.RequestID); got != u["2"]+":"+u["4"] {
		t.Fatalf("reviewer and subject parent snapshots: %s", got)
	}
	// Duplicate open request: CONFLICT.
	if r, _ := e.hr.CreateRequest(ctx, rollout.Controlled, hrCreate("key-hr-tr-002", "transfer_seller", u["6"], ptr(u["5"]), e.hrActor("3"))); r.Status != result.Conflict {
		t.Fatalf("duplicate: %+v", r)
	}
	// Supervisor A requests termination of seller Y: reviewer = senior (3).
	tm, _ := e.hr.CreateRequest(ctx, rollout.Controlled, hrCreate("key-hr-tm-001", "terminate_seller", u["7"], nil, e.hrActor("4")))
	if tm.Status != result.Success || tm.RequestStatus != "pending_review" {
		t.Fatalf("terminate create: %+v", tm)
	}
	dec := func(key, id, d string, ver int64, a string) hrrequest.Result {
		r, err := e.hr.DecideRequest(ctx, rollout.Controlled, hrrequest.Decide{RequestID: id, Decision: d, ExpectedVersion: ver, IdempotencyKey: key,
			CorrelationID: "corr-" + key, Actor: e.hrActor(a)})
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	// Wrong reviewer / higher rank without being the reviewer: denied.
	if r := dec("key-hr-d-001", tm.RequestID, "approve", 1, "2"); r.Status != result.Unauthorized {
		t.Fatalf("manager is not the current reviewer: %+v", r)
	}
	if r := dec("key-hr-d-002", tm.RequestID, "approve", 1, "1"); r.Status != result.Blocked {
		t.Fatalf("HR final decision blocked (OPD-05): %+v", r)
	}
	// Senior approves the step: pending_review → manager.
	s1 := dec("key-hr-d-003", tm.RequestID, "approve", 1, "3")
	if s1.Status != result.Success || s1.RequestStatus != "pending_review" || s1.Version != 2 {
		t.Fatalf("step: %+v", s1)
	}
	// Two reviewers on the same version: the stale one conflicts.
	if r := dec("key-hr-d-004", tm.RequestID, "approve", 1, "2"); r.Status != result.Conflict {
		t.Fatalf("stale decision: %+v", r)
	}
	// Manager (no parent) approves: pending_hr — NOT applied.
	s2 := dec("key-hr-d-005", tm.RequestID, "approve", 2, "2")
	if s2.RequestStatus != "pending_hr" {
		t.Fatalf("to HR: %+v", s2)
	}
	// Final apply (pending_hr): BLOCKED; the seller stays employed and placed.
	if r := dec("key-hr-d-006", tm.RequestID, "approve", 3, "2"); r.Status != result.Blocked {
		t.Fatalf("final apply blocked: %+v", r)
	}
	if e.md5(t, "identity.user_account", "identity.role_grant", "organization.org_assignment", "organization.employment") != idFP {
		t.Fatal("an HR request changed identity, grants, hierarchy or employment (OPD-05 / D-42)")
	}
	// Manager rejects the transfer request.
	if r := dec("key-hr-d-007", tr.RequestID, "reject", 1, "2"); r.RequestStatus != "rejected" {
		t.Fatalf("reject: %+v", r)
	}
	if n := e.count(t, `SELECT count(*) FROM organization.hr_request_event`); n != 5 {
		t.Fatalf("history events: %d", n)
	}
	// Write-back: creates first, then decisions with the legacy request id.
	e.drain(t)
	for i := 0; i < 4; i++ {
		e.step(t)
	}
	var pending int64 = e.count(t, `SELECT count(*) FROM integration.writeback_command WHERE state <> 'APPLIED'`)
	if pending != 0 {
		t.Fatalf("all HR write-backs applied: %d left; %s", pending, e.q(t, `SELECT string_agg(event_type || ':' || state, ',') FROM integration.writeback_command`))
	}
	if got := e.q(t, `SELECT string_agg(parity, ',' ORDER BY received_at) FROM integration.writeback_command`); strings.Count(got, "MATCH") != 5 || strings.Contains(got, "MISMATCH") {
		t.Fatalf("HR parity: %s", got)
	}
	var legacyTerm legacytest.SyntheticHRRequest
	for id := range e.legacy.Requests {
		if r := e.legacy.Request(id); r.RequestType == "terminate_seller" {
			legacyTerm = r
		}
	}
	if legacyTerm.Status != "pending_hr" || legacyTerm.Subject != "7" || len(legacyTerm.Path) != 3 {
		t.Fatalf("legacy request: %+v", legacyTerm)
	}
}

// Cross-cutting Slice 11 invariants: ambiguous legacy results open
// reconciliation (never success); no ACK / SLA / phone column exists
// (OPD-02, D-29/D-31); command identities never collide across cases; HR
// timeouts after commit are reconciled without a second insert.
func TestWritesInvariants(t *testing.T) {
	e := setupWrites(t)
	ctx := context.Background()
	u := e.users
	ptr := func(s string) *string { return &s }

	// HR create: timeout after the legacy commit → UNKNOWN → observed → APPLIED, one legacy row.
	e.legacy.Inject("hr_insert", legacytest.TimeoutAfterCommit)
	r, _ := e.hr.CreateRequest(ctx, rollout.Controlled, hrCreate("key-inv-hr-01", "transfer_seller", u["6"], ptr(u["5"]), e.hrActor("3")))
	e.drain(t)
	e.step(t)
	if got := e.state(t, r.CommandID); got != "UNKNOWN:-" {
		t.Fatalf("hr timeout: %s", got)
	}
	e.step(t)
	if got := e.state(t, r.CommandID); got != "APPLIED:MATCH" || len(e.legacy.Requests) != 1 || e.legacy.Calls["hr_insert"] != 1 {
		t.Fatalf("hr reconcile: %s requests=%d calls=%d", got, len(e.legacy.Requests), e.legacy.Calls["hr_insert"])
	}

	// A second create whose legacy result turns out duplicated (two candidate
	// rows for one intent) is AMBIGUOUS → reconciliation, never success.
	e.legacy.Inject("hr_insert", legacytest.TimeoutAfterCommit)
	r2, _ := e.hr.CreateRequest(ctx, rollout.Controlled, hrCreate("key-inv-hr-02", "terminate_seller", u["9"], nil, e.hrActor("8")))
	e.drain(t)
	e.step(t)
	e.legacy.Duplicate(r2.CommandID)
	e.step(t)
	if got := e.state(t, r2.CommandID); got != "RECONCILIATION_REQUIRED:-" {
		t.Fatalf("ambiguous legacy: %s", got)
	}
	if got := e.q(t, `SELECT reason || ':' || next_safe_action FROM integration.writeback_reconciliation WHERE command_id = $1 AND reason = 'LEGACY_AMBIGUOUS'`,
		r2.CommandID); got != "LEGACY_AMBIGUOUS:DOMAIN_OWNER_REVIEW" {
		t.Fatal(got)
	}

	// No acceptance / SLA / escalation / phone column anywhere in the Slice 11 tables.
	if n := e.count(t, `SELECT count(*) FROM information_schema.columns WHERE (table_schema, table_name) IN
		(('cases','contact_outcome_current'),('cases','contact_outcome_event'),('cases','command_record'),('organization','hr_request'),
		 ('organization','hr_request_event'),('organization','command_record'),('integration','writeback_command'),('integration','writeback_attempt'),
		 ('integration','writeback_reconciliation'),('integration','writeback_parity'))
		AND column_name ~* '(accepted_at|due_at|sla|escalat|phone|mobile|password|credential|capabilit|role_key)'`); n != 0 {
		t.Fatalf("forbidden columns: %d", n)
	}

	// Distinct cases → distinct command identities, even with a shared key
	// shape; nothing is merged or collapsed.
	a, _ := e.cases.RecordContactOutcome(ctx, rollout.Controlled, outcome("key-inv-c-001", e.caseIDs["101"], "NO_ANSWER", e.caseVersion(t, "101"), e.actor("6")))
	b, _ := e.cases.RecordContactOutcome(ctx, rollout.Controlled, outcome("key-inv-c-002", e.caseIDs["102"], "NO_ANSWER", e.caseVersion(t, "102"), e.actor("7")))
	if a.Status != result.Success || b.Status != result.Success || a.CommandID == b.CommandID {
		t.Fatalf("command identity: %+v %+v", a, b)
	}
	if n := e.count(t, `SELECT count(DISTINCT case_id) FROM cases.contact_outcome_event`); n != 2 {
		t.Fatalf("cases kept distinct: %d", n)
	}
	// Current custody, attribution facets and the historical actor are untouched.
	if n := e.count(t, `SELECT count(*) FROM custody.custody_event`); n != 0 {
		t.Fatalf("no custody event was written by a contact outcome: %d", n)
	}
}
