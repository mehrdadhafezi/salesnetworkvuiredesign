// Package architecture enforces the Slice 0 import and literal rules (E0 §9F,
// §12 criteria 3, 7 and 9) with a small go/ast checker:
//
//   - internal/kernel/** imports only the standard library and other kernel
//     packages; no database, network, OS, file or process packages.
//   - kernel code never reads the system clock or the machine-local zone.
//   - kernel code contains no float32/float64.
//   - the money package exports no conversion function.
//   - legacy table literals appear only under internal/integration/legacy.
//   - refs.NewLegacyRef is called only from internal/integration/legacy.
//   - no dot imports anywhere in target code: a dot import would hide a
//     package-qualified call (e.g. a bare NewLegacyRef) from the rules above.
//
// Slice 1 rules (Architecture V1 §5, §6, §8.11, §21):
//
//   - runtime-ddl: schema-mutating SQL literals (CREATE/ALTER/DROP of
//     tables, schemas, roles, ...; TRUNCATE; GRANT/REVOKE) appear only in the
//     migration tooling and the test-only pgtest harness. Test files are exempt.
//   - platform-import: internal/platform/** never imports a business module.
//   - foreign-store: internal/<a>/store is imported only from internal/<a>/**.
//   - pgtest-import: internal/platform/pgtest (and, from Slice 3, the
//     synthetic-snapshot builder internal/integration/legacy/legacytest) is
//     imported only from _test.go files.
//
// Slice 3 rules (Architecture V1 §24):
//
//   - legacy-import: internal/integration/legacy/** is imported only from
//     itself and the offline cmd/crm-legacy-import tool; cmd/crm-migrate may
//     import exactly its migrations package (the only DDL path). The API
//     server, platform and business modules never reach the legacy connector.
//   - legacy-literal also covers "woocommerce" in Go string literals and
//     wp_/sn_/woocommerce text in .sql and .tmpl files outside the legacy tree.
//
// Slice 4 rules (Architecture V1 §5, §6, §18, §21, §24):
//
//   - module-boundary: identity, organization and authorization reach each
//     other only through published api packages: identity never imports
//     organization; organization imports identity/api only; authorization
//     imports identity/api and organization/api only; a module's http
//     adapter may also use the authorization evaluator (authorization and
//     authorization/http). The integration connector reaches target modules
//     only through their api packages and the authorization evaluator, so
//     it can never call a module's store or write its tables directly.
//   - foreign-schema-write: an INSERT/UPDATE/DELETE SQL literal naming a
//     module schema appears only inside that module's own tree.
//   - read-path-import: read handlers (internal/<m>/http), the API server
//     and cmd/crm-api never import an import command (internal/<m>/imports).
//
// Slice 5 rules (customer identity shadow; Gate-0 phone invariants):
//
//   - module-boundary also covers customers: it imports no other business
//     module except identity/api (the capability vocabulary, from its http
//     adapter) and the authorization evaluator (from its http adapter); no
//     other module imports customers except through customers/api.
//   - foreign-schema-write covers the customers schema (customers only).
//   - merge-path: no identifier anywhere (tests included) names a merge,
//     winner-selection or phone-canonicalisation operation
//     (ResolveMerge, MergeCustomers, ApplyMerge, CanonicalizeByPhone, ...).
//     Slice 5 is read/shadow only and D-30 (merge policy) is open.
//
// Slice 6 rules (sales read shadow; FPR append-only, INV-004, INV-029):
//
//   - module-boundary also covers cases and custody: custody reaches cases
//     only through cases/api; cases never imports custody; neither imports
//     organization at all (historical attribution never consults the
//     current hierarchy); their http adapters may use identity/api (the
//     capability vocabulary) and the authorization evaluator.
//   - foreign-schema-write covers the cases and custody schemas.
//   - append-only-write: no SQL literal anywhere UPDATEs or DELETEs a
//     custody event, a case state history row or an attribution snapshot.
//   - phone-key: no string literal in internal/cases or internal/custody
//     mentions a phone (phone is never a Case or customer key or join).
//
// Slice 7 rules (billing + payments read shadow; TAD #19/#20, D-25 open):
//
//   - module-boundary also covers billing and payments: payments reaches
//     billing only through billing/api (invoice resolution); billing reaches
//     payments only through payments/api and only from its http adapter
//     (payment sums and coverage for the read surface); billing may read
//     customers/api and cases/api; their http adapters may use identity/api
//     and the authorization evaluator; no other module imports them.
//   - foreign-schema-write covers the billing and payments schemas, so
//     payments never writes billing and billing never writes payments.
//   - append-only-write also covers payments.attempt_state_history.
//   - phone-key also covers internal/billing and internal/payments.
//   - status-fallback: no SQL literal outside the legacy connector chains
//     COALESCE(NULLIF(...status...)) and no identifier names a generic
//     invoice main-status resolver (resolveMainStatus, canonicalInvoiceStatus,
//     effectiveStatus, ...): the canonical invoice status is deferred (TAD
//     #20) and the legacy display expression lives only in the connector.
//   - money-unit-conversion: internal/billing and internal/payments never
//     multiply or divide by the literal 10 (the legacy Toman↔Rial factor)
//     and declare no function whose name suggests a unit conversion.
//
// Slice 8 rules (MIS / reporting read; INV-021, D-23, D-24, D-45):
//
//   - module-boundary also covers reporting: it reaches owners only through
//     their published api packages and only from its offline projection
//     rebuild (internal/reporting/projections); its http adapter may use
//     identity/api and the authorization evaluator; no module imports
//     reporting.
//   - read-path-import also forbids reporting/http importing the projection
//     rebuild (a GET never rebuilds, backfills or refreshes).
//   - foreign-schema-write covers the reporting schema; append-only-write
//     covers every reporting table; phone-key covers internal/reporting.
//   - reporting-foreign-read: no SQL literal in internal/reporting reads an
//     owner schema (reporting reads only reporting.*).
//   - metric-formula: aggregate SQL (count/sum/avg/min/max) in
//     internal/reporting appears only in the shared query spec package
//     (internal/reporting/query) and the metric contract registry
//     (internal/reporting/domain, formula text) — nowhere else.
//   - phone-key covers SQL literals in internal/reporting.
//   - export-endpoint: no string literal in internal/reporting or the API
//     server names an export, CSV, XLSX or download route or media type (no
//     Invoice Review export, D-23).
//
// Slice 9 rules (finance SHADOW ledger; D-41, OPD-06, OPD-09, F06, F07):
//
//   - module-boundary also covers finance (and a future commission module):
//     finance reaches billing and payments only through their published api
//     packages, its http adapter may use identity/api and the evaluator, and
//     only integration (the normalized wallet source port) and nothing else
//     imports finance/api; commission never imports finance (posting
//     requests will be events, Slice 13).
//   - foreign-schema-write covers the finance schema (only internal/finance
//     writes it; commission or reporting never), so finance never writes
//     billing or payments; append-only-write covers every finance table
//     (no UPDATE / DELETE of a ledger fact anywhere).
//   - finance-foreign-read: no SQL literal in internal/finance reads another
//     module's schema (finance reads only finance.*).
//   - finance-read-write: the finance read path (finance/http, finance/query)
//     holds no INSERT / UPDATE / DELETE literal; read-path-import forbids it
//     (and the API server) importing the offline shadow rebuild: a GET never
//     derives, posts or records anything (F07).
//   - finance-write-route: no string literal in the API server or finance
//     names a finance posting / approval / reversal / settlement / refund /
//     credit / debit route (GET-only read surface, F06).
//   - finance-float: no float32 / float64 anywhere in internal/finance.
//   - money-unit-conversion also covers internal/finance (no x10 / /10, no
//     conversion-like function: OPD-06).
//   - finance-cross-unit-sum: every SQL sum in internal/finance is scoped by
//     unit (units are never summed together).
//   - finance-business-key: every ledger Entry literal in the finance domain
//     and the shadow rebuild sets BusinessKey (also NOT NULL + UNIQUE in the
//     database).
//   - phone-key covers every literal and identifier in internal/finance:
//     phone / mobile are never an account key, filter or join.
//   - legacy-import also allows cmd/crm-finance-shadow (the offline job that
//     composes the staged wallet source); finance itself never imports it.
//
// Slice 10 rules (Biawin / fulfillment READ shadow; D-35 and D-38 open):
//
//   - module-boundary covers biawin and fulfillment: they reach identity
//     only through identity/api from their http packages, never another
//     module's internals; biawin and fulfillment never import each other.
//   - foreign-schema-write covers the biawin and fulfillment schemas (only
//     their own trees write them), so biawin never mutates finance, billing
//     or payments and fulfillment never mutates billing or a wallet;
//     append-only-write covers every biawin and fulfillment table.
//   - biawin-foreign-sql: no SQL literal in internal/biawin or
//     internal/fulfillment names another schema (read or write).
//   - biawin-read-write: the read path (biawin|fulfillment /http, /query)
//     holds no write SQL (no GET writes); read-path-import also forbids the
//     import commands there.
//   - biawin-provider-call: no create_order / charge_wallet (or any
//     CreateOrder / ChargeWallet identifier) outside internal/integration/
//     biawin, and nothing but its own tests imports internal/integration/
//     biawin (no provider is wired into the composed system).
//   - biawin-forbidden-command: no activation / suspension / termination,
//     direct Dot completion, project-membership write, shipping, assignment,
//     service completion, evidence upload, upgrade execution or commission
//     execution identifier or route literal in the Slice 10 trees.
//   - phone-key covers every literal and identifier in internal/biawin and
//     internal/fulfillment: phone is never a subscription / fulfillment
//     identity, join, dedupe or provider key.
//   - legacy-import also allows cmd/crm-biawin-shadow (the offline job).
//
// Test files are composition roots for module-boundary (as cmd/ is); every
// other rule applies to them as before.
package architecture

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Violation is one broken rule at a file position.
type Violation struct {
	Rule string
	File string // slash-separated, relative to the checked root
	Line int
	Text string
}

