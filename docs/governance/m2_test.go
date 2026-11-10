package governance

// M2 (authorization decision intake) checks: the M2 package and the engineering readiness note add no decision,
// no approval, no name, no date, no selected option and no new decision ID; the blank intake forms for D-42,
// D-34 and R5-SEC-01 never validate as records; the three decisions stay READY_FOR_REVIEW in the tracker.

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

var m2Docs = []string{"m2-authorization-decision-intake-fa.md", "m2-engineering-readiness.md"}

// R5 required approver roles per M2 decision, each an independent approval.
var m2Approvers = map[string][]string{
	"D-42":                     {"Product owner", "HR owner", "Sales owner", "Security owner"},
	"D-34":                     {"Security owner", "HR owner"},
	"R5-SEC-01 (proposed ADR)": {"Security owner", "HR owner", "Legal owner", "Product owner"},
}

func TestM2TrackerUnchanged(t *testing.T) {
	seen := map[string]bool{}
	for _, r := range loadTracker(t).Records {
		if _, ok := m2Approvers[r.ID]; !ok {
			continue
		}
		seen[r.ID] = true
		if r.Status != "READY_FOR_REVIEW" || r.ApprovalRecord != "" || r.Verification != "NOT_STARTED" || r.Meeting != "M2" {
			t.Errorf("%s: status %s, record %q, verification %s, meeting %s; want READY_FOR_REVIEW, none, NOT_STARTED, M2",
				r.ID, r.Status, r.ApprovalRecord, r.Verification, r.Meeting)
		}
		for k, v := range r.Outcome {
			if v != "" {
				t.Errorf("%s: outcome %s is filled", r.ID, k)
			}
		}
	}
	for id := range m2Approvers {
		if !seen[id] {
			t.Errorf("%s missing from the tracker", id)
		}
	}
}

func TestM2IntakeBlanksNeverValidate(t *testing.T) {
	dir := filepath.Join(root, "docs/governance/intake/m2")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	types := map[string]map[string]bool{}
	for _, e := range entries {
		rec, err := loadApproval(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("%s: %v", e.Name(), err)
		}
		roles, ok := m2Approvers[rec.DecisionID]
		if !ok {
			t.Errorf("%s: decision_id %q is not an M2 tracker ID", e.Name(), rec.DecisionID)
			continue
		}
		if types[rec.DecisionID] == nil {
			types[rec.DecisionID] = map[string]bool{}
		}
		types[rec.DecisionID][rec.RecordType] = true
		if want := strings.TrimSuffix(recordFileName(rec), ".json") + ".BLANK.json"; e.Name() != want {
			t.Errorf("%s must be named %s", e.Name(), want)
		}
		if lvl, _ := validateApprovalRecord(rec, true); lvl != levelIncomplete {
			t.Errorf("%s validates as %s; a blank must never be a record", e.Name(), lvl)
		}
		filled := rec.Outcome != "" && rec.Outcome != "DEFERRED" || rec.SelectedOption != "" || rec.DecisionText != "" ||
			rec.DecisionDate != "" || rec.EffectiveDate != "" || rec.NoticeDate != "" || rec.DeferralDate != "" ||
			rec.RevisitDate != "" || rec.DeferralReason != "" || rec.RevisitCondition != "" || rec.Conditions != "" ||
			rec.EvidenceSource.Reference != "" || rec.ImmutableRef != "" || rec.RecordedBy.Name != "" || rec.HumanVerification != nil
		if rec.Initiator != nil && (rec.Initiator.Name != "" || rec.Initiator.AuthorityEvidence.Reference != "") {
			filled = true
		}
		for _, a := range rec.Approvers {
			if a.Name != "" || a.AuthorityEvidence.Reference != "" {
				filled = true
			}
		}
		if filled {
			t.Errorf("%s: a blank intake form carries filled decision, name, date or evidence content", e.Name())
		}
		if rec.RecordType == "DECISION" {
			var got []string
			for _, a := range rec.Approvers {
				got = append(got, a.Role)
			}
			if strings.Join(got, ",") != strings.Join(roles, ",") || strings.Join(rec.IndependentApprovals, ",") != strings.Join(roles, ",") {
				t.Errorf("%s: approvers %v / independent %v, want the R5 roles %v", e.Name(), got, rec.IndependentApprovals, roles)
			}
		}
	}
	for id := range m2Approvers {
		for _, want := range []string{"REVIEW_NOTICE", "DECISION", "DEFERRAL"} {
			if !types[id][want] {
				t.Errorf("%s: missing blank %s form", id, want)
			}
		}
	}
}

