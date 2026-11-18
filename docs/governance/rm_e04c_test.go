package governance

// RM-E04C (Seller authorization architecture, decision package, screen designs, API proposal, traceability,
// roadmap) consistency checks. The RM-E04C documents are proposals: they must never reach runtime, never change a
// tracker status and never reference a test that is not defined.
//
//   - the candidate own-scope capabilities are not registered and the seller bundle stays empty;
//   - no proposed /me operation exists in the OpenAPI contract;
//   - every RM-E04C document carries a not-approved marker and Production NO-GO;
//   - the decisions the package covers keep their quoted tracker status, with no outcome and no approval record;
//   - every NT-xx referenced by the traceability matrix is defined, and every SR-01..SR-16 is covered.

import (
	"regexp"
	"strings"
	"testing"

	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
)

var rmE04CDocs = []string{
	"docs/implementation/rm-e04c-seller-authorization-adr-PROPOSED.md",
	"docs/implementation/rm-e04c-seller-api-contract-PROPOSAL.md",
	"docs/implementation/rm-e04c-seller-implementation-roadmap.md",
	"docs/security/rm-e04c-seller-screen-designs.md",
	"docs/security/rm-e04c-seller-traceability-matrix.md",
	"docs/governance/proposals/rm-e04c-seller-decision-package.fa.md",
}

// Tracker status quoted by the RM-E04C decision package (baseline c0d7f4b).
var rmE04CQuotedStatus = map[string]string{
	"D-42":                     "READY_FOR_REVIEW",
	"D-34":                     "READY_FOR_REVIEW",
	"R5-SEC-01 (proposed ADR)": "READY_FOR_REVIEW",
	"ADR #18":                  "READY_FOR_REVIEW",
	"D-41":                     "EVIDENCE_MISSING",
	"OPD-09":                   "EVIDENCE_MISSING",
	"D-36":                     "EVIDENCE_MISSING",
	"OPD-10":                   "READY_FOR_REVIEW",
	"D-30":                     "EVIDENCE_MISSING",
	"D-45":                     "READY_FOR_REVIEW",
}

func TestRME04CProposalsStayOutOfRuntime(t *testing.T) {
	for _, c := range []identityapi.Capability{"cases.own.read", "billing.own.read", "payments.own.read"} {
		if identityapi.KnownCapability(c) {
			t.Errorf("candidate capability %s is registered; registration needs approved M2 records (roadmap B2)", c)
		}
	}
	seller, ok := identityapi.LookupRole(identityapi.RoleSeller)
	if !ok {
		t.Fatal("seller role missing from the vocabulary")
	}
	if len(seller.Capabilities) != 0 {
		t.Errorf("seller bundle holds %v; populating it needs approved M2 records (roadmap B2)", seller.Capabilities)
	}
	spec := read(t, "api/openapi/openapi.yaml")
	for _, s := range []string{"/api/v1/me/", "listMyCases", "getMyCase", "listMyCaseHistory", "getMyCaseContactOutcomes",
		"listMyCaseAttribution", "listMyInvoices", "getMyInvoice", "listMyInvoiceStages", "getMyInvoiceTotals", "listMyInvoicePayments"} {
		if strings.Contains(spec, s) {
			t.Errorf("OpenAPI contract contains proposed %q; RM-E04C operations are proposals only", s)
		}
	}
}

func TestRME04CDocumentsAreMarkedNotApproved(t *testing.T) {
	marker := regexp.MustCompile(`(DRAFT — NOT APPROVED|PROPOSED — NOT ACCEPTED|PROPOSAL — NOT IMPLEMENTED, NOT APPROVED|PLAN ONLY)`)
	for _, f := range rmE04CDocs {
		d := read(t, f)
		head := d
		if len(head) > 1500 {
			head = head[:1500]
		}
		if !marker.MatchString(head) {
			t.Errorf("%s: no not-approved marker near the top", f)
		}
		if !strings.Contains(d, "NO-GO") {
			t.Errorf("%s: Production NO-GO not stated", f)
		}
	}
}

func TestRME04CTrackerUnchanged(t *testing.T) {
	seen := map[string]bool{}
	for _, r := range loadTracker(t).Records {
		want, ok := rmE04CQuotedStatus[r.ID]
		if !ok {
			continue
		}
		seen[r.ID] = true
		if r.Status != want || r.ApprovalRecord != "" || r.Verification != "NOT_STARTED" {
			t.Errorf("%s: status %s, record %q, verification %s; the package quotes %s, none, NOT_STARTED",
				r.ID, r.Status, r.ApprovalRecord, r.Verification, want)
		}
		for k, v := range r.Outcome {
			if v != "" {
				t.Errorf("%s: outcome %s is filled", r.ID, k)
			}
		}
	}
	for id := range rmE04CQuotedStatus {
		if !seen[id] {
			t.Errorf("%s missing from the tracker", id)
		}
	}
}

func TestRME04CTraceabilityIsClosed(t *testing.T) {
	matrix := read(t, "docs/security/rm-e04c-seller-traceability-matrix.md")
	defined := map[string]bool{}
	for _, m := range regexp.MustCompile(`(?m)^### (NT-\d{2}) `).FindAllStringSubmatch(read(t, "docs/security/rm-e04a-seller-negative-test-plan.md"), -1) {
		defined[m[1]] = true
	}
	for _, m := range regexp.MustCompile(`(?m)^\| \*\*(NT-\d{2})\*\* \|`).FindAllStringSubmatch(matrix, -1) {
		if defined[m[1]] {
			t.Errorf("%s defined twice", m[1])
		}
		defined[m[1]] = true
	}
	for _, f := range append([]string{"docs/security/rm-e04c-seller-traceability-matrix.md"}, rmE04CDocs...) {
		for _, id := range regexp.MustCompile(`NT-\d{2}`).FindAllString(read(t, f), -1) {
			if !defined[id] {
				t.Errorf("%s references undefined %s", f, id)
			}
		}
	}
	cov := matrix[strings.Index(matrix, "## 4. Requirement coverage"):]
	for n := 1; n <= 16; n++ {
		sr := "SR-" + map[bool]string{true: "0", false: ""}[n < 10] + itoa(n)
		if !regexp.MustCompile(`(?m)^\| ` + sr + ` .*\| NT-\d{2}`).MatchString(cov) {
			t.Errorf("%s has no covering test in §4", sr)
		}
	}
}