func (v Violation) String() string {
	return fmt.Sprintf("%s:%d: [%s] %s", v.File, v.Line, v.Rule, v.Text)
}

// Rule names.
const (
	RuleKernelImport     = "kernel-import"
	RuleKernelClock      = "kernel-clock"
	RuleKernelFloat      = "kernel-float"
	RuleMoneyConversion  = "money-conversion"
	RuleLegacyLiteral    = "legacy-literal"
	RuleLegacyRefFactory = "legacyref-factory"
	RuleDotImport        = "dot-import"
	RuleRuntimeDDL       = "runtime-ddl"
	RulePlatformImport   = "platform-import"
	RuleForeignStore     = "foreign-store"
	RulePgtestImport     = "pgtest-import"
	RuleLegacyImport     = "legacy-import"
	RuleModuleBoundary   = "module-boundary"
	RuleForeignWrite     = "foreign-schema-write"
	RuleReadPathImport   = "read-path-import"
	RuleMergePath        = "merge-path"
	RuleAppendOnlyWrite  = "append-only-write"
	RulePhoneKey         = "phone-key"
	RuleStatusFallback   = "status-fallback"
	RuleUnitConversion   = "money-unit-conversion"
	RuleReportingRead    = "reporting-foreign-read"
	RuleMetricFormula    = "metric-formula"
	RuleExportEndpoint   = "export-endpoint"
	RuleFinanceRead      = "finance-foreign-read"
	RuleFinanceReadWrite = "finance-read-write"
	RuleFinanceRoute     = "finance-write-route"
	RuleFinanceFloat     = "finance-float"
	RuleCrossUnitSum     = "finance-cross-unit-sum"
	RuleBusinessKey      = "finance-business-key"
	RuleBiawinSQL        = "biawin-foreign-sql"
	RuleBiawinReadWrite  = "biawin-read-write"
	RuleProviderCall     = "biawin-provider-call"
	RuleForbiddenCommand = "biawin-forbidden-command"
)

