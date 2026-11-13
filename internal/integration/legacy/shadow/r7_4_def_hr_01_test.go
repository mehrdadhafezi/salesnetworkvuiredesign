package shadow_test

// DEF-HR-01 REGRESSIONS (default suite) — malformed reporting evidence must
// never make an HR request's requester its own reviewer.
//
// hrrequest documents "ambiguous or missing hierarchy evidence is BLOCKED,
// never guessed", and the organization import already classifies a
// SELF_PARENT line and a reporting CYCLE as CONFLICT findings
// (internal/organization/domain/hierarchy.go). Before the fix the HR service
// accepted such a line as a valid parent, so the requester became the
// reviewer of their own request, immediately (self-parent) or after an
// approval step (cycle). The two TestR74Defect* tests are the original R7-4
// reproductions (previously behind the r74defects build tag); the TestDEFHR01*
// tests cover the boundaries of the fix (docs/remediation/r7-4-domain-hardening.md).
//
// BACKEND-ONLY COVERAGE on the synthetic writes scenario (disposable
// PostgreSQL). Hierarchy shapes are fixture updates of the imported
// organization evidence, each restored and proven restored. This does NOT
// resolve person-level separation of duties (R5-SEC-01 remains open).
//
// Synthetic hierarchy (legacytest.WritesScenario):
//
//	2 manager (root) ← 3 senior (301) ← 4 supervisor A (302) ← 6, 7 sellers (304, 305)
//	2 manager        ← 8 supervisor C (306)

import (
	"context"
	"fmt"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/api"
)

// Supervisor A's own line (302) points at supervisor A: SELF_PARENT evidence.
func TestR74DefectHRSelfParentReviewer(t *testing.T) {
	h := newHR(t)
	u := h.users
	h.withLink("302", bptr(true), &[]string{u["4"]}[0], func() {
		r := h.doCreate(h.create("4", api.HRTerminateSeller, u["7"], nil))
		if r.Status == result.Success {
			t.Errorf("DEF-HR-01a: requester became the reviewer of their own request (reviewer=%s, requester=%s)",
				h.q(t, `SELECT current_reviewer_user_id::text FROM organization.hr_request WHERE request_id = $1`, r.RequestID), u["4"])
			return
		}
		expectHR(t, "self-parent evidence", r, result.Blocked, reasoncodes.LinkageConflict)
	})
}

// Senior (301) re-pointed under supervisor A: A → senior → A (CYCLE). After
// one step the request returns to its requester.
func TestR74DefectHRCycleReturnsToRequester(t *testing.T) {
	h := newHR(t)
	u := h.users
	h.withLink("301", bptr(true), &[]string{u["4"]}[0], func() {
		r := h.doCreate(h.create("4", api.HRTerminateSeller, u["7"], nil))
		if r.Status != result.Success {
			expectHR(t, "cycle above the requester", r, result.Blocked, reasoncodes.LinkageConflict)
			return
		}
		step := h.doDecide(h.decide("3", r.RequestID, "approve", 1))
		if step.Status == result.Success {
			if got := h.q(t, `SELECT current_reviewer_user_id::text FROM organization.hr_request WHERE request_id = $1`, r.RequestID); got == u["4"] {
				t.Errorf("DEF-HR-01b: the request came back to its requester as reviewer")
			}
			return
		}
		expectHR(t, "cycle on the reviewer path", step, result.Blocked, reasoncodes.LinkageConflict)
	})
}

// Self-parent and cycles at creation: the requester, an ancestor, or a loop
// that never involves the requester. Every refusal writes nothing.
func TestDEFHR01CreationEvidence(t *testing.T) {
	h := newHR(t)
	u := h.users
	g := h.hrGuard(t)
	term := api.HRTerminateSeller
	h.withLink("302", bptr(true), &[]string{u["4"]}[0], func() { // A → A
		h.refusedCreate("immediate self-parent (requester)", h.create("4", term, u["7"], nil), result.Blocked, reasoncodes.LinkageConflict)
	})
	h.withLink("301", bptr(true), &[]string{u["3"]}[0], func() { // senior → senior, above A
		h.refusedCreate("self-parent above the requester", h.create("4", term, u["7"], nil), result.Blocked, reasoncodes.LinkageConflict)
	})
	h.withLink("301", bptr(true), &[]string{u["4"]}[0], func() { // A → senior → A
		h.refusedCreate("cycle through the requester", h.create("4", term, u["7"], nil), result.Blocked, reasoncodes.LinkageConflict)
	})
	h.withLink("301", bptr(true), &[]string{u["8"]}[0], func() { // senior → C
		h.withLink("306", bptr(true), &[]string{u["3"]}[0], func() { // C → senior: A → senior → C → senior …
			h.refusedCreate("multi-node cycle not involving the requester", h.create("4", term, u["7"], nil), result.Blocked, reasoncodes.LinkageConflict)
		})
	})
	if h.count(t, `SELECT count(*) FROM organization.hr_request`) != 0 {
		t.Fatal("a refused creation left a request behind")
	}
	h.noSideEffects(t, "DEF-HR-01 creation", g)
}

