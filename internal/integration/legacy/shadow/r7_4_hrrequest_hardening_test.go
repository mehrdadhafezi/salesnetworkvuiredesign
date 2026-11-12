package shadow_test

// R7-4 HR request (Slice 11) domain hardening on the synthetic writes
// scenario (disposable PostgreSQL):
//
//	2 manager (no parent) ─┬─ 3 senior ─┬─ 4 supervisor A ─┬─ 6 seller X
//	                       │            │                  ├─ 7 seller Y
//	                       │            │                  └─ 10 seller W (unmapped status)
//	                       │            └─ 5 supervisor B
//	                       └─ 8 supervisor C ── 9 seller Z            1 = HR
//
// BACKEND-ONLY COVERAGE: the service is called with explicit synthetic
// actors; the HTTP test uses the TEST-ONLY testSubjects overlay (a direct
// synthetic grant). None of it is evidence of production authorization: no
// role bundle carries organization.hr_request.* and D-42 / R5-SEC-01 are
// open. Person-level separation of duties cannot be proven (no person
// identity model) and is reported BLOCKED, not tested.
//
// Hierarchy edge cases are produced by fixture updates of the imported
// organization evidence in the disposable database (as a later legacy
// import would carry them), each restored and proven restored afterwards.
// The service never writes that evidence.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"

	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/hrrequest"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/rollout"
)

// hrOtherSchemas must never change because of an HR request command.
var hrOtherSchemas = []string{"identity", "custody", "cases", "billing", "payments", "finance", "commission", "integration", "fulfillment", "biawin"}

// hrEvidenceTables is the organization evidence an HR request must never write.
var hrEvidenceTables = []string{"organization.person", "organization.employment", "organization.org_assignment", "organization.position",
	"organization.level", "organization.org_unit"}

// hrRequestTables are everything an HR request command may write.
var hrRequestTables = []string{"organization.hr_request", "organization.hr_request_event", "organization.command_record", "organization.outbox_event",
	"organization.idempotency_record"}

type hrGuard struct {
	others   map[string]tableState
	evidence string
}

func (e *writesEnv) hrGuard(t *testing.T) hrGuard {
	t.Helper()
	return hrGuard{others: schemaSnapshot(t, e.admin, hrOtherSchemas), evidence: e.md5(t, hrEvidenceTables...)}
}

// noSideEffects proves no other module and no organization evidence changed.
func (e *writesEnv) noSideEffects(t *testing.T, label string, g hrGuard) {
	t.Helper()
	for _, d := range diffFinancial(g.others, schemaSnapshot(t, e.admin, hrOtherSchemas)) {
		t.Errorf("%s: %s (an HR request touched another module)", label, d)
	}
	if e.md5(t, hrEvidenceTables...) != g.evidence {
		t.Errorf("%s: organization evidence (employment / hierarchy) changed", label)
	}
}

func expectHR(t *testing.T, label string, r hrrequest.Result, st result.Status, code reasoncodes.Code, blockedBy ...string) {
	t.Helper()
	if r.Status != st || r.StatusName != st.String() || !reflect.DeepEqual(r.Reasons, []reasoncodes.Code{code}) {
		t.Fatalf("%s: got %s %v, want %s [%s] (%+v)", label, r.StatusName, r.Reasons, st, code, r)
	}
	if len(blockedBy) == 0 {
		blockedBy = nil
	}
	if !reflect.DeepEqual(r.BlockedBy, blockedBy) {
		t.Fatalf("%s: blocked_by %v, want %v", label, r.BlockedBy, blockedBy)
	}
	if r.Applied || r.CommandID != "" || r.RequestID != "" || r.WritebackState != hrrequest.WritebackNotRequested {
		t.Fatalf("%s: a refusal applies, records and writes back nothing: %+v", label, r)
	}
}

type hrHarness struct {
	*writesEnv
	t   *testing.T
	seq int
}

func newHR(t *testing.T) *hrHarness { return &hrHarness{writesEnv: setupWrites(t), t: t} }

func (h *hrHarness) key(tag string) string { h.seq++; return fmt.Sprintf("r74-hr-%s-%04d", tag, h.seq) }

func (h *hrHarness) create(actor, typ, subject string, target *string) hrrequest.Create {
	return hrrequest.Create{RequestType: typ, SubjectUserID: subject, TargetParentUserID: target, Reason: "synthetic r7-4 reason",
		IdempotencyKey: h.key("c"), CorrelationID: "corr-r74-hr", Actor: h.hrActor(actor)}
}

func (h *hrHarness) decide(actor, id, decision string, v int64) hrrequest.Decide {
	return hrrequest.Decide{RequestID: id, Decision: decision, ExpectedVersion: v, IdempotencyKey: h.key("d"), CorrelationID: "corr-r74-hr", Actor: h.hrActor(actor)}
}

func (h *hrHarness) doCreate(c hrrequest.Create) hrrequest.Result {
	h.t.Helper()
	r, err := h.hr.CreateRequest(context.Background(), rollout.Controlled, c)
	if err != nil {
		h.t.Fatalf("create: %v", err)
	}
	return r
}

func (h *hrHarness) doDecide(d hrrequest.Decide) hrrequest.Result {
	h.t.Helper()
	r, err := h.hr.DecideRequest(context.Background(), rollout.Controlled, d)
	if err != nil {
		h.t.Fatalf("decide: %v", err)
	}
	return r
}

// refusedCreate / refusedDecide assert the exact refusal and that no request table changed.
func (h *hrHarness) refusedCreate(label string, c hrrequest.Create, st result.Status, code reasoncodes.Code, by ...string) {
	h.t.Helper()
	before := h.md5(h.t, hrRequestTables...)
	expectHR(h.t, label, h.doCreate(c), st, code, by...)
	if h.md5(h.t, hrRequestTables...) != before {
		h.t.Fatalf("%s: a refusal wrote a request, event, record, outbox row or idempotency key", label)
	}
}