// slice10Modules are the Slice 10 module trees.
var slice10Modules = []string{"internal/biawin", "internal/fulfillment"}

// slice10Trees are every tree where a Slice 10 command could hide.
var slice10Trees = []string{"internal/biawin", "internal/fulfillment", "internal/integration/biawin", "internal/platform/apiserver", "cmd/crm-api",
	"cmd/crm-biawin-shadow"}

// providerPkg is the only tree that may know the Biawin provider.
const providerPkg = "internal/integration/biawin"

// providerWrite matches a Biawin provider write call or name.
var providerWrite = regexp.MustCompile(`(?i)create_?order|charge_?wallet`)

// biawinForeignSQL matches SQL naming a schema other than biawin / fulfillment.
var biawinForeignSQL = regexp.MustCompile(`(?i)\b(from|join|into|update)\s+(identity|organization|customers|cases|custody|billing|payments|integration|audit|platform_featureflags|reporting|finance|commission)\.`)

// forbiddenCommand matches identifiers of Slice 10 forbidden commands.
var forbiddenCommand = regexp.MustCompile(`(?i)^(activate|suspend|terminate|reactivate)(subscription|membership|item)?s?$|^cancel(subscription|membership|item)s?$|^complete(dot|dotcase|case|service|item|fulfillment)$|` +
	`^(create|update|write|upsert|insert|save|sync)(project)?memberships?$|^(run|execute|calculate|process|pay)commissions?$|^(ship|dispatch|deliver)(order|case)?$|` +
	`^assign(case|expert|manager|handler|fulfillment)$|^(upload|attach)(service)?evidence$|^(execute|run|apply)upgrade$|^(charge|debit|credit|topup)wallet$`)

// forbiddenRoute matches a Slice 10 write / command route literal.
var forbiddenRoute = regexp.MustCompile(`(?i)/(biawin|fulfillment)/[a-z0-9_/{}.-]*\b(activate|suspend|terminate|cancel|order|orders|charge|wallet|ship|shipping|assign|complete|completion|evidence|upload|upgrade|commission|sync|import|refresh)\b`)

// writeSQL matches a mutating SQL statement.
var writeSQL = regexp.MustCompile(`(?i)\b(insert\s+into|update\s+[a-z_]+\.|delete\s+from)\b`)

// financeWriteRoute matches a finance write / posting route.
var financeWriteRoute = regexp.MustCompile(`(?i)/finance/[a-z0-9_/{}.-]*\b(post|posting|approve|reverse|settle|settlement|refund|credit|debit|adjust)\b`)

// sumSQL matches an SQL sum.
var sumSQL = regexp.MustCompile(`(?i)\bsum\s*\(`)

// phoneIdent matches a phone-like identifier.
var phoneIdent = regexp.MustCompile(`(?i)phone|mobile`)

// ownerSchemaRead matches SQL reading another module's schema.
var ownerSchemaRead = regexp.MustCompile(`(?i)\b(from|join)\s+(identity|organization|customers|cases|custody|billing|payments|integration|audit|platform_featureflags)\.`)

// financeForeignRead matches SQL reading a schema other than finance.
var financeForeignRead = regexp.MustCompile(`(?i)\b(from|join)\s+(identity|organization|customers|cases|custody|billing|payments|integration|audit|platform_featureflags|reporting)\.`)

// sqlish matches a string literal that is SQL (where a phone join could hide).
var sqlish = regexp.MustCompile(`(?i)\b(select|where|join|insert\s+into|update)\b`)

// aggregateSQL matches aggregate SQL functions (metric formulas).
var aggregateSQL = regexp.MustCompile(`(?i)\b(count|sum|avg|min|max)\s*\(`)

