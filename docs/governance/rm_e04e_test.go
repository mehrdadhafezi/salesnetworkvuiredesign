package governance

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// RM-E04E: Seller M2 registration preparation. Documentation only. These tests keep the preparation material
// unmistakably unapproved: no preselection, no claimed sign-off, no invented decision id, and candidate drafts
// that can never pass as approval records. They do not freeze the tracker, the gate or the S4 manifest, so a
// legitimate future registration and approval is not blocked by them.

var rmE04EDocs = []string{
	"docs/governance/proposals/rm-e04e-decision-dashboard.fa.md",
	"docs/governance/proposals/rm-e04e-m2-record-1-seller-capability-form.md",
	"docs/governance/proposals/rm-e04e-m2-record-2-seller-row-scope-form.md",
	"docs/governance/proposals/rm-e04e-authority-and-registration-packet.md",
	"docs/implementation/rm-e04e-codex-security-review-handoff.md",
}

const rmE04EDraftDir = "docs/governance/proposals/rm-e04e-candidate-records"

// Outcome rows of the two forms: the value column must stay empty until a real, verified decision exists.
var rmE04EOutcomeRows = []string{"Selected option", "Selected predicates", "Full decision text",
	"Approvers (name, role, authority evidence)", "Decision date", "Effective date", "Conditions",
	"Evidence reference / immutable_ref", "Human verification"}

func TestRME04EDocumentsMarkedNotApproved(t *testing.T) {
	for _, f := range rmE04EDocs {
		d := read(t, f)
		head := d
		if len(head) > 1500 {
			head = head[:1500]
		}
		if !strings.Contains(head, "DRAFT — NOT APPROVED") {
			t.Errorf("%s: no DRAFT — NOT APPROVED marker near the top", f)
		}
		if !strings.Contains(d, "NO-GO") {
			t.Errorf("%s: Production NO-GO not stated", f)
		}
	}
}

func TestRME04ENoPreselectionOrFilledOutcome(t *testing.T) {
	for _, f := range rmE04EDocs {
		d := read(t, f)
		if strings.ContainsAny(d, "☑☒✔✓") || strings.Contains(d, "[x]") || strings.Contains(d, "[X]") {
			t.Errorf("%s: a choice or checklist item is pre-ticked", f)
		}
		for i, l := range strings.Split(d, "\n") {
			for _, row := range rmE04EOutcomeRows {
				if strings.HasPrefix(l, "| "+row+" |") &&
					strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(l, "| "+row+" |"), "|")) != "" {
					t.Errorf("%s:%d outcome row %q is filled", f, i+1, row)
				}
			}
		}
	}
	for _, f := range []string{rmE04EDocs[1], rmE04EDocs[2]} {
		d := read(t, f)
		if !strings.Contains(d, "| Official decision ID | **None.**") {
			t.Errorf("%s: must state that no official decision ID exists", f)
		}
		if strings.Count(d, "| Selected ") < 1 {
			t.Errorf("%s: outcome table missing", f)
		}
	}
}

func TestRME04ENoClaimedSignOff(t *testing.T) {
	for _, f := range rmE04EDocs {
		for i, l := range strings.Split(read(t, f), "\n") {
			if claimedSignOff.MatchString(l) && !regexp.MustCompile(`(?i)pending|not (obtained|recorded|claimed)|no .*(sign-off|review)|required|owed|requested`).MatchString(l) {
				t.Errorf("%s:%d claims a sign-off: %s", f, i+1, strings.TrimSpace(l))
			}
		}
	}
	h := read(t, "docs/implementation/rm-e04e-codex-security-review-handoff.md")
	if !strings.Contains(h, "| Status | **CODEX_REVIEW_PENDING.**") {
		t.Error("the handoff must state CODEX_REVIEW_PENDING until Codex records a review itself")
	}
}

