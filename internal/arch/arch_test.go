package arch

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const modulePath = "github.com/Dashulya-coder/CaseTaskNotifier"

var skipDirs = map[string]bool{
	"vendor":            true,
	".git":              true,
	".ops":              true,
	".idea":             true,
	"node_modules":      true,
	"test-results":      true,
	"playwright-report": true,
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}

func buildGraph(t *testing.T) map[string][]string {
	t.Helper()
	root := moduleRoot(t)
	fset := token.NewFileSet()
	graph := map[string][]string{}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}

		relDir, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return err
		}
		pkg := filepath.ToSlash(relDir)
		if pkg == "." {
			pkg = ""
		}
		if _, ok := graph[pkg]; !ok {
			graph[pkg] = nil
		}

		for _, imp := range f.Imports {
			p := strings.Trim(imp.Path.Value, `"`)
			if !strings.HasPrefix(p, modulePath) {
				continue
			}
			rel := strings.TrimPrefix(p, modulePath+"/")
			if rel == pkg {
				continue
			}
			graph[pkg] = appendUnique(graph[pkg], rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return graph
}

func layerOf(pkg string) string {
	switch {
	case pkg == "internal/app":
		return "app"
	case pkg == "main":
		return "main"
	case strings.HasPrefix(pkg, "internal/http/"):
		return "delivery"
	case oneOf(pkg, "internal/subscription", "internal/release", "internal/scanner", "internal/saga"):
		return "domain"
	case oneOf(pkg, "internal/repository", "internal/github", "internal/client/notification", "internal/notification/publisher"):
		return "adapter"
	case oneOf(pkg, "internal/repo", "internal/token", "internal/urlbuilder", "internal/validator", "internal/metrics", "internal/logger", "internal/config", "internal/mailer"):
		return "kernel"
	case pkg == "notifier/internal/app":
		return "n_app"
	case pkg == "notifier/main":
		return "n_main"
	case pkg == "notifier/internal/delivery":
		return "n_domain"
	case oneOf(pkg, "notifier/internal/server", "notifier/internal/rest", "notifier/internal/consumer"):
		return "n_inbound"
	case oneOf(pkg, "notifier/internal/store", "notifier/internal/smtp"):
		return "n_adapter"
	case oneOf(pkg, "notifier/internal/config", "notifier/internal/metrics"):
		return "n_kernel"
	case pkg == "notifier/contract" || strings.HasPrefix(pkg, "notifier/gen"):
		return "contract"
	}
	return ""
}

var allowedImports = map[string]map[string]bool{
	"kernel":    set("kernel", "contract"),
	"domain":    set("kernel", "domain", "adapter", "contract"),
	"adapter":   set("kernel", "domain", "adapter", "contract"),
	"delivery":  set("kernel", "domain", "delivery", "contract"),
	"app":       set("kernel", "domain", "adapter", "delivery", "app", "contract"),
	"main":      set("app", "kernel", "contract"),
	"n_kernel":  set("n_kernel", "contract"),
	"n_domain":  set("contract"),
	"n_adapter": set("n_kernel", "n_domain", "n_adapter", "contract"),
	"n_inbound": set("n_kernel", "n_domain", "n_inbound", "contract"),
	"n_app":     set("n_kernel", "n_domain", "n_adapter", "n_inbound", "n_app", "contract"),
	"n_main":    set("n_app", "contract"),
	"contract":  set("contract"),
}

func TestEveryInternalPackageHasALayer(t *testing.T) {
	graph := buildGraph(t)
	for pkg := range graph {
		if !classifiable(pkg) {
			continue
		}
		if layerOf(pkg) == "" {
			t.Errorf("package %q is not assigned to any architectural layer; add it to layerOf and allowedImports", pkg)
		}
	}
}

func TestLayerDependencyRules(t *testing.T) {
	graph := buildGraph(t)
	for _, pkg := range sortedKeys(graph) {
		from := layerOf(pkg)
		if from == "" {
			continue
		}
		for _, imp := range graph[pkg] {
			to := layerOf(imp)
			if to == "" {
				continue
			}
			if !allowedImports[from][to] {
				t.Errorf("illegal dependency: %s (%s) -> %s (%s)", pkg, from, imp, to)
			}
		}
	}
}

func TestServiceBoundaryIsContractOnly(t *testing.T) {
	graph := buildGraph(t)
	for _, pkg := range sortedKeys(graph) {
		for _, imp := range graph[pkg] {
			if strings.HasPrefix(pkg, "internal/") && strings.HasPrefix(imp, "notifier/internal/") {
				t.Errorf("monolith package %q imports notifier internals %q; cross only via notifier/contract or notifier/gen", pkg, imp)
			}
			if strings.HasPrefix(pkg, "notifier/") && strings.HasPrefix(imp, "internal/") {
				t.Errorf("notifier package %q imports monolith internals %q; the notifier must not depend on the monolith", pkg, imp)
			}
		}
	}
}

func TestNotifierDomainHasNoInternalDeps(t *testing.T) {
	graph := buildGraph(t)
	for _, imp := range graph["notifier/internal/delivery"] {
		if layerOf(imp) != "contract" {
			t.Errorf("notifier/internal/delivery must stay transport-agnostic but imports %q", imp)
		}
	}
}

func TestCompositionRootImportedOnlyByMain(t *testing.T) {
	graph := buildGraph(t)
	roots := map[string]string{
		"internal/app":          "main",
		"notifier/internal/app": "notifier/main",
	}
	for root, onlyImporter := range roots {
		for _, pkg := range sortedKeys(graph) {
			if pkg == onlyImporter {
				continue
			}
			for _, imp := range graph[pkg] {
				if imp == root {
					t.Errorf("composition root %q imported by %q; only %q may wire it", root, pkg, onlyImporter)
				}
			}
		}
	}
}

func classifiable(pkg string) bool {
	if strings.HasPrefix(pkg, "internal/") || pkg == "main" {
		return true
	}
	if strings.HasPrefix(pkg, "notifier/internal/") || pkg == "notifier/main" {
		return true
	}
	return false
}

func oneOf(v string, opts ...string) bool {
	for _, o := range opts {
		if v == o {
			return true
		}
	}
	return false
}

func set(vs ...string) map[string]bool {
	m := make(map[string]bool, len(vs))
	for _, v := range vs {
		m[v] = true
	}
	return m
}

func appendUnique(s []string, v string) []string {
	for _, x := range s {
		if x == v {
			return s
		}
	}
	return append(s, v)
}

func sortedKeys(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