func (h *hrHarness) refusedDecide(label string, d hrrequest.Decide, st result.Status, code reasoncodes.Code, by ...string) hrrequest.Result {
	h.t.Helper()
	before := h.md5(h.t, hrRequestTables...)
	r := h.doDecide(d)
	expectHR(h.t, label, r, st, code, by...)
	if h.md5(h.t, hrRequestTables...) != before {
		h.t.Fatalf("%s: a refusal wrote a request, event, record, outbox row or idempotency key", label)
	}
	return r
}

func (h *hrHarness) invalidCreate(label string, c hrrequest.Create) {
	h.t.Helper()
	before := h.md5(h.t, hrRequestTables...)
	if _, err := h.hr.CreateRequest(context.Background(), rollout.Controlled, c); !errors.Is(err, hrrequest.ErrInvalid) {
		h.t.Fatalf("%s: want ErrInvalid, got %v", label, err)
	}
	if h.md5(h.t, hrRequestTables...) != before {
		h.t.Fatalf("%s: an invalid command wrote something", label)
	}
}

func (h *hrHarness) invalidDecide(label string, d hrrequest.Decide) {
	h.t.Helper()
	if _, err := h.hr.DecideRequest(context.Background(), rollout.Controlled, d); !errors.Is(err, hrrequest.ErrInvalid) {
		h.t.Fatalf("%s: want ErrInvalid, got %v", label, err)
	}
}

func okCreate(t *testing.T, label string, r hrrequest.Result, status string) {
	t.Helper()
	if r.Status != result.Success || !r.Applied || r.RequestStatus != status || r.Version != 1 || r.RequestID == "" || r.CommandID == "" ||
		r.WritebackState != hrrequest.WritebackPending || r.Replayed {
		t.Fatalf("%s: %+v", label, r)
	}
}

func okDecide(t *testing.T, label string, r hrrequest.Result, status string, version int64) {
	t.Helper()
	if r.Status != result.Success || !r.Applied || r.RequestStatus != status || r.Version != version || r.WritebackState != hrrequest.WritebackPending || r.Replayed {
		t.Fatalf("%s: %+v", label, r)
	}
}

// fixture runs an update of the imported organization evidence and returns
// the statement that restores the exact previous values.
func (h *hrHarness) fixture(sql string, args ...any) {
	h.t.Helper()
	if _, err := h.admin.Exec(context.Background(), sql, args...); err != nil {
		h.t.Fatalf("fixture: %v", err)
	}
}

func (h *hrHarness) empRow(user string) (status string, active *bool, link string) {
	h.t.Helper()
	if err := h.admin.QueryRow(context.Background(), `SELECT employment_status, is_active, user_link_state FROM organization.employment WHERE user_id = $1`,
		user).Scan(&status, &active, &link); err != nil {
		h.t.Fatal(err)
	}
	return
}

// withEmployment changes one user's employment evidence for fn and restores it.
func (h *hrHarness) withEmployment(user, status string, active *bool, link string, fn func()) {
	h.t.Helper()
	s0, a0, l0 := h.empRow(user)
	h.fixture(`UPDATE organization.employment SET employment_status = $2, is_active = $3, user_link_state = $4 WHERE user_id = $1`, user, status, active, link)
	fn()
	h.fixture(`UPDATE organization.employment SET employment_status = $2, is_active = $3, user_link_state = $4 WHERE user_id = $1`, user, s0, a0, l0)
}

// withLink changes the current-flag or parent of one REPORTS_TO link (by
// legacy assignment key) for fn and restores it.
func (h *hrHarness) withLink(sourceKey string, current *bool, parentUser *string, fn func()) {
	h.t.Helper()
	var cur0 *bool
	var parent0 string
	if err := h.admin.QueryRow(context.Background(), `SELECT is_current, parent_employment_id::text FROM organization.org_assignment WHERE source_key = $1`,
		sourceKey).Scan(&cur0, &parent0); err != nil {
		h.t.Fatal(err)
	}
	parent := parent0
	if parentUser != nil {
		if err := h.admin.QueryRow(context.Background(), `SELECT employment_id::text FROM organization.employment WHERE user_id = $1`, *parentUser).Scan(&parent); err != nil {
			h.t.Fatal(err)
		}
	}
	h.fixture(`UPDATE organization.org_assignment SET is_current = $2, parent_employment_id = $3::uuid WHERE source_key = $1`, sourceKey, current, parent)
	fn()
	h.fixture(`UPDATE organization.org_assignment SET is_current = $2, parent_employment_id = $3::uuid WHERE source_key = $1`, sourceKey, cur0, parent0)
}

func bptr(b bool) *bool { return &b }

const (
	hrHRBlock = hrrequest.DecisionHRRuntime
)

// ---- A. request creation -----------------------------------------------------

