package governance

// RM-P10A identity and access decision-closure package: documentation
// consistency only. These tests read the package and the decision tracker;
// they enforce no runtime behaviour and approve nothing.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const (
	rmP10AMain   = "docs/decisions/rm-p10a-identity-access-closure-package.md"
	rmP10AMatrix = "docs/decisions/rm-p10a/decision-dependency-matrix.md"
	rmP10AGraph  = "docs/governance/proposals/rm-e04i-dependency-graph.json"
)

var (
	rmP10AFiles = []string{
		rmP10AMain,
		"docs/decisions/rm-p10a/identity-person-account-contract.md",
		"docs/decisions/rm-p10a/row-scope-contract.md",
		"docs/decisions/rm-p10a/d34-account-provisioning-contract.md",
		"docs/decisions/rm-p10a/d42-grant-revoke-contract.md",
		rmP10AMatrix,
		"docs/decisions/rm-p10a/pilot-security-controls.md",
		"docs/decisions/rm-p10a/rm-p10b-delivery-plan.md",
	}
	// The only status labels the package may use (task RM-P10A §7).
	rmP10AStatuses = []string{"EXISTING_VERIFIED", "PROPOSED", "REQUIRES_APPROVAL", "BLOCKED", "UNKNOWN"}
	rmP10AApproved = regexp.MustCompile(`\bAPPROVED\b`)
	// Tracker id → short id used in the matrix first column.
	rmP10ADecisions = map[string]string{
		"R5-SEC-01 (proposed ADR)": "R5-SEC-01", "D-34": "D-34", "D-42": "D-42", "D-32": "D-32",
		"D-45": "D-45", "D-23": "D-23", "ADR #18": "ADR #18", "OPD-05": "OPD-05", "SD-04": "SD-04",
	}
)

func rmP10AStatusOK(cell string) bool {
	cell = strings.TrimSpace(cell)
	for _, s := range rmP10AStatuses {
		if strings.HasPrefix(cell, "`"+s+"`") {
			return true
		}
	}
	return false
}

