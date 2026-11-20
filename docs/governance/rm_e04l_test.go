package governance

// RM-E04L — Seller own-case owner preference (documentation-level guards only).
//
// The project owner's preference (read-only own cases; CUSTODY_CURRENT_HOLDER) is annotated in the two existing M2
// forms. These tests keep it a non-authoritative annotation: it is marked NOT APPROVED, never claims a verified
// state, and its report labels are consumed by no gate, tracker, code or contract file. They do not freeze the
// tracker or the gate, so a legitimate future registration and approval is not blocked here.

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const (
	rmE04LHandoff = "docs/implementation/rm-e04l-seller-own-case-preference-handoff.md"
	rmE04LLabel   = "OWNER_PREFERRED_"
)

// form → heading of its RM-E04L section (the section runs to the end of the file).
var rmE04LSections = map[string]string{
	"docs/governance/proposals/rm-e04e-m2-record-1-seller-capability-form.md": "## 13. RM-E04L owner preference",
	"docs/governance/proposals/rm-e04e-m2-record-2-seller-row-scope-form.md":  "## 12. RM-E04L owner preference",
}

// A verified / approved claim that is not the negated governance state NOT_APPROVED_VERIFIED.
var rmE04LClaim = regexp.MustCompile(`(^|[^_A-Z])(APPROVED_VERIFIED|APPROVED —|ACCEPTED|SIGNED|GRANTED)\b`)

func rmE04LSectionProblems(s string) []string {
	var out []string
	for _, m := range []string{"OWNER PREFERENCE — NOT APPROVED", "NOT_APPROVED_VERIFIED", "NO-GO", rmE04LHandoff[len("docs/"):]} {
		if !strings.Contains(s, m) {
			out = append(out, "missing marker "+m)
		}
	}
	for i, l := range strings.Split(s, "\n") {
		if rmE04LClaim.MatchString(l) {
			out = append(out, "line "+strconv.Itoa(i+1)+" claims an approved state: "+strings.TrimSpace(l))
		}
	}
	if strings.ContainsAny(s, "☑☒✔✓") || strings.Contains(s, "[x]") || strings.Contains(s, "[X]") {
		out = append(out, "a choice is ticked")
	}
	return out
}

func TestRME04LPreferenceIsAnnotationOnly(t *testing.T) {
	for f, h := range rmE04LSections {
		d := read(t, f)
		i := strings.Index(d, h)
		if i < 0 {
			t.Errorf("%s: RM-E04L section %q missing", f, h)
			continue
		}
		sec := d[i:]
		for _, p := range rmE04LSectionProblems(sec) {
			t.Errorf("%s: %s", f, p)
		}
		// Negative fixtures: the rule must catch a preference promoted to a decision.
		for name, m := range map[string]string{
			"verified":   strings.Replace(sec, "**`NOT_APPROVED_VERIFIED`**", "APPROVED_VERIFIED", 1),
			"unmarked":   strings.Replace(sec, "OWNER PREFERENCE — NOT APPROVED", "OWNER PREFERENCE", 1),
			"ticked":     sec + "\n☑ B\n",
			"granted":    sec + "\ncases.own.read GRANTED\n",
			"accepted":   sec + "\nADR ACCEPTED\n",
			"production": strings.ReplaceAll(sec, "NO-GO", "GO"),
		} {
			if len(rmE04LSectionProblems(m)) == 0 {
				t.Errorf("%s: negative fixture %q passed the rule", f, name)
			}
		}
	}
	h := read(t, rmE04LHandoff)
	if !strings.Contains(h[:min(len(h), 400)], "DRAFT — NOT APPROVED") || !strings.Contains(h, "NO-GO") {
		t.Errorf("%s: DRAFT — NOT APPROVED / NO-GO not stated", rmE04LHandoff)
	}
	for _, m := range []string{"M2_REC_1 = NOT_APPROVED_VERIFIED", "M2_REC_2 = NOT_APPROVED_VERIFIED", "UNASSIGNED",
		"D-42 approval alone authorizes no Seller read"} {
		if !strings.Contains(h, m) {
			t.Errorf("%s: missing %q", rmE04LHandoff, m)
		}
	}
}

// The preference labels exist only in prose. No gate, tracker, manifest, code, contract or migration may use them.
func TestRME04LPreferenceIsNotConsumed(t *testing.T) {
	scanned := map[string]bool{".go": true, ".json": true, ".sql": true, ".yaml": true, ".yml": true, ".ts": true,
		".tsx": true, ".js": true, ".sh": true, ".toml": true}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if n := d.Name(); n == ".git" || n == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if !scanned[filepath.Ext(p)] || rel == "docs/governance/rm_e04l_test.go" {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if strings.Contains(string(b), rmE04LLabel) {
			t.Errorf("%s uses the owner-preference label %q; a preference is never machine-consumable", rel, rmE04LLabel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