func TestHRRequestR74Creation(t *testing.T) {
	h := newHR(t)
	u := h.users
	ptr := func(s string) *string { return &s }
	g := h.hrGuard(t)
	transfer, terminate := api.HRTransferSeller, api.HRTerminateSeller

	// Contract errors (400): exact lowercase enum, field pairing, sizes.
	for label, c := range map[string]hrrequest.Create{
		"upper-case enum":         h.create("4", "TERMINATE_SELLER", u["7"], nil),
		"transfer without target": h.create("3", transfer, u["6"], nil),
		"terminate with a target": h.create("4", terminate, u["7"], ptr(u["5"])),
		"subject not a uuid":      h.create("4", terminate, "seller-7", nil),
		"target not a uuid":       h.create("3", transfer, u["6"], ptr("supervisor-b")),
		"missing reason":          func() hrrequest.Create { c := h.create("4", terminate, u["7"], nil); c.Reason = ""; return c }(),
		"reason over 1000": func() hrrequest.Create {
			c := h.create("4", terminate, u["7"], nil)
			c.Reason = strings.Repeat("r", 1001)
			return c
		}(),
		"short idempotency key": func() hrrequest.Create {
			c := h.create("4", terminate, u["7"], nil)
			c.IdempotencyKey = "short"
			return c
		}(),
		"missing correlation id": func() hrrequest.Create { c := h.create("4", terminate, u["7"], nil); c.CorrelationID = ""; return c }(),
		"correlation id over 128": func() hrrequest.Create {
			c := h.create("4", terminate, u["7"], nil)
			c.CorrelationID = strings.Repeat("c", 129)
			return c
		}(),
		"unknown request type":       h.create("4", "promote_seller", u["7"], nil),
		"transfer with empty target": h.create("3", transfer, u["6"], ptr("")),
	} {
		h.invalidCreate(label, c)
	}

	// Authority and scope refusals.
	h.refusedCreate("requester is the subject", h.create("4", terminate, u["4"], nil), result.Unauthorized, reasoncodes.RowScopeDenied)
	h.refusedCreate("subject is not a seller", h.create("3", terminate, u["4"], nil), result.Unauthorized, reasoncodes.RowScopeDenied)
	h.refusedCreate("seller outside the requester's hierarchy", h.create("5", terminate, u["6"], nil), result.Unauthorized, reasoncodes.RowScopeDenied)
	h.refusedCreate("sibling supervisor's seller", h.create("8", terminate, u["7"], nil), result.Unauthorized, reasoncodes.RowScopeDenied)
	h.refusedCreate("seller as requester", h.create("6", terminate, u["7"], nil), result.Unauthorized, reasoncodes.ActionDenied)
	h.refusedCreate("HR as requester", h.create("1", terminate, u["6"], nil), result.Blocked, reasoncodes.PolicyUnresolved, hrHRBlock, hrrequest.DecisionOPD05)
	h.refusedCreate("unknown subject", h.create("4", terminate, "00000000-0000-4000-8000-0000000000c4", nil), result.Blocked, reasoncodes.LinkageConflict)
	h.refusedCreate("unknown requester", hrrequest.Create{RequestType: terminate, SubjectUserID: u["7"], Reason: "x", IdempotencyKey: h.key("c"),
		CorrelationID: "c", Actor: hrrequest.Actor{Principal: "synthetic:unknown", UserID: "00000000-0000-4000-8000-0000000000c5"}},
		result.Blocked, reasoncodes.LinkageConflict)
	h.refusedCreate("unmapped subject status", h.create("4", terminate, u["10"], nil), result.Blocked, reasoncodes.LinkageConflict)
	h.refusedCreate("unknown destination", h.create("3", transfer, u["6"], ptr("00000000-0000-4000-8000-0000000000c6")), result.Blocked, reasoncodes.LinkageConflict)
	h.refusedCreate("destination is a seller", h.create("3", transfer, u["6"], ptr(u["7"])), result.Blocked, reasoncodes.RecipientIneligible)
	h.refusedCreate("destination is the senior", h.create("2", transfer, u["6"], ptr(u["3"])), result.Blocked, reasoncodes.RecipientIneligible)
	h.refusedCreate("no-op transfer to the current parent", h.create("3", transfer, u["6"], ptr(u["4"])), result.Blocked, reasoncodes.PolicyUnresolved,
		hrrequest.DecisionSD04)
	h.refusedCreate("destination outside the requester's scope", h.create("3", transfer, u["6"], ptr(u["8"])), result.Blocked, reasoncodes.PolicyUnresolved,
		hrrequest.DecisionSD04)
	h.refusedCreate("anonymous actor", hrrequest.Create{RequestType: terminate, SubjectUserID: u["7"], Reason: "x", IdempotencyKey: h.key("c"), CorrelationID: "c"},
		result.Unauthorized, reasoncodes.RowScopeDenied)
	if r, err := h.hr.CreateRequest(context.Background(), rollout.Disabled, h.create("4", terminate, u["7"], nil)); err != nil || r.Mode != rollout.Disabled {
		t.Fatalf("disabled: %+v %v", r, err)
	} else {
		expectHR(t, "disabled", r, result.Blocked, reasoncodes.FeatureDisabled)
	}

	// Positive: supervisor A requests termination of seller Y → reviewer = senior (3).
	c := h.create("4", terminate, u["7"], nil)
	r := h.doCreate(c)
	okCreate(t, "terminate", r, api.HRPendingReview)
	if got := h.q(t, `SELECT request_type || '|' || status || '|' || current_reviewer_user_id || '|' || subject_parent_user_id || '|' || requested_by_user_id
		|| '|' || version FROM organization.hr_request WHERE request_id = $1`, r.RequestID); got !=
		strings.Join([]string{"terminate_seller", "pending_review", u["3"], u["4"], u["4"], "1"}, "|") {
		t.Fatalf("request row: %s", got)
	}
	if h.count(t, `SELECT count(*) FROM organization.hr_request_event WHERE request_id = $1 AND action = 'created' AND to_status = 'pending_review'
		AND after_version = 1 AND actor_user_id = $2`, r.RequestID, u["4"]) != 1 ||
		h.count(t, `SELECT count(*) FROM organization.command_record WHERE command_id = $1 AND applied`, r.CommandID) != 1 ||
		h.count(t, `SELECT count(*) FROM organization.outbox_event WHERE event_type = $1`, api.EventHRRequestCreated) != 1 {
		t.Fatal("created event, command record and one outbox event")
	}
	// Replay, key reuse, duplicate open request.
	if rp := h.doCreate(c); !rp.Replayed || rp.RequestID != r.RequestID || rp.CommandID != r.CommandID {
		t.Fatalf("replay: %+v", rp)
	}
	reuse := c
	reuse.Reason = "another reason"
	h.refusedCreate("same key, different body", reuse, result.Conflict, reasoncodes.ContractInvalid)
	h.refusedCreate("duplicate open request", h.create("4", terminate, u["7"], nil), result.Conflict, reasoncodes.StaleState)
	h.refusedCreate("duplicate open request by another creator", h.create("3", terminate, u["7"], nil), result.Conflict, reasoncodes.StaleState)
	// The duplicate guard is per (subject, type): a transfer of the same seller is a separate request.
	okCreate(t, "transfer of the same seller", h.doCreate(h.create("3", transfer, u["7"], ptr(u["5"]))), api.HRPendingReview)
	// A requester with no parent sends the request straight to HR (pending_hr, no reviewer).
	m := h.doCreate(h.create("2", terminate, u["9"], nil))
	okCreate(t, "manager request", m, api.HRPendingHR)
	if h.count(t, `SELECT count(*) FROM organization.hr_request WHERE request_id = $1 AND current_reviewer_user_id IS NULL`, m.RequestID) != 1 {
		t.Fatal("pending_hr has no reviewer")
	}
	// SHADOW: evaluated and recorded, nothing created.
	sh, err := h.hr.CreateRequest(context.Background(), rollout.Shadow, h.create("4", terminate, u["6"], nil))
	if err != nil || sh.Status != result.Success || sh.Applied || sh.WritebackState != hrrequest.WritebackNotRequested ||
		h.count(t, `SELECT count(*) FROM organization.hr_request WHERE request_id = $1`, sh.RequestID) != 0 ||
		h.count(t, `SELECT count(*) FROM organization.command_record WHERE command_id = $1 AND NOT applied`, sh.CommandID) != 1 {
		t.Fatalf("shadow: %+v %v", sh, err)
	}
	h.noSideEffects(t, "creation", g)
}

