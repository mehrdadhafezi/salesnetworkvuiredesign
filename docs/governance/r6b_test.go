package governance

// R6-B (M1 intake) checks: the M1 decision pack, the blank ADR #18 intake forms, the P-GOV worksheets and
// the staging rehearsal checklist add no decision, no approval, no name, no date and no selected option,
// and the R6 tracker stays at its baseline.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var r6bDocs = []string{"r6b-m1-adr18-decision-pack-fa.md", "r6b-m1-pgov-worksheets-fa.md",
	"r6b-m1-staging-rehearsal-checklist-fa.md", "r6b-m1-engineering-eligibility.md"}

// The tracker is unchanged by R6-B: 26 / 30 / 5, ADR #18 review-ready, D-03…D-22 evidence-missing.
func TestR6BTrackerUnchanged(t *testing.T) {
	counts := map[string]int{}
	for _, r := range loadTracker(t).Records {
		counts[r.Status]++
		switch {
		case r.ID == "ADR #18" && r.Status != "READY_FOR_REVIEW":
			t.Errorf("ADR #18 status %s, want READY_FOR_REVIEW", r.Status)
		case r.R5Package == "P-GOV" && r.Status != "EVIDENCE_MISSING":
			t.Errorf("%s status %s, want EVIDENCE_MISSING", r.ID, r.Status)
		}
	}
	want := map[string]int{"READY_FOR_REVIEW": 26, "EVIDENCE_MISSING": 30, "DRAFT": 5}
	for k, v := range want {
		if counts[k] != v {
			t.Errorf("%s: %d entries, want %d", k, counts[k], v)
		}
	}
	if len(counts) != len(want) {
		t.Errorf("unexpected statuses %v", counts)
	}
}

// Blank intake forms are not records: they never validate, carry no name, date, outcome or verification,
// and live outside approvals/.
func TestR6BIntakeBlanksNeverValidate(t *testing.T) {
	dir := filepath.Join(root, "docs/governance/intake/m1")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	types := map[string]bool{}
	for _, e := range entries {
		rec, err := loadApproval(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("%s: %v", e.Name(), err)
		}
		types[rec.RecordType] = true
		if want := strings.TrimSuffix(recordFileName(rec), ".json") + ".BLANK.json"; e.Name() != want {
			t.Errorf("%s must be named %s", e.Name(), want)
		}
		if rec.DecisionID != "ADR #18" {
			t.Errorf("%s: decision_id %q", e.Name(), rec.DecisionID)
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
	}
	for _, want := range []string{"REVIEW_NOTICE", "DECISION", "DEFERRAL"} {
		if !types[want] {
			t.Errorf("missing blank %s form", want)
		}
	}
	// The DECISION blank names exactly the R5 approver roles for ADR #18, each as an independent approval.
	var dec approvalRecord
	b, _ := os.ReadFile(filepath.Join(dir, "ADR_18-v1.0-DECISION.BLANK.json"))
	if err := json.Unmarshal(b, &dec); err != nil {
		t.Fatal(err)
	}
	roles := []string{"Product owner", "Security owner", "Legal owner"}
	if len(dec.Approvers) != 3 || strings.Join(dec.IndependentApprovals, ",") != strings.Join(roles, ",") {
		t.Errorf("ADR #18 blank must list Product, Security and Legal as independent approvals: %v", dec.IndependentApprovals)
	}
}

// P-GOV: twenty non-authoritative worksheets, three unticked options each, no outcome filled.
func TestR6BPGovWorksheets(t *testing.T) {
	doc := read(t, "docs/remediation/r6b-m1-pgov-worksheets-fa.md")
	for _, id := range pgovIDs() {
		if !strings.Contains(doc, "| "+id+" | EVIDENCE_MISSING | هیچ | منتظر مالک مجاز |") {
			t.Errorf("%s: intake status row missing or changed", id)
		}
		parts := strings.Split(doc, "\n### "+id+"\n")
		if len(parts) != 2 {
			t.Errorf("%s: expected one worksheet", id)
			continue
		}
		ws := strings.SplitN(parts[1], "\n### ", 2)[0]
		if !strings.Contains(ws, "EVIDENCE NOT RECOVERED — HUMAN REVIEW REQUIRED") {
			t.Errorf("%s: worksheet lost its non-authoritative marker", id)
		}
		if strings.Count(ws, "☐") != 3 {
			t.Errorf("%s: want exactly three unticked options", id)
		}
	}
}

// No R6-B document pre-selects an option or fills an outcome, answer, check or verifier.
func TestR6BNoPreselectionOrFilledOutcome(t *testing.T) {
	for _, f := range r6bDocs {
		d := read(t, "docs/remediation/"+f)
		if strings.ContainsAny(d, "☑☒✓✔") || strings.Contains(d, "[x]") {
			t.Errorf("%s: a choice is pre-selected", f)
		}
		for _, l := range strings.Split(d, "\n") {
			for _, lab := range outcomeLabels {
				if strings.HasPrefix(l, "| "+lab+" |") && strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(l, "| "+lab+" |"), "|")) != "" {
					t.Errorf("%s: outcome field %q is filled", f, lab)
				}
			}
			cols := strings.Split(strings.Trim(strings.TrimSpace(l), "|"), "|")
			// Management questions (Q1…Q9): answer, respondent and evidence columns stay empty.
			if strings.HasPrefix(l, "| Q") && len(cols) == 5 {
				for _, c := range cols[2:] {
					if strings.TrimSpace(c) != "" {
						t.Errorf("%s: question %s has a filled answer: %q", f, strings.TrimSpace(cols[0]), l)
					}
				}
			}
			// Rehearsal checklist rows: box unticked, evidence and verifier empty.
			if f == "r6b-m1-staging-rehearsal-checklist-fa.md" && len(cols) == 7 && strings.TrimSpace(cols[4]) == "☐" && strings.TrimSpace(cols[0]) != "#" {
				if strings.TrimSpace(cols[5]) != "" || strings.TrimSpace(cols[6]) != "" {
					t.Errorf("checklist row %s is filled", strings.TrimSpace(cols[0]))
				}
			}
		}
	}
}
