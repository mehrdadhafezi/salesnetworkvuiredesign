//go:build r74defects

package shadow_test

// R7-4 DEFECT REGRESSIONS — these tests FAIL on the current code by design.
// They are excluded from every CI job by the build tag and are run with:
//
//	go test -tags r74defects -count=1 -run 'TestR74Defect' ./internal/integration/legacy/shadow/
//
// Each asserts the behaviour the EXISTING contract already requires; a fix
// needs independent approval (see docs/remediation/r7-4-domain-hardening.md,
// DEF-HR-01) and then moves the test into the default suite.
//
// DEF-HR-01: hrrequest documents "ambiguous or missing hierarchy evidence is
// BLOCKED, never guessed", and the organization import already classifies a
// SELF_PARENT line and a reporting CYCLE as CONFLICT findings
// (internal/organization/domain/hierarchy.go). The HR service still accepts
// such a line as a valid parent, so the requester becomes the reviewer of
// their own request and records approval steps on it.

import (
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