// exportLiteral matches an export / file-download route or media type.
var exportLiteral = regexp.MustCompile(`(?i)(/export\b|\.csv\b|\.xlsx\b|text/csv|spreadsheetml|/download\b|attachment;)`)

// statusFallbackSQL matches a COALESCE chain over status columns.
var statusFallbackSQL = regexp.MustCompile(`(?i)coalesce\s*\(\s*nullif\s*\(\s*[a-z_.]*status`)

// statusResolver matches identifiers of a generic invoice main-status
// resolver (TAD #20 deferred).
var statusResolver = regexp.MustCompile(`(?i)^(resolve|derive|compute|pick|select)?(canonical|effective|main|current)(invoice)?status(resolver|fallback)?$|^(resolve|derive|compute)(invoice)?(main)?status`)

// moneyModules are the trees where amounts live with explicit units.
var moneyModules = []string{"internal/billing", "internal/payments", "internal/finance"}

// appendOnlyWrite matches SQL that would mutate an append-only table.
var appendOnlyWrite = regexp.MustCompile(`(?i)\b(update|delete\s+from)\s+(custody\.custody_event|cases\.case_state_history|cases\.attribution_snapshot|payments\.attempt_state_history|reporting\.[a-z_]+|finance\.[a-z_]+|biawin\.[a-z_]+|fulfillment\.[a-z_]+)\b`)

// phoneLiteral matches a phone reference in cases/custody string literals.
var phoneLiteral = regexp.MustCompile(`(?i)phone`)

// mergePath matches identifiers of forbidden merge / canonicalisation
// operations (Slice 5; D-30 open). Matching is case-insensitive.
var mergePath = regexp.MustCompile(`(?i)^(resolvemerge|mergecustomers?|applymerge|canonicali[sz]ebyphone|canonicali[sz]ecustomers?|selectcanonical|pickwinner|mergebyphone|collapsecandidates?|reassigncustomer)`)

// slice4Modules are the business modules of Slice 4.
var slice4Modules = map[string]bool{"identity": true, "organization": true, "authorization": true, "customers": true,
	"cases": true, "custody": true, "billing": true, "payments": true, "reporting": true, "finance": true, "commission": true,
	"biawin": true, "fulfillment": true}

// schemaOwners maps a module schema to the only tree allowed to write it.
var schemaOwners = map[string]string{
	"identity":              "internal/identity",
	"organization":          "internal/organization",
	"customers":             "internal/customers",
	"cases":                 "internal/cases",
	"custody":               "internal/custody",
	"billing":               "internal/billing",
	"payments":              "internal/payments",
	"reporting":             "internal/reporting",
	"finance":               "internal/finance",
	"biawin":                "internal/biawin",
	"fulfillment":           "internal/fulfillment",
	"integration":           "internal/integration",
	"audit":                 "internal/platform/audit",
	"platform_featureflags": "internal/platform/featureflags",
}

var foreignWrite = regexp.MustCompile(`(?i)\b(insert\s+into|update|delete\s+from)\s+(identity|organization|customers|cases|custody|billing|payments|reporting|finance|biawin|fulfillment|integration|audit|platform_featureflags)\.`)

// readPathDirs never import an import command.
var readPathDirs = []string{"internal/platform/apiserver", "cmd/crm-api"}

// moduleImportAllowed reports whether a file of module top (rel) may import
// internal/<other>/<rest> under the Slice 4 module-boundary rule.
func moduleImportAllowed(rel, top, other, rest string) bool {
	if other == top {
		return true
	}
	switch {
	case top == "integration":
		return rest == "api" || other == "authorization" && rest == ""
	case !slice4Modules[top] || !slice4Modules[other]:
		return true
	case top == "identity" && other == "organization":
		return false
	case top == "customers" && other == "identity":
		return rest == "api" && under(rel, "internal/customers/http")
	case (top == "cases" || top == "custody") && other == "identity":
		return rest == "api" && under(rel, "internal/"+top+"/http")
	case top == "custody" && other == "cases":
		return rest == "api"
	case (top == "billing" || top == "payments") && other == "identity":
		return rest == "api" && under(rel, "internal/"+top+"/http")
	case top == "payments" && other == "billing":
		return rest == "api"
	case top == "billing" && other == "payments":
		return rest == "api" && under(rel, "internal/billing/http")
	case top == "billing" && (other == "customers" || other == "cases"):
		return rest == "api"
	case top == "reporting" && other == "identity":
		return rest == "api" && under(rel, "internal/reporting/http")
	case top == "reporting" && (other == "cases" || other == "custody" || other == "billing" || other == "payments" || other == "customers"):
		return rest == "api" && under(rel, "internal/reporting/projections")
	case top == "finance" && other == "identity":
		return rest == "api" && under(rel, "internal/finance/http")
	case top == "finance" && (other == "billing" || other == "payments"):
		return rest == "api" && (under(rel, "internal/finance/shadow") || under(rel, "internal/finance/domain"))
	case top == "commission" && other == "finance":
		return false
	case (top == "biawin" || top == "fulfillment") && other == "identity":
		return rest == "api" && under(rel, "internal/"+top+"/http")
	case other == "authorization" && (rest == "" || rest == "http"):
		return under(rel, "internal/"+top+"/http")
	case top == "authorization" || top == "organization" && other == "identity":
		return rest == "api"
	}
	return false
}

