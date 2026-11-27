package server_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

type railDay struct {
	Day      string            `json:"day"`
	Written  bool              `json:"written"`
	Title    string            `json:"title"`
	Glimpses map[string]string `json:"glimpses"`
}

type railMonth struct {
	Month      string    `json:"month"`
	Days       []railDay `json:"days"`
	NewerMonth *string   `json:"newerMonth"`
	OlderMonth *string   `json:"olderMonth"`
}

func (b *browser) month(month string) (railMonth, response) {
	b.app.t.Helper()
	res := b.do(http.MethodGet, "/api/v1/days?month="+month, nil)
	var m railMonth
	if res.status == http.StatusOK {
		res.json(b.app.t, &m)
	}
	return m, res
}

func TestANewUsersRailIsJustTodayWithItsTitleAndGlimpses(t *testing.T) {
	app := startApp(t, tokyoMorning) // 05:00 on 26 Nov in Tokyo
	b := app.browser()
	b.signup(uniqueEmail("rail"), "Asia/Tokyo")
	b.save("2026-11-26", "my-journal", helloDoc, 0)

	got, res := b.month("2026-11")
	if res.status != http.StatusOK {
		t.Fatalf("GET /days?month=2026-11 = %d %s, want 200", res.status, res.body)
	}
	if got.Month != "2026-11" || got.NewerMonth != nil || got.OlderMonth != nil {
		t.Errorf("month %q newer %v older %v, want 2026-11 with no cursors", got.Month, got.NewerMonth, got.OlderMonth)
	}
	if len(got.Days) != 1 {
		t.Fatalf("days = %+v, want only today (signup was today)", got.Days)
	}
	today := got.Days[0]
	if today.Day != "2026-11-26" || !today.Written || today.Title != "A quiet Thursday" {
		t.Errorf("today = %+v, want 2026-11-26, written, titled \"A quiet Thursday\"", today)
	}
	if g := today.Glimpses["my-journal"]; g != "A quiet Thursday · Rain all morning." {
		t.Errorf("My Journal glimpse = %q, want the Page's lines joined with \" · \"", g)
	}
	if g, ok := today.Glimpses["i-learned"]; !ok || g != "" {
		t.Errorf("I Learned glimpse = %q (present %v), want an empty string for an unwritten Page", g, ok)
	}
}

// 03:00 on 29 Sep 2026 in Tokyo.
var tokyoSeptember = time.Date(2026, 9, 28, 18, 0, 0, 0, time.UTC)

// writeOn moves the clock forward to 12:00 on day in Tokyo, signs in again in
// case the session lapsed, and writes text as day's Page in notebook.
func (b *browser) writeOn(email, day, notebook, text string) {
	b.app.t.Helper()
	noon, err := time.Parse(time.RFC3339, day+"T12:00:00+09:00")
	if err != nil {
		b.app.t.Fatal(err)
	}
	b.app.clock.Advance(noon.Sub(b.app.clock.Now(b.app.t.Context())))
	b.login(email)
	if res := b.save(day, notebook, docOf(text), 0); res.status != http.StatusOK {
		b.app.t.Fatalf("writing %s on %s = %d %s", notebook, day, res.status, res.body)
	}
}

func days(ms []railMonth) (out []string) {
	for _, m := range ms {
		for _, d := range m.Days {
			out = append(out, d.Day)
		}
	}
	return out
}

func TestTheRailPagesByMonthFromAnyAnchorInBothDirections(t *testing.T) {
	app := startApp(t, tokyoSeptember)
	b := app.browser()
	email := uniqueEmail("history")
	b.signup(email, "Asia/Tokyo")
	b.writeOn(email, "2026-09-29", "my-journal", "First Day")
	b.writeOn(email, "2026-10-15", "i-learned", "Octopuses taste with their arms.")
	b.writeOn(email, "2026-11-26", "my-journal", "Today")

	// Jump to October, then walk to both ends by the cursors.
	oct, res := b.month("2026-10")
	if res.status != http.StatusOK {
		t.Fatalf("GET October = %d %s", res.status, res.body)
	}
	if len(oct.Days) != 31 || oct.Days[0].Day != "2026-10-31" || oct.Days[30].Day != "2026-10-01" {
		t.Errorf("October = %d Days from %s, want all 31, newest first", len(oct.Days), oct.Days[0].Day)
	}
	if oct.NewerMonth == nil || *oct.NewerMonth != "2026-11" || oct.OlderMonth == nil || *oct.OlderMonth != "2026-09" {
		t.Errorf("October cursors = newer %v older %v, want 2026-11 and 2026-09", oct.NewerMonth, oct.OlderMonth)
	}
	learned := oct.Days[31-15]
	if learned.Day != "2026-10-15" || !learned.Written || learned.Title != "" || learned.Glimpses["i-learned"] != "Octopuses taste with their arms." {
		t.Errorf("15 Oct = %+v, want written with no Title (only I Learned has text) and its glimpse", learned)
	}
	if blank := oct.Days[0]; blank.Written || blank.Title != "" || blank.Glimpses["my-journal"] != "" {
		t.Errorf("31 Oct = %+v, want an unwritten Day with nothing to glimpse", blank)
	}

	nov, _ := b.month(*oct.NewerMonth)
	if len(nov.Days) != 26 || nov.Days[0].Day != "2026-11-26" || nov.NewerMonth != nil {
		t.Errorf("November = %d Days from %v, newer %v, want 26 Days up to today and no newer month", len(nov.Days), days([]railMonth{nov}), nov.NewerMonth)
	}
	sep, _ := b.month(*oct.OlderMonth)
	if got := days([]railMonth{sep}); len(got) != 2 || got[0] != "2026-09-30" || got[1] != "2026-09-29" || sep.OlderMonth != nil {
		t.Errorf("September = %v older %v, want 30 and 29 Sep (signup) and no older month", got, sep.OlderMonth)
	}
	if sep.Days[1].Title != "First Day" {
		t.Errorf("29 Sep Title = %q, want \"First Day\"", sep.Days[1].Title)
	}

	for _, month := range []string{"2026-08", "2026-12", "2026-13", "September", ""} {
		if _, res := b.month(month); res.status != http.StatusNotFound || res.errorCode(t) != "not_found" {
			t.Errorf("GET /days?month=%s = %d %s, want 404 not_found (before the first Day, the future, not a month)", month, res.status, res.body)
		}
	}

	// Another user never sees these Days.
	other := app.browser()
	other.signup(uniqueEmail("other"), "Asia/Tokyo")
	if mine, _ := other.month("2026-11"); len(mine.Days) != 1 || mine.Days[0].Written {
		t.Errorf("a new user's November = %+v, want only their unwritten today", mine.Days)
	}
}

