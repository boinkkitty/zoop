// Package archtest enforces ADR 0002: core packages never import HTTP, the
// database or an adapter. Add every new core package to core.
package archtest_test

import (
	"go/build"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const module = "github.com/boinkkitty/zoop/"

// core holds the application's rules.
var core = []string{
	"internal/auth",
	"internal/auth/password",
	"internal/days",
	"internal/pages",
}

// forbidden are import path prefixes that belong to adapters, never the core.
var forbidden = []string{
	"net/http",
	"database/sql",
	"github.com/go-chi/",
	"github.com/jackc/",
	module + "internal/db",
	module + "internal/postgres",
	module + "internal/server",
	module + "internal/app",
	module + "internal/testenv",
	module + "web",
}

func TestCorePackagesDontImportAdapters(t *testing.T) {
	_, here, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(here), "..", "..")
	for _, pkg := range core {
		p, err := build.Default.ImportDir(filepath.Join(root, pkg), 0)
		if err != nil {
			t.Errorf("%s: %v", pkg, err)
			continue
		}
		for _, imp := range p.Imports {
			for _, bad := range forbidden {
				bad = strings.TrimSuffix(bad, "/")
				if imp == bad || strings.HasPrefix(imp, bad+"/") {
					t.Errorf("core package %s imports %s (ADR 0002: the core must not depend on adapters)", pkg, imp)
				}
			}
		}
	}
}
