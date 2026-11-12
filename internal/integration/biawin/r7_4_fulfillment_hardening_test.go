package biawin_test

// R7-4 fulfillment domain hardening (backend + disposable PostgreSQL).
//
// BACKEND-ONLY COVERAGE: commands are issued straight to work.Service with
// explicit synthetic actors ("synthetic:<uuid>"). No capability gate runs
// here, so nothing in this file is evidence of production authorization
// (no role bundle carries fulfillment.work.*; see R7-3 for the real
// refusal). Authority asserted here is the catalog per-service row scope,
// which the service enforces itself inside the transaction.
//
// Provider behaviour uses integrationbiawin.NewSynthetic(): a deterministic
// TEST-ONLY port, never a production connector, never a network call.
//
// Every refusal is asserted with its exact status, reason codes and
// blocked_by, and with an unchanged fingerprint of the fulfillment,
// integration, biawin and catalog schemas (a refusal stores nothing, not
// even its idempotency key). Positive paths assert the exact rows written.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	biawinapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin/api"
	catalogapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/catalog/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/catalog/registry"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment/work"
	integrationbiawin "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/biawin"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/providerops"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/outbox"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/rollout"
)

// ---- helpers ----------------------------------------------------------------

// guardedSchemas are fingerprinted around every refusal.
var guardedSchemas = []string{"fulfillment", "integration", "biawin", "catalog"}

// fingerprint is one md5 per base table of the guarded schemas (row order
// independent). Table names only ever reach failure output.
func (e *env) fingerprint(t *testing.T) map[string]string {
	t.Helper()
	ctx := context.Background()
	rows, err := e.admin(t).Query(ctx, `SELECT quote_ident(table_schema) || '.' || quote_ident(table_name) FROM information_schema.tables
		WHERE table_type = 'BASE TABLE' AND table_schema = ANY($1) ORDER BY 1`, guardedSchemas)
	if err != nil {
		t.Fatal(err)
	}
	var tables []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		tables = append(tables, s)
	}
	rows.Close()
	out := map[string]string{}
	for _, tbl := range tables {
		var h *string
		if err := e.admin(t).QueryRow(ctx, `SELECT md5(coalesce(string_agg(x::text, '|' ORDER BY x::text), '')) FROM `+tbl+` x`).Scan(&h); err != nil {
			t.Fatal(err)
		}
		out[tbl] = *h
	}
	return out
}

func sameState(t *testing.T, label string, before, after map[string]string) {
	t.Helper()
	for k, v := range before {
		if after[k] != v {
			t.Errorf("%s: %s changed (a refusal must store nothing)", label, k)
		}
	}
}

// expectRefusal asserts the exact refusal contract.
func expectRefusal(t *testing.T, label string, r work.Result, st result.Status, code reasoncodes.Code, blockedBy ...string) {
	t.Helper()
	if r.Status != st || r.StatusName != st.String() {
		t.Fatalf("%s: status %s, want %s (%+v)", label, r.StatusName, st, r)
	}
	if !reflect.DeepEqual(r.Reasons, []reasoncodes.Code{code}) {
		t.Fatalf("%s: reasons %v, want [%s]", label, r.Reasons, code)
	}
	if len(blockedBy) == 0 {
		blockedBy = nil
	}
	if !reflect.DeepEqual(r.BlockedBy, blockedBy) {
		t.Fatalf("%s: blocked_by %v, want %v", label, r.BlockedBy, blockedBy)
	}
	if r.Applied || r.CommandID != "" || r.Replayed {
		t.Fatalf("%s: a refusal is never applied, recorded or replayed: %+v", label, r)
	}
}

// refused runs c expecting a refusal and an unchanged database.
func (e *env) refused(t *testing.T, label string, c work.Command, st result.Status, code reasoncodes.Code, blockedBy ...string) work.Result {
	t.Helper()
	before := e.fingerprint(t)
	r := e.cmd(t, c)
	expectRefusal(t, label, r, st, code, blockedBy...)
	sameState(t, label, before, e.fingerprint(t))
	return r
}

// invalidCmd runs c expecting the contract error and an unchanged database.
func (e *env) invalidCmd(t *testing.T, label string, c work.Command) {
	t.Helper()
	if c.IdempotencyKey == "" {
		c.IdempotencyKey = key()
	}
	if c.CorrelationID == "" {
		c.CorrelationID = "corr-invalid"
	}
	before := e.fingerprint(t)
	if _, err := e.work.Execute(context.Background(), rollout.Controlled, c); !errors.Is(err, work.ErrInvalid) {
		t.Fatalf("%s: want ErrInvalid, got %v", label, err)
	}
	sameState(t, label, before, e.fingerprint(t))
}

func expectSuccess(t *testing.T, label string, r work.Result, state string, version int64) {
	t.Helper()
	if r.Status != result.Success || !r.Applied || r.Mode != rollout.Controlled || len(r.Reasons) != 0 || r.CommandID == "" ||
		r.State != state || r.CaseVersion != version {
		t.Fatalf("%s: want SUCCESS applied %s v%d, got %+v", label, state, version, r)
	}
}

// grant records a catalog scope change (GRANT / REVOKE) under a synthetic approval.
func (e *env) grant(t *testing.T, service, user, kind, action string) {
	t.Helper()
	if err := e.cat.Record(context.Background(), registry.Recording{ApprovalRef: "SYN-R7-4-" + action, RecordedBy: "test:r7-4",
		Scopes: []registry.ScopeRecord{{ServiceKey: service, UserID: user, Kind: kind, Action: action}}}); err != nil {
		t.Fatal(err)
	}
}

// newCase activates one more synthetic subscription of a one-service product
// and returns its single QUEUED v1 work case.
func (e *env) newCase(t *testing.T, product string) string {
	t.Helper()
	customer := uid(t)
	o := e.activate(t, paid(t, product, biawinapi.RouteAdditionalServices, "RESOLVED", &customer))
	e.relay(t)
	cs := e.cases(t, *o.SubscriptionID)
	if len(cs) != 1 {
		t.Fatalf("expected one case, got %d", len(cs))
	}
	return cs[0]
}

// injectItem delivers one synthetic SubscriptionItemActivated message to the
// fulfillment intake consumer (fixture at the inbox boundary; the consumer
// code is the production one) and returns the created case.
func (e *env) injectItem(t *testing.T, serviceKey string) string {
	t.Helper()
	item := uid(t)
	payload, _ := json.Marshal(biawinapi.SubscriptionItemActivated{ActivationID: uid(t), SubscriptionID: uid(t), ItemID: item, Seq: 1, ServiceKey: serviceKey,
		ServiceVersion: 1, CustomerID: uid(t), RolloutMode: "SHADOW_COMPARE", CorrelationID: "corr-inject", CausationID: "r74", OccurredAt: e.clock.Now().Time()})
	out, err := e.work.OnItemActivated(context.Background(), outbox.Record{Event: outbox.Event{ID: uid(t), Type: biawinapi.EventItemActivated, Version: 1,
		Payload: payload, OccurredAt: e.clock.Now()}})
	if err != nil || out.Decision != "CASE_CREATED" || out.CaseID == nil {
		t.Fatalf("inject %s: %+v %v", serviceKey, out, err)
	}
	return *out.CaseID
}