// The hierarchy changes between creation and decision. Approval re-resolves
// the next reviewer from the live chain and fails closed when it is cyclic or
// leads back to the requester; rejection does not route and stays available.
func TestDEFHR01HierarchyChangesBeforeDecision(t *testing.T) {
	h := newHR(t)
	u := h.users
	g := h.hrGuard(t)
	notes := func(s string) *string { return &s }
	row := func(id string) string {
		return h.q(t, `SELECT status || '|' || coalesce(current_reviewer_user_id::text, '-') || '|' || version FROM organization.hr_request WHERE request_id = $1`, id)
	}

	// Acyclic, but the next reviewer would be the requester: A re-pointed under
	// C and the senior re-pointed under A (senior → A → C → manager).
	a := h.doCreate(h.create("4", api.HRTerminateSeller, u["7"], nil)) // reviewer 3
	okCreate(t, "create (valid chain)", a, api.HRPendingReview)
	h.withLink("302", bptr(true), &[]string{u["8"]}[0], func() {
		h.withLink("301", bptr(true), &[]string{u["4"]}[0], func() {
			h.refusedDecide("approval would route back to the requester", h.decide("3", a.RequestID, "approve", 1), result.Blocked, reasoncodes.LinkageConflict)
			if got := row(a.RequestID); got != "pending_review|"+u["3"]+"|1" {
				t.Fatalf("refused approval moved the request: %s", got)
			}
			rj := h.decide("3", a.RequestID, "reject", 1)
			rj.Notes = notes("synthetic rejection under broken evidence")
			okDecide(t, "rejection is still available", h.doDecide(rj), api.HRRejected, 2)
		})
	})
	h.refusedDecide("terminal after rejection", h.decide("3", a.RequestID, "approve", 2), result.Conflict, reasoncodes.StaleState)

	// A cycle above the current reviewer that does not involve the requester.
	b := h.doCreate(h.create("4", api.HRTerminateSeller, u["6"], nil)) // reviewer 3
	okCreate(t, "create (valid chain)", b, api.HRPendingReview)
	h.withLink("301", bptr(true), &[]string{u["8"]}[0], func() {
		h.withLink("306", bptr(true), &[]string{u["3"]}[0], func() {
			h.refusedDecide("approval into a cycle", h.decide("3", b.RequestID, "approve", 1), result.Blocked, reasoncodes.LinkageConflict)
		})
	})
	if got := row(b.RequestID); got != "pending_review|"+u["3"]+"|1" {
		t.Fatalf("refused approval moved the request: %s", got)
	}
	// Evidence restored → the same step succeeds and progression is unchanged.
	okDecide(t, "senior approves after the evidence is restored", h.doDecide(h.decide("3", b.RequestID, "approve", 1)), api.HRPendingReview, 2)
	if got := row(b.RequestID); got != "pending_review|"+u["2"]+"|2" {
		t.Fatalf("after step: %s", got)
	}
	h.noSideEffects(t, "DEF-HR-01 decisions", g)
}

// Valid chains are unchanged: multi-step progression to pending_hr with one
// event per step, and a root-level requester goes straight to pending_hr.
func TestDEFHR01ValidChainsUnchanged(t *testing.T) {
	h := newHR(t)
	u := h.users
	g := h.hrGuard(t)
	r := h.doCreate(h.create("4", api.HRTerminateSeller, u["7"], nil))
	okCreate(t, "supervisor creates", r, api.HRPendingReview)
	if got := h.q(t, `SELECT current_reviewer_user_id::text FROM organization.hr_request WHERE request_id = $1`, r.RequestID); got != u["3"] {
		t.Fatalf("first reviewer %s, want the senior", got)
	}
	okDecide(t, "senior approves", h.doDecide(h.decide("3", r.RequestID, "approve", 1)), api.HRPendingReview, 2)
	okDecide(t, "manager approves", h.doDecide(h.decide("2", r.RequestID, "approve", 2)), api.HRPendingHR, 3)
	if got := h.q(t, `SELECT string_agg(action || ':' || coalesce(next_reviewer_user_id::text, '-'), ',' ORDER BY coalesce(before_version, 0))
		FROM organization.hr_request_event WHERE request_id = $1`, r.RequestID); got != "created:"+u["3"]+",approved_step:"+u["2"]+",approved_step:-" {
		t.Fatalf("event chain: %s", got)
	}
	root := h.doCreate(h.create("2", api.HRTerminateSeller, u["6"], nil))
	okCreate(t, "root-level requester", root, api.HRPendingHR)
	if got := h.q(t, `SELECT coalesce(current_reviewer_user_id::text, '-') FROM organization.hr_request WHERE request_id = $1`, root.RequestID); got != "-" {
		t.Fatalf("root-level reviewer: %s", got)
	}
	h.noSideEffects(t, "DEF-HR-01 valid chains", g)
}