// Only tracker or R5 inventory ids may be cited; the M2 records are referred to by working labels, never by an
// invented id.
func TestRME04ENoInventedDecisionID(t *testing.T) {
	known := ids(read(t, "docs/remediation/r5-decision-inventory.md"))
	for _, r := range loadTracker(t).Records {
		known[strings.SplitN(r.ID, " ", 2)[0]] = true
		for _, part := range strings.Split(r.ID, " / ") {
			known[strings.TrimSpace(part)] = true
		}
	}
	idPattern := regexp.MustCompile(`\b(?:D-[0-9]+[a-e]?|OPD-[0-9]+|R5-[A-Z]+-[0-9]+)\b`)
	for _, f := range rmE04EDocs {
		var unknown []string
		for _, id := range idPattern.FindAllString(read(t, f), -1) {
			if !known[id] && !known[strings.TrimRight(id, "abcde")] {
				unknown = append(unknown, id)
			}
		}
		sort.Strings(unknown)
		if len(unknown) > 0 {
			t.Errorf("%s cites decision IDs not in the tracker: %v", f, unknown)
		}
	}
}

// Candidate drafts live outside approvals/, carry a placeholder id, select nothing, and are rejected by the R6
// validator in repository mode even when every other field is filled and verified.
func TestRME04EDraftsNeverValidate(t *testing.T) {
	entries, err := os.ReadDir(filepath.Join(root, rmE04EDraftDir))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("%s: want exactly the two M2 candidate drafts, found %d", rmE04EDraftDir, len(entries))
	}
	recs := trackerByID(t)
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".CANDIDATE-DECISION.DRAFT.json") {
			t.Errorf("%s: candidate drafts must be named *.CANDIDATE-DECISION.DRAFT.json", name)
		}
		if _, err := os.Stat(filepath.Join(root, "docs/governance/approvals", name)); err == nil {
			t.Errorf("%s copied into approvals/", name)
		}
		r, err := loadApproval(filepath.Join(root, rmE04EDraftDir, name))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if _, tracked := recs[r.DecisionID]; tracked || !strings.Contains(r.DecisionID, "PLACEHOLDER") {
			t.Errorf("%s: decision_id %q must be a placeholder, never a tracker id", name, r.DecisionID)
		}
		if r.Outcome != "" || r.SelectedOption != "" || r.DecisionText != "" || len(r.Approvers) > 0 ||
			len(r.IndependentApprovals) > 0 || r.DecisionDate != "" || r.HumanVerification != nil {
			t.Errorf("%s: an outcome, option, approver or verification is pre-filled", name)
		}
		if lvl, p := validateApprovalRecord(r, true); lvl != levelIncomplete || len(p) == 0 {
			t.Errorf("%s: blank draft validates (%s)", name, lvl)
		}
		// Negative control: complete every field (as a forger would) except the placeholder id.
		filled := r
		filled.Outcome, filled.SelectedOption, filled.DecisionText, filled.Conditions = "APPROVED", "A", "text", "none"
		filled.DecisionDate, filled.EffectiveDate = "2030-01-01", "2030-01-01"
		filled.EvidenceSource.Type, filled.EvidenceSource.Reference, filled.ImmutableRef = "signed minutes", "ref", "abc1234"
		filled.RecordedBy = person{Name: "Recorder", Role: "Recorder"}
		for _, role := range rmE04CGateFloorRoles {
			a := authorizedPerson{Name: "Signer " + role, Role: role}
			a.AuthorityEvidence.Type, a.AuthorityEvidence.Reference = "delegation", "auth-"+role
			filled.Approvers = append(filled.Approvers, a)
			filled.IndependentApprovals = append(filled.IndependentApprovals, role)
		}
		if lvl, _ := validateApprovalRecord(filled, false); lvl != levelRecorded {
			t.Fatalf("%s: negative control miscalibrated: completed draft should be structurally complete outside the repository (%s)", name, lvl)
		}
		if lvl, p := validateApprovalRecord(filled, true); lvl != levelIncomplete || !strings.Contains(strings.Join(p, ";"), "PLACEHOLDER") {
			t.Errorf("%s: a completed draft with its placeholder id is accepted as a repository record (%s)", name, lvl)
		}
	}
}