func (e *env) caseState(t *testing.T, caseID string) string {
	t.Helper()
	var s string
	if err := e.admin(t).QueryRow(context.Background(), `SELECT state || '|' || coalesce(assignee_user_id::text, '-') || '|v' || version
		FROM fulfillment.work_case WHERE case_id = $1`, caseID).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

func at(s string) *time.Time { v, _ := time.Parse(time.RFC3339, s); return &v }

func (e *env) q(t *testing.T, sql string, args ...any) string {
	t.Helper()
	var v *string
	if err := e.admin(t).QueryRow(context.Background(), sql, args...).Scan(&v); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	if v == nil {
		return ""
	}
	return *v
}

// ---- A. assignment and row scope ---------------------------------------------

func TestR74FulfillmentAssignmentScope(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	sup, sup2, ex, ex2, supT, exT, outsider := uid(t), uid(t), uid(t), uid(t), uid(t), uid(t), uid(t)
	e.seed(t, "r74-as", 1, map[string]string{"r74-as.svc-01": sup}, map[string]string{"r74-as.svc-01": ex}, nil)
	e.seed(t, "r74-at", 1, map[string]string{"r74-at.svc-01": supT}, map[string]string{"r74-at.svc-01": exT}, nil)
	e.grant(t, "r74-as.svc-01", sup2, catalogapi.ScopeSupervisor, "GRANT")
	e.grant(t, "r74-as.svc-01", ex2, catalogapi.ScopeExecutor, "GRANT")
	c := e.newCase(t, "r74-as")
	assign := func(actorID, assignee string, v int64) work.Command {
		return work.Command{Type: work.TypeAssign, CaseID: c, AssigneeUserID: assignee, ExpectedVersion: v, Actor: actor(actorID)}
	}

	// Refusals on the QUEUED v1 case: nothing is stored.
	e.refused(t, "supervisor of another service", assign(supT, ex, 1), result.Unauthorized, reasoncodes.RowScopeDenied)
	e.refused(t, "executor is not a supervisor", assign(ex, ex, 1), result.Unauthorized, reasoncodes.RowScopeDenied)
	e.refused(t, "no scope at all", assign(outsider, ex, 1), result.Unauthorized, reasoncodes.RowScopeDenied)
	e.refused(t, "assignee is an executor of another service", assign(sup, exT, 1), result.Blocked, reasoncodes.RecipientIneligible)
	e.refused(t, "assignee is a supervisor, not an executor", assign(sup, sup2, 1), result.Blocked, reasoncodes.RecipientIneligible)
	e.refused(t, "assignee with no scope", assign(sup, outsider, 1), result.Blocked, reasoncodes.RecipientIneligible)
	stale := e.refused(t, "stale If-Match", assign(sup, ex, 2), result.Conflict, reasoncodes.StaleState)
	if stale.Current != 1 {
		t.Fatalf("stale refusal reports the current version: %+v", stale)
	}
	noPrincipal := assign(sup, ex, 1)
	noPrincipal.Actor.Principal = ""
	e.refused(t, "actor without principal", noPrincipal, result.Unauthorized, reasoncodes.RowScopeDenied)
	if r, err := e.work.Execute(ctx, rollout.Disabled, work.Command{Type: work.TypeAssign, CaseID: c, AssigneeUserID: ex, ExpectedVersion: 1,
		IdempotencyKey: key(), CorrelationID: "c", Actor: actor(sup)}); err != nil || r.Mode != rollout.Disabled {
		t.Fatalf("disabled: %+v %v", r, err)
	} else {
		expectRefusal(t, "disabled mode", r, result.Blocked, reasoncodes.FeatureDisabled)
	}
	before := e.fingerprint(t)
	if _, err := e.work.Execute(ctx, rollout.Controlled, work.Command{Type: work.TypeAssign, CaseID: "00000000-0000-4000-8000-0000000000a4",
		AssigneeUserID: ex, ExpectedVersion: 1, IdempotencyKey: key(), CorrelationID: "c", Actor: actor(sup)}); !errors.Is(err, work.ErrNotFound) {
		t.Fatalf("unknown case: %v", err)
	}
	sameState(t, "unknown case", before, e.fingerprint(t))
	e.invalidCmd(t, "malformed assignee", work.Command{Type: work.TypeAssign, CaseID: c, AssigneeUserID: "not-a-uuid", ExpectedVersion: 1, Actor: actor(sup)})
	e.invalidCmd(t, "expected version 0", work.Command{Type: work.TypeAssign, CaseID: c, AssigneeUserID: ex, ExpectedVersion: 0, Actor: actor(sup)})
	e.invalidCmd(t, "short idempotency key", work.Command{Type: work.TypeAssign, CaseID: c, AssigneeUserID: ex, ExpectedVersion: 1, Actor: actor(sup),
		IdempotencyKey: "short"})

	// Positive: the current supervisor assigns a current executor.
	r := e.cmd(t, assign(sup, ex, 1))
	expectSuccess(t, "assign", r, api.WorkAssigned, 2)
	if got := e.caseState(t, c); got != "ASSIGNED|"+ex+"|v2" {
		t.Fatalf("case row: %s", got)
	}
	if n := e.count(t, `SELECT count(*) FROM fulfillment.work_event WHERE case_id = $1 AND kind = 'ASSIGNED' AND before_version = 1 AND after_version = 2
		AND actor_user_id::text = $2 AND supervisors ? $2 AND supervisors ? $3 AND command_id::text = $4`, c, sup, sup2, r.CommandID); n != 1 {
		t.Fatalf("assigned event with the supervisors of its time: %d", n)
	}
	if n := e.count(t, `SELECT count(*) FROM fulfillment.work_assignment WHERE case_id = $1 AND assignee_user_id = $2::uuid AND previous_assignee IS NULL
		AND assigned_by_user_id = $3::uuid`, c, ex, sup); n != 1 {
		t.Fatalf("assignment row: %d", n)
	}
	if n := e.count(t, `SELECT count(*) FROM fulfillment.work_command_record WHERE command_id = $1::uuid AND applied AND before_version = 1 AND after_version = 2
		AND rollout_mode = 'CONTROLLED_WRITEBACK'`, r.CommandID); n != 1 {
		t.Fatalf("command record: %d", n)
	}
	if n := e.count(t, `SELECT count(*) FROM fulfillment.outbox_event WHERE event_type = $1`, api.EventWorkCaseCompleted); n != 0 {
		t.Fatalf("assignment emits no completion event: %d", n)
	}

	// Reassignment before work starts is allowed; the previous assignee is kept.
	r = e.cmd(t, assign(sup2, ex2, 2))
	expectSuccess(t, "reassign while ASSIGNED", r, api.WorkAssigned, 3)
	if n := e.count(t, `SELECT count(*) FROM fulfillment.work_assignment WHERE case_id = $1 AND assignee_user_id = $2::uuid AND previous_assignee = $3::uuid`,
		c, ex2, ex); n != 1 {
		t.Fatal("reassignment keeps the previous assignee")
	}
	// OBSERVED (governance question G-F1, not a defect claim): assigning the
	// current assignee again is not refused; it records a new ASSIGNED event
	// and bumps the version. No contract defines a no-op refusal.
	r = e.cmd(t, assign(sup2, ex2, 3))
	expectSuccess(t, "no-op assignment (observed)", r, api.WorkAssigned, 4)
	if n := e.count(t, `SELECT count(*) FROM fulfillment.work_assignment WHERE case_id = $1`, c); n != 3 {
		t.Fatalf("assignment history rows: %d", n)
	}

	// Work started → reassignment is BLOCKED by OPD-05 (handover undecided).
	expectSuccess(t, "note", e.cmd(t, work.Command{Type: work.TypeAction, CaseID: c, Action: api.ActionNote, Note: ptr("started"), ExpectedVersion: 4,
		Actor: actor(ex2)}), api.WorkInProgress, 5)
	e.refused(t, "reassign after work started", assign(sup, ex, 5), result.Blocked, reasoncodes.PolicyUnresolved, api.BlockOPD05)
	// The OPD-05 block precedes the version check: a stale reassignment still names the decision.
	e.refused(t, "stale reassign after work started", assign(sup, ex, 1), result.Blocked, reasoncodes.PolicyUnresolved, api.BlockOPD05)

	// Revoked scope is enforced at command time; history keeps the old facts.
	e.grant(t, "r74-as.svc-01", sup2, catalogapi.ScopeSupervisor, "REVOKE")
	e.grant(t, "r74-as.svc-01", ex2, catalogapi.ScopeExecutor, "REVOKE")
	c2 := e.newCase(t, "r74-as")
	e.refused(t, "revoked supervisor", work.Command{Type: work.TypeAssign, CaseID: c2, AssigneeUserID: ex, ExpectedVersion: 1, Actor: actor(sup2)},
		result.Unauthorized, reasoncodes.RowScopeDenied)
	e.refused(t, "revoked executor as assignee", work.Command{Type: work.TypeAssign, CaseID: c2, AssigneeUserID: ex2, ExpectedVersion: 1, Actor: actor(sup)},
		result.Blocked, reasoncodes.RecipientIneligible)
	e.refused(t, "assignee whose executor scope was revoked", work.Command{Type: work.TypeAction, CaseID: c, Action: api.ActionNoAnswer, ExpectedVersion: 5,
		Actor: actor(ex2)}, result.Unauthorized, reasoncodes.RowScopeDenied)
	if n := e.count(t, `SELECT count(*) FROM fulfillment.work_event WHERE case_id = $1 AND actor_user_id = $2::uuid`, c, sup2); n != 2 {
		t.Fatalf("history of the revoked supervisor is preserved: %d", n)
	}
}

// ---- B. operational actions --------------------------------------------------

func TestR74FulfillmentOperationalActions(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	sup, ex, ex2 := uid(t), uid(t), uid(t)
	e.seed(t, "r74-ac", 1, map[string]string{"r74-ac.svc-01": sup}, map[string]string{"r74-ac.svc-01": ex}, nil)
	e.grant(t, "r74-ac.svc-01", ex2, catalogapi.ScopeExecutor, "GRANT")
	queued := e.newCase(t, "r74-ac")
	c := e.newCase(t, "r74-ac")
	expectSuccess(t, "assign", e.cmd(t, work.Command{Type: work.TypeAssign, CaseID: c, AssigneeUserID: ex, ExpectedVersion: 1, Actor: actor(sup)}), api.WorkAssigned, 2)
	act := func(who, action string, v int64) work.Command {
		return work.Command{Type: work.TypeAction, CaseID: c, Action: action, ExpectedVersion: v, Actor: actor(who)}
	}

	e.refused(t, "unassigned case", work.Command{Type: work.TypeAction, CaseID: queued, Action: api.ActionNoAnswer, ExpectedVersion: 1, Actor: actor(ex)},
		result.Unauthorized, reasoncodes.RowScopeDenied)
	e.refused(t, "executor who is not the assignee", act(ex2, api.ActionNoAnswer, 2), result.Unauthorized, reasoncodes.RowScopeDenied)
	e.refused(t, "supervisor never substitutes", act(sup, api.ActionNoAnswer, 2), result.Unauthorized, reasoncodes.RowScopeDenied)
	e.invalidCmd(t, "unknown action", act(ex, "CALL_BACK", 2))
	e.invalidCmd(t, "FOLLOW_UP without a date", act(ex, api.ActionFollowUp, 2))
	e.invalidCmd(t, "NOTE without a note", act(ex, api.ActionNote, 2))
	empty := act(ex, api.ActionNote, 2)
	empty.Note = ptr("")
	e.invalidCmd(t, "empty note", empty)
	long := act(ex, api.ActionNote, 2)
	long.Note = ptr(strings.Repeat("n", 2001))
	e.invalidCmd(t, "note over 2000", long)

	k1 := key()
	first := act(ex, api.ActionNoAnswer, 2)
	first.IdempotencyKey = k1
	r1 := e.cmd(t, first)
	expectSuccess(t, "NO_ANSWER", r1, api.WorkInProgress, 3)
	fu := act(ex, api.ActionFollowUp, 3)
	fu.FollowUpAt = at("2026-11-10T08:00:00Z")
	expectSuccess(t, "FOLLOW_UP", e.cmd(t, fu), api.WorkInProgress, 4)
	note := act(ex, api.ActionNote, 4)
	note.Note = ptr("synthetic note")
	expectSuccess(t, "NOTE", e.cmd(t, note), api.WorkInProgress, 5)
	if got := e.q(t, `SELECT string_agg(before_version || '>' || after_version || ':' || coalesce(detail->>'action', '') || ':' || coalesce(detail->>'follow_up_at', '-'),
		',' ORDER BY after_version) FROM fulfillment.work_event WHERE case_id = $1 AND kind = 'ACTION_RECORDED'`, c); got !=
		"2>3:NO_ANSWER:-,3>4:FOLLOW_UP:2026-11-10T08:00:00Z,4>5:NOTE:-" {
		t.Fatalf("action history chain: %s", got)
	}

	stale := e.refused(t, "stale version", act(ex, api.ActionNoAnswer, 3), result.Conflict, reasoncodes.StaleState)
	if stale.Current != 5 {
		t.Fatalf("current version: %+v", stale)
	}
	// Replay: same key + same command returns the stored result; nothing new.
	before := e.fingerprint(t)
	rp := e.cmd(t, first)
	if !rp.Replayed || rp.CommandID != r1.CommandID || rp.CaseVersion != 3 || rp.Status != result.Success {
		t.Fatalf("replay: %+v", rp)
	}
	sameState(t, "replay", before, e.fingerprint(t))
	// Same key, other payload / other actor: CONFLICT, never another actor's stored result.
	other := first
	other.Action, other.Note = api.ActionNote, ptr("different")
	e.refused(t, "same key, different payload", other, result.Conflict, reasoncodes.ContractInvalid)
	byOther := first
	byOther.Actor = actor(ex2)
	e.refused(t, "same key, different actor", byOther, result.Conflict, reasoncodes.ContractInvalid)
	// A new key with identical content is a new fact (actions are deduplicated by key only).
	dup := act(ex, api.ActionNoAnswer, 5)
	expectSuccess(t, "same content, new key", e.cmd(t, dup), api.WorkInProgress, 6)
	// A refused command does not consume its key: the same key later succeeds.
	kr := key()
	wrong := act(ex2, api.ActionNoAnswer, 6)
	wrong.IdempotencyKey = kr
	e.refused(t, "refused with key K", wrong, result.Unauthorized, reasoncodes.RowScopeDenied)
	if n := e.count(t, `SELECT count(*) FROM fulfillment.idempotency_record WHERE idem_key = $1`, kr); n != 0 {
		t.Fatalf("a refusal stores no idempotency record: %d", n)
	}
	right := act(ex, api.ActionNoAnswer, 6)
	right.IdempotencyKey = kr
	expectSuccess(t, "key K after the refusal", e.cmd(t, right), api.WorkInProgress, 7)

	// SHADOW: evaluated and recorded as not applied; the case does not move.
	beforeCase := e.caseState(t, c)
	sh, err := e.work.Execute(ctx, rollout.Shadow, work.Command{Type: work.TypeAction, CaseID: c, Action: api.ActionNoAnswer, ExpectedVersion: 7,
		IdempotencyKey: key(), CorrelationID: "c", Actor: actor(ex)})
	if err != nil || sh.Status != result.Success || sh.Applied || sh.Mode != rollout.Shadow || sh.CaseVersion != 7 {
		t.Fatalf("shadow: %+v %v", sh, err)
	}
	if e.caseState(t, c) != beforeCase || e.count(t, `SELECT count(*) FROM fulfillment.work_command_record WHERE command_id = $1::uuid AND NOT applied
		AND after_version IS NULL`, sh.CommandID) != 1 {
		t.Fatal("shadow must record the command as not applied and leave the case unchanged")
	}

	// Closed case: no further action.
	expectSuccess(t, "close", e.cmd(t, work.Command{Type: work.TypeComplete, CaseID: c, Outcome: catalogapi.OutcomeNotExecuted, Reason: ptr("synthetic"),
		ExpectedVersion: 7, Actor: actor(ex)}), api.WorkNotExecuted, 8)
	e.refused(t, "action on a closed case", act(ex, api.ActionNoAnswer, 8), result.Conflict, reasoncodes.StaleState)
}

// ---- C. evidence metadata ----------------------------------------------------

func TestR74FulfillmentEvidence(t *testing.T) {
	e := setup(t)
	sup, ex, ex2 := uid(t), uid(t), uid(t)
	e.seed(t, "r74-ev", 1, map[string]string{"r74-ev.svc-01": sup}, map[string]string{"r74-ev.svc-01": ex}, func(v *catalogapi.ServiceVersion) {
		v.Evidence = []catalogapi.EvidenceRequirement{{Type: "service-form", Required: true}, {Type: "site-photo", Required: false}}
	})
	e.grant(t, "r74-ev.svc-01", ex2, catalogapi.ScopeExecutor, "GRANT")
	c, other := e.newCase(t, "r74-ev"), e.newCase(t, "r74-ev")
	for _, id := range []string{c, other} {
		expectSuccess(t, "assign", e.cmd(t, work.Command{Type: work.TypeAssign, CaseID: id, AssigneeUserID: ex, ExpectedVersion: 1, Actor: actor(sup)}), api.WorkAssigned, 2)
	}
	ev := func(caseID, typ, sha, label string, v int64, who string) work.Command {
		return work.Command{Type: work.TypeEvidence, CaseID: caseID, EvidenceType: typ, ContentSHA: sha, Label: label, ExpectedVersion: v, Actor: actor(who)}
	}
	sha := shaOf("synthetic-form-1")

	e.invalidCmd(t, "type outside the pattern", ev(c, "Service Form", sha, "x", 2, ex))
	e.invalidCmd(t, "sha256 too short", ev(c, "service-form", sha[:63], "x", 2, ex))
	e.invalidCmd(t, "sha256 upper-case", ev(c, "service-form", strings.ToUpper(sha), "x", 2, ex))
	e.invalidCmd(t, "sha256 not hex", ev(c, "service-form", strings.Repeat("g", 64), "x", 2, ex))
	e.invalidCmd(t, "empty label", ev(c, "service-form", sha, "", 2, ex))
	e.invalidCmd(t, "label over 200", ev(c, "service-form", sha, strings.Repeat("l", 201), 2, ex))
	e.refused(t, "type the service does not define", ev(c, "payment-receipt", sha, "x", 2, ex), result.Blocked, reasoncodes.ContractInvalid)
	e.refused(t, "not the assignee", ev(c, "service-form", sha, "x", 2, ex2), result.Unauthorized, reasoncodes.RowScopeDenied)
	e.refused(t, "supervisor", ev(c, "service-form", sha, "x", 2, sup), result.Unauthorized, reasoncodes.RowScopeDenied)

	k := key()
	first := ev(c, "service-form", sha, "signed form", 2, ex)
	first.IdempotencyKey = k
	r := e.cmd(t, first)
	expectSuccess(t, "required evidence", r, api.WorkInProgress, 3)
	if n := e.count(t, `SELECT count(*) FROM fulfillment.work_evidence WHERE case_id = $1 AND evidence_type = 'service-form' AND encode(content_sha256, 'hex') = $2
		AND submitted_by = $3::uuid AND command_id = $4::uuid`, c, sha, ex, r.CommandID); n != 1 {
		t.Fatalf("evidence row: %d", n)
	}
	if rp := e.cmd(t, first); !rp.Replayed || rp.CommandID != r.CommandID {
		t.Fatalf("replay: %+v", rp)
	}
	e.refused(t, "duplicate type + sha", ev(c, "service-form", sha, "again", 3, ex), result.Conflict, reasoncodes.ContractInvalid)
	// Duplicate detection is per (case, type, sha): another type or another case is a distinct record.
	expectSuccess(t, "same sha, optional type", e.cmd(t, ev(c, "site-photo", sha, "photo", 3, ex)), api.WorkInProgress, 4)
	expectSuccess(t, "same sha, other case", e.cmd(t, ev(other, "service-form", sha, "form", 2, ex)), api.WorkInProgress, 3)
	if n := e.count(t, `SELECT count(*) FROM fulfillment.work_evidence WHERE encode(content_sha256, 'hex') = $1`, sha); n != 3 {
		t.Fatalf("evidence rows: %d", n)
	}
	// After completion no evidence is accepted.
	expectSuccess(t, "complete", e.cmd(t, work.Command{Type: work.TypeComplete, CaseID: c, Outcome: catalogapi.OutcomeExecuted, ExpectedVersion: 4, Actor: actor(ex)}),
		api.WorkCompleted, 5)
	e.refused(t, "evidence after completion", ev(c, "site-photo", shaOf("late"), "late", 5, ex), result.Conflict, reasoncodes.StaleState)
}

// ---- D. completion -----------------------------------------------------------

func TestR74FulfillmentCompletion(t *testing.T) {
	e := setup(t)
	sup, ex := uid(t), uid(t)
	e.seed(t, "r74-co", 1, map[string]string{"r74-co.svc-01": sup}, map[string]string{"r74-co.svc-01": ex}, nil)
	e.seed(t, "r74-ex-only", 1, map[string]string{"r74-ex-only.svc-01": sup}, map[string]string{"r74-ex-only.svc-01": ex}, func(v *catalogapi.ServiceVersion) {
		v.CompletionOutcomes = []string{catalogapi.OutcomeExecuted}
	})
	e.seed(t, "r74-credit", 1, map[string]string{"r74-credit.svc-01": sup}, map[string]string{"r74-credit.svc-01": ex}, func(v *catalogapi.ServiceVersion) {
		v.FinancialBehaviour = catalogapi.FinancialConsumesCredit
		v.Evidence = []catalogapi.EvidenceRequirement{}
	})
	e.seed(t, "r74-ledger", 1, map[string]string{"r74-ledger.svc-01": sup}, map[string]string{"r74-ledger.svc-01": ex}, func(v *catalogapi.ServiceVersion) {
		v.FinancialBehaviour = catalogapi.FinancialChargesLedger
		v.Evidence = []catalogapi.EvidenceRequirement{}
	})
	assigned := func(product string) string {
		c := e.newCase(t, product)
		expectSuccess(t, "assign "+product, e.cmd(t, work.Command{Type: work.TypeAssign, CaseID: c, AssigneeUserID: ex, ExpectedVersion: 1, Actor: actor(sup)}),
			api.WorkAssigned, 2)
		return c
	}
	done := func(c, outcome string, reason *string, v int64) work.Command {
		return work.Command{Type: work.TypeComplete, CaseID: c, Outcome: outcome, Reason: reason, ExpectedVersion: v, Actor: actor(ex)}
	}
	completedEvents := func() int {
		return e.count(t, `SELECT count(*) FROM fulfillment.outbox_event WHERE event_type = $1`, api.EventWorkCaseCompleted)
	}

	c := assigned("r74-co")
	e.invalidCmd(t, "EXECUTED with a reason", done(c, catalogapi.OutcomeExecuted, ptr("x"), 2))
	e.invalidCmd(t, "NOT_EXECUTED without reason", done(c, catalogapi.OutcomeNotExecuted, nil, 2))
	e.invalidCmd(t, "NOT_EXECUTED empty reason", done(c, catalogapi.OutcomeNotExecuted, ptr(""), 2))
	e.invalidCmd(t, "NOT_EXECUTED reason over 1000", done(c, catalogapi.OutcomeNotExecuted, ptr(strings.Repeat("r", 1001)), 2))
	e.invalidCmd(t, "unknown outcome", done(c, "DONE", nil, 2))
	queued := e.newCase(t, "r74-co")
	e.refused(t, "QUEUED case (no assignee)", done(queued, catalogapi.OutcomeExecuted, nil, 1), result.Unauthorized, reasoncodes.RowScopeDenied)
	e.refused(t, "EXECUTED without required evidence", done(c, catalogapi.OutcomeExecuted, nil, 2), result.Blocked, reasoncodes.CoverageIncomplete,
		api.BlockEvidenceMissing)
	// The evidence rule precedes the version check.
	e.refused(t, "stale EXECUTED without evidence", done(c, catalogapi.OutcomeExecuted, nil, 9), result.Blocked, reasoncodes.CoverageIncomplete,
		api.BlockEvidenceMissing)
	expectSuccess(t, "evidence", e.cmd(t, work.Command{Type: work.TypeEvidence, CaseID: c, EvidenceType: "service-form", ContentSHA: shaOf("co"), Label: "form",
		ExpectedVersion: 2, Actor: actor(ex)}), api.WorkInProgress, 3)
	e.refused(t, "stale If-Match on a valid completion", done(c, catalogapi.OutcomeExecuted, nil, 2), result.Conflict, reasoncodes.StaleState)
	r := e.cmd(t, done(c, catalogapi.OutcomeExecuted, nil, 3))
	expectSuccess(t, "EXECUTED", r, api.WorkCompleted, 4)
	if completedEvents() != 1 || e.count(t, `SELECT count(*) FROM fulfillment.outbox_event WHERE event_type = $1 AND payload->>'case_id' = $2
		AND payload->>'outcome' = 'EXECUTED' AND payload->>'command_id' = $3`, api.EventWorkCaseCompleted, c, r.CommandID) != 1 {
		t.Fatal("exactly one completion event for the case")
	}
	e.refused(t, "repeated completion", done(c, catalogapi.OutcomeExecuted, nil, 4), result.Conflict, reasoncodes.StaleState)
	e.refused(t, "NOT_EXECUTED after completion", done(c, catalogapi.OutcomeNotExecuted, ptr("late"), 4), result.Conflict, reasoncodes.StaleState)
	if completedEvents() != 1 {
		t.Fatal("a refused completion emits nothing")
	}

	// NOT_EXECUTED needs no evidence and may close an ASSIGNED case directly.
	n := assigned("r74-co")
	expectSuccess(t, "NOT_EXECUTED from ASSIGNED", e.cmd(t, done(n, catalogapi.OutcomeNotExecuted, ptr("customer unreachable"), 2)), api.WorkNotExecuted, 3)
	if got := e.q(t, `SELECT detail->>'reason' FROM fulfillment.work_event WHERE case_id = $1 AND kind = 'COMPLETED'`, n); got != "customer unreachable" {
		t.Fatalf("reason kept in history: %s", got)
	}
	// An outcome the service version does not approve.
	x := assigned("r74-ex-only")
	e.refused(t, "outcome not approved by the service", done(x, catalogapi.OutcomeNotExecuted, ptr("x"), 2), result.Blocked, reasoncodes.ContractInvalid)
	// Financially restricted services. The production path never creates
	// their case: biawin activation refuses them first (D-35).
	for _, p := range []string{"r74-credit", "r74-ledger"} {
		customer := uid(t)
		o := e.activate(t, paid(t, p, biawinapi.RouteAdditionalServices, "RESOLVED", &customer))
		if o.SubscriptionID != nil || !strings.Contains(strings.Join(o.BlockedBy, ","), "D-35") {
			t.Fatalf("%s: activation of a financial service must be refused: %+v", p, o)
		}
	}
	e.relay(t)
	if n := e.count(t, `SELECT count(*) FROM fulfillment.work_case WHERE service_key IN ('r74-credit.svc-01', 'r74-ledger.svc-01')`); n != 0 {
		t.Fatalf("no case for a refused activation: %d", n)
	}
	// Defence in depth: an item event delivered straight to the fulfillment
	// inbox (fixture; the real consumer code path) still cannot complete.
	for _, p := range []string{"r74-credit", "r74-ledger"} {
		f := e.injectItem(t, p+".svc-01")
		expectSuccess(t, "assign "+p, e.cmd(t, work.Command{Type: work.TypeAssign, CaseID: f, AssigneeUserID: ex, ExpectedVersion: 1, Actor: actor(sup)}),
			api.WorkAssigned, 2)
		e.refused(t, p+" EXECUTED", done(f, catalogapi.OutcomeExecuted, nil, 2), result.Blocked, reasoncodes.PolicyUnresolved, api.BlockD35, api.BlockD38)
		e.refused(t, p+" NOT_EXECUTED", done(f, catalogapi.OutcomeNotExecuted, ptr("x"), 2), result.Blocked, reasoncodes.PolicyUnresolved, api.BlockD35, api.BlockD38)
	}
	if completedEvents() != 2 {
		t.Fatalf("completion events: %d", completedEvents())
	}
}

// ---- D/7. provider-dependent completion and the provider boundary -------------

// providerReader fails on demand (an unavailable integration read).
type providerReader struct{ err error }

func (p providerReader) Confirmed(context.Context, string) (bool, error) { return false, p.err }

func TestR74FulfillmentProviderBoundary(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	sup, ex := uid(t), uid(t)
	e.seed(t, "r74-pv", 1, map[string]string{"r74-pv.svc-01": sup}, map[string]string{"r74-pv.svc-01": ex}, func(v *catalogapi.ServiceVersion) {
		v.IntegrationProvider = catalogapi.ProviderBiawin
		v.Evidence = []catalogapi.EvidenceRequirement{}
	})
	port := integrationbiawin.NewSynthetic() // TEST-ONLY synthetic adapter
	secret := []byte("synthetic-r7-4-callback-secret")
	assigned := func() string {
		c := e.newCase(t, "r74-pv")
		expectSuccess(t, "assign", e.cmd(t, work.Command{Type: work.TypeAssign, CaseID: c, AssigneeUserID: ex, ExpectedVersion: 1, Actor: actor(sup)}), api.WorkAssigned, 2)
		return c
	}
	request := func(c, k string) providerops.Operation {
		op, _, err := e.ops.RequestOperation(ctx, providerops.OperationRequest{OperationKey: "case:" + c + "|op:" + k, Provider: "BIAWIN",
			OpType: providerops.OpServiceOrder, SubjectCaseID: &c, CorrelationID: "corr-r74", CausationID: "r74-" + k})
		if err != nil {
			t.Fatal(err)
		}
		return op
	}
	execd := func(c string) work.Command {
		return work.Command{Type: work.TypeComplete, CaseID: c, Outcome: catalogapi.OutcomeExecuted, ExpectedVersion: 2, Actor: actor(ex)}
	}
	blocked := func(label, c string) {
		t.Helper()
		e.refused(t, label, execd(c), result.Blocked, reasoncodes.DependencyUnknown, api.BlockProviderConfirmation)
	}

	c := assigned()
	blocked("no provider operation", c)
	op := request(c, "SERVICE_ORDER")
	blocked("PREPARED", c)
	// The production (disabled) port sends nothing.
	if _, err := e.ops.Dispatch(ctx, integrationbiawin.DisabledPort{}, op.OperationID); !errors.Is(err, providerops.ErrPortDisabled) {
		t.Fatalf("disabled port: %v", err)
	}
	blocked("PREPARED after a disabled dispatch", c)
	ack, err := e.ops.Dispatch(ctx, port, op.OperationID)
	if err != nil || ack.State != providerops.StateAcknowledged {
		t.Fatalf("synthetic ack: %+v %v", ack, err)
	}
	blocked("ACKNOWLEDGED is not CONFIRMED", c)

	// UNKNOWN (timeout after a possible creation) and RECONCILIATION_REQUIRED stay blocked.
	u := assigned()
	opU := request(u, "SERVICE_ORDER")
	port.Script(opU.Reference, integrationbiawin.BehaveTimeoutCreated)
	if got, _ := e.ops.Dispatch(ctx, port, opU.OperationID); got.State != providerops.StateUnknown {
		t.Fatalf("unknown: %+v", got)
	}
	blocked("UNKNOWN", u)
	rq := assigned()
	opR := request(rq, "SERVICE_ORDER")
	port.Script(opR.Reference, integrationbiawin.BehaveTimeoutNoLookup)
	e.ops.Dispatch(ctx, port, opR.OperationID)
	rec, _ := e.ops.Reconcile(ctx, port, opR.OperationID)
	if rec.State != providerops.StateReconciliation {
		t.Fatalf("reconciliation: %+v", rec)
	}
	blocked("RECONCILIATION_REQUIRED", rq)

	// Reconciliation restrictions (human resolution).
	resolve := func(id string, v int64, present bool, ref string) error {
		_, err := e.ops.Resolve(ctx, providerops.Resolution{OperationID: id, Present: present, ExternalRef: ref, EvidenceNote: "synthetic operator note",
			ExpectedVersion: v, ActorUserID: sup})
		return err
	}
	cur, _ := e.ops.Get(ctx, op.OperationID)
	if err := resolve(op.OperationID, cur.Version, true, "x"); !errors.Is(err, providerops.ErrState) {
		t.Fatalf("resolving an ACKNOWLEDGED operation: %v", err)
	}
	if err := resolve(opR.OperationID, rec.Version+7, true, "x"); !errors.Is(err, providerops.ErrState) {
		t.Fatalf("stale resolution: %v", err)
	}
	if err := resolve(opR.OperationID, rec.Version, true, ""); !errors.Is(err, providerops.ErrInvalid) {
		t.Fatalf("present without reference: %v", err)
	}
	if err := resolve(opR.OperationID, rec.Version, false, "x"); !errors.Is(err, providerops.ErrInvalid) {
		t.Fatalf("absent with reference: %v", err)
	}
	if err := resolve("00000000-0000-4000-8000-0000000000b4", 1, true, "x"); !errors.Is(err, providerops.ErrNotFound) {
		t.Fatalf("unknown operation: %v", err)
	}
	blocked("RECONCILIATION_REQUIRED after refused resolutions", rq)

	// Unavailable / absent confirmation reader: never a success.
	noReader, err := work.New(e.fR, e.cat, nil, e.clock)
	if err != nil {
		t.Fatal(err)
	}
	before := e.fingerprint(t)
	r, err := noReader.Execute(ctx, rollout.Controlled, work.Command{Type: work.TypeComplete, CaseID: c, Outcome: catalogapi.OutcomeExecuted, ExpectedVersion: 2,
		IdempotencyKey: key(), CorrelationID: "c", Actor: actor(ex)})
	if err != nil {
		t.Fatal(err)
	}
	expectRefusal(t, "no confirmation reader", r, result.Blocked, reasoncodes.DependencyUnknown, api.BlockProviderConfirmation)
	boom := errors.New("synthetic integration read outage")
	failing, _ := work.New(e.fR, e.cat, providerReader{err: boom}, e.clock)
	if _, err := failing.Execute(ctx, rollout.Controlled, work.Command{Type: work.TypeComplete, CaseID: c, Outcome: catalogapi.OutcomeExecuted, ExpectedVersion: 2,
		IdempotencyKey: key(), CorrelationID: "c", Actor: actor(ex)}); !errors.Is(err, boom) {
		t.Fatalf("an unavailable confirmation read is an error, never SUCCESS: %v", err)
	}
	sameState(t, "unavailable provider read", before, e.fingerprint(t))

	// A confirmation of ANOTHER case never unblocks this one.
	other := assigned()
	opO := request(other, "SERVICE_ORDER")
	e.ops.Dispatch(ctx, port, opO.OperationID)
	curO, _ := e.ops.Get(ctx, opO.OperationID)
	if d, err := e.ops.HandleCallback(ctx, secret, signedCallback(t, secret, providerops.CallbackEnvelope{CallbackID: "r74-cb-other", Reference: opO.Reference,
		ExternalRef: *curO.ExternalRef, Status: "CONFIRMED"})); err != nil || d != "APPLIED" {
		t.Fatalf("confirm other: %s %v", d, err)
	}
	blocked("confirmation belongs to another case", c)
	// NOT_EXECUTED is not provider-gated.
	expectSuccess(t, "NOT_EXECUTED without confirmation", e.cmd(t, work.Command{Type: work.TypeComplete, CaseID: u, Outcome: catalogapi.OutcomeNotExecuted,
		Reason: ptr("provider unavailable"), ExpectedVersion: 2, Actor: actor(ex)}), api.WorkNotExecuted, 3)

	// Confirmed via the synthetic signed callback → completion allowed.
	cb := signedCallback(t, secret, providerops.CallbackEnvelope{CallbackID: "r74-cb-1", Reference: op.Reference, ExternalRef: *ack.ExternalRef, Status: "CONFIRMED"})
	if d, err := e.ops.HandleCallback(ctx, secret, cb); err != nil || d != "APPLIED" {
		t.Fatalf("confirm: %s %v", d, err)
	}
	// Duplicate and stale callbacks never change a final operation.
	if d, _ := e.ops.HandleCallback(ctx, secret, cb); d != "DUPLICATE" {
		t.Fatalf("duplicate: %s", d)
	}
	late := signedCallback(t, secret, providerops.CallbackEnvelope{CallbackID: "r74-cb-late", Reference: op.Reference, ExternalRef: *ack.ExternalRef, Status: "REJECTED"})
	if d, _ := e.ops.HandleCallback(ctx, secret, late); d != "ALREADY_FINAL" {
		t.Fatalf("stale REJECTED after CONFIRMED: %s", d)
	}
	reused := signedCallback(t, secret, providerops.CallbackEnvelope{CallbackID: "r74-cb-1", Reference: op.Reference, ExternalRef: *ack.ExternalRef, Status: "REJECTED"})
	if d, _ := e.ops.HandleCallback(ctx, secret, reused); d != "ID_CONFLICT" {
		t.Fatalf("same callback id, other content: %s", d)
	}
	unmatched := signedCallback(t, secret, providerops.CallbackEnvelope{CallbackID: "r74-cb-unmatched", Reference: "no-such-reference", ExternalRef: "x", Status: "CONFIRMED"})
	if d, _ := e.ops.HandleCallback(ctx, secret, unmatched); d != "UNMATCHED" {
		t.Fatalf("unmatched: %s", d)
	}
	if got, _ := e.ops.Get(ctx, op.OperationID); got.State != providerops.StateConfirmed {
		t.Fatalf("final state changed: %+v", got)
	}
	expectSuccess(t, "EXECUTED after CONFIRMED", e.cmd(t, execd(c)), api.WorkCompleted, 3)

	// A REJECTED provider answer never unblocks completion.
	rj := assigned()
	opJ := request(rj, "SERVICE_ORDER")
	if d, _ := e.ops.HandleCallback(ctx, secret, signedCallback(t, secret, providerops.CallbackEnvelope{CallbackID: "r74-cb-rej", Reference: opJ.Reference,
		ExternalRef: "syn-rej", Status: "REJECTED"})); d != "EARLY_APPLIED" {
		t.Fatalf("rejected callback: %s", d)
	}
	blocked("provider REJECTED", rj)
	// The synthetic port was never asked to do more than this test scripted.
	if port.Calls(op.Reference) != 1 || port.Calls(opJ.Reference) != 0 {
		t.Fatalf("unexpected provider calls: %d %d", port.Calls(op.Reference), port.Calls(opJ.Reference))
	}
}

// ---- E. concurrency ----------------------------------------------------------

// race runs n commands at once (independent pooled connections) behind a barrier.
func (e *env) race(t *testing.T, cmds []work.Command) []work.Result {
	t.Helper()
	out := make([]work.Result, len(cmds))
	errs := make([]error, len(cmds))
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range cmds {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			out[i], errs[i] = e.work.Execute(context.Background(), rollout.Controlled, cmds[i])
		}(i)
	}
	close(start)
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("command %d: %v", i, err)
		}
	}
	return out
}