// DDL is allowed only in migration tooling and the test-only harness. The
// lint package itself is exempt because its patterns spell DDL keywords.
var ddlAllowedDirs = []string{"internal/platform/migrations", "cmd/crm-migrate", "internal/platform/pgtest", "internal/architecture"}

var runtimeDDL = regexp.MustCompile(`(?i)\b(create|alter|drop)\s+(table|schema|database|index|role|user|function|procedure|trigger|type|view|sequence|extension|policy)\b|\btruncate\b|\b(grant|revoke)\s+(all|select|insert|update|delete|truncate|references|trigger|usage|create|connect|temporary|execute|maintain|set)\b`)

// platformAllowed are the internal/<top> trees platform code may import.
var platformAllowed = map[string]bool{"kernel": true, "platform": true}

const (
	platformDir = "internal/platform"
	pgtestPkg   = "internal/platform/pgtest"
	// legacyTestPkg builds synthetic legacy snapshots; test files only.
	legacyTestPkg = "internal/integration/legacy/legacytest"
)

const (
	kernelDir      = "internal/kernel"
	moneyDir       = "internal/kernel/money"
	refsDir        = "internal/kernel/refs"
	legacyInteg    = "internal/integration/legacy"
	legacyRefMaker = "NewLegacyRef"
)

// skipDirs are never scanned: VCS data, the legacy plugin snapshot, docs,
// the prototype and test fixtures.
var skipDirs = map[string]bool{".git": true, "plugin": true, "prototype": true, "docs": true, "node_modules": true, "vendor": true, "testdata": true}

// forbiddenStd are standard-library import prefixes the kernel must not use.
var forbiddenStd = []string{"database", "net", "os", "syscall", "unsafe", "plugin", "io/fs", "path/filepath", "runtime/cgo"}

// testOnlyStd are allowed in kernel tests but not in kernel production code.
var testOnlyStd = []string{"math/rand", "time/tzdata", "testing"}

var legacyLiteral = regexp.MustCompile(`(^|[^A-Za-z0-9_])(wp|sn)_[A-Za-z0-9_]`)

// wooLiteral matches WooCommerce internals named in target code. The lint
// package itself is exempt (it spells the pattern).
var wooLiteral = regexp.MustCompile(`(?i)woo` + `commerce`)

// legacyImporters may import internal/integration/legacy/**.
var legacyImporters = []string{legacyInteg, "cmd/crm-legacy-import", "cmd/crm-finance-shadow", "cmd/crm-biawin-shadow"}

// legacyMigrationsImporter may import only the legacy migrations package.
const (
	legacyMigrationsImporter = "cmd/crm-migrate"
	legacyMigrationsPkg      = "integration/legacy/migrations"
)

// scannedTextExt are non-Go files scanned for legacy literals.
var scannedTextExt = []string{".sql", ".tmpl"}

const architectureDir = "internal/architecture"

var conversionName = regexp.MustCompile(`(?i)convert|toirr|toirt|ratio|rate|rial|toman`)

// Check scans every Go file under root. modulePath is the go.mod module path.
func Check(root, modulePath string) ([]Violation, error) {
	var out []Violation
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root && skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if !strings.HasSuffix(path, ".go") {
			vs, err := checkText(path, filepath.ToSlash(rel))
			if err != nil {
				return err
			}
			out = append(out, vs...)
			return nil
		}
		vs, err := checkFile(path, filepath.ToSlash(rel), modulePath)
		if err != nil {
			return err
		}
		out = append(out, vs...)
		return nil
	})
	sort.Slice(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Line < out[j].Line
	})
	return out, err
}

func under(rel, dir string) bool { return rel == dir || strings.HasPrefix(rel, dir+"/") }

func hasPrefixPath(p, prefix string) bool { return p == prefix || strings.HasPrefix(p, prefix+"/") }

