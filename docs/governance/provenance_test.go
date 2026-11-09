package governance

// Evidence provenance checks (R5 review finding G2).
//
// What is verified by automation, per evidence item with level TEXT_AT_COMMIT_VERIFIED:
//   - every commit named in "commit" exists and is an ancestor of the tested HEAD;
//   - the cited path exists in each of those commits;
//   - the exact "anchor" text occurs in that path at the first commit named.
//
// What is NOT verified: who wrote or approved the text, whether the "fact" summary is a complete and
// correct reading, and anything outside git (GitHub threads, owner statements, negative claims). Passing
// these checks is text-at-commit provenance, never independent authentication.
//
// The checks need full git history. Without it they are reported as NOT VERIFIED and skipped, unless
// R5_REQUIRE_PROVENANCE=1 (set in CI), in which case the test fails.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

const (
	levelVerified    = "TEXT_AT_COMMIT_VERIFIED"
	levelNotVerified = "NOT_VERIFIED"
)

var commitToken = regexp.MustCompile(`\b[0-9a-f]{7,40}\b`)

type evidenceItem struct {
	ID          string `json:"id"`
	Path        string `json:"path"`
	Commit      string `json:"commit"`
	Anchor      string `json:"anchor"`
	Provenance  string `json:"provenance"`
	NotVerified string `json:"not_verified"`
}

type evidenceIndex struct {
	VerificationScope string            `json:"verification_scope"`
	Levels            map[string]string `json:"provenance_levels"`
	Evidence          []evidenceItem    `json:"evidence"`
}

func git(args ...string) (string, error) {
	out, err := exec.Command("git", append([]string{"-C", root}, args...)...).Output()
	return string(out), err
}

// requireHistory skips (NOT VERIFIED) or fails when full git history is unavailable.
func requireHistory(t *testing.T) {
	t.Helper()
	out, err := git("rev-parse", "--is-shallow-repository")
	reason := ""
	switch {
	case err != nil:
		reason = "git history is unavailable: " + err.Error()
	case strings.TrimSpace(out) != "false":
		reason = "the clone is shallow"
	}
	if reason == "" {
		return
	}
	if os.Getenv("R5_REQUIRE_PROVENANCE") == "1" {
		t.Fatalf("evidence provenance cannot be verified (%s) and R5_REQUIRE_PROVENANCE=1", reason)
	}
	t.Skipf("evidence provenance NOT VERIFIED: %s (set R5_REQUIRE_PROVENANCE=1 to fail instead)", reason)
}

// verifyProvenance checks one evidence item against git history. It returns an error when the item's
// provenance level claims more than the repository proves.
func verifyProvenance(e evidenceItem) error {
	switch e.Provenance {
	case levelVerified:
		if strings.TrimSpace(e.Anchor) == "" {
			return errors.New("level TEXT_AT_COMMIT_VERIFIED without an anchor")
		}
	case levelNotVerified:
		if e.Anchor != "" {
			return errors.New("anchor present but level NOT_VERIFIED")
		}
		return nil // nothing beyond path existence and commit syntax is claimed
	default:
		return fmt.Errorf("unknown provenance level %q", e.Provenance)
	}
	commits := commitToken.FindAllString(e.Commit, -1)
	if len(commits) == 0 {
		return fmt.Errorf("no commit reference in %q", e.Commit)
	}
	for _, c := range commits {
		if _, err := git("cat-file", "-e", c+"^{commit}"); err != nil {
			return fmt.Errorf("commit %s does not exist", c)
		}
		if _, err := git("merge-base", "--is-ancestor", c, "HEAD"); err != nil {
			return fmt.Errorf("commit %s is not an ancestor of HEAD", c)
		}
		if _, err := git("cat-file", "-e", c+":"+e.Path); err != nil {
			return fmt.Errorf("path %s does not exist at %s", e.Path, c)
		}
	}
	content, err := git("show", commits[0]+":"+e.Path)
	if err != nil {
		return fmt.Errorf("cannot read %s at %s", e.Path, commits[0])
	}
	if !strings.Contains(content, e.Anchor) {
		return fmt.Errorf("anchor %q not found in %s at %s", e.Anchor, e.Path, commits[0])
	}
	return nil
}