func tally(rs []work.Result) (wins []work.Result, refusals []work.Result) {
	for _, r := range rs {
		if r.Status == result.Success {
			wins = append(wins, r)
		} else {
			refusals = append(refusals, r)
		}
	}
	return wins, refusals
}

func TestR74FulfillmentConcurrency(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	sup, sup2, ex, ex2 := uid(t), uid(t), uid(t), uid(t)
	e.seed(t, "r74-cc", 1, map[string]string{"r74-cc.svc-01": sup}, map[string]string{"r74-cc.svc-01": ex}, nil)
	e.grant(t, "r74-cc.svc-01", sup2, catalogapi.ScopeSupervisor, "GRANT")
	e.grant(t, "r74-cc.svc-01", ex2, catalogapi.ScopeExecutor, "GRANT")
	const n = 8
	cmd := func(c work.Command, k string) work.Command {
		c.IdempotencyKey, c.CorrelationID = k, "corr-race"
		return c
	}

	// E1: two supervisors assign different executors to the same v1 case.
	c := e.newCase(t, "r74-cc")
	var cmds []work.Command
	for i := 0; i < n; i++ {
		who, to := sup, ex
		if i%2 == 1 {
			who, to = sup2, ex2
		}
		cmds = append(cmds, cmd(work.Command{Type: work.TypeAssign, CaseID: c, AssigneeUserID: to, ExpectedVersion: 1, Actor: actor(who)}, fmt.Sprintf("race-assign-%02d", i)))
	}
	wins, lost := tally(e.race(t, cmds))
	if len(wins) != 1 {
		t.Fatalf("E1: %d winners", len(wins))
	}
	for _, r := range lost {
		expectRefusal(t, "E1 loser", r, result.Conflict, reasoncodes.StaleState)
	}
	if winner := wins[0]; winner.CaseVersion != 2 || winner.State != api.WorkAssigned {
		t.Fatalf("E1 winner: %+v", winner)
	}
	if e.count(t, `SELECT count(*) FROM fulfillment.work_event WHERE case_id = $1 AND kind = 'ASSIGNED'`, c) != 1 ||
		e.count(t, `SELECT count(*) FROM fulfillment.work_assignment WHERE case_id = $1`, c) != 1 ||
		e.count(t, `SELECT count(*) FROM fulfillment.work_command_record WHERE case_id = $1`, c) != 1 || !strings.HasSuffix(e.caseState(t, c), "|v2") {
		t.Fatalf("E1: exactly one authoritative transition: %s", e.caseState(t, c))
	}
	assignee := strings.Split(e.caseState(t, c), "|")[1]

	// E2: the assignee submits n actions against the same version at once.
	cmds = nil
	for i := 0; i < n; i++ {
		cmds = append(cmds, cmd(work.Command{Type: work.TypeAction, CaseID: c, Action: api.ActionNoAnswer, ExpectedVersion: 2, Actor: actor(assignee)},
			fmt.Sprintf("race-action-%02d", i)))
	}
	wins, lost = tally(e.race(t, cmds))
	if len(wins) != 1 || wins[0].CaseVersion != 3 {
		t.Fatalf("E2: %d winners", len(wins))
	}
	for _, r := range lost {
		expectRefusal(t, "E2 loser", r, result.Conflict, reasoncodes.StaleState)
	}
	if e.count(t, `SELECT count(*) FROM fulfillment.work_event WHERE case_id = $1 AND kind = 'ACTION_RECORDED'`, c) != 1 {
		t.Fatal("E2: one action event")
	}

	// E3: two completions (EXECUTED and NOT_EXECUTED) against the same version.
	expectSuccess(t, "evidence", e.cmd(t, work.Command{Type: work.TypeEvidence, CaseID: c, EvidenceType: "service-form", ContentSHA: shaOf("race"), Label: "f",
		ExpectedVersion: 3, Actor: actor(assignee)}), api.WorkInProgress, 4)
	cmds = nil
	for i := 0; i < n; i++ {
		x := work.Command{Type: work.TypeComplete, CaseID: c, Outcome: catalogapi.OutcomeExecuted, ExpectedVersion: 4, Actor: actor(assignee)}
		if i%2 == 1 {
			x.Outcome, x.Reason = catalogapi.OutcomeNotExecuted, ptr("race")
		}
		cmds = append(cmds, cmd(x, fmt.Sprintf("race-complete-%02d", i)))
	}
	wins, lost = tally(e.race(t, cmds))
	if len(wins) != 1 {
		t.Fatalf("E3: %d winners", len(wins))
	}
	for _, r := range lost {
		expectRefusal(t, "E3 loser", r, result.Conflict, reasoncodes.StaleState)
	}
	if e.count(t, `SELECT count(*) FROM fulfillment.outbox_event WHERE event_type = $1 AND payload->>'case_id' = $2`, api.EventWorkCaseCompleted, c) != 1 ||
		e.count(t, `SELECT count(*) FROM fulfillment.work_event WHERE case_id = $1 AND kind = 'COMPLETED'`, c) != 1 || !strings.HasSuffix(e.caseState(t, c), "|v5") {
		t.Fatal("E3: one completion, one completion event, version 5")
	}

	// E4: the same key and the same command n times at once: one execution, n-1 replays.
	c4 := e.newCase(t, "r74-cc")
	same := cmd(work.Command{Type: work.TypeAssign, CaseID: c4, AssigneeUserID: ex, ExpectedVersion: 1, Actor: actor(sup)}, "race-same-key-0001")
	cmds = nil
	for i := 0; i < n; i++ {
		cmds = append(cmds, same)
	}
	rs := e.race(t, cmds)
	executed := 0
	for _, r := range rs {
		if r.Status != result.Success || r.CommandID != rs[0].CommandID || r.CaseVersion != 2 {
			t.Fatalf("E4: every caller sees the one result: %+v", r)
		}
		if !r.Replayed {
			executed++
		}
	}
	if executed != 1 || e.count(t, `SELECT count(*) FROM fulfillment.work_command_record WHERE case_id = $1`, c4) != 1 ||
		e.count(t, `SELECT count(*) FROM fulfillment.work_event WHERE case_id = $1 AND kind = 'ASSIGNED'`, c4) != 1 {
		t.Fatalf("E4: executed %d times", executed)
	}

	// E5: the same key with different payloads at once: one wins, the others CONFLICT (never replayed).
	c5 := e.newCase(t, "r74-cc")
	cmds = nil
	for i := 0; i < n; i++ {
		to := ex
		if i%2 == 1 {
			to = ex2
		}
		cmds = append(cmds, cmd(work.Command{Type: work.TypeAssign, CaseID: c5, AssigneeUserID: to, ExpectedVersion: 1, Actor: actor(sup)}, "race-diff-key-0001"))
	}
	rs = e.race(t, cmds)
	wins, lost = tally(rs)
	if len(wins) == 0 {
		t.Fatal("E5: no winner")
	}
	for _, w := range wins {
		if w.CommandID != wins[0].CommandID {
			t.Fatal("E5: two different executions under one key")
		}
	}
	winTo := strings.Split(e.caseState(t, c5), "|")[1]
	for i, r := range rs {
		if r.Status == result.Success && cmds[i].AssigneeUserID != winTo {
			t.Fatalf("E5: a different payload was answered with the winner's result: %+v", r)
		}
	}
	for _, r := range lost {
		expectRefusal(t, "E5 loser", r, result.Conflict, reasoncodes.ContractInvalid)
	}
	if e.count(t, `SELECT count(*) FROM fulfillment.work_command_record WHERE case_id = $1`, c5) != 1 {
		t.Fatal("E5: one command record")
	}

	// E6: concurrent intake of one item — the same message redelivered and a
	// second message for the same item — creates exactly one case.
	item, sub, cust := uid(t), uid(t), uid(t)
	payload, _ := json.Marshal(biawinapi.SubscriptionItemActivated{ActivationID: uid(t), SubscriptionID: sub, ItemID: item, Seq: 1, ServiceKey: "r74-cc.svc-01",
		ServiceVersion: 1, CustomerID: cust, RolloutMode: "SHADOW_COMPARE", CorrelationID: "corr-intake", CausationID: "r74", OccurredAt: e.clock.Now().Time()})
	msgA, msgB := uid(t), uid(t)
	recs := []outbox.Record{}
	for i := 0; i < n; i++ {
		id := msgA
		if i%2 == 1 {
			id = msgB
		}
		recs = append(recs, outbox.Record{Event: outbox.Event{ID: id, Type: biawinapi.EventItemActivated, Version: 1, Payload: payload, OccurredAt: e.clock.Now()}})
	}
	outs := make([]work.IntakeOutcome, n)
	errs := make([]error, n)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range recs {
		wg.Add(1)
		go func(i int) { defer wg.Done(); <-start; outs[i], errs[i] = e.work.OnItemActivated(ctx, recs[i]) }(i)
	}
	close(start)
	wg.Wait()
	created := 0
	for i, err := range errs {
		if err != nil {
			t.Fatalf("E6 intake %d: %v", i, err)
		}
		if outs[i].Decision == "CASE_CREATED" && !outs[i].Duplicate {
			created++
		}
	}
	if created != 1 || e.count(t, `SELECT count(*) FROM fulfillment.work_case WHERE item_id = $1`, item) != 1 ||
		e.count(t, `SELECT count(*) FROM fulfillment.work_event WHERE case_id = $1 AND kind = 'CREATED'`, work.CaseID(item)) != 1 ||
		e.count(t, `SELECT count(*) FROM fulfillment.outbox_event WHERE event_type = $1 AND payload->>'item_id' = $2`, api.EventWorkCaseCreated, item) != 1 ||
		e.count(t, `SELECT count(*) FROM fulfillment.work_intake WHERE item_id = $1`, item) != 2 {
		t.Fatalf("E6: created=%d", created)
	}
}

