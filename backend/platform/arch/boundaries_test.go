// Package arch tests the dependency rules the structure depends on. Go's
// internal/ rule already stops a service reaching into a sibling, but
// nothing stops platform/ from growing a dependency on a feature — which
// would invert the T and make the shared layer un-shareable. This fails
// the build if that happens.
package arch_test

import (
	"go/build"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Dir(filepath.Dir(wd))
}

// walkImports visits every Go package under dir and reports its imports.
func walkImports(t *testing.T, root, dir string, visit func(pkgDir string, imports []string)) {
	t.Helper()
	err := filepath.Walk(filepath.Join(root, dir), func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			return err
		}
		pkg, perr := build.ImportDir(path, 0)
		if perr != nil {
			return nil // directory holds no Go files
		}
		rel, _ := filepath.Rel(root, path)
		visit(rel, append(pkg.Imports, pkg.TestImports...))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// platform is the horizontal bar of the T: shared by every service and
// aware of none of them.
func TestPlatformDoesNotDependOnAnyService(t *testing.T) {
	root := repoRoot(t)
	walkImports(t, root, "platform", func(pkgDir string, imports []string) {
		for _, imp := range imports {
			if strings.Contains(imp, "campaigntrackerpro/services/") {
				t.Errorf("%s imports %s — platform must not depend on a service", pkgDir, imp)
			}
		}
	})
}

// A service may read another service's public contract, but never its
// implementation. Go enforces this for internal/; this covers the case
// where someone adds a non-internal package to a service and a sibling
// starts reaching for it.
func TestServicesDoNotImportSiblingInternals(t *testing.T) {
	root := repoRoot(t)
	walkImports(t, root, "services", func(pkgDir string, imports []string) {
		owner := strings.Split(pkgDir, string(filepath.Separator))[1]
		for _, imp := range imports {
			if !strings.Contains(imp, "/internal/") {
				continue
			}
			if strings.Contains(imp, "campaigntrackerpro/services/") &&
				!strings.Contains(imp, "services/"+owner+"/internal/") {
				t.Errorf("%s imports %s — a service must use a sibling's public contract, not its internals", pkgDir, imp)
			}
		}
	})
}
