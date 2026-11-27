package server_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

type pageBody struct {
	Day           string          `json:"day"`
	Notebook      string          `json:"notebook"`
	Doc           json.RawMessage `json:"doc"`
	SchemaVersion int             `json:"schemaVersion"`
	Version       int             `json:"version"`
	UpdatedAt     *string         `json:"updatedAt"`
	Editable      bool            `json:"editable"`
	LocksAt       *string         `json:"locksAt"`
}

func (b *browser) page(day, notebook string) (pageBody, response) {
	b.app.t.Helper()
	res := b.do(http.MethodGet, "/api/v1/pages/"+day+"/"+notebook, nil)
	var p pageBody
	if res.status == http.StatusOK {
		res.json(b.app.t, &p)
	}
	return p, res
}

const emptyDoc = `{"type":"doc","content":[{"type":"paragraph"}]}`

func sameJSON(t *testing.T, got json.RawMessage, want string) bool {
	t.Helper()
	var g, w any
	if err := json.Unmarshal(got, &g); err != nil {
		t.Fatalf("doc %s is not JSON: %v", got, err)
	}
	if err := json.Unmarshal([]byte(want), &w); err != nil {
		t.Fatal(err)
	}
	gb, _ := json.Marshal(g)
	wb, _ := json.Marshal(w)
	return string(gb) == string(wb)
}

func TestTodaysUnwrittenPageIsAnEmptyDocAtVersionZero(t *testing.T) {
	app := startApp(t, tokyoMorning)
	b := app.browser()
	b.signup(uniqueEmail("fresh"), "Asia/Tokyo")

	got, res := b.page("2026-11-26", "my-journal")
	if res.status != http.StatusOK {
		t.Fatalf("GET today's page = %d %s, want 200", res.status, res.body)
	}
	if got.Day != "2026-11-26" || got.Notebook != "my-journal" {
		t.Errorf("day/notebook = %q/%q, want 2026-11-26/my-journal", got.Day, got.Notebook)
	}
	if !sameJSON(t, got.Doc, emptyDoc) {
		t.Errorf("doc = %s, want %s", got.Doc, emptyDoc)
	}
	if got.Version != 0 || got.UpdatedAt != nil {
		t.Errorf("version %d updatedAt %v, want 0 and null", got.Version, got.UpdatedAt)
	}
	if got.SchemaVersion != 1 {
		t.Errorf("schemaVersion = %d, want 1", got.SchemaVersion)
	}
	if !got.Editable {
		t.Error("editable = false, want true for today")
	}
}

func TestTheServerDecidesWhichDayIsToday(t *testing.T) {
	app := startApp(t, tokyoMorning) // 05:00 on 26 Nov in Tokyo
	b := app.browser()
	b.signup(uniqueEmail("today"), "Asia/Tokyo")

	today, _ := b.page("2026-11-26", "my-journal")
	// A Free User's today locks at the end of tomorrow's Grace Window: 02:00 on 27 Nov in Tokyo.
	if today.LocksAt == nil || *today.LocksAt != "2026-11-26T17:00:00Z" {
		t.Errorf("locksAt = %v, want 2026-11-26T17:00:00Z", today.LocksAt)
	}
	// Reading didn't create the Page, so the first save is still over version 0.
	if res := b.save("2026-11-26", "my-journal", helloDoc, 0); res.status != http.StatusOK {
		t.Errorf("first save after a GET = %d %s, want 200", res.status, res.body)
	}

	for _, path := range []string{"2026-11-27/my-journal", "2026-11-25/my-journal", "2026-11-26/diary", "26-11-2026/my-journal"} {
		if res := b.do(http.MethodGet, "/api/v1/pages/"+path, nil); res.status != http.StatusNotFound {
			t.Errorf("GET %s = %d, want 404 (tomorrow, before the first Day, unknown Notebook, bad date)", path, res.status)
		}
	}

	app.clock.Advance(24 * time.Hour) // 05:00 on 27 Nov in Tokyo
	yesterday, res := b.page("2026-11-26", "my-journal")
	if res.status != http.StatusOK || yesterday.Editable || yesterday.LocksAt != nil {
		t.Errorf("yesterday = %d editable %v locksAt %v, want 200, read-only, no lock time", res.status, yesterday.Editable, yesterday.LocksAt)
	}
	if !sameJSON(t, yesterday.Doc, helloDoc) {
		t.Errorf("yesterday's doc = %s, want what was written", yesterday.Doc)
	}
	if now, _ := b.page("2026-11-27", "my-journal"); !now.Editable {
		t.Error("the new today isn't editable")
	}
}