func checkFile(path, rel, modulePath string) ([]Violation, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	isTest := strings.HasSuffix(rel, "_test.go")
	inKernel := under(rel, kernelDir)
	var out []Violation
	add := func(rule string, pos token.Pos, text string) {
		out = append(out, Violation{Rule: rule, File: rel, Line: fset.Position(pos).Line, Text: text})
	}

	for _, imp := range f.Imports {
		p, _ := strconv.Unquote(imp.Path.Value)
		if imp.Name != nil && imp.Name.Name == "." {
			add(RuleDotImport, imp.Pos(), "dot import of "+p+"; use a package-qualified import")
		}
		if inKernel {
			if msg := kernelImportProblem(p, modulePath, isTest); msg != "" {
				add(RuleKernelImport, imp.Pos(), msg)
			}
		}
		internalTop, internalRest, isInternal := splitInternal(p, modulePath)
		if !isInternal {
			continue
		}
		if under(rel, platformDir) && !platformAllowed[internalTop] {
			add(RulePlatformImport, imp.Pos(), "platform imports business module "+p)
		}
		if internalRest == "store" || strings.HasPrefix(internalRest, "store/") {
			if !under(rel, "internal/"+internalTop) {
				add(RuleForeignStore, imp.Pos(), "imports another module's store package "+p)
			}
		}
		for _, testOnly := range []string{pgtestPkg, legacyTestPkg} {
			if internalTop+"/"+internalRest == strings.TrimPrefix(testOnly, "internal/") && !isTest && !under(rel, testOnly) {
				add(RulePgtestImport, imp.Pos(), "test-only "+p+" imported from production code")
			}
		}
		if full := internalTop + "/" + internalRest; hasPrefixPath(full, strings.TrimPrefix(legacyInteg, "internal/")) && !legacyImportAllowed(rel, full) {
			add(RuleLegacyImport, imp.Pos(), "imports the legacy connector "+p+" outside "+legacyInteg+" and the offline jobs (crm-legacy-import, crm-finance-shadow, crm-biawin-shadow)")
		}
		// Test files are composition roots (like cmd/) and may wire modules.
		if relTop, ok := internalTopOf(rel); ok && !isTest && internalTop != "kernel" && internalTop != "platform" {
			if !moduleImportAllowed(rel, relTop, internalTop, internalRest) {
				add(RuleModuleBoundary, imp.Pos(), "internal/"+relTop+" imports "+p+"; cross-module access goes through the published api")
			}
		}
		if !isTest && internalTop == "reporting" && internalRest == "projections" && (under(rel, "internal/reporting/http") || under(rel, "internal/reporting/query") ||
			under(rel, "internal/platform/apiserver") || under(rel, "cmd/crm-api")) {
			add(RuleReadPathImport, imp.Pos(), "read path imports the projection rebuild "+p)
		}
		if !isTest && internalTop == "finance" && internalRest == "shadow" && (under(rel, "internal/finance/http") || under(rel, "internal/finance/query") ||
			under(rel, "internal/platform/apiserver") || under(rel, "cmd/crm-api")) {
			add(RuleReadPathImport, imp.Pos(), "read path imports the offline shadow rebuild "+p)
		}
		if full := internalTop + "/" + internalRest; hasPrefixPath(full, strings.TrimPrefix(providerPkg, "internal/")) && !under(rel, providerPkg) {
			add(RuleProviderCall, imp.Pos(), "imports the Biawin provider "+p+"; no provider is wired in Slice 10")
		}
		if internalRest == "imports" || strings.HasPrefix(internalRest, "imports/") {
			readPath := false
			for _, d := range readPathDirs {
				readPath = readPath || under(rel, d)
			}
			if top, ok := internalTopOf(rel); ok && (under(rel, "internal/"+top+"/http") || (top == "biawin" || top == "fulfillment") && under(rel, "internal/"+top+"/query")) {
				readPath = true
			}
			if readPath {
				add(RuleReadPathImport, imp.Pos(), "read path imports the import command "+p)
			}
		}
	}

	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.BasicLit:
			if x.Kind == token.STRING && !under(rel, legacyInteg) {
				if s, err := strconv.Unquote(x.Value); err == nil && legacyLiteral.MatchString(s) {
					add(RuleLegacyLiteral, x.Pos(), "legacy table literal "+x.Value+" outside "+legacyInteg)
				} else if err == nil && wooLiteral.MatchString(s) && !under(rel, architectureDir) {
					add(RuleLegacyLiteral, x.Pos(), "WooCommerce literal "+x.Value+" outside "+legacyInteg)
				}
			}
			if x.Kind == token.STRING && !isTest && !ddlAllowed(rel) {
				if s, err := strconv.Unquote(x.Value); err == nil && runtimeDDL.MatchString(s) {
					add(RuleRuntimeDDL, x.Pos(), "schema-mutating SQL outside migrations: "+runtimeDDL.FindString(s))
				}
			}
			if x.Kind == token.STRING && !under(rel, architectureDir) {
				if s, err := strconv.Unquote(x.Value); err == nil {
					if m := appendOnlyWrite.FindString(s); m != "" && !isTest {
						add(RuleAppendOnlyWrite, x.Pos(), "mutates an append-only table: "+m)
					}
					reportingSQL := under(rel, "internal/reporting") && sqlish.MatchString(s)
					if !isTest && (under(rel, "internal/cases") || under(rel, "internal/custody") || under(rel, "internal/billing") ||
						under(rel, "internal/payments") || reportingSQL) && phoneLiteral.MatchString(s) {
						add(RulePhoneKey, x.Pos(), "phone in a cases/custody/billing/payments literal; phone is never a key or join")
					}
					if !isTest && under(rel, "internal/reporting") && ownerSchemaRead.MatchString(s) {
						add(RuleReportingRead, x.Pos(), "reporting reads an owner schema: "+ownerSchemaRead.FindString(s))
					}
					if !isTest && under(rel, "internal/reporting") && !under(rel, "internal/reporting/query") && !under(rel, "internal/reporting/domain") &&
						aggregateSQL.MatchString(s) {
						add(RuleMetricFormula, x.Pos(), "aggregate SQL outside the shared report query spec: "+aggregateSQL.FindString(s))
					}
					if !isTest && (under(rel, "internal/reporting") || under(rel, "internal/platform/apiserver")) && exportLiteral.MatchString(s) {
						add(RuleExportEndpoint, x.Pos(), "export/download literal: "+exportLiteral.FindString(s))
					}
					inFinance := !isTest && under(rel, "internal/finance")
					if inFinance && financeForeignRead.MatchString(s) {
						add(RuleFinanceRead, x.Pos(), "finance reads another module's schema: "+financeForeignRead.FindString(s))
					}
					if inFinance && (under(rel, "internal/finance/http") || under(rel, "internal/finance/query")) && writeSQL.MatchString(s) {
						add(RuleFinanceReadWrite, x.Pos(), "write SQL on the finance read path: "+writeSQL.FindString(s))
					}
					if !isTest && (under(rel, "internal/finance") || under(rel, "internal/platform/apiserver")) && financeWriteRoute.MatchString(s) {
						add(RuleFinanceRoute, x.Pos(), "finance write route literal: "+financeWriteRoute.FindString(s))
					}
					if inFinance && sumSQL.MatchString(s) && !strings.Contains(strings.ToLower(s), "unit") {
						add(RuleCrossUnitSum, x.Pos(), "SQL sum not scoped by unit in finance")
					}
					if inFinance && phoneIdent.MatchString(s) {
						add(RulePhoneKey, x.Pos(), "phone / mobile in a finance literal; never an account key, filter or join")
					}
					inSlice10 := !isTest && inAny(rel, slice10Modules)
					if inSlice10 && biawinForeignSQL.MatchString(s) {
						add(RuleBiawinSQL, x.Pos(), "biawin / fulfillment SQL names another schema: "+biawinForeignSQL.FindString(s))
					}
					if inSlice10 && slice10ReadPath(rel) && writeSQL.MatchString(s) {
						add(RuleBiawinReadWrite, x.Pos(), "write SQL on the biawin / fulfillment read path: "+writeSQL.FindString(s))
					}
					if inSlice10 && phoneIdent.MatchString(s) {
						add(RulePhoneKey, x.Pos(), "phone / mobile in a biawin / fulfillment literal; never an identity, join, dedupe or provider key")
					}
					if !isTest && !under(rel, providerPkg) && providerWrite.MatchString(s) {
						add(RuleProviderCall, x.Pos(), "Biawin provider write literal "+providerWrite.FindString(s)+" outside "+providerPkg)
					}
					if !isTest && inAny(rel, slice10Trees) && forbiddenRoute.MatchString(s) {
						add(RuleForbiddenCommand, x.Pos(), "Slice 10 command route literal: "+forbiddenRoute.FindString(s))
					}
					if !under(rel, legacyInteg) && statusFallbackSQL.MatchString(s) {
						add(RuleStatusFallback, x.Pos(), "COALESCE status fallback outside the legacy connector (TAD #20 deferred)")
					}
				}
			}
			if x.Kind == token.STRING && !isTest && !under(rel, architectureDir) {
				if s, err := strconv.Unquote(x.Value); err == nil {
					for _, m := range foreignWrite.FindAllStringSubmatch(s, -1) {
						if owner := schemaOwners[strings.ToLower(m[2])]; !under(rel, owner) {
							add(RuleForeignWrite, x.Pos(), "writes schema "+m[2]+" outside "+owner)
						}
					}
				}
			}
		case *ast.BinaryExpr:
			if !isTest && inMoney(rel) && (x.Op == token.MUL || x.Op == token.QUO) {
				for _, side := range []ast.Expr{x.X, x.Y} {
					if lit, ok := side.(*ast.BasicLit); ok && lit.Kind == token.INT && lit.Value == "10" {
						add(RuleUnitConversion, x.Pos(), "arithmetic by 10 in billing/payments; units are never converted")
					}
				}
			}
		case *ast.Ident:
			if !under(rel, architectureDir) && !under(rel, legacyInteg) && statusResolver.MatchString(x.Name) {
				add(RuleStatusFallback, x.Pos(), "generic invoice status resolver "+x.Name+" (TAD #20 deferred)")
			}
			if !under(rel, architectureDir) && mergePath.MatchString(x.Name) {
				add(RuleMergePath, x.Pos(), "merge/canonicalisation identifier "+x.Name+" (Slice 5 is read/shadow only; D-30 open)")
			}
			if inKernel && (x.Name == "float32" || x.Name == "float64") {
				add(RuleKernelFloat, x.Pos(), x.Name+" in kernel")
			}
			if under(rel, "internal/finance") && (x.Name == "float32" || x.Name == "float64") {
				add(RuleFinanceFloat, x.Pos(), x.Name+" in finance (money is int64 + unit)")
			}
			if !isTest && under(rel, "internal/finance") && phoneIdent.MatchString(x.Name) {
				add(RulePhoneKey, x.Pos(), "phone / mobile identifier "+x.Name+" in finance")
			}
			if !isTest && inAny(rel, slice10Modules) && phoneIdent.MatchString(x.Name) {
				add(RulePhoneKey, x.Pos(), "phone / mobile identifier "+x.Name+" in biawin / fulfillment")
			}
			if !isTest && !under(rel, architectureDir) && !under(rel, providerPkg) && providerWrite.MatchString(x.Name) {
				add(RuleProviderCall, x.Pos(), "Biawin provider write identifier "+x.Name+" outside "+providerPkg)
			}
			if !isTest && inAny(rel, slice10Trees) && forbiddenCommand.MatchString(x.Name) {
				add(RuleForbiddenCommand, x.Pos(), "Slice 10 forbidden command identifier "+x.Name)
			}
		case *ast.SelectorExpr:
			if x.Sel.Name == legacyRefMaker && !under(rel, legacyInteg) && !under(rel, refsDir) {
				add(RuleLegacyRefFactory, x.Pos(), legacyRefMaker+" called outside "+legacyInteg)
			}
			if inKernel {
				if id, ok := x.X.(*ast.Ident); ok && id.Name == "time" {
					switch x.Sel.Name {
					case "Now", "Since", "Until":
						add(RuleKernelClock, x.Pos(), "time."+x.Sel.Name+" in kernel; inject a Clock")
					}
				} else if x.Sel.Name == "Local" && !isTest {
					add(RuleKernelClock, x.Pos(), ".Local() converts to the machine-local zone")
				}
			}
		case *ast.CompositeLit:
			if !isTest && (under(rel, "internal/finance/domain") || under(rel, "internal/finance/shadow")) && isEntryType(x.Type) && len(x.Elts) > 0 {
				has := false
				for _, e := range x.Elts {
					if kv, ok := e.(*ast.KeyValueExpr); ok {
						if k, ok := kv.Key.(*ast.Ident); ok && k.Name == "BusinessKey" {
							has = true
						}
					}
				}
				if !has {
					add(RuleBusinessKey, x.Pos(), "ledger Entry literal without a BusinessKey")
				}
			}
		case *ast.FuncDecl:
			if inMoney(rel) && !isTest && conversionName.MatchString(x.Name.Name) {
				add(RuleUnitConversion, x.Pos(), "conversion-like function "+x.Name.Name+" in billing/payments/finance")
			}
			if under(rel, moneyDir) && !isTest && x.Name.IsExported() && conversionName.MatchString(x.Name.Name) {
				add(RuleMoneyConversion, x.Pos(), "money exports conversion-like function "+x.Name.Name)
			}
		}
		return true
	})
	return out, nil
}