func rmP10ACells(line string) []string {
	parts := strings.Split(strings.Trim(strings.TrimSpace(line), "|"), "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// Every file exists, carries the DRAFT and NO-GO markers, pre-ticks nothing
// and never uses APPROVED as a status (only in "NOT APPROVED").
func TestRMP10APackageMarkers(t *testing.T) {
	for _, f := range rmP10AFiles {
		doc := read(t, f)
		for _, m := range []string{"DRAFT — NOT APPROVED", "GROUP B CLOSED — PRODUCTION NO-GO"} {
			if !strings.Contains(doc, m) {
				t.Errorf("%s: missing marker %q", f, m)
			}
		}
		if strings.ContainsAny(doc, "☑☒") || strings.Contains(doc, "[x]") {
			t.Errorf("%s: a choice is pre-ticked", f)
		}
		for _, loc := range rmP10AApproved.FindAllStringIndex(doc, -1) {
			if !strings.HasSuffix(doc[:loc[0]], "NOT ") {
				line := doc[strings.LastIndex(doc[:loc[0]], "\n")+1:]
				if i := strings.Index(line, "\n"); i >= 0 {
					line = line[:i]
				}
				t.Errorf("%s: APPROVED used as a status: %q", f, line)
			}
		}
	}
}

// Every table column headed exactly "وضعیت" holds one of the five labels.
func TestRMP10AStatusVocabulary(t *testing.T) {
	for _, f := range rmP10AFiles {
		col, rows, inCode := -1, 0, false
		for _, l := range strings.Split(read(t, f), "\n") {
			if strings.HasPrefix(strings.TrimSpace(l), "```") {
				inCode = !inCode
				continue
			}
			if inCode || !strings.HasPrefix(l, "|") {
				col = -1
				continue
			}
			cells := rmP10ACells(l)
			if col == -1 {
				col = -2 // header seen, no status column unless found
				for i, c := range cells {
					if c == "وضعیت" {
						col = i
					}
				}
				continue
			}
			if col < 0 || strings.HasPrefix(cells[0], "---") {
				continue
			}
			if col >= len(cells) || !rmP10AStatusOK(cells[col]) {
				t.Errorf("%s: status cell not in vocabulary: %q", f, l)
			}
			rows++
		}
		if rows == 0 {
			t.Errorf("%s: no status-labelled table row", f)
		}
	}
}

// While a decision has no approval record, the matrix may only call it
// REQUIRES_APPROVAL or BLOCKED, and the tracker must not claim it verified.
func TestRMP10ADecisionsStayUnapproved(t *testing.T) {
	matrix := read(t, rmP10AMatrix)
	seen := map[string]bool{}
	for _, r := range loadTracker(t).Records {
		short, ok := rmP10ADecisions[r.ID]
		if !ok {
			continue
		}
		seen[r.ID] = true
		if r.ApprovalRecord != "" {
			continue // a real record exists; this package no longer governs the label
		}
		if strings.HasSuffix(r.Status, "_VERIFIED") || strings.TrimSpace(r.Outcome["selected_option"]) != "" {
			t.Errorf("%s: tracker claims %q / option %q without an approval record", r.ID, r.Status, r.Outcome["selected_option"])
		}
		row := ""
		for _, l := range strings.Split(matrix, "\n") {
			if strings.HasPrefix(l, "| "+short+" |") {
				row = l
			}
		}
		if row == "" {
			t.Errorf("%s: missing from %s", short, rmP10AMatrix)
			continue
		}
		cells := rmP10ACells(row)
		last := cells[len(cells)-1]
		if !strings.HasPrefix(last, "`REQUIRES_APPROVAL`") && !strings.HasPrefix(last, "`BLOCKED`") {
			t.Errorf("%s: matrix status %q although no approval record exists", short, last)
		}
	}
	for id := range rmP10ADecisions {
		if !seen[id] {
			t.Errorf("%s: not found in the tracker", id)
		}
	}
}

// The package proposes a design resolution of OF-4 but must not present it as
// a governance resolution: while the tracker prerequisites stay circular, the
// RM-E04I graph keeps both cycles UNRESOLVED and the package says so.
func TestRMP10ACycleRecordedNotResolved(t *testing.T) {
	var tr struct {
		Records []struct{ ID, Prerequisites string } `json:"records"`
	}
	if err := json.Unmarshal([]byte(read(t, "docs/governance/r6-decision-tracker.json")), &tr); err != nil {
		t.Fatal(err)
	}
	prereq := map[string]string{}
	for _, r := range tr.Records {
		prereq[r.ID] = r.Prerequisites
	}
	circular := strings.Contains(prereq["R5-SEC-01 (proposed ADR)"], "D-34") &&
		strings.Contains(prereq["R5-SEC-01 (proposed ADR)"], "D-42") &&
		strings.Contains(prereq["D-34"], "R5-SEC-01") && strings.Contains(prereq["D-42"], "R5-SEC-01")
	if !circular {
		t.Skip("tracker prerequisites no longer circular; HQ-10 was decided through a record")
	}
	var g struct {
		Conflicts []struct{ ID, Status string } `json:"unresolved_conflicts"`
	}
	if err := json.Unmarshal([]byte(read(t, rmP10AGraph)), &g); err != nil {
		t.Fatal(err)
	}
	status := map[string]string{}
	for _, c := range g.Conflicts {
		status[c.ID] = c.Status
	}
	for _, id := range []string{"OF-4a", "OF-4b"} {
		if status[id] != "UNRESOLVED" {
			t.Errorf("%s: %q in %s while the tracker is still circular", id, status[id], rmP10AGraph)
		}
	}
	main := read(t, rmP10AMain)
	for _, m := range []string{"HQ-10", "GOVERNANCE_UNRESOLVED", "BOOT-1"} {
		if !strings.Contains(main, m) {
			t.Errorf("%s: missing %q", rmP10AMain, m)
		}
	}
}

// The decision package directory holds documents only: no code, migration,
// configuration or machine-readable record that a gate could consume.
func TestRMP10ANoImplementationArtifacts(t *testing.T) {
	err := filepath.WalkDir(filepath.Join(root, "docs/decisions"), func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(p) != ".md" {
			t.Errorf("%s: only Markdown documents belong in docs/decisions", p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