type savePage struct {
	Doc           json.RawMessage `json:"doc"`
	SchemaVersion int             `json:"schemaVersion"`
	BaseVersion   int             `json:"baseVersion"`
}

func (b *browser) save(day, notebook, doc string, baseVersion int) response {
	b.app.t.Helper()
	return b.do(http.MethodPut, "/api/v1/pages/"+day+"/"+notebook,
		savePage{Doc: json.RawMessage(doc), SchemaVersion: 1, BaseVersion: baseVersion})
}

const helloDoc = `{"type":"doc","content":[
	{"type":"paragraph","content":[{"type":"text","text":"A quiet Thursday"}]},
	{"type":"paragraph","content":[{"type":"text","text":"Rain all morning."}]}]}`

func TestASavedPageIsReadBackAtTheNextVersion(t *testing.T) {
	app := startApp(t, tokyoMorning)
	b := app.browser()
	b.signup(uniqueEmail("writer"), "Asia/Tokyo")

	res := b.save("2026-11-26", "my-journal", helloDoc, 0)
	if res.status != http.StatusOK {
		t.Fatalf("PUT = %d %s, want 200", res.status, res.body)
	}
	var saved pageBody
	res.json(t, &saved)
	if saved.Version != 1 || !sameJSON(t, saved.Doc, helloDoc) {
		t.Errorf("PUT returned version %d doc %s, want version 1 and the doc", saved.Version, saved.Doc)
	}

	got, _ := b.page("2026-11-26", "my-journal")
	if got.Version != 1 || !sameJSON(t, got.Doc, helloDoc) {
		t.Errorf("GET after save = version %d doc %s, want version 1 and the saved doc", got.Version, got.Doc)
	}
	if got.UpdatedAt == nil || *got.UpdatedAt != "2026-11-25T20:00:00Z" {
		t.Errorf("updatedAt = %v, want 2026-11-25T20:00:00Z", got.UpdatedAt)
	}

	app.clock.Advance(time.Minute)
	if res := b.save("2026-11-26", "my-journal", emptyDoc, 1); res.status != http.StatusOK {
		t.Fatalf("second PUT = %d %s, want 200", res.status, res.body)
	}
	if got, _ := b.page("2026-11-26", "my-journal"); got.Version != 2 || !sameJSON(t, got.Doc, emptyDoc) {
		t.Errorf("after second save = version %d doc %s, want version 2 and the emptied doc", got.Version, got.Doc)
	}
	if other, _ := b.page("2026-11-26", "i-learned"); other.Version != 0 {
		t.Errorf("I Learned version = %d, want 0: saving My Journal must not touch it", other.Version)
	}
}

func TestASaveWithADocOutsideTheSchemaIsRefused(t *testing.T) {
	app := startApp(t, tokyoMorning)
	b := app.browser()
	b.signup(uniqueEmail("bold"), "Asia/Tokyo")

	bold := `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Hi","marks":[{"type":"bold"}]}]}]}`
	res := b.save("2026-11-26", "my-journal", bold, 0)
	if res.status != http.StatusBadRequest || res.errorCode(t) != "validation" {
		t.Errorf("PUT with a bold mark = %d %s, want 400 validation", res.status, res.body)
	}
	if got, _ := b.page("2026-11-26", "my-journal"); got.Version != 0 {
		t.Errorf("version after refused save = %d, want 0", got.Version)
	}
}

func docOf(text string) string {
	b, _ := json.Marshal(text)
	return `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":` + string(b) + `}]}]}`
}

