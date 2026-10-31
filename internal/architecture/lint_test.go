package architecture

import (
	"bufio"
	"os"
	"os/exec"
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
		RuleLegacyLiteral:    4, // kernel, business package, WooCommerce literal, .sql file
		RuleLegacyRefFactory: 1,
		RuleDotImport:        1, // internal/sales/dotimport.go
		RuleRuntimeDDL:       3, // platform/bad CREATE TABLE; sales/ddl alter table + GRANT
		RulePlatformImport:   1, // platform/bad imports billing
		RulePgtestImport:     2, // pgtest from platform/bad; legacytest from the import tool
		RuleForeignStore:     3, // sales imports billing/store; authorization imports organization/store; connector imports customers/store
		RuleLegacyImport:     3, // sales, crm-api and crm-migrate (non-migrations) import the connector
		RuleModuleBoundary:   7, // identity→organization/api, organization→identity/query, authorization→organization/store, connector→organization/imports; customers→organization/query, customers/http→identity/query, connector→customers/store
		RuleForeignWrite:     2, // connector INSERT INTO organization.*, UPDATE customers.*
		RuleReadPathImport:   3, // organization/http, customers/http and crm-api import an import command
		RuleMergePath:        2, // customers MergeCustomers, canonicalizeByPhone
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

// TestSliceThreeRules checks the legacy connector boundary (Architecture V1
// §24): the connector is imported only by itself and the offline import
// tool (crm-migrate only its migrations), and WooCommerce/legacy names never
// appear in target Go strings or SQL/template files outside it.
func TestSliceThreeRules(t *testing.T) {
	vs, err := Check(filepath.Join("testdata", "violating"), "example.test/mod")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"internal/sales/legacy_import.go":  RuleLegacyImport,
		"cmd/crm-api/legacy.go":            RuleLegacyImport,
		"cmd/crm-migrate/legacy.go":        RuleLegacyImport,
		"internal/billing/woo.go":          RuleLegacyLiteral,
		"internal/sales/schema.sql":        RuleLegacyLiteral,
		"cmd/crm-legacy-import/fixture.go": RulePgtestImport,
	}
	for file, rule := range want {
		found := false
		for _, v := range vs {
			if v.File == file && v.Rule == rule {
				found = true
			}
		}
		if !found {
			t.Errorf("rule %s did not fire in %s", rule, file)
		}
	}
	allowed, err := Check(filepath.Join("testdata", "allowed"), "example.test/mod")
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range allowed {
		t.Errorf("allowed fixture flagged: %s", v)
	}
	for _, s := range []string{"WooCommerce", "woocommerce_order_items"} {
		if !wooLiteral.MatchString(s) {
			t.Errorf("%q not detected", s)
		}
	}
	for _, s := range []string{"commerce", "woo", "workflow"} {
		if wooLiteral.MatchString(s) {
			t.Errorf("%q falsely detected", s)
		}
	}
}

// TestAPIServerHasNoLegacyDependency proves, transitively, that the target
// API server binary cannot reach the legacy connector.
func TestAPIServerHasNoLegacyDependency(t *testing.T) {
	mod := modulePath(t)
	cmd := exec.Command("go", "list", "-deps", "./cmd/crm-api")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	for _, p := range strings.Fields(string(out)) {
		if hasPrefixPath(p, mod+"/internal/integration") {
			t.Errorf("crm-api depends on %s", p)
		}
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

// TestSliceFourRules checks the Slice 4 module boundaries (Architecture V1
// §5, §6, §21, §24): published-api-only access between identity,
// organization, authorization and the connector; no foreign schema writes;
// no import command on a read path.
func TestSliceFourRules(t *testing.T) {
	vs, err := Check(filepath.Join("testdata", "violating"), "example.test/mod")
	if err != nil {
		t.Fatal(err)
	}
	want := []struct{ file, rule string }{
		{"internal/identity/org.go", RuleModuleBoundary},
		{"internal/organization/internals.go", RuleModuleBoundary},
		{"internal/authorization/engine.go", RuleModuleBoundary},
		{"internal/integration/legacy/shadowbad/write.go", RuleModuleBoundary},
		{"internal/integration/legacy/shadowbad/write.go", RuleForeignWrite},
		{"internal/organization/http/handler.go", RuleReadPathImport},
		{"cmd/crm-api/imports.go", RuleReadPathImport},
	}
	for _, w := range want {
		found := false
		for _, v := range vs {
			if v.File == w.file && v.Rule == w.rule {
				found = true
			}
		}
		if !found {
			t.Errorf("rule %s did not fire in %s", w.rule, w.file)
		}
	}
	allowed, err := Check(filepath.Join("testdata", "allowed"), "example.test/mod")
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range allowed {
		t.Errorf("allowed fixture flagged: %s", v)
	}
	for _, s := range []string{"INSERT INTO identity.user_account", "update  organization.person set", "DELETE FROM audit.audit_event"} {
		if !foreignWrite.MatchString(s) {
			t.Errorf("%q not detected", s)
		}
	}
	for _, s := range []string{"SELECT * FROM identity.user_account", "updated organization.person", "the identity. update"} {
		if foreignWrite.MatchString(s) {
			t.Errorf("%q falsely detected", s)
		}
	}
}

func TestSliceFiveRules(t *testing.T) {
	vs, err := Check(filepath.Join("testdata", "violating"), "example.test/mod")
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		file, rule string
		n          int
	}{
		{"internal/customers/merge.go", RuleModuleBoundary, 1},
		{"internal/customers/merge.go", RuleMergePath, 2},
		{"internal/customers/http/handler.go", RuleReadPathImport, 1},
		{"internal/customers/http/handler.go", RuleModuleBoundary, 1},
		{"internal/integration/legacy/custbad/write.go", RuleForeignWrite, 1},
		{"internal/integration/legacy/custbad/write.go", RuleModuleBoundary, 1},
	}
	for _, w := range want {
		n := 0
		for _, v := range vs {
			if v.File == w.file && v.Rule == w.rule {
				n++
			}
		}
		if n != w.n {
			t.Errorf("rule %s fired %d times in %s, want %d", w.rule, n, w.file, w.n)
		}
	}
	for _, name := range []string{"ResolveMerge", "MergeCustomers", "ApplyMerge", "CanonicalizeByPhone", "canonicaliseByPhone", "PickWinner", "SelectCanonicalCustomer"} {
		if !mergePath.MatchString(name) {
			t.Errorf("%s not detected", name)
		}
	}
	for _, name := range []string{"MergeCase", "MergeCaseState", "ListMergeCases", "mergeCaseID", "GetMergeCase"} {
		if mergePath.MatchString(name) {
			t.Errorf("%s falsely detected", name)
		}
	}
}

// TestAPIServerHasNoImportDependency proves, transitively, that the API
// server binary reaches no import command and no migration source: every
// GET path is read-only by construction.
func TestAPIServerHasNoImportDependency(t *testing.T) {
	mod := modulePath(t)
	cmd := exec.Command("go", "list", "-deps", "./cmd/crm-api")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	for _, p := range strings.Fields(string(out)) {
		if !strings.HasPrefix(p, mod+"/internal/") {
			continue
		}
		if strings.HasSuffix(p, "/imports") || strings.HasSuffix(p, "/migrations") {
			t.Errorf("crm-api depends on %s", p)
		}
	}
}