// The reviewer chain is bounded by the existing maxHierDepth (12): twelve
// ancestors above the requester are accepted, a thirteenth is not evidence.
func TestDEFHR01DepthBoundary(t *testing.T) {
	h := newHR(t)
	u := h.users
	// A has two ancestors (senior, manager); extra roots above the manager.
	h.withAncestorsAboveManager(repeat("2", 10), func() {
		okCreate(t, "12 ancestors", h.doCreate(h.create("4", api.HRTerminateSeller, u["7"], nil)), api.HRPendingReview)
	})
	h.withAncestorsAboveManager(repeat("2", 11), func() {
		h.refusedCreate("13 ancestors (depth exceeded)", h.create("4", api.HRTerminateSeller, u["6"], nil), result.Blocked, reasoncodes.LinkageConflict)
	})
}

// A second (non-PRESENT) employment of the requester above the chain: no
// cycle by employment, but the request would reach its requester at the top.
func TestDEFHR01RequesterAboveOwnChain(t *testing.T) {
	h := newHR(t)
	g := h.hrGuard(t)
	h.withAncestorsAboveManager([]string{"4"}, func() {
		h.refusedCreate("requester's other employment above the chain", h.create("4", api.HRTerminateSeller, h.users["7"], nil),
			result.Blocked, reasoncodes.LinkageConflict)
	})
	h.noSideEffects(t, "DEF-HR-01 requester above own chain", g)
}

func repeat(s string, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = s
	}
	return out
}

// withAncestorsAboveManager inserts synthetic ancestor employments above the
// root manager for fn (bottom to top, each a clone of the named fixture
// user's employment) and deletes them afterwards. The clones are
// ABSENT_IN_LATEST, so no user's own PRESENT employment evidence changes.
func (h *hrHarness) withAncestorsAboveManager(cloneOf []string, fn func()) {
	h.t.Helper()
	ctx := context.Background()
	before := h.md5(h.t, hrEvidenceTables...)
	var prev string
	if err := h.admin.QueryRow(ctx, `SELECT employment_id::text FROM organization.employment WHERE user_id = $1 AND presence = 'PRESENT'`,
		h.users["2"]).Scan(&prev); err != nil {
		h.t.Fatal(err)
	}
	var people, emps, links []string
	defer func() {
		for _, s := range []struct {
			sql string
			ids []string
		}{
			{`DELETE FROM organization.org_assignment WHERE assignment_id = ANY($1::uuid[])`, links},
			{`DELETE FROM organization.employment WHERE employment_id = ANY($1::uuid[])`, emps},
			{`DELETE FROM organization.person WHERE person_id = ANY($1::uuid[])`, people},
		} {
			h.fixture(s.sql, s.ids)
		}
		if h.md5(h.t, hrEvidenceTables...) != before {
			h.t.Fatal("depth fixture was not removed exactly")
		}
	}()
	for i, user := range cloneOf {
		key := fmt.Sprint(990001 + i)
		var src, pid, eid, aid string
		if err := h.admin.QueryRow(ctx, `SELECT employment_id::text FROM organization.employment WHERE user_id = $1 AND presence = 'PRESENT'`,
			h.users[user]).Scan(&src); err != nil {
			h.t.Fatal(err)
		}
		if err := h.admin.QueryRow(ctx, `INSERT INTO organization.person SELECT (jsonb_populate_record(NULL::organization.person,
			to_jsonb(p) || jsonb_build_object('person_id', gen_random_uuid(), 'source_key', $2::text, 'presence', 'ABSENT_IN_LATEST'))).*
			FROM organization.person p JOIN organization.employment e ON e.person_id = p.person_id WHERE e.employment_id = $1::uuid
			RETURNING person_id::text`, src, key).Scan(&pid); err != nil {
			h.t.Fatalf("person clone: %v", err)
		}
		people = append(people, pid)
		if err := h.admin.QueryRow(ctx, `INSERT INTO organization.employment SELECT (jsonb_populate_record(NULL::organization.employment,
			to_jsonb(e) || jsonb_build_object('employment_id', gen_random_uuid(), 'person_id', $2::uuid, 'source_key', $3::text, 'presence', 'ABSENT_IN_LATEST'))).*
			FROM organization.employment e WHERE e.employment_id = $1::uuid RETURNING employment_id::text`, src, pid, key).Scan(&eid); err != nil {
			h.t.Fatalf("employment clone: %v", err)
		}
		emps = append(emps, eid)
		if err := h.admin.QueryRow(ctx, `INSERT INTO organization.org_assignment SELECT (jsonb_populate_record(NULL::organization.org_assignment,
			to_jsonb(a) || jsonb_build_object('assignment_id', gen_random_uuid(), 'source_key', $1::text, 'child_employment_id', $2::uuid,
			'parent_employment_id', $3::uuid, 'is_current', true))).*
			FROM organization.org_assignment a WHERE a.source_key = '301' RETURNING assignment_id::text`, key, prev, eid).Scan(&aid); err != nil {
			h.t.Fatalf("assignment clone: %v", err)
		}
		links = append(links, aid)
		prev = eid
	}
	fn()
}