// ---- A (evidence). hierarchy and identity evidence edge cases -----------------

func TestHRRequestR74HierarchyEvidence(t *testing.T) {
	h := newHR(t)
	u := h.users
	g := h.hrGuard(t)
	terminate := api.HRTerminateSeller
	inactive := false

	h.withEmployment(u["4"], "TERMINATED", nil, "LINKED", func() {
		h.refusedCreate("terminated requester", h.create("4", terminate, u["7"], nil), result.Unauthorized, reasoncodes.ActionDenied)
	})
	h.withEmployment(u["4"], "ACTIVE", &inactive, "LINKED", func() {
		h.refusedCreate("requester flagged inactive", h.create("4", terminate, u["7"], nil), result.Unauthorized, reasoncodes.ActionDenied)
	})
	h.withEmployment(u["4"], "UNMAPPED", nil, "LINKED", func() {
		h.refusedCreate("requester with unmapped status", h.create("4", terminate, u["7"], nil), result.Blocked, reasoncodes.LinkageConflict)
	})
	h.withEmployment(u["7"], "ACTIVE", nil, "SHARED_USER", func() {
		h.refusedCreate("ambiguous subject identity (shared user)", h.create("4", terminate, u["7"], nil), result.Blocked, reasoncodes.LinkageConflict)
	})
	h.withEmployment(u["4"], "ACTIVE", nil, "SHARED_USER", func() {
		h.refusedCreate("ambiguous requester identity (shared user)", h.create("4", terminate, u["7"], nil), result.Blocked, reasoncodes.LinkageConflict)
	})
	h.withEmployment(u["5"], "SUSPENDED", nil, "LINKED", func() {
		h.refusedCreate("suspended destination", h.create("3", api.HRTransferSeller, u["6"], &[]string{u["5"]}[0]), result.Blocked, reasoncodes.RecipientIneligible)
	})
	// 305 = seller Y → supervisor A.
	h.withLink("305", nil, nil, func() {
		h.refusedCreate("undecidable current flag on the subject's line", h.create("4", terminate, u["7"], nil), result.Blocked, reasoncodes.LinkageConflict)
	})
	h.withLink("305", bptr(false), nil, func() {
		h.refusedCreate("no current reporting line (no scope evidence)", h.create("4", terminate, u["7"], nil), result.Unauthorized, reasoncodes.RowScopeDenied)
	})
	// 302 = supervisor A → senior: an undecidable line above the requester blocks the reviewer.
	h.withLink("302", nil, nil, func() {
		h.refusedCreate("undecidable reviewer line", h.create("4", terminate, u["7"], nil), result.Blocked, reasoncodes.LinkageConflict)
	})
	// Cycle: senior (301) re-pointed under supervisor A → 6 → A → senior → A … never reaches C.
	h.withLink("301", bptr(true), &[]string{u["4"]}[0], func() {
		h.refusedCreate("hierarchy cycle (depth exceeded)", h.create("8", terminate, u["6"], nil), result.Blocked, reasoncodes.LinkageConflict)
	})
	if h.md5(t, hrEvidenceTables...) != g.evidence {
		t.Fatal("fixtures were not restored exactly")
	}
	// OBSERVED (G-H1, governance question): the subject's own employment
	// status is not checked, so a request about an already TERMINATED seller
	// is accepted. No contract states whether it must be refused.
	h.withEmployment(u["7"], "TERMINATED", nil, "LINKED", func() {
		okCreate(t, "request about a terminated seller (observed)", h.doCreate(h.create("4", terminate, u["7"], nil)), api.HRPendingReview)
	})
	h.noSideEffects(t, "hierarchy evidence", g)
}

