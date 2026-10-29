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
		RuleDotImport:        1, // internal/sales/dotimport.go
		RuleRuntimeDDL:       3, // platform/bad CREATE TABLE; sales/ddl alter table + GRANT
		RulePlatformImport:   1, // platform/bad imports billing
		RulePgtestImport:     1, // platform/bad imports pgtest from production code
		RuleForeignStore:     1, // sales imports billing/store
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

// TestDotImportRule proves the dot-import rule closes the gap where a bare
// NewLegacyRef call escapes the selector-based legacyref-factory rule.
func TestDotImportRule(t *testing.T) {
	vs, err := Check(filepath.Join("testdata", "violating"), "example.test/mod")
	if err != nil {
		t.Fatal(err)
	}
	const file = "internal/sales/dotimport.go"
	var dot, factory int
	for _, v := range vs {
		if v.File != file {
			continue
		}
		switch v.Rule {
		case RuleDotImport:
			dot++
		case RuleLegacyRefFactory:
			factory++
		}
	}
	if dot != 1 {
		t.Fatalf("dot-import violations in %s = %d, want 1", file, dot)
	}
	if factory != 0 {
		t.Fatalf("bare call is invisible to %s (got %d); the dot-import rule is what catches it", RuleLegacyRefFactory, factory)
	}
	allowed, err := Check(filepath.Join("testdata", "allowed", "internal", "sales"), "example.test/mod")
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range allowed {
		t.Errorf("named/blank imports must be allowed: %s", v)
	}
}

// TestSliceOneRules checks each Slice 1 rule fires where expected and
// stays silent on the allowed fixture (migration DDL, own store, kernel and
// platform imports from platform, pgtest and DDL from test files).
func TestSliceOneRules(t *testing.T) {
	vs, err := Check(filepath.Join("testdata", "violating"), "example.test/mod")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		RulePlatformImport: "internal/platform/bad/bad.go",
		RulePgtestImport:   "internal/platform/bad/bad.go",
		RuleForeignStore:   "internal/sales/store_import.go",
	}
	for rule, file := range want {
		found := false
		for _, v := range vs {
			if v.Rule == rule && v.File == file {
				found = true
			}
		}
		if !found {
			t.Errorf("rule %s did not fire in %s", rule, file)
		}
	}
	ddlFiles := map[string]int{}
	for _, v := range vs {
		if v.Rule == RuleRuntimeDDL {
			ddlFiles[v.File]++
		}
	}
	if ddlFiles["internal/platform/bad/bad.go"] != 1 || ddlFiles["internal/sales/ddl.go"] != 2 {
		t.Errorf("runtime-ddl hits = %v", ddlFiles)
	}
	allowed, err := Check(filepath.Join("testdata", "allowed"), "example.test/mod")
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range allowed {
		t.Errorf("allowed fixture flagged: %s", v)
	}
}

func TestRuntimeDDLPattern(t *testing.T) {
	for _, s := range []string{"CREATE TABLE x", "create  schema y", "Drop Index z", "ALTER ROLE r", "TRUNCATE t", "grant select on t to r", "REVOKE ALL ON t FROM r", "create extension pgcrypto"} {
		if !runtimeDDL.MatchString(s) {
			t.Errorf("%q not detected", s)
		}
	}
	for _, s := range []string{"SELECT * FROM t", "INSERT INTO t VALUES (1)", "UPDATE t SET a = 1", "SET LOCAL ROLE r", "created_at", "table_name", "dropped", "grants", "the grant was"} {
		if runtimeDDL.MatchString(s) {
			t.Errorf("%q falsely detected", s)
		}
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
