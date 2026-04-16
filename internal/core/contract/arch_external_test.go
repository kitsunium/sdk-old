//go:build archcheck

// Architecture boundary enforcement. Runs only under `go test -tags=archcheck`
// because Bazel's sandboxed test execution does not expose the full source
// tree this walker needs. CI runs it alongside `bazel test` via a Make target.
package contract_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const modulePrefix = "github.com/kitsunium/sdk/"

// layerRule describes what a given source subtree may and may not import.
type layerRule struct {
	name          string
	root          string
	allowedExtra  []string
	forbiddenPkgs []string
	externalAllow bool
}

// rules defines the architecture contract. New subtrees must be added here.
var rules = []layerRule{
	{
		name:          "internal/kernel",
		root:          "internal/kernel",
		allowedExtra:  []string{"internal/kernel/"},
		forbiddenPkgs: []string{"internal/core/", "components/", "adapters/", "ports/"},
		externalAllow: false,
	},
	{
		name:          "internal/core",
		root:          "internal/core",
		allowedExtra:  []string{"internal/kernel/", "internal/core/"},
		forbiddenPkgs: []string{"components/", "adapters/", "ports/"},
		externalAllow: true,
	},
	{
		name:          "ports",
		root:          "ports",
		allowedExtra:  []string{"ports"},
		forbiddenPkgs: []string{"internal/", "components/", "adapters/"},
		externalAllow: false,
	},
	{
		name:          "components",
		root:          "components",
		allowedExtra:  []string{"ports/", "internal/kernel/", "internal/core/", "components/"},
		forbiddenPkgs: []string{"adapters/"},
		externalAllow: true,
	},
	{
		name:          "adapters",
		root:          "adapters",
		allowedExtra:  []string{"ports/", "internal/kernel/", "internal/core/", "adapters/"},
		forbiddenPkgs: []string{"components/"},
		externalAllow: true,
	},
}

// TestArchitectureBoundaries walks every Go file under each layer root and
// rejects imports that violate the documented dependency rules.
func TestArchitectureBoundaries(t *testing.T) {
	repoRoot := findRepoRoot(t)

	for _, r := range rules {
		t.Run(r.name, func(t *testing.T) {
			root := filepath.Join(repoRoot, r.root)
			if !dirExists(root) {
				t.Logf("skip %s: directory absent (not yet created)", r.root)
				return
			}
			walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() || !strings.HasSuffix(path, ".go") {
					return nil
				}
				checkFile(t, path, r)
				return nil
			})
			if walkErr != nil {
				t.Fatalf("walk %s: %v", root, walkErr)
			}
		})
	}
}

// checkFile parses a single Go source file and validates its import list.
// Test files relax the externalAllow rule (third-party assertion libraries
// like testify are standard) but the cross-layer forbiddenPkgs check stays
// strict — a test under internal/kernel still may not import components/.
func checkFile(t *testing.T, path string, r layerRule) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	isTest := strings.HasSuffix(path, "_test.go")
	for _, imp := range f.Imports {
		raw := strings.Trim(imp.Path.Value, `"`)
		violation := classifyImport(raw, r, isTest)
		if violation != "" {
			t.Errorf("%s imports %q: %s", stripPrefix(path), raw, violation)
		}
	}
}

// classifyImport returns an empty string if the import is allowed under the
// given rule, or a human-readable violation reason otherwise.
func classifyImport(imp string, r layerRule, isTest bool) string {
	if !strings.Contains(firstSegment(imp), ".") {
		return ""
	}
	if !strings.HasPrefix(imp, modulePrefix) {
		if r.externalAllow || isTest {
			return ""
		}
		return "external (non-stdlib) imports are forbidden in this layer"
	}
	relative := strings.TrimPrefix(imp, modulePrefix)
	for _, bad := range r.forbiddenPkgs {
		if strings.HasPrefix(relative, bad) {
			return "forbidden cross-layer import (" + bad + ")"
		}
	}
	for _, good := range r.allowedExtra {
		trimmed := strings.TrimSuffix(good, "/")
		if relative == trimmed || strings.HasPrefix(relative, trimmed+"/") {
			return ""
		}
	}
	return "not in the allowed subtree for this layer"
}

// firstSegment returns the portion of p before the first '/'.
func firstSegment(p string) string {
	head, _, _ := strings.Cut(p, "/")
	return head
}

// stripPrefix trims the repo prefix from a path for readable error messages.
func stripPrefix(p string) string {
	for _, marker := range []string{"/internal/", "/components/", "/ports/", "/adapters/"} {
		if i := strings.Index(p, marker); i >= 0 {
			return p[i+1:]
		}
	}
	return p
}

// findRepoRoot walks upward from the test binary CWD until it finds go.mod.
func findRepoRoot(t *testing.T) string {
	t.Helper()
	start, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	dir := start
	for {
		if fileExists(filepath.Join(dir, "go.mod")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not locate go.mod starting from %s", start)
		}
		dir = parent
	}
}

// dirExists reports whether path is an existing directory.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// fileExists reports whether path is an existing regular file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