func TestGlimpsesAreCutAtAbout80CharactersAndBlankLinesDontCount(t *testing.T) {
	app := startApp(t, tokyoMorning)
	b := app.browser()
	b.signup(uniqueEmail("glimpse"), "Asia/Tokyo")

	long := `{"type":"doc","content":[
		{"type":"paragraph","content":[{"type":"text","text":"Swim"}]},
		{"type":"paragraph"},
		{"type":"paragraph","content":[{"type":"text","text":"Forty lengths. The lane next to me had a man doing butterfly very slowly, like a sad dolphin."}]}]}`
	b.save("2026-11-26", "my-journal", long, 0)
	// Three empty lines: the Page was emptied, so it isn't writing.
	b.save("2026-11-26", "i-learned", `{"type":"doc","content":[{"type":"paragraph"},{"type":"paragraph"},{"type":"paragraph"}]}`, 0)

	got, _ := b.month("2026-11")
	today := got.Days[0]
	want := "Swim · Forty lengths. The lane next to me had a man doing butterfly very slowly,…"
	if g := today.Glimpses["my-journal"]; g != want {
		t.Errorf("glimpse = %q, want %q (blank lines skipped, cut at 80 characters)", g, want)
	}
	if today.Title != "Swim" {
		t.Errorf("Title = %q, want \"Swim\"", today.Title)
	}
	if g := today.Glimpses["i-learned"]; g != "" {
		t.Errorf("I Learned glimpse = %q, want none for a Page of empty lines", g)
	}

	b.save("2026-11-26", "my-journal", `{"type":"doc","content":[{"type":"paragraph"},{"type":"paragraph"}]}`, 1)
	if got, _ := b.month("2026-11"); got.Days[0].Written {
		t.Errorf("today = %+v after both Pages were emptied, want an unwritten Day", got.Days[0])
	}
}

type railSummary struct {
	FirstDay string `json:"firstDay"`
	Today    string `json:"today"`
	Months   []struct {
		Month       string `json:"month"`
		WrittenDays int    `json:"writtenDays"`
	} `json:"months"`
}

func TestTheSummaryCountsWrittenDaysInEveryMonthOfTheHistory(t *testing.T) {
	app := startApp(t, tokyoSeptember)
	b := app.browser()
	email := uniqueEmail("summary")
	b.signup(email, "Asia/Tokyo")
	b.writeOn(email, "2026-09-29", "my-journal", "First Day")
	b.writeOn(email, "2026-09-30", "my-journal", "Second")
	b.writeOn(email, "2026-09-30", "i-learned", "Both Notebooks still make one Day")
	b.writeOn(email, "2026-11-25", "i-learned", "Leeks need washing twice")
	b.writeOn(email, "2026-11-26", "my-journal", "Today")

	res := b.do(http.MethodGet, "/api/v1/days/summary", nil)
	if res.status != http.StatusOK {
		t.Fatalf("GET /days/summary = %d %s, want 200", res.status, res.body)
	}
	var got railSummary
	res.json(t, &got)
	if got.FirstDay != "2026-09-29" || got.Today != "2026-11-26" {
		t.Errorf("firstDay %s today %s, want 2026-09-29 and 2026-11-26", got.FirstDay, got.Today)
	}
	want := "2026-11:2 2026-10:0 2026-09:2"
	var have []string
	for _, m := range got.Months {
		have = append(have, fmt.Sprintf("%s:%d", m.Month, m.WrittenDays))
	}
	if strings.Join(have, " ") != want {
		t.Errorf("months = %v, want %s (every month, newest first, Days not Pages)", have, want)
	}
}
