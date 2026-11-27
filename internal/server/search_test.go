package server_test

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
)

type segment struct {
	Text  string `json:"text"`
	Match bool   `json:"match"`
}

type searchResult struct {
	Day      string    `json:"day"`
	Notebook string    `json:"notebook"`
	Title    string    `json:"title"`
	Snippet  []segment `json:"snippet"`
}

type searchPage struct {
	Results    []searchResult `json:"results"`
	NextCursor *string        `json:"nextCursor"`
}

func (b *browser) search(q string, cursor ...string) (searchPage, response) {
	b.app.t.Helper()
	v := url.Values{"q": {q}}
	if len(cursor) > 0 {
		v.Set("cursor", cursor[0])
	}
	res := b.do(http.MethodGet, "/api/v1/search?"+v.Encode(), nil)
	var p searchPage
	if res.status == http.StatusOK {
		res.json(b.app.t, &p)
	}
	return p, res
}

func TestSearchFindsAWordStartAndMarksTheWholeWord(t *testing.T) {
	app := startApp(t, tokyoMorning)
	b := app.browser()
	b.signup(uniqueEmail("search"), "Asia/Tokyo")
	b.save("2026-11-26", "my-journal", helloDoc, 0)

	got, res := b.search("rai")
	if res.status != http.StatusOK {
		t.Fatalf("GET /search?q=rai = %d %s, want 200", res.status, res.body)
	}
	if len(got.Results) != 1 {
		t.Fatalf("results = %+v, want the one Page", got.Results)
	}
	r := got.Results[0]
	if r.Day != "2026-11-26" || r.Notebook != "my-journal" || r.Title != "A quiet Thursday" {
		t.Errorf("result = %s/%s titled %q, want 2026-11-26/my-journal titled \"A quiet Thursday\"", r.Day, r.Notebook, r.Title)
	}
	want := []segment{{"A quiet Thursday\n", false}, {"Rain", true}, {" all morning.", false}}
	if !sameSegments(r.Snippet, want) {
		t.Errorf("snippet = %+v, want %+v", r.Snippet, want)
	}
	if got.NextCursor != nil {
		t.Errorf("nextCursor = %q, want null: that's every match", *got.NextCursor)
	}
}

func sameSegments(got, want []segment) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestEveryWordMustMatchTheStartOfAWord(t *testing.T) {
	app := startApp(t, tokyoMorning)
	b := app.browser()
	b.signup(uniqueEmail("search-and"), "Asia/Tokyo")
	b.save("2026-11-26", "my-journal", docOf("Baked bread with Mum"), 0)
	b.save("2026-11-26", "i-learned", docOf("Bread rises because of yeast"), 0)

	for q, want := range map[string][]string{
		"bread":         {"my-journal", "i-learned"},
		"BREAD mu":      {"my-journal"},
		"yeast, bread!": {"i-learned"},
		"um":            nil, // inside "Mum", not its start
		"read":          nil,
		"bread cake":    nil,
	} {
		got, res := b.search(q)
		if res.status != http.StatusOK {
			t.Fatalf("search %q = %d %s, want 200", q, res.status, res.body)
		}
		var notebooks []string
		for _, r := range got.Results {
			notebooks = append(notebooks, r.Notebook)
		}
		if !slices.Equal(notebooks, want) {
			t.Errorf("search %q found %v, want %v", q, notebooks, want)
		}
	}
}

func TestDayDateWordsMatchLikeText(t *testing.T) {
	app := startApp(t, tokyoSeptember)
	b := app.browser()
	email := uniqueEmail("search-dates")
	b.signup(email, "Asia/Tokyo")
	b.writeOn(email, "2026-10-15", "my-journal", "Lunch with Mum") // a Thursday
	b.writeOn(email, "2026-11-02", "my-journal", "Called Mum")     // a Monday

	for q, want := range map[string][]string{
		"nov":        {"2026-11-02"},
		"thursday":   {"2026-10-15"},
		"2026":       {"2026-11-02", "2026-10-15"},
		"nov mum":    {"2026-11-02"},
		"15 oct":     {"2026-10-15"},
		"2 nov":      {"2026-11-02"},
		"mon called": {"2026-11-02"},
		"sep":        nil,
	} {
		got, _ := b.search(q)
		var found []string
		for _, r := range got.Results {
			found = append(found, r.Day)
		}
		if !slices.Equal(found, want) {
			t.Errorf("search %q found %v, want %v", q, found, want)
		}
	}

	// Only the date matched, so the snippet is the start of the Page with nothing marked.
	got, _ := b.search("october")
	if len(got.Results) != 1 {
		t.Fatalf("search october = %+v, want one result", got.Results)
	}
	if want := []segment{{"Lunch with Mum", false}}; !sameSegments(got.Results[0].Snippet, want) {
		t.Errorf("date-only snippet = %+v, want %+v", got.Results[0].Snippet, want)
	}
	// A date word and a text word: only the text is marked.
	got, _ = b.search("nov mum")
	if want := []segment{{"Called ", false}, {"Mum", true}}; len(got.Results) != 1 || !sameSegments(got.Results[0].Snippet, want) {
		t.Errorf("nov mum = %+v, want the snippet %+v", got.Results, want)
	}
}