// isEntryType reports a composite literal of the ledger Entry type.
func isEntryType(t ast.Expr) bool {
	switch v := t.(type) {
	case *ast.Ident:
		return v.Name == "Entry"
	case *ast.SelectorExpr:
		return v.Sel.Name == "Entry"
	}
	return false
}

// slice10ReadPath reports the biawin / fulfillment read path (http, query).
func slice10ReadPath(rel string) bool {
	for _, m := range slice10Modules {
		if under(rel, m+"/http") || under(rel, m+"/query") {
			return true
		}
	}
	return false
}

func inAny(rel string, dirs []string) bool {
	for _, d := range dirs {
		if under(rel, d) {
			return true
		}
	}
	return false
}

func inMoney(rel string) bool {
	for _, d := range moneyModules {
		if under(rel, d) {
			return true
		}
	}
	return false
}

// internalTopOf returns the internal/<top> of a repository-relative path.
func internalTopOf(rel string) (string, bool) {
	if !strings.HasPrefix(rel, "internal/") {
		return "", false
	}
	top, _, _ := strings.Cut(strings.TrimPrefix(rel, "internal/"), "/")
	return top, top != ""
}

// splitInternal splits "<module>/internal/<top>/<rest>" into top and rest.
func splitInternal(p, modulePath string) (top, rest string, ok bool) {
	prefix := modulePath + "/internal/"
	if !strings.HasPrefix(p, prefix) {
		return "", "", false
	}
	top, rest, _ = strings.Cut(strings.TrimPrefix(p, prefix), "/")
	return top, rest, top != ""
}

