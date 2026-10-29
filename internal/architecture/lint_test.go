package architecture

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const repoRoot = "../.."

func modulePath(t *testing.T) string {
	t.Helper()
	f, err := os.Open(filepath.Join(repoRoot, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		if line := strings.TrimSpace(s.Text()); strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	t.Fatal("module line not found")
	return ""
}

// TestRepositoryPassesArchitectureLint is the architecture gate run in CI.
func TestRepositoryPassesArchitectureLint(t *testing.T) {
	vs, err := Check(repoRoot, modulePath(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range vs {
		t.Error(v)
	}
}

// TestViolatingFixtureFails proves every rule fires on a deliberately bad tree.
func TestViolatingFixtureFails(t *testing.T) {
	vs, err := Check(filepath.Join("testdata", "violating"), "example.test/mod")
	if err != nil {
		t.Fatal(err)
	}
	hits := map[string]int{}
	for _, v := range vs {
		hits[v.Rule]++
	}
	want := map[string]int{
		RuleKernelImport:     4, // business module, net/http, os, math/rand in production code
		RuleKernelClock:      2, // time.Now and .Local()
		RuleKernelFloat:      2, // float64 parameter and float32 conversion
		RuleMoneyConversion:  1,
		RuleLegacyLiteral:    2, // one in kernel, one in a business package
		RuleLegacyRefFactory: 1,
	}
	for rule, n := range want {
		if hits[rule] != n {
			t.Errorf("rule %s: got %d violations, want %d", rule, hits[rule], n)
		}
	}
	for rule := range hits {
		if _, ok := want[rule]; !ok {
			t.Errorf("unexpected rule %s", rule)
		}
	}
	if t.Failed() {
		for _, v := range vs {
			t.Log(v)
		}
	}
}

// TestFixtureAllowedCases proves the allowed paths stay clean.
func TestFixtureAllowedCases(t *testing.T) {
	vs, err := Check(filepath.Join("testdata", "allowed"), "example.test/mod")
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range vs {
		t.Errorf("unexpected violation in allowed fixture: %s", v)
	}
}

func TestViolationString(t *testing.T) {
	v := Violation{Rule: "r", File: "a/b.go", Line: 3, Text: "x"}
	if v.String() != "a/b.go:3: [r] x" {
		t.Fatalf("String = %q", v.String())
	}
}

func TestCheckMissingRoot(t *testing.T) {
	if _, err := Check(filepath.Join("testdata", "does-not-exist"), "m"); err == nil {
		t.Fatal("missing root must error")
	}
}