func loadIndex(t *testing.T) evidenceIndex {
	t.Helper()
	var idx evidenceIndex
	if err := json.Unmarshal([]byte(read(t, "docs/governance/r5-evidence-index.json")), &idx); err != nil {
		t.Fatal(err)
	}
	return idx
}

// The index states the limits of its own verification and claims no authentication.
func TestEvidenceVerificationScope(t *testing.T) {
	idx := loadIndex(t)
	for _, want := range []string{"does NOT authenticate", "No item is independently authenticated"} {
		if !strings.Contains(idx.VerificationScope, want) {
			t.Errorf("verification_scope must state %q", want)
		}
	}
	if len(idx.Levels) != 2 || idx.Levels[levelVerified] == "" || idx.Levels[levelNotVerified] == "" {
		t.Errorf("provenance_levels must define exactly %s and %s", levelVerified, levelNotVerified)
	}
	raw := read(t, "docs/governance/r5-evidence-index.json")
	if regexp.MustCompile(`"(authenticated|independently_authenticated)"\s*:`).MatchString(raw) {
		t.Error("the evidence index must not carry an authentication field")
	}
}

// Every evidence item's provenance level is backed by git history.
func TestEvidenceProvenance(t *testing.T) {
	requireHistory(t)
	idx := loadIndex(t)
	verified := 0
	for _, e := range idx.Evidence {
		if err := verifyProvenance(e); err != nil {
			t.Errorf("%s: %v", e.ID, err)
			continue
		}
		if e.Provenance == levelVerified {
			verified++
		}
	}
	t.Logf("text-at-commit provenance verified for %d of %d items (not independent authentication)", verified, len(idx.Evidence))
}

// Regression: the verifier rejects every way an item can claim more than git proves.
func TestProvenanceVerifierRejects(t *testing.T) {
	requireHistory(t)
	const arch = "docs/implementation/CRM-STANDALONE-TARGET-ARCHITECTURE-V1.md"
	const anchor = "**ODR D-35 a–e:** (a) cash credit"
	ok := evidenceItem{ID: "ok", Path: arch, Commit: "560ca6c", Anchor: anchor, Provenance: levelVerified}
	if err := verifyProvenance(ok); err != nil {
		t.Fatalf("a genuine item must verify: %v", err)
	}
	mutate := func(f func(*evidenceItem)) evidenceItem { e := ok; f(&e); return e }
	cases := map[string]evidenceItem{
		"unknown commit":        mutate(func(e *evidenceItem) { e.Commit = "0000000" }),
		"unknown second commit": mutate(func(e *evidenceItem) { e.Commit = "560ca6c / 1111111" }),
		"malformed commit":      mutate(func(e *evidenceItem) { e.Commit = "HEAD~1" }),
		"path absent at commit": mutate(func(e *evidenceItem) { e.Path = "docs/remediation/r5-decision-inventory.md" }),
		"anchor absent":         mutate(func(e *evidenceItem) { e.Anchor = "text R5 invented that the architecture never contained" }),
		"anchor only in later rev": mutate(func(e *evidenceItem) {
			// The R4 addendum exists in go-no-go-report.md at 9d6b49b, not in the Slice 15 version.
			e.Path, e.Commit, e.Anchor = "docs/cutover/go-no-go-report.md", "7f1eb94", "## Addendum — R4 sales and finance readiness (2026-11-09)"
		}),
		"verified without anchor": mutate(func(e *evidenceItem) { e.Anchor = "" }),
		"unknown level":           mutate(func(e *evidenceItem) { e.Provenance = "AUTHENTICATED" }),
		"anchor but NOT_VERIFIED": mutate(func(e *evidenceItem) { e.Provenance = levelNotVerified }),
	}
	for name, e := range cases {
		if err := verifyProvenance(e); err == nil {
			t.Errorf("%s: verifier accepted an unproven item", name)
		}
	}
}