// ---- B. decisions --------------------------------------------------------------

func TestHRRequestR74Decisions(t *testing.T) {
	h := newHR(t)
	ctx := context.Background()
	u := h.users
	g := h.hrGuard(t)
	notes := func(s string) *string { return &s }
	hrBlock := []string{hrrequest.DecisionHRRuntime, hrrequest.DecisionOPD05, hrrequest.DecisionSD04}

	tm := h.doCreate(h.create("4", api.HRTerminateSeller, u["7"], nil)) // reviewer 3
	okCreate(t, "create", tm, api.HRPendingReview)
	id := tm.RequestID
	for label, d := range map[string]hrrequest.Decide{
		"upper-case decision": h.decide("3", id, "APPROVE", 1),
		"unknown decision":    h.decide("3", id, "escalate", 1),
		"empty notes":         func() hrrequest.Decide { d := h.decide("3", id, "approve", 1); d.Notes = notes(""); return d }(),
		"notes over 1000": func() hrrequest.Decide {
			d := h.decide("3", id, "approve", 1)
			d.Notes = notes(strings.Repeat("n", 1001))
			return d
		}(),
		"version 0":  h.decide("3", id, "approve", 0),
		"request id": h.decide("3", "request-1", "approve", 1),
	} {
		h.invalidDecide(label, d)
	}
	if _, err := h.hr.DecideRequest(ctx, rollout.Controlled, h.decide("3", "00000000-0000-4000-8000-0000000000d4", "approve", 1)); !errors.Is(err, hrrequest.ErrNotFound) {
		t.Fatalf("unknown request: %v", err)
	}
	h.refusedDecide("higher rank, not the reviewer", h.decide("2", id, "approve", 1), result.Unauthorized, reasoncodes.RowScopeDenied)
	h.refusedDecide("requester decides own request", h.decide("4", id, "approve", 1), result.Unauthorized, reasoncodes.RowScopeDenied)
	h.refusedDecide("subject decides", h.decide("7", id, "reject", 1), result.Unauthorized, reasoncodes.RowScopeDenied)
	h.refusedDecide("HR decides a step", h.decide("1", id, "approve", 1), result.Blocked, reasoncodes.PolicyUnresolved, hrBlock...)
	stale := h.refusedDecide("stale version", h.decide("3", id, "approve", 2), result.Conflict, reasoncodes.StaleState)
	if stale.CurrentVersion != 1 {
		t.Fatalf("current version: %+v", stale)
	}
	if r, err := h.hr.DecideRequest(ctx, rollout.Disabled, h.decide("3", id, "approve", 1)); err != nil {
		t.Fatal(err)
	} else {
		expectHR(t, "disabled", r, result.Blocked, reasoncodes.FeatureDisabled)
	}
	// Reviewer evidence is re-checked at decision time.
	h.withEmployment(u["3"], "RESIGNED", nil, "LINKED", func() {
		h.refusedDecide("reviewer no longer active", h.decide("3", id, "approve", 1), result.Unauthorized, reasoncodes.ActionDenied)
	})
	h.withEmployment(u["3"], "ACTIVE", nil, "SHARED_USER", func() {
		h.refusedDecide("reviewer identity ambiguous", h.decide("3", id, "approve", 1), result.Blocked, reasoncodes.LinkageConflict)
	})
	// 301 = senior → manager: an undecidable next reviewer blocks approval, not rejection logic.
	h.withLink("301", nil, nil, func() {
		h.refusedDecide("approve with an undecidable next reviewer", h.decide("3", id, "approve", 1), result.Blocked, reasoncodes.LinkageConflict)
	})

	// Step approval by the correct reviewer → next reviewer = the reviewer's parent.
	d1 := h.decide("3", id, "approve", 1)
	s1 := h.doDecide(d1)
	okDecide(t, "senior approves", s1, api.HRPendingReview, 2)
	if got := h.q(t, `SELECT status || '|' || current_reviewer_user_id || '|' || version FROM organization.hr_request WHERE request_id = $1`, id); got !=
		"pending_review|"+u["2"]+"|2" {
		t.Fatalf("after step: %s", got)
	}
	if h.count(t, `SELECT count(*) FROM organization.hr_request_event WHERE request_id = $1 AND action = 'approved_step' AND from_status = 'pending_review'
		AND to_status = 'pending_review' AND next_reviewer_user_id = $2 AND before_version = 1 AND after_version = 2`, id, u["2"]) != 1 {
		t.Fatal("step event")
	}
	if rp := h.doDecide(d1); !rp.Replayed || rp.CommandID != s1.CommandID || rp.Version != 2 {
		t.Fatalf("replay: %+v", rp)
	}
	reuse := d1
	reuse.Decision = "reject"
	h.refusedDecide("same key, other decision", reuse, result.Conflict, reasoncodes.ContractInvalid)
	h.refusedDecide("duplicate approval by the previous reviewer", h.decide("3", id, "approve", 2), result.Unauthorized, reasoncodes.RowScopeDenied)
	// Manager (no parent) approves → pending_hr: recorded, NOT applied.
	okDecide(t, "manager approves", h.doDecide(h.decide("2", id, "approve", 2)), api.HRPendingHR, 3)
	h.refusedDecide("final approval at pending_hr", h.decide("2", id, "approve", 3), result.Blocked, reasoncodes.PolicyUnresolved, hrBlock...)
	h.refusedDecide("rejection at pending_hr", h.decide("2", id, "reject", 3), result.Blocked, reasoncodes.PolicyUnresolved, hrBlock...)
	h.refusedDecide("HR final decision", h.decide("1", id, "approve", 3), result.Blocked, reasoncodes.PolicyUnresolved, hrBlock...)

	// Rejection is terminal.
	tr := h.doCreate(h.create("3", api.HRTransferSeller, u["6"], &[]string{u["5"]}[0])) // reviewer 2
	rj := h.decide("2", tr.RequestID, "reject", 1)
	rj.Notes = notes("synthetic rejection note")
	okDecide(t, "reject", h.doDecide(rj), api.HRRejected, 2)
	if got := h.q(t, `SELECT status || '|' || coalesce(current_reviewer_user_id::text, '-') || '|' || reviewer_notes FROM organization.hr_request
		WHERE request_id = $1`, tr.RequestID); got != "rejected|-|synthetic rejection note" {
		t.Fatalf("rejected row: %s", got)
	}
	h.refusedDecide("decision after rejection", h.decide("2", tr.RequestID, "approve", 2), result.Conflict, reasoncodes.StaleState)
	// A rejected request no longer blocks a new request for the same seller.
	okCreate(t, "new transfer after rejection", h.doCreate(h.create("3", api.HRTransferSeller, u["6"], &[]string{u["5"]}[0])), api.HRPendingReview)

	// OBSERVED (G-H2, governance question): the reviewer is the snapshot taken
	// at the previous transition. After the hierarchy changes, the recorded
	// reviewer still decides and the new parent cannot; the NEXT reviewer is
	// computed from the live hierarchy at decision time.
	y := h.doCreate(h.create("4", api.HRTerminateSeller, u["6"], nil)) // reviewer 3
	h.withLink("302", bptr(true), &[]string{u["8"]}[0], func() {       // supervisor A now reports to supervisor C
		h.refusedDecide("new parent is not the recorded reviewer", h.decide("8", y.RequestID, "approve", 1), result.Unauthorized, reasoncodes.RowScopeDenied)
		okDecide(t, "recorded reviewer decides after a hierarchy change (observed)", h.doDecide(h.decide("3", y.RequestID, "approve", 1)), api.HRPendingReview, 2)
	})
	// SHADOW decision: evaluated and recorded, the request does not move.
	sh, err := h.hr.DecideRequest(ctx, rollout.Shadow, h.decide("2", y.RequestID, "approve", 2))
	if err != nil || sh.Status != result.Success || sh.Applied || sh.RequestStatus != api.HRPendingReview || sh.Version != 2 ||
		h.q(t, `SELECT status || '|' || version FROM organization.hr_request WHERE request_id = $1`, y.RequestID) != "pending_review|2" {
		t.Fatalf("shadow: %+v %v", sh, err)
	}
	h.noSideEffects(t, "decisions", g)
}