func TestPagesAreCappedAt100kCharactersAnd512KBBodies(t *testing.T) {
	app := startApp(t, tokyoMorning)
	b := app.browser()
	b.signup(uniqueEmail("long"), "Asia/Tokyo")

	// 100,000 characters is allowed even when each is 3 bytes (300 KB).
	if res := b.save("2026-11-26", "my-journal", docOf(strings.Repeat("日", 100_000)), 0); res.status != http.StatusOK {
		t.Fatalf("PUT of 100,000 characters = %d %s, want 200", res.status, res.body)
	}
	res := b.save("2026-11-26", "my-journal", docOf(strings.Repeat("a", 100_001)), 1)
	if res.status != http.StatusRequestEntityTooLarge || res.errorCode(t) != "too_long" {
		t.Errorf("PUT of 100,001 characters = %d %s, want 413 too_long", res.status, res.body)
	}

	// Under 100k characters of text, but over 512 KB on the wire.
	many := `{"type":"doc","content":[{"type":"paragraph"}` + strings.Repeat(`,{"type":"paragraph"}`, 30_000) + `]}`
	res = b.save("2026-11-26", "my-journal", many, 1)
	if res.status != http.StatusRequestEntityTooLarge || res.errorCode(t) != "too_long" {
		t.Errorf("PUT of a %d KB body = %d %s, want 413 too_long", len(many)>>10, res.status, res.body)
	}
	if got, _ := b.page("2026-11-26", "my-journal"); got.Version != 1 {
		t.Errorf("version = %d after refused saves, want 1", got.Version)
	}
}

func TestASaveFromAnOlderSchemaIsClientOutdated(t *testing.T) {
	app := startApp(t, tokyoMorning)
	b := app.browser()
	b.signup(uniqueEmail("old-tab"), "Asia/Tokyo")

	res := b.do(http.MethodPut, "/api/v1/pages/2026-11-26/my-journal",
		savePage{Doc: json.RawMessage(helloDoc), SchemaVersion: 0, BaseVersion: 0})
	if res.status != http.StatusConflict || res.errorCode(t) != "client_outdated" {
		t.Errorf("PUT with schemaVersion 0 = %d %s, want 409 client_outdated", res.status, res.body)
	}
	res = b.do(http.MethodPut, "/api/v1/pages/2026-11-26/my-journal",
		savePage{Doc: json.RawMessage(helloDoc), SchemaVersion: 2, BaseVersion: 0})
	if res.status != http.StatusBadRequest || res.errorCode(t) != "validation" {
		t.Errorf("PUT with a schemaVersion from the future = %d %s, want 400 validation", res.status, res.body)
	}
	if got, _ := b.page("2026-11-26", "my-journal"); got.Version != 0 {
		t.Errorf("version = %d after refused saves, want 0", got.Version)
	}
}

func TestASaveOverAStaleVersionIsAConflictCarryingTheCurrentPage(t *testing.T) {
	app := startApp(t, tokyoMorning)
	laptop := app.browser()
	email := uniqueEmail("two-devices")
	laptop.signup(email, "Asia/Tokyo")
	phone := app.browser()
	phone.login(email)

	if res := laptop.save("2026-11-26", "my-journal", helloDoc, 0); res.status != http.StatusOK {
		t.Fatalf("laptop PUT = %d %s", res.status, res.body)
	}
	// The phone opened the Page before the laptop saved, so it still thinks it's at version 0.
	res := phone.save("2026-11-26", "my-journal", emptyDoc, 0)
	if res.status != http.StatusConflict || res.errorCode(t) != "version_conflict" {
		t.Fatalf("stale PUT = %d %s, want 409 version_conflict", res.status, res.body)
	}
	var body struct {
		Page pageBody `json:"page"`
	}
	res.json(t, &body)
	if body.Page.Version != 1 || !sameJSON(t, body.Page.Doc, helloDoc) || !body.Page.Editable {
		t.Errorf("conflict carried %+v, want the laptop's version 1", body.Page)
	}

	if res := phone.save("2026-11-26", "my-journal", emptyDoc, 5); res.status != http.StatusConflict {
		t.Errorf("PUT from a version ahead of the server = %d, want 409", res.status)
	}
	if got, _ := laptop.page("2026-11-26", "my-journal"); got.Version != 1 || !sameJSON(t, got.Doc, helloDoc) {
		t.Errorf("after refused saves = version %d doc %s, want the laptop's version 1 untouched", got.Version, got.Doc)
	}
}
