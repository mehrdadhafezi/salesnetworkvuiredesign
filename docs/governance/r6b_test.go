package governance

// R6-B (M1 intake) checks: the M1 decision pack, the blank ADR #18 intake forms, the P-GOV worksheets and
// the staging rehearsal checklist add no decision, no approval, no name, no date and no selected option,
// and the R6 tracker stays at its baseline.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
	for _, p := range validateRehearsalChecklist(read(t, "docs/remediation/"+rehearsalChecklist)) {
		t.Errorf("%s: %s", rehearsalChecklist, p)
	}
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
		}
	}
}

const (
	rehearsalChecklist      = "r6b-m1-staging-rehearsal-checklist-fa.md"
	rehearsalChecklistItems = 24
)

// validateRehearsalChecklist checks every row of the rehearsal checklist table, whatever its checkbox value:
// exactly rows 1…24, each once, seven columns, checkbox exactly ☐, evidence and verifier empty. The header
// (first cell "#") and the separator are excluded explicitly; any other row is reported as malformed.
func validateRehearsalChecklist(doc string) []string {
	var p []string
	number := regexp.MustCompile(`^[1-9][0-9]*$`)
	separator := regexp.MustCompile(`^\|(\s*:?-+:?\s*\|)+$`)
	seen := map[int]int{}
	header := 0
	for i, l := range strings.Split(doc, "\n") {
		l = strings.TrimSpace(l)
		if !strings.HasPrefix(l, "|") {
			continue
		}
		if separator.MatchString(l) {
			continue
		}
		cols := strings.Split(strings.TrimSuffix(strings.TrimPrefix(l, "|"), "|"), "|")
		for k := range cols {
			cols[k] = strings.TrimSpace(cols[k])
		}
		if cols[0] == "#" {
			header++
			continue
		}
		where := fmt.Sprintf("line %d", i+1)
		if !number.MatchString(cols[0]) {
			p = append(p, fmt.Sprintf("%s: malformed checklist row number %q", where, cols[0]))
			continue
		}
		n := 0
		fmt.Sscanf(cols[0], "%d", &n)
		seen[n]++
		if n < 1 || n > rehearsalChecklistItems {
			p = append(p, fmt.Sprintf("%s: unexpected checklist row %d", where, n))
		}
		if len(cols) != 7 {
			p = append(p, fmt.Sprintf("row %d: %d columns, want 7", n, len(cols)))
			continue
		}
		if cols[4] != "☐" {
			p = append(p, fmt.Sprintf("row %d: checkbox %q, want exactly ☐", n, cols[4]))
		}
		if cols[5] != "" {
			p = append(p, fmt.Sprintf("row %d: evidence reference is filled (%q)", n, cols[5]))
		}
		if cols[6] != "" {
			p = append(p, fmt.Sprintf("row %d: verifier is filled (%q)", n, cols[6]))
		}
	}
	if header != 1 {
		p = append(p, fmt.Sprintf("%d checklist header rows, want 1", header))
	}
	for n := 1; n <= rehearsalChecklistItems; n++ {
		switch seen[n] {
		case 0:
			p = append(p, fmt.Sprintf("checklist row %d is missing", n))
		case 1:
		default:
			p = append(p, fmt.Sprintf("checklist row %d appears %d times", n, seen[n]))
		}
	}
	return p
}

// Regression: every way the checklist can be altered is rejected, independently of the checkbox value.
func TestR6BRehearsalChecklistRejectsAlterations(t *testing.T) {
	doc := read(t, "docs/remediation/"+rehearsalChecklist)
	if p := validateRehearsalChecklist(doc); len(p) > 0 {
		t.Fatalf("committed checklist must be valid: %v", p)
	}
	row := func(n int) string {
		for _, l := range strings.Split(doc, "\n") {
			if strings.HasPrefix(l, fmt.Sprintf("| %d | ", n)) {
				return l
			}
		}
		t.Fatalf("row %d not found", n)
		return ""
	}
	cell := func(l string, i int, v string) string {
		c := strings.Split(l, " | ")
		c[i] = v
		return strings.Join(c, " | ")
	}
	r1, r5 := row(1), row(5)
	cases := map[string]string{
		"checkbox changed to X":    strings.Replace(doc, r1, cell(r1, 4, "X"), 1),
		"checkbox checked":         strings.Replace(doc, r1, cell(r1, 4, "☑"), 1),
		"row removed":              strings.Replace(doc, r5+"\n", "", 1),
		"row duplicated":           strings.Replace(doc, r5, r5+"\n"+r5, 1),
		"row number changed":       strings.Replace(doc, r5, strings.Replace(r5, "| 5 |", "| 25 |", 1), 1),
		"row number malformed":     strings.Replace(doc, r5, strings.Replace(r5, "| 5 |", "| 5a |", 1), 1),
		"evidence field filled":    strings.Replace(doc, r1, cell(r1, 5, "ref-1"), 1),
		"verifier field filled":    strings.Replace(doc, r1, strings.TrimSuffix(r1, "|  |")+"| Someone |", 1),
		"X with evidence (bypass)": strings.Replace(doc, r1, cell(cell(r1, 4, "X"), 5, "fabricated-ref"), 1),
		"header duplicated":        strings.Replace(doc, "| # | ", "| # | x |\n| # | ", 1),
	}
	for name, d := range cases {
		if d == doc {
			t.Fatalf("%s: mutation did not apply", name)
		}
		p := validateRehearsalChecklist(d)
		if len(p) == 0 {
			t.Errorf("%s: accepted", name)
			continue
		}
		t.Logf("%s → rejected: %s", name, strings.Join(p, "; "))
	}
}