// ---- C / D. authority, separation of duties, no apply ---------------------------

// Every recorded decision leaves employment, hierarchy, identity (grants,
// credentials), custody, cases and every financial schema untouched, and
// emits only organization.hr_request_* events.
func TestHRRequestR74NoApplyEffects(t *testing.T) {
	h := newHR(t)
	u := h.users
	g := h.hrGuard(t)
	grants := h.md5(t, "identity.role_grant")
	tm := h.doCreate(h.create("4", api.HRTerminateSeller, u["7"], nil))
	okDecide(t, "senior", h.doDecide(h.decide("3", tm.RequestID, "approve", 1)), api.HRPendingReview, 2)
	okDecide(t, "manager", h.doDecide(h.decide("2", tm.RequestID, "approve", 2)), api.HRPendingHR, 3)
	tr := h.doCreate(h.create("3", api.HRTransferSeller, u["6"], &[]string{u["5"]}[0]))
	okDecide(t, "reject transfer", h.doDecide(h.decide("2", tr.RequestID, "reject", 1)), api.HRRejected, 2)

	h.noSideEffects(t, "approved to pending_hr and rejected", g)
	if h.md5(t, "identity.role_grant") != grants || h.count(t, `SELECT count(*) FROM identity.role_grant WHERE status = 'ACTIVE'`) != 0 {
		t.Fatal("no ACTIVE grant may exist or be created")
	}
	// Seller Y is still employed and still reports to supervisor A; seller X still under A.
	for _, k := range []string{"7", "6"} {
		if got := h.q(t, `SELECT e.employment_status || '|' || pe.user_id FROM organization.employment e JOIN organization.org_assignment a
			ON a.child_employment_id = e.employment_id AND a.is_current JOIN organization.employment pe ON pe.employment_id = a.parent_employment_id
			WHERE e.user_id = $1`, u[k]); got != "ACTIVE|"+u["4"] {
			t.Fatalf("seller %s employment / parent changed: %s", k, got)
		}
	}
	if n := h.count(t, `SELECT count(*) FROM organization.outbox_event WHERE event_type NOT IN ($1, $2)`, api.EventHRRequestCreated, api.EventHRRequestDecided); n != 0 {
		t.Fatalf("only HR request events may be emitted: %d others", n)
	}
	// No status beyond the recorded steps: nothing is approved / applied.
	if n := h.count(t, `SELECT count(*) FROM organization.hr_request WHERE status IN ('approved', 'failed', 'cancelled')`); n != 0 {
		t.Fatalf("no request may reach an applied status: %d", n)
	}
}

// ---- E. concurrency ----------------------------------------------------------------