// legacyImportAllowed reports whether rel may import the legacy package full
// ("integration/legacy/...").
func legacyImportAllowed(rel, full string) bool {
	for _, d := range legacyImporters {
		if under(rel, d) {
			return true
		}
	}
	return under(rel, legacyMigrationsImporter) && full == legacyMigrationsPkg
}

// checkText scans a non-Go text file (.sql, .tmpl) outside the legacy tree
// for legacy table names and WooCommerce internals.
func checkText(path, rel string) ([]Violation, error) {
	scan := false
	for _, ext := range scannedTextExt {
		if strings.HasSuffix(rel, ext) {
			scan = true
		}
	}
	if !scan || under(rel, legacyInteg) || under(rel, architectureDir) {
		return nil, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out []Violation
	for i, line := range strings.Split(string(b), "\n") {
		if legacyLiteral.MatchString(line) || wooLiteral.MatchString(line) {
			out = append(out, Violation{Rule: RuleLegacyLiteral, File: rel, Line: i + 1, Text: "legacy name in " + filepath.Ext(rel) + " file outside " + legacyInteg})
		}
	}
	return out, nil
}

func ddlAllowed(rel string) bool {
	for _, d := range ddlAllowedDirs {
		if under(rel, d) {
			return true
		}
	}
	return false
}

func kernelImportProblem(p, modulePath string, isTest bool) string {
	first := strings.SplitN(p, "/", 2)[0]
	if strings.Contains(first, ".") {
		if hasPrefixPath(p, modulePath+"/"+kernelDir) {
			return ""
		}
		return "kernel imports non-kernel package " + p
	}
	for _, bad := range forbiddenStd {
		if hasPrefixPath(p, bad) {
			return "kernel imports forbidden std package " + p
		}
	}
	if !isTest {
		for _, t := range testOnlyStd {
			if hasPrefixPath(p, t) {
				return "kernel production code imports test-only package " + p
			}
		}
	}
	return ""
}
