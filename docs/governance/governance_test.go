// Package governance holds R5 documentation consistency checks. It has no
// production code: it only reads the decision documents and fails CI when a
// decision ID is missing from the inventory, when a DRAFT record carries an
// approval it cannot have, or when the evidence index is inconsistent.
package governance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

const root = "../.."

var (
	// Decision identifiers. D-xx must not be part of OPD-xx (letters before it).
	decisionID  = regexp.MustCompile(`(?:^|[^A-Za-z0-9])((?:D-[0-9]{2}[a-e]?)|(?:OPD-[0-9]{2})|(?:ADR-[0-9]{2})|(?:ADR #[0-9]+)|(?:SD-0[0-9])|(?:F(?:0[1-9]|10))|(?:E-[1-7])|(?:TAD #[0-9]+))(?:[^0-9A-Za-z]|$)`)
	blankFields = []string{"Approval date", "Named approvers", "Selected option", "Conditions of approval", "Evidence of approval", "Effective date"}
	packages    = []string{"r5-sales-approval-package-DRAFT.md", "r5-finance-approval-packages-DRAFT.md", "r5-person-identity-sod-DRAFT.md",
		"r5-hr-handover-approval-DRAFT.md", "r5-biawin-reporting-governance-DRAFT.md"}
	// The only decisions with an authoritative approval on main (E0 §7A).
	approvedAllowed = map[string]bool{"D-01": true, "D-02": true, "MONEY (E0)": true}
)

func read(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func markdownFiles(t *testing.T) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(filepath.Join(root, "docs"), func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(p, ".md") {
			out = append(out, p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return append(out, filepath.Join(root, "AGENTS.md"))
}

func ids(s string) map[string]bool {
	out := map[string]bool{}
	for _, m := range decisionID.FindAllStringSubmatch(s, -1) {
		out[m[1]] = true
	}
	return out
}

// Every decision ID referenced anywhere in the documentation has a row in
// the R5 inventory: nothing is silently omitted.
func TestEveryDecisionIDIsInventoried(t *testing.T) {
	inventory := ids(read(t, "docs/remediation/r5-decision-inventory.md"))
	var missing []string
	for _, f := range markdownFiles(t) {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for id := range ids(string(b)) {
			if !inventory[id] {
				missing = append(missing, id+" ("+strings.TrimPrefix(f, root+"/")+")")
			}
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("decision IDs referenced in docs but missing from r5-decision-inventory.md:\n%s", strings.Join(missing, "\n"))
	}
	for n := 3; n <= 22; n++ {
		id := "D-" + map[bool]string{true: "0", false: ""}[n < 10] + itoa(n)
		if !inventory[id] {
			t.Fatalf("%s must have its own inventory row", id)
		}
	}
}

func itoa(n int) string {
	const digits = "0123456789"
	if n < 10 {
		return digits[n : n+1]
	}
	return itoa(n/10) + digits[n%10:n%10+1]
}

// Every R5 decision record is DRAFT — NOT APPROVED and its approval fields
// are empty (no approval evidence exists on main).
func TestDraftRecordsCarryNoApproval(t *testing.T) {
	total := 0
	for _, p := range packages {
		doc := read(t, "docs/remediation/"+p)
		blocks := strings.Split(doc, "\n### ")[1:]
		if len(blocks) == 0 {
			t.Fatalf("%s has no decision record", p)
		}
		for _, b := range blocks {
			title := strings.SplitN(b, "\n", 2)[0]
			if !strings.Contains(b, "**DRAFT — NOT APPROVED**") {
				t.Errorf("%s: record %q is not marked DRAFT — NOT APPROVED", p, title)
			}
			for _, f := range blankFields {
				line := ""
				for _, l := range strings.Split(b, "\n") {
					if strings.HasPrefix(l, "| "+f+" |") {
						line = l
					}
				}
				if line == "" {
					t.Errorf("%s: record %q lacks the field %q", p, title, f)
					continue
				}
				if v := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "| "+f+" |"), "|")); v != "" {
					t.Errorf("%s: record %q has a non-empty %q (%q) without approval evidence", p, title, f, v)
				}
			}
			total++
		}
	}
	var idx struct {
		Records []struct{ ID string } `json:"decision_records"`
	}
	if err := json.Unmarshal([]byte(read(t, "docs/governance/r5-evidence-index.json")), &idx); err != nil {
		t.Fatal(err)
	}
	if total != len(idx.Records) {
		t.Fatalf("package records %d != evidence-index records %d", total, len(idx.Records))
	}
}

// No status in the inventory says APPROVED except the E0 approvals.
func TestNoUnapprovedApprovedStatus(t *testing.T) {
	for _, l := range strings.Split(read(t, "docs/remediation/r5-decision-inventory.md"), "\n") {
		if !strings.HasPrefix(l, "| ") || strings.HasPrefix(l, "| ID ") || strings.HasPrefix(l, "|---") {
			continue
		}
		cols := strings.Split(l, " | ")
		if len(cols) < 4 {
			continue
		}
		id, status := strings.TrimPrefix(cols[0], "| "), cols[3]
		if strings.Contains(status, "APPROVED") && !strings.Contains(status, "NOT APPROVED") && !approvedAllowed[id] {
			t.Errorf("inventory row %s claims %q without an authoritative approval record", id, status)
		}
	}
	// P-GOV: no owner choice is pre-filled.
	gov := read(t, "docs/remediation/r5-d03-d22-evidence-recovery.md")
	if strings.ContainsAny(gov, "☑☒") || strings.Contains(gov, "[x]") {
		t.Fatal("a P-GOV choice is pre-filled")
	}
}

// The machine-readable evidence index is consistent.
func TestEvidenceIndex(t *testing.T) {
	var idx struct {
		Classes  []string `json:"classes"`
		Evidence []struct {
			ID, Fact, Cls, Path, Section, Commit string
		} `json:"evidence"`
		Records []struct {
			ID, Package, Status, Readiness string
			Evidence                       []string
		} `json:"decision_records"`
	}
	if err := json.Unmarshal([]byte(read(t, "docs/governance/r5-evidence-index.json")), &idx); err != nil {
		t.Fatal(err)
	}
	classes := map[string]bool{}
	for _, c := range idx.Classes {
		classes[c] = true
	}
	seen := map[string]bool{}
	commit := regexp.MustCompile(`^[0-9a-f]{7,40}\b`)
	for _, e := range idx.Evidence {
		if seen[e.ID] {
			t.Errorf("duplicate evidence id %s", e.ID)
		}
		seen[e.ID] = true
		if !classes[e.Cls] {
			t.Errorf("%s: unknown class %q", e.ID, e.Cls)
		}
		if _, err := os.Stat(filepath.Join(root, e.Path)); err != nil {
			t.Errorf("%s: path %s does not exist", e.ID, e.Path)
		}
		if !commit.MatchString(e.Commit) {
			t.Errorf("%s: commit %q is not a commit reference", e.ID, e.Commit)
		}
	}
	for _, r := range idx.Records {
		if r.Status != "DRAFT — NOT APPROVED" {
			t.Errorf("record %s status %q", r.ID, r.Status)
		}
		for _, ev := range r.Evidence {
			if !seen[ev] {
				t.Errorf("record %s cites unknown evidence %s", r.ID, ev)
			}
		}
	}
}
