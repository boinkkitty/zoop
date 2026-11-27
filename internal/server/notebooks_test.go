package server_test

import (
	"net/http"
	"testing"
)

// I Learned has no Title line, so its first line may be blank.
const learnedDoc = `{"type":"doc","content":[
	{"type":"paragraph"},
	{"type":"paragraph","content":[{"type":"text","text":"Octopuses have three hearts."}]}]}`

func TestILearnedIsItsOwnPageOnTheSameDay(t *testing.T) {
	app := startApp(t, tokyoMorning)
	b := app.browser()
	b.signup(uniqueEmail("learner"), "Asia/Tokyo")

	empty, res := b.page("2026-11-26", "i-learned")
	if res.status != http.StatusOK || empty.Notebook != "i-learned" || empty.Version != 0 || !sameJSON(t, empty.Doc, emptyDoc) {
		t.Fatalf("GET today's I Learned = %d %+v, want 200, an empty doc at version 0", res.status, empty)
	}
	if res := b.save("2026-11-26", "i-learned", learnedDoc, 0); res.status != http.StatusOK {
		t.Fatalf("PUT I Learned = %d %s, want 200", res.status, res.body)
	}
	if got, _ := b.page("2026-11-26", "i-learned"); got.Version != 1 || !sameJSON(t, got.Doc, learnedDoc) {
		t.Errorf("I Learned after save = version %d doc %s, want version 1 and the saved doc", got.Version, got.Doc)
	}
	if journal, _ := b.page("2026-11-26", "my-journal"); journal.Version != 0 {
		t.Errorf("My Journal version = %d, want 0: saving I Learned must not touch it", journal.Version)
	}
}

func TestILearnedIsBlockPlus(t *testing.T) {
	app := startApp(t, tokyoMorning)
	b := app.browser()
	b.signup(uniqueEmail("no-blocks"), "Asia/Tokyo")

	res := b.save("2026-11-26", "i-learned", `{"type":"doc","content":[]}`, 0)
	if res.status != http.StatusBadRequest || res.errorCode(t) != "validation" {
		t.Errorf("PUT I Learned with no blocks = %d %s, want 400 validation", res.status, res.body)
	}
}

func TestAnUnknownNotebookIsNotFound(t *testing.T) {
	app := startApp(t, tokyoMorning)
	b := app.browser()
	b.signup(uniqueEmail("diary"), "Asia/Tokyo")

	for _, slug := range []string{"diary", "My-Journal", "i-learned%20"} {
		if _, res := b.page("2026-11-26", slug); res.status != http.StatusNotFound || res.errorCode(t) != "not_found" {
			t.Errorf("GET %s = %d %s, want 404 not_found", slug, res.status, res.body)
		}
		if res := b.save("2026-11-26", slug, helloDoc, 0); res.status != http.StatusNotFound || res.errorCode(t) != "not_found" {
			t.Errorf("PUT %s = %d %s, want 404 not_found", slug, res.status, res.body)
		}
	}
}
