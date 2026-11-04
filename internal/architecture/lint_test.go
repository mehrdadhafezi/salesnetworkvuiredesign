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
		RuleKernelImport:       4, // business module, net/http, os, math/rand in production code
		RuleKernelClock:        2, // time.Now and .Local()
		RuleKernelFloat:        2, // float64 parameter and float32 conversion
		RuleMoneyConversion:    1,
		RuleLegacyLiteral:      4, // kernel, business package, WooCommerce literal, .sql file
		RuleLegacyRefFactory:   1,
		RuleDotImport:          1,  // internal/sales/dotimport.go
		RuleRuntimeDDL:         3,  // platform/bad CREATE TABLE; sales/ddl alter table + GRANT
		RulePlatformImport:     1,  // platform/bad imports billing
		RulePgtestImport:       2,  // pgtest from platform/bad; legacytest from the import tool
		RuleForeignStore:       3,  // sales imports billing/store; authorization imports organization/store; connector imports customers/store
		RuleLegacyImport:       4,  // + cases/http imports the legacy write-back connector; sales, crm-api and crm-migrate (non-migrations) import the connector
		RuleModuleBoundary:     23, // + catalog→biawin/api, fulfillment/work→biawin/activation, fulfillment/work→catalog/registry; + cases/command→billing/api; + biawin→billing/query, fulfillment→biawin/query; + finance→billing/query, commission→finance/api; + reporting→cases/query, reporting/http→payments/api, billing→reporting/query; + payments→billing/query, billing/query→payments/api; custody→cases/query, custody→organization/api, cases/http→custody/query; identity→organization/api, organization→identity/query, authorization→organization/store, connector→organization/imports; customers→organization/query, customers/http→identity/query, connector→customers/store
		RuleForeignWrite:       9,  // + catalog UPDATE fulfillment.work_case; + biawin UPDATE finance.*, fulfillment INSERT INTO billing.*; connector INSERT INTO organization.*, UPDATE customers.*, INSERT INTO cases.*; payments INSERT INTO billing.*; finance UPDATE billing.*; commission INSERT INTO finance.*
		RuleReadPathImport:     9,  // + commission/http imports the offline commission shadow job; + biawin/http and fulfillment/query import an import command; organization/http, customers/http, cases/http and crm-api import an import command; reporting/http imports the projection rebuild; finance/http imports the shadow rebuild
		RuleMergePath:          2,  // customers MergeCustomers, canonicalizeByPhone
		RuleAppendOnlyWrite:    10, // + catalog UPDATE fulfillment.work_case, fulfillment/work DELETE FROM work_case + UPDATE work_event; + biawin UPDATE finance.ledger_entry, DELETE FROM biawin.subscription, fulfillment/query UPDATE fulfillment.*; custody UPDATE custody.custody_event; payments UPDATE payments.attempt_state_history; reporting UPDATE reporting.case_fact; finance DELETE FROM finance.ledger_entry
		RulePhoneKey:           10, // + cases/command byPhone; + biawin phone join literal, phoneJoin and byPhone identifiers; custody, payments and reporting customer_phone joins; finance phone key literal + byPhone and mobileKey identifiers
		RuleStatusFallback:     2,  // billing COALESCE status chain and resolveMainStatus
		RuleUnitConversion:     5,  // payments toRial and toman * 10; finance ratioOf, toRial and v * 10
		RuleReportingRead:      1,  // reporting SELECT FROM billing.invoice
		RuleMetricFormula:      1,  // reporting count( outside the query spec
		RuleExportEndpoint:     1,  // reporting /invoice-review/export
		RuleFinanceRead:        2,  // finance SELECT FROM payments.payment_attempt; finance/review SELECT FROM payments.submitted_attempt
		RuleFinanceReadWrite:   1,  // finance/http INSERT INTO finance.shadow_run
		RuleFinanceRoute:       1,  // finance/http /ledger/{entry_id}/reverse
		RuleFinanceFloat:       2,  // finance float64 parameter and result
		RuleCrossUnitSum:       1,  // finance sum( without unit
		RuleBusinessKey:        1,  // finance/domain Entry literal without BusinessKey
		RuleBiawinSQL:          3,  // biawin UPDATE finance.* and FROM billing.*; fulfillment INTO billing.*
		RuleBiawinReadWrite:    2,  // biawin/http INSERT; fulfillment/query UPDATE
		RuleProviderCall:       4,  // biawin imports the provider, charge_wallet literal, ChargeWallet identifier; crm-api imports the provider
		RuleS11FinancialImport: 1,  // cases/command → billing/api
		RuleS11Forbidden:       5,  // IssueInvoice, ReturnCase, MarkReady, ApplyHRRequest + /cases/{case_id}/return
		RuleS11HRIdentity:      1,  // GrantRole
		RuleS11RankShortcut:    2,  // IsHigherRank, "manage_options"
		RuleS11ReadPathSQL:     1,  // SQL in cases/http
		RuleS11Outbox:          1,  // command write without outbox append
		RuleS11DomainConnector: 2,  // cases/command → integration/writeback; cases/http → legacy/writeback
		RuleS11Bulk:            1,  // BulkReassign
		RuleS11BlindRetry:      1,  // Apply outside apply()
		RuleForbiddenCommand:   11, // ChargeWallet, ActivateSubscription, CompleteDot, WriteProjectMembership, RunCommission, AssignExpert, ShipOrder, UploadEvidence + two command routes
		RuleS12Forbidden:       3,  // ApproveAttempt, IssueNextStage, an approve route literal
		RuleS12GatewayWiring:   1,  // cmd/crm-api-s12 imports a gateway adapter
		RuleS12Network:         2,  // http.Get in payments; net.Dial in a gateway adapter
		RuleS12EvidencePath:    1,  // payments imports os
		RuleS12Session:         2,  // AutoLogin in payments; NewSession in a gateway adapter
		RuleS12TargetReview:    1,  // payments SQL sets a target attempt ACCEPTED
		RuleS12ReviewPure:      2,  // review imports platform/db; SQL in the review gate
		RuleS13Forbidden:       3,  // ApproveRun, PostCommission, a commission approve route literal
		RuleS13Dispatch:        1,  // commission outbox.Append
		RuleS13ForeignSQL:      2,  // commission INSERT INTO finance.*; SELECT FROM organization.*
		RuleS13Float:           1,  // commission float64
		RuleS13DomainPure:      2,  // domain imports platform/db; domain reads time.Now
		RuleS13ReadWrite:       1,  // commission/http INSERT INTO commission.*
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
// API server binary cannot reach the legacy connector or the Biawin
// provider. Slice 11 lets it read the integration-owned write-back
// diagnostics (internal/integration/writeback, read service + http) only.
func TestAPIServerHasNoLegacyDependency(t *testing.T) {
	mod := modulePath(t)
	cmd := exec.Command("go", "list", "-deps", "./cmd/crm-api")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	for _, p := range strings.Fields(string(out)) {
		// Slice 14: the provider-operation ledger read + human reconciliation
		// (no port, no provider, no legacy connector; checked transitively).
		allowed := p == mod+"/internal/integration" || p == mod+"/internal/integration/writeback" || p == mod+"/internal/integration/writeback/http" ||
			p == mod+"/internal/integration/providerops" || p == mod+"/internal/integration/providerops/http"
		if hasPrefixPath(p, mod+"/internal/integration") && !allowed {
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

func TestSliceSixRules(t *testing.T) {
	vs, err := Check(filepath.Join("testdata", "violating"), "example.test/mod")
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		file, rule string
		n          int
	}{
		{"internal/custody/bad.go", RuleModuleBoundary, 2},
		{"internal/custody/bad.go", RuleAppendOnlyWrite, 1},
		{"internal/custody/bad.go", RulePhoneKey, 1},
		{"internal/cases/http/handler.go", RuleModuleBoundary, 1},
		{"internal/cases/http/handler.go", RuleReadPathImport, 1},
		{"internal/integration/legacy/salesbad/write.go", RuleForeignWrite, 1},
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
	for _, s := range []string{"UPDATE custody.custody_event SET x = 1", "delete from cases.attribution_snapshot", "update  cases.case_state_history set"} {
		if !appendOnlyWrite.MatchString(s) {
			t.Errorf("%q not detected", s)
		}
	}
	for _, s := range []string{"INSERT INTO custody.custody_event", "SELECT * FROM cases.attribution_snapshot", "UPDATE custody.assignment SET parity = $1"} {
		if appendOnlyWrite.MatchString(s) {
			t.Errorf("%q falsely detected", s)
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
		// No import command, migration, projection rebuild or finance shadow
		// rebuild is reachable from the API server (a GET never derives).
		if strings.HasSuffix(p, "/imports") || strings.HasSuffix(p, "/migrations") || strings.HasSuffix(p, "/projections") ||
			strings.HasSuffix(p, "/finance/shadow") || strings.Contains(p, "/integration/legacy/") || strings.HasSuffix(p, "/integration/biawin") {
			t.Errorf("crm-api depends on %s", p)
		}
	}
}

// TestNoProviderInComposedSystem proves no binary reaches the Biawin
// provider package: Slice 10 makes no external call of any kind.
func TestNoProviderInComposedSystem(t *testing.T) {
	mod := modulePath(t)
	cmd := exec.Command("go", "list", "-deps", "./cmd/...")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	for _, p := range strings.Fields(string(out)) {
		if p == mod+"/internal/integration/biawin" {
			t.Errorf("a command depends on the Biawin provider %s", p)
		}
	}
}

func TestSliceSevenRules(t *testing.T) {
	vs, err := Check(filepath.Join("testdata", "violating"), "example.test/mod")
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		file, rule string
		n          int
	}{
		{"internal/payments/bad.go", RuleModuleBoundary, 1},
		{"internal/payments/bad.go", RuleAppendOnlyWrite, 1},
		{"internal/payments/bad.go", RuleForeignWrite, 1},
		{"internal/payments/bad.go", RulePhoneKey, 1},
		{"internal/payments/bad.go", RuleUnitConversion, 2},
		{"internal/billing/query/status.go", RuleStatusFallback, 2},
		{"internal/billing/query/status.go", RuleModuleBoundary, 1},
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
	allowed, err := Check(filepath.Join("testdata", "allowed"), "example.test/mod")
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range allowed {
		t.Errorf("allowed fixture flagged: %s", v)
	}
	for _, name := range []string{"resolveMainStatus", "CanonicalInvoiceStatus", "effectiveStatus", "deriveInvoiceStatus", "mainStatusFallback"} {
		if !statusResolver.MatchString(name) {
			t.Errorf("%s not detected", name)
		}
	}
	for _, name := range []string{"stageState", "StateRaw", "legacyDisplayStatus", "StatusPill", "InvoiceStateField", "statusLabel"} {
		if statusResolver.MatchString(name) {
			t.Errorf("%s falsely detected", name)
		}
	}
}

func TestSliceEightRules(t *testing.T) {
	vs, err := Check(filepath.Join("testdata", "violating"), "example.test/mod")
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		file, rule string
		n          int
	}{
		{"internal/reporting/bad.go", RuleModuleBoundary, 1},
		{"internal/reporting/bad.go", RuleReportingRead, 1},
		{"internal/reporting/bad.go", RuleAppendOnlyWrite, 1},
		{"internal/reporting/bad.go", RuleMetricFormula, 1},
		{"internal/reporting/bad.go", RuleExportEndpoint, 1},
		{"internal/reporting/bad.go", RulePhoneKey, 1},
		{"internal/reporting/http/handler.go", RuleReadPathImport, 1},
		{"internal/reporting/http/handler.go", RuleModuleBoundary, 1},
		{"internal/billing/uses_reporting.go", RuleModuleBoundary, 1},
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
	allowed, err := Check(filepath.Join("testdata", "allowed"), "example.test/mod")
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range allowed {
		t.Errorf("allowed fixture flagged: %s", v)
	}
	for _, s := range []string{"/api/v1/invoice-review/export", "report.csv", "x.xlsx", "text/csv", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"} {
		if !exportLiteral.MatchString(s) {
			t.Errorf("%q not detected", s)
		}
	}
	for _, s := range []string{"/api/v1/invoice-review", "exported_at", "/api/v1/reports/invoices"} {
		if exportLiteral.MatchString(s) {
			t.Errorf("%q falsely detected", s)
		}
	}
}

func TestSliceNineRules(t *testing.T) {
	vs, err := Check(filepath.Join("testdata", "violating"), "example.test/mod")
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		file, rule string
		n          int
	}{
		{"internal/finance/bad.go", RuleModuleBoundary, 1},
		{"internal/finance/bad.go", RuleFinanceRead, 1},
		{"internal/finance/bad.go", RuleForeignWrite, 1},
		{"internal/finance/bad.go", RuleAppendOnlyWrite, 1},
		{"internal/finance/bad.go", RuleCrossUnitSum, 1},
		{"internal/finance/bad.go", RulePhoneKey, 3},
		{"internal/finance/bad.go", RuleFinanceFloat, 2},
		{"internal/finance/bad.go", RuleUnitConversion, 3},
		{"internal/finance/http/handler.go", RuleReadPathImport, 1},
		{"internal/finance/http/handler.go", RuleFinanceReadWrite, 1},
		{"internal/finance/http/handler.go", RuleFinanceRoute, 1},
		{"internal/finance/domain/entry.go", RuleBusinessKey, 1},
		{"internal/commission/bad.go", RuleModuleBoundary, 1},
		{"internal/commission/bad.go", RuleForeignWrite, 1},
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
	for _, s := range []string{"/api/v1/finance/post", "/api/v1/finance/approve", "/api/v1/finance/ledger/{entry_id}/reverse", "/api/v1/finance/settle",
		"/api/v1/finance/refund", "/api/v1/finance/credit", "/api/v1/finance/debit"} {
		if !financeWriteRoute.MatchString(s) {
			t.Errorf("%q not detected", s)
		}
	}
	for _, s := range []string{"/api/v1/finance/ledger", "/api/v1/finance/balances", "/api/v1/finance/reconciliation", "/api/v1/finance/shadow-runs/{run_id}"} {
		if financeWriteRoute.MatchString(s) {
			t.Errorf("%q wrongly flagged", s)
		}
	}
}

// TestSliceTenRules proves each Slice 10 rule fires where intended and the
// read routes of the contract stay clean.
func TestSliceTenRules(t *testing.T) {
	vs, err := Check(filepath.Join("testdata", "violating"), "example.test/mod")
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		file, rule string
		n          int
	}{
		{"internal/biawin/bad.go", RuleModuleBoundary, 1},
		{"internal/biawin/bad.go", RuleProviderCall, 3},
		{"internal/biawin/bad.go", RuleForeignWrite, 1},
		{"internal/biawin/bad.go", RuleAppendOnlyWrite, 2},
		{"internal/biawin/bad.go", RuleBiawinSQL, 2},
		{"internal/biawin/bad.go", RulePhoneKey, 3},
		{"internal/biawin/bad.go", RuleForbiddenCommand, 5},
		{"internal/biawin/http/handler.go", RuleReadPathImport, 1},
		{"internal/biawin/http/handler.go", RuleBiawinReadWrite, 1},
		{"internal/biawin/http/handler.go", RuleForbiddenCommand, 1},
		{"internal/fulfillment/query/read.go", RuleReadPathImport, 1},
		{"internal/fulfillment/query/read.go", RuleBiawinReadWrite, 1},
		{"internal/fulfillment/query/read.go", RuleAppendOnlyWrite, 1},
		{"internal/fulfillment/query/read.go", RuleForbiddenCommand, 4},
		{"internal/fulfillment/bad.go", RuleModuleBoundary, 1},
		{"internal/fulfillment/bad.go", RuleForeignWrite, 1},
		{"internal/fulfillment/bad.go", RuleBiawinSQL, 1},
		{"cmd/crm-api/biawin.go", RuleProviderCall, 1},
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
	for _, s := range []string{"/api/v1/biawin/subscriptions/{subscription_id}/suspend", "/api/v1/biawin/orders", "/api/v1/biawin/wallet/charge",
		"/api/v1/fulfillment/cases/{case_id}/assign", "/api/v1/fulfillment/cases/{case_id}/ship", "/api/v1/fulfillment/cases/{case_id}/evidence",
		"/api/v1/fulfillment/upgrade", "/api/v1/biawin/sync", "/api/v1/biawin/import"} {
		if !forbiddenRoute.MatchString(s) {
			t.Errorf("%q not detected", s)
		}
	}
	for _, s := range []string{"/api/v1/biawin/subscriptions", "/api/v1/biawin/subscriptions/{subscription_id}", "/api/v1/biawin/subscriptions/{subscription_id}/items",
		"/api/v1/biawin/parity", "/api/v1/fulfillment/cases", "/api/v1/fulfillment/cases/{case_id}", "/api/v1/fulfillment/cases/{case_id}/history"} {
		if forbiddenRoute.MatchString(s) {
			t.Errorf("%q wrongly flagged", s)
		}
	}
	for _, id := range []string{"CreateOrder", "createOrder", "chargeWallet"} {
		if !providerWrite.MatchString(id) {
			t.Errorf("%q not detected", id)
		}
	}
	for _, id := range []string{"cancel", "assignments", "Activated", "ctx", "History"} {
		if forbiddenCommand.MatchString(id) {
			t.Errorf("%q wrongly flagged", id)
		}
	}
}

// TestSliceElevenRules proves each Slice 11 rule fires where intended and the
// evidenced command routes stay clean.
func TestSliceElevenRules(t *testing.T) {
	vs, err := Check(filepath.Join("testdata", "violating"), "example.test/mod")
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		file, rule string
		n          int
	}{
		{"internal/cases/command/bad.go", RuleS11FinancialImport, 1},
		{"internal/cases/command/bad.go", RuleS11DomainConnector, 1},
		{"internal/cases/command/bad.go", RuleS11Outbox, 1},
		{"internal/cases/command/bad.go", RuleS11Forbidden, 3},
		{"internal/cases/command/bad.go", RuleS11RankShortcut, 1},
		{"internal/cases/command/bad.go", RuleS11Bulk, 1},
		{"internal/cases/command/bad.go", RulePhoneKey, 1},
		{"internal/organization/hrrequest/bad.go", RuleS11HRIdentity, 1},
		{"internal/organization/hrrequest/bad.go", RuleS11RankShortcut, 1},
		{"internal/organization/hrrequest/bad.go", RuleS11Forbidden, 1},
		{"internal/cases/http/command.go", RuleS11ReadPathSQL, 1},
		{"internal/cases/http/command.go", RuleS11Forbidden, 1},
		{"internal/cases/http/command.go", RuleS11DomainConnector, 1},
		{"internal/integration/writeback/retry.go", RuleS11BlindRetry, 1},
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
	for _, s := range []string{"/api/v1/cases/{case_id}/return", "/api/v1/cases/{case_id}/ready", "/api/v1/invoices/{invoice_id}/issue",
		"/api/v1/custody/transfers", "/api/v1/custody/allocations", "/api/v1/hr/requests/{request_id}/apply", "/api/v1/cases/{case_id}/transitions",
		"/api/v1/cases/bulk"} {
		if !s11Route.MatchString(s) {
			t.Errorf("%q not detected", s)
		}
	}
	for _, s := range []string{"/api/v1/cases/{case_id}/contact-outcomes", "/api/v1/hr/requests", "/api/v1/hr/requests/{request_id}/decisions",
		"/api/v1/cases/{case_id}/history", "/api/v1/custody/cases/{case_id}"} {
		if s11Route.MatchString(s) {
			t.Errorf("%q wrongly flagged", s)
		}
	}
}

func TestSliceTwelveRoutes(t *testing.T) {
	for _, s := range []string{"/api/v1/finance/review/attempts/{attempt_id}/decisions", "/api/v1/payments/attempts/{attempt_id}/approve",
		"/api/v1/finance/ledger/postings", "/api/v1/payments/refunds", "/api/v1/billing/invoices/{invoice_id}/next-stage",
		"/api/v1/payments/gateway/callback", "/api/v1/finance/review/bulk-approve", "/api/v1/payments/settlements"} {
		if !s12Route.MatchString(s) {
			t.Errorf("%q not detected", s)
		}
	}
	for _, s := range []string{"/api/v1/payments/uploads", "/api/v1/payments/invoices/{invoice_id}/submission", "/api/v1/payments/invoices/{invoice_id}/batches",
		"/api/v1/payments/batches/{batch_id}", "/api/v1/payments/evidence", "/api/v1/finance/review/attempts", "/api/v1/finance/review/attempts/{attempt_id}"} {
		if s12Route.MatchString(s) {
			t.Errorf("%q wrongly flagged", s)
		}
	}
}

// TestSliceFourteenRules proves the Slice 14 boundaries: the catalog
// depends on nothing, work intake reads only published api packages, the
// work case is the single mutable table (owner package only, never deleted)
// and only the exact work-case command routes escape forbiddenRoute.
func TestSliceFourteenRules(t *testing.T) {
	vs, err := Check(filepath.Join("testdata", "violating"), "example.test/mod")
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		file, rule string
		n          int
	}{
		{"internal/catalog/bad.go", RuleModuleBoundary, 1},
		{"internal/catalog/bad.go", RuleForeignWrite, 1},
		{"internal/catalog/bad.go", RuleAppendOnlyWrite, 1},
		{"internal/fulfillment/work/s14.go", RuleModuleBoundary, 2},
		{"internal/fulfillment/work/s14.go", RuleAppendOnlyWrite, 2},
		{"internal/fulfillment/work/s14.go", RuleForbiddenCommand, 1},
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
	for _, s := range []string{"/api/v1/fulfillment/work-cases/{case_id}/complete", "/api/v1/fulfillment/work-cases/{case_id}/assign",
		"/api/v1/fulfillment/work-cases/{case_id}/cancel", "/api/v1/biawin/subscriptions/{subscription_id}/activate", "/api/v1/biawin/wallet/charge",
		"/api/v1/fulfillment/work-cases/{case_id}/evidence/upload"} {
		if !forbiddenRoute.MatchString(s) || s14Routes[s] {
			t.Errorf("%q not detected", s)
		}
	}
	if !s14MutableWrite("internal/fulfillment/work/work.go", "UPDATE fulfillment.work_case") ||
		s14MutableWrite("internal/fulfillment/query/read.go", "UPDATE fulfillment.work_case") ||
		s14MutableWrite("internal/fulfillment/work/work.go", "DELETE FROM fulfillment.work_case") ||
		s14MutableWrite("internal/fulfillment/work/work.go", "UPDATE fulfillment.work_event") {
		t.Error("s14MutableWrite admits more than the work case update from its owner")
	}
}