// Outcome unknown: an idempotency record left in progress (simulated by a
// fixture update, as after an interrupted writer) is never executed blindly.
func TestR74FulfillmentOutcomeUnknownIsNotRetriedBlindly(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	sup, ex := uid(t), uid(t)
	e.seed(t, "r74-ou", 1, map[string]string{"r74-ou.svc-01": sup}, map[string]string{"r74-ou.svc-01": ex}, nil)
	c := e.newCase(t, "r74-ou")
	k := key()
	cmdA := work.Command{Type: work.TypeAssign, CaseID: c, AssigneeUserID: ex, ExpectedVersion: 1, IdempotencyKey: k, Actor: actor(sup)}
	expectSuccess(t, "assign", e.cmd(t, cmdA), api.WorkAssigned, 2)
	if _, err := e.admin(t).Exec(ctx, `UPDATE fulfillment.idempotency_record SET state = 'in_progress', completed_at = NULL, result = NULL
		WHERE scope = $1 AND idem_key = $2`, work.TypeAssign, k); err != nil {
		t.Fatal(err)
	}
	r := e.refused(t, "in-progress key", cmdA, result.Unknown, reasoncodes.OutcomeUnknown)
	if r.Mode != rollout.Controlled {
		t.Fatalf("mode: %+v", r)
	}
}