func TestResultsComeNewestDayFirstTwentyAtATimeByCursor(t *testing.T) {
	app := startApp(t, tokyoSeptember)
	b := app.browser()
	email := uniqueEmail("search-pages")
	b.signup(email, "Asia/Tokyo")
	var want []string // newest Day first, My Journal before I Learned
	for d := 1; d <= 13; d++ {
		day := fmt.Sprintf("2026-10-%02d", d)
		b.writeOn(email, day, "i-learned", "Green tea has less caffeine")
		b.writeOn(email, day, "my-journal", "Tea in the garden")
		want = append([]string{day + "/my-journal", day + "/i-learned"}, want...)
	}
	b.writeOn(email, "2026-10-14", "my-journal", "Coffee instead")

	first, res := b.search("tea")
	if res.status != http.StatusOK {
		t.Fatalf("search = %d %s", res.status, res.body)
	}
	if len(first.Results) != 20 || first.NextCursor == nil {
		t.Fatalf("first page = %d results, cursor %v, want 20 and a cursor", len(first.Results), first.NextCursor)
	}
	second, res := b.search("tea", *first.NextCursor)
	if res.status != http.StatusOK {
		t.Fatalf("second page = %d %s", res.status, res.body)
	}
	if second.NextCursor != nil {
		t.Errorf("second page cursor = %q, want null after the last match", *second.NextCursor)
	}
	var got []string
	for _, r := range append(first.Results, second.Results...) {
		got = append(got, r.Day+"/"+r.Notebook)
	}
	if !slices.Equal(got, want) {
		t.Errorf("results = %v, want %v", got, want)
	}

	if _, res := b.search("tea", "not-a-cursor"); res.status != http.StatusBadRequest || res.errorCode(t) != "validation" {
		t.Errorf("bad cursor = %d %s, want 400 validation", res.status, res.body)
	}
}

func TestSearchOnlySeesYourOwnPages(t *testing.T) {
	app := startApp(t, tokyoMorning)
	alice, bob := app.browser(), app.browser()
	alice.signup(uniqueEmail("search-alice"), "Asia/Tokyo")
	bob.signup(uniqueEmail("search-bob"), "Asia/Tokyo")
	alice.save("2026-11-26", "my-journal", docOf("Secret marmalade recipe"), 0)

	if got, _ := bob.search("marmalade"); len(got.Results) != 0 {
		t.Errorf("Bob's search found %+v, want nothing of Alice's", got.Results)
	}
	if got, _ := alice.search("marmalade"); len(got.Results) != 1 {
		t.Errorf("Alice's search found %d results, want her Page", len(got.Results))
	}
	if _, res := app.browser().search("marmalade"); res.status != http.StatusUnauthorized {
		t.Errorf("signed-out search = %d, want 401", res.status)
	}
}

func TestQuerySyntaxIsTakenAsPlainText(t *testing.T) {
	app := startApp(t, tokyoMorning)
	b := app.browser()
	b.signup(uniqueEmail("search-syntax"), "Asia/Tokyo")
	b.save("2026-11-26", "my-journal", docOf("It's a café: tea & cake!"), 0)

	for q, want := range map[string]int{
		"it's":         1,
		"café":         1,
		"tea & cake":   1,
		"tea | coffee": 0, // | is not OR
		"!tea":         1, // ! is not NOT
		"'":            0,
		"&:*()<->":     0,
		"":             0,
		"   ":          0,
	} {
		got, res := b.search(q)
		if res.status != http.StatusOK {
			t.Errorf("search %q = %d %s, want 200", q, res.status, res.body)
			continue
		}
		if got.Results == nil || len(got.Results) != want {
			t.Errorf("search %q = %v, want %d results (an empty list, never null)", q, got.Results, want)
		}
	}
}

func TestALongPagesSnippetIsAWindowAroundTheMatch(t *testing.T) {
	app := startApp(t, tokyoMorning)
	b := app.browser()
	b.signup(uniqueEmail("search-long"), "Asia/Tokyo")
	text := strings.Repeat("nothing much happened today ", 100) + "then a heron landed " + strings.Repeat("and left ", 100)
	b.save("2026-11-26", "my-journal", docOf(text), 0)

	got, _ := b.search("heron")
	if len(got.Results) != 1 {
		t.Fatalf("results = %d, want 1", len(got.Results))
	}
	var snippet string
	var marked []string
	for _, s := range got.Results[0].Snippet {
		snippet += s.Text
		if s.Match {
			marked = append(marked, s.Text)
		}
	}
	if !slices.Equal(marked, []string{"heron"}) {
		t.Errorf("marked = %v, want [heron]", marked)
	}
	if n := len(strings.Fields(snippet)); n > 40 {
		t.Errorf("snippet is %d words, want a window of at most 40: %q", n, snippet)
	}
}
