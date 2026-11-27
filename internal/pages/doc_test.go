package pages_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/boinkkitty/zoop/internal/pages"
)

// The fixtures are shared with the client's schema tests (web/src/page).
const fixtures = "../../testdata/page-docs"

type fixture struct {
	Notebook  string          `json:"notebook"`
	Doc       json.RawMessage `json:"doc"`
	PlainText string          `json:"plainText"`
	Why       string          `json:"why"`
}

func loadFixtures(t *testing.T, kind string) map[string]fixture {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(fixtures, kind, "*.json"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no %s fixtures in %s (%v)", kind, fixtures, err)
	}
	out := map[string]fixture{}
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var fx fixture
		if err := json.Unmarshal(raw, &fx); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		out[filepath.Base(f)] = fx
	}
	return out
}

func TestValidDocsAreAcceptedWithTheirPlainText(t *testing.T) {
	for name, fx := range loadFixtures(t, "valid") {
		t.Run(name, func(t *testing.T) {
			got, err := pages.ParseDoc(fx.Notebook, fx.Doc)
			if err != nil {
				t.Fatalf("ParseDoc refused a valid doc: %v", err)
			}
			if got != fx.PlainText {
				t.Errorf("plain text = %q, want %q", got, fx.PlainText)
			}
		})
	}
}

func TestInvalidDocsAreRefused(t *testing.T) {
	for name, fx := range loadFixtures(t, "invalid") {
		t.Run(name, func(t *testing.T) {
			_, err := pages.ParseDoc(fx.Notebook, fx.Doc)
			if err == nil {
				t.Fatalf("ParseDoc accepted a doc that should be refused: %s", fx.Why)
			}
			t.Logf("refused (%s): %v", fx.Why, err)
		})
	}
}

func TestSchemaVersionMatchesTheSharedFixtures(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(fixtures, "schema-version.json"))
	if err != nil {
		t.Fatal(err)
	}
	var v struct{ SchemaVersion int }
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	if pages.SchemaVersion != v.SchemaVersion {
		t.Errorf("pages.SchemaVersion = %d, fixtures say %d", pages.SchemaVersion, v.SchemaVersion)
	}
}