func TestHRRequestR74Concurrency(t *testing.T) {
	h := newHR(t)
	ctx := context.Background()
	u := h.users
	const n = 8
	runCreates := func(cs []hrrequest.Create) []hrrequest.Result {
		out := make([]hrrequest.Result, len(cs))
		errs := make([]error, len(cs))
		start := make(chan struct{})
		var wg sync.WaitGroup
		for i := range cs {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				<-start
				out[i], errs[i] = h.hr.CreateRequest(ctx, rollout.Controlled, cs[i])
			}(i)
		}
		close(start)
		wg.Wait()
		for _, err := range errs {
			if err != nil {
				t.Fatal(err)
			}
		}
		return out
	}

	// Distinct keys, same subject and type: one open request, the rest CONFLICT.
	var cs []hrrequest.Create
	for i := 0; i < n; i++ {
		actor := "4"
		if i%2 == 1 {
			actor = "3"
		}
		cs = append(cs, h.create(actor, api.HRTerminateSeller, u["7"], nil))
	}
	wins := 0
	for _, r := range runCreates(cs) {
		if r.Status == result.Success {
			wins++
			continue
		}
		expectHR(t, "concurrent duplicate", r, result.Conflict, reasoncodes.StaleState)
	}
	if wins != 1 || h.count(t, `SELECT count(*) FROM organization.hr_request WHERE subject_user_id = $1 AND request_type = 'terminate_seller'`, u["7"]) != 1 {
		t.Fatalf("one open request: %d winners", wins)
	}
	open := h.q(t, `SELECT request_id::text FROM organization.hr_request WHERE subject_user_id = $1`, u["7"])

	// Same key, same body: executed once, replayed n-1 times.
	same := h.create("3", api.HRTransferSeller, u["6"], &[]string{u["5"]}[0])
	cs = nil
	for i := 0; i < n; i++ {
		cs = append(cs, same)
	}
	executed := 0
	for _, r := range runCreates(cs) {
		if r.Status != result.Success {
			t.Fatalf("same key: %+v", r)
		}
		if !r.Replayed {
			executed++
		}
	}
	if executed != 1 || h.count(t, `SELECT count(*) FROM organization.hr_request WHERE subject_user_id = $1`, u["6"]) != 1 {
		t.Fatalf("same key executed %d times", executed)
	}

	// Same key, different bodies: one execution; a different body is never answered with it.
	k := h.key("same-diff")
	cs = nil
	for i := 0; i < n; i++ {
		c := h.create("8", api.HRTerminateSeller, u["9"], nil)
		c.IdempotencyKey, c.Reason = k, fmt.Sprintf("reason %d", i%2)
		cs = append(cs, c)
	}
	rs := runCreates(cs)
	winReason := h.q(t, `SELECT reason FROM organization.hr_request WHERE subject_user_id = $1`, u["9"])
	for i, r := range rs {
		switch {
		case r.Status == result.Success && cs[i].Reason != winReason:
			t.Fatalf("a different body was answered with the winner's result: %+v", r)
		case r.Status != result.Success:
			expectHR(t, "same key, different body", r, result.Conflict, reasoncodes.ContractInvalid)
		}
	}

	// Concurrent decisions on version 1 (approve and reject): exactly one wins.
	// No winner is assumed. A loser's answer depends on what the winner did:
	// STALE_STATE (rejected), ROW_SCOPE_DENIED (an approval moved the
	// reviewer up) or the HR-apply block (the approval reached pending_hr).
	// Losers never see SUCCESS and never add an event.
	// The reviewer depends on which creator won above: decide as the recorded one.
	reviewerUser := h.q(t, `SELECT current_reviewer_user_id::text FROM organization.hr_request WHERE request_id = $1`, open)
	reviewer := ""
	for k, v := range u {
		if v == reviewerUser {
			reviewer = k
		}
	}
	if reviewer == "" {
		t.Fatal("recorded reviewer is not a scenario user")
	}
	ds := make([]hrrequest.Decide, n)
	for i := range ds {
		decision := "approve"
		if i%2 == 1 {
			decision = "reject"
		}
		ds[i] = h.decide(reviewer, open, decision, 1)
	}
	out := make([]hrrequest.Result, n)
	errs := make([]error, n)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range ds {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			out[i], errs[i] = h.hr.DecideRequest(ctx, rollout.Controlled, ds[i])
		}(i)
	}
	close(start)
	wg.Wait()
	wins = 0
	for i, r := range out {
		if errs[i] != nil {
			t.Fatal(errs[i])
		}
		switch {
		case r.Status == result.Success:
			wins++
		case r.Status == result.Conflict:
			expectHR(t, "decision loser", r, result.Conflict, reasoncodes.StaleState)
		case r.Status == result.Blocked:
			expectHR(t, "decision loser", r, result.Blocked, reasoncodes.PolicyUnresolved, hrrequest.DecisionHRRuntime, hrrequest.DecisionOPD05,
				hrrequest.DecisionSD04)
		default:
			expectHR(t, "decision loser", r, result.Unauthorized, reasoncodes.RowScopeDenied)
		}
	}
	if wins != 1 || h.count(t, `SELECT count(*) FROM organization.hr_request_event WHERE request_id = $1 AND action <> 'created'`, open) != 1 ||
		h.q(t, `SELECT version::text FROM organization.hr_request WHERE request_id = $1`, open) != "2" {
		t.Fatalf("one decision: %d winners", wins)
	}
}

// ---- C. capability at the HTTP boundary (test-only overlay) ----------------------