// No M2 document pre-selects an option or fills an outcome or a management answer.
func TestM2NoPreselectionOrFilledOutcome(t *testing.T) {
	for _, f := range m2Docs {
		d := read(t, "docs/remediation/"+f)
		if !strings.Contains(d, "DRAFT — NOT APPROVED") {
			t.Errorf("%s: missing DRAFT — NOT APPROVED marker", f)
		}
		if strings.ContainsAny(d, "☑☒✓✔") || strings.Contains(d, "[x]") {
			t.Errorf("%s: a choice is pre-selected", f)
		}
		for _, l := range strings.Split(d, "\n") {
			for _, lab := range outcomeLabels {
				if strings.HasPrefix(l, "| "+lab+" |") && strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(l, "| "+lab+" |"), "|")) != "" {
					t.Errorf("%s: outcome field %q is filled", f, lab)
				}
			}
			// Management questions Q1…Qn: answer, respondent and evidence columns stay empty.
			cols := strings.Split(strings.Trim(strings.TrimSpace(l), "|"), "|")
			if strings.HasPrefix(l, "| Q") && len(cols) == 6 {
				for _, c := range cols[3:] {
					if strings.TrimSpace(c) != "" {
						t.Errorf("%s: question %s has a filled answer: %q", f, strings.TrimSpace(cols[0]), l)
					}
				}
			}
		}
	}
	// The three existing decisions keep exactly their R5 option counts, all unticked.
	d := read(t, "docs/remediation/m2-authorization-decision-intake-fa.md")
	for id, n := range map[string]int{"### ۳-۱. D-42": 3, "### ۳-۲. D-34": 3, "### ۳-۳. R5-SEC-01": 4} {
		parts := strings.SplitN(d, id, 2)
		if len(parts) != 2 {
			t.Errorf("%s: section missing", id)
			continue
		}
		sec := strings.SplitN(parts[1], "\n### ", 2)[0]
		for _, l := range strings.Split(sec, "\n") {
			if strings.HasPrefix(l, "| گزینه‌ها (عین R5") {
				if c := strings.Count(l, "☐"); c != n {
					t.Errorf("%s: %d options, want the %d R5 options", id, c, n)
				}
			}
		}
	}
}

// Every decision ID cited by the M2 documents already exists in the tracker: the documents invent no ID, and
// the two proposed decisions carry no ID.
func TestM2NoInventedDecisionID(t *testing.T) {
	known := map[string]bool{}
	for _, r := range loadTracker(t).Records {
		known[strings.SplitN(r.ID, " ", 2)[0]] = true
		for _, part := range strings.Split(r.ID, " / ") {
			known[strings.TrimSpace(part)] = true
		}
	}
	idPattern := regexp.MustCompile(`\b(?:D-[0-9]+[a-e]?|OPD-[0-9]+|R5-[A-Z]+-[0-9]+)\b`)
	for _, f := range m2Docs {
		var unknown []string
		for _, id := range idPattern.FindAllString(read(t, "docs/remediation/"+f), -1) {
			if !known[id] && !known[strings.TrimRight(id, "abcde")] {
				unknown = append(unknown, id)
			}
		}
		sort.Strings(unknown)
		if len(unknown) > 0 {
			t.Errorf("%s cites decision IDs not in the tracker: %v", f, unknown)
		}
	}
	d := read(t, "docs/remediation/m2-authorization-decision-intake-fa.md")
	for _, sec := range []string{"### ۴-۱.", "### ۴-۲."} {
		parts := strings.SplitN(d, sec, 2)
		if len(parts) != 2 || !strings.Contains(strings.SplitN(parts[1], "\n### ", 2)[0], "| شناسه | **ندارد**") {
			t.Errorf("%s: the proposed decision must state that it has no ID", sec)
		}
	}
}
