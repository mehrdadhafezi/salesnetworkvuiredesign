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
)

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
var legacyImporters = []string{legacyInteg, "cmd/crm-legacy-import"}

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
			add(RuleLegacyImport, imp.Pos(), "imports the legacy connector "+p+" outside "+legacyInteg+" and cmd/crm-legacy-import")
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
		case *ast.Ident:
			if inKernel && (x.Name == "float32" || x.Name == "float64") {
				add(RuleKernelFloat, x.Pos(), x.Name+" in kernel")
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
		case *ast.FuncDecl:
			if under(rel, moneyDir) && !isTest && x.Name.IsExported() && conversionName.MatchString(x.Name.Name) {
				add(RuleMoneyConversion, x.Pos(), "money exports conversion-like function "+x.Name.Name)
			}
		}
		return true
	})
	return out, nil
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