func TestHRRequestR74HTTPCapability(t *testing.T) {
	e := setupWrites(t)
	subs := testSubjects{users: e.users, caps: map[string][]identityapi.Capability{
		"4":  {identityapi.CapHRRequestRead},                                   // read only
		"3":  {identityapi.CapHRRequestCreate},                                 // create, no review
		"6":  {identityapi.CapHRRequestCreate, identityapi.CapHRRequestReview}, // seller holding capabilities
		"2":  {identityapi.CapHRRequestCreate, identityapi.CapHRRequestReview}, // manager
		"8":  {identityapi.CapHRRequestCreate, identityapi.CapHRRequestRead},   // supervisor C
		"5":  {identityapi.CapContactOutcomeRecord, identityapi.CapCaseRead},   // unrelated capabilities
		"7":  {identityapi.CapHRRequestReview, identityapi.CapHRRequestRead},   // review only
		"1":  {identityapi.CapHRRequestCreate, identityapi.CapHRRequestReview}, // HR
		"9":  {identityapi.CapHRRequestCreate, identityapi.CapHRRequestReview}, // seller Z
		"10": {identityapi.CapHRRequestCreate, identityapi.CapHRRequestReview}, // seller W
	}}
	srv := e.writesServer(t, subs)
	u := e.users
	e.setFlags(t, map[string]bool{"organization.hr_request_write": true, "organization.hr_request_writeback": true})
	g := (&hrHarness{writesEnv: e, t: t}).hrGuard(t)
	reqTables := e.md5(t, hrRequestTables...)
	body := func(subject string) string {
		return `{"request_type":"terminate_seller","subject_user_id":"` + subject + `","reason":"synthetic r7-4 http"}`
	}
	hdr := func(k string) map[string]string { return map[string]string{"Idempotency-Key": k} }

	// No create capability → 403 at the gate, whatever the position.
	if r := call(t, srv, http.MethodPost, "/api/v1/hr/requests", "4", body(u["7"]), hdr("r74-http-0001")); r.code != 403 {
		t.Fatalf("supervisor without the create capability: %d %s", r.code, r.body)
	}
	if r := call(t, srv, http.MethodPost, "/api/v1/hr/requests", "5", body(u["6"]), hdr("r74-http-0002")); r.code != 403 {
		t.Fatalf("unrelated capabilities: %d %s", r.code, r.body)
	}
	if r := call(t, srv, http.MethodPost, "/api/v1/hr/requests", "", body(u["7"]), hdr("r74-http-0003")); r.code != 403 {
		t.Fatalf("anonymous: %d %s", r.code, r.body)
	}
	// The capability alone never grants: a seller holding it is refused by the hierarchy rules.
	if r := call(t, srv, http.MethodPost, "/api/v1/hr/requests", "6", body(u["7"]), hdr("r74-http-0004")); r.code != 403 ||
		!strings.Contains(r.body, `"UNAUTHORIZED"`) || !strings.Contains(r.body, `"ACTION_DENIED"`) {
		t.Fatalf("seller with the capability: %d %s", r.code, r.body)
	}
	if e.md5(t, hrRequestTables...) != reqTables {
		t.Fatal("a refused HTTP command wrote something")
	}
	// Capability + position + hierarchy → created (backend-only proof).
	r := call(t, srv, http.MethodPost, "/api/v1/hr/requests", "8", body(u["9"]), hdr("r74-http-0005"))
	if r.code != 200 || r.json["outcome"] != "SUCCESS" {
		t.Fatalf("supervisor C creates: %d %s", r.code, r.body)
	}
	id := cmdOf(r)["request_id"].(string)
	path := "/api/v1/hr/requests/" + id + "/decisions"
	dhdr := func(k string) map[string]string { return map[string]string{"Idempotency-Key": k, "If-Match": "1"} }
	// Review needs its own capability; create does not imply it.
	if r := call(t, srv, http.MethodPost, path, "3", `{"decision":"approve"}`, dhdr("r74-http-0006")); r.code != 403 {
		t.Fatalf("create-only cannot review: %d %s", r.code, r.body)
	}
	// Review capability without being the recorded reviewer → 403 ROW_SCOPE_DENIED.
	if r := call(t, srv, http.MethodPost, path, "7", `{"decision":"approve"}`, dhdr("r74-http-0007")); r.code != 403 || !strings.Contains(r.body, "ROW_SCOPE_DENIED") {
		t.Fatalf("review capability without row scope: %d %s", r.code, r.body)
	}
	// Recorded reviewer (manager, no parent) approves → pending_hr; HR cannot finalise.
	if r := call(t, srv, http.MethodPost, path, "2", `{"decision":"approve"}`, dhdr("r74-http-0008")); r.code != 200 || cmdOf(r)["request_status"] != "pending_hr" {
		t.Fatalf("manager approves: %d %s", r.code, r.body)
	}
	if r := call(t, srv, http.MethodPost, path, "1", `{"decision":"approve"}`, map[string]string{"Idempotency-Key": "r74-http-0009", "If-Match": "2"}); r.code != 200 ||
		r.json["outcome"] != "BLOCKED" || !strings.Contains(r.body, "HR_APPLY_IMPLEMENTATION_BLOCKED") {
		t.Fatalf("HR final apply: %d %s", r.code, r.body)
	}
	// At pending_hr the decision block precedes the version check, even with a stale If-Match.
	if r := call(t, srv, http.MethodPost, path, "2", `{"decision":"reject"}`, dhdr("r74-http-0010")); r.code != 200 || r.json["outcome"] != "BLOCKED" ||
		!strings.Contains(r.body, "OPD-05") {
		t.Fatalf("rejection at pending_hr: %d %s", r.code, r.body)
	}
	// A second open request for the same seller and type → 409 CONFLICT / STALE_STATE.
	r2 := call(t, srv, http.MethodPost, "/api/v1/hr/requests", "8", body(u["9"]), hdr("r74-http-0011"))
	if r2.code != 409 || !strings.Contains(r2.body, "STALE_STATE") {
		t.Fatalf("duplicate open request over HTTP: %d %s", r2.code, r2.body)
	}
	(&hrHarness{writesEnv: e, t: t}).noSideEffects(t, "http", g)
}
