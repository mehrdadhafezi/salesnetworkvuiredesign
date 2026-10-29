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
package architecture

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
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
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
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
		if inKernel {
			if msg := kernelImportProblem(p, modulePath, isTest); msg != "" {
				add(RuleKernelImport, imp.Pos(), msg)
			}
		}
	}

	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.BasicLit:
			if x.Kind == token.STRING && !under(rel, legacyInteg) {
				if s, err := strconv.Unquote(x.Value); err == nil && legacyLiteral.MatchString(s) {
					add(RuleLegacyLiteral, x.Pos(), "legacy table literal "+x.Value+" outside "+legacyInteg)
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
