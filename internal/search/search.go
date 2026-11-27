// Package search finds a user's Pages by word starts and date words, newest Day first.
package search

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
)

// PageSize is how many results one search request returns.
const PageSize = 20

// Match markers wrap each match in a Hit's Headline. They are private-use
// characters, so the snippet travels as plain text and is never parsed as HTML.
const (
	MatchStart = ""
	MatchEnd   = ""
)

// Store is the port search needs from persistence. internal/postgres implements it.
type Store interface {
	// Search returns up to limit of userID's Pages matching query (a to_tsquery
	// string), newest Day first then by Notebook, strictly after the cursor if any.
	Search(ctx context.Context, userID uuid.UUID, query string, after *Cursor, limit int) ([]Hit, error)
}

// Cursor is the position of the last result on a page.
type Cursor struct {
	Day      time.Time
	Position int // the Notebook's position on the side divider
}

// Hit is one matching Page as the Store finds it.
type Hit struct {
	Cursor
	Notebook string
	Head     string // the start of the Page's plain text
	Headline string // a snippet with matches wrapped in MatchStart … MatchEnd
}

// Segment is a run of snippet text, marked if it matched.
type Segment struct {
	Text  string
	Match bool
}

// Result is one matching Page.
type Result struct {
	Day      time.Time
	Notebook string
	Title    string // empty for I Learned
	Snippet  []Segment
}

// Results is one page of results.
type Results struct {
	Results []Result
	Next    string // the cursor for the next page; "" when that's every match
}

// ErrBadCursor is returned for a cursor this package didn't make.
var ErrBadCursor = errors.New("search: bad cursor")

// String encodes c as "2026-11-26.1": the Day, then the Notebook's position.
func (c Cursor) String() string {
	return c.Day.Format(time.DateOnly) + "." + strconv.Itoa(c.Position)
}

func parseCursor(s string) (*Cursor, error) {
	day, pos, ok := strings.Cut(s, ".")
	d, err := time.Parse(time.DateOnly, day)
	if !ok || err != nil {
		return nil, ErrBadCursor
	}
	p, err := strconv.Atoi(pos)
	if err != nil || p < 0 {
		return nil, ErrBadCursor
	}
	return &Cursor{Day: d, Position: p}, nil
}

type Service struct{ store Store }

func New(store Store) *Service { return &Service{store: store} }

// Search returns a page of userID's Pages matching every word start in q, after cursor
// ("" for the first page), or ErrBadCursor.
func (s *Service) Search(ctx context.Context, userID uuid.UUID, q, cursor string) (Results, error) {
	var after *Cursor
	if cursor != "" {
		var err error
		if after, err = parseCursor(cursor); err != nil {
			return Results{}, err
		}
	}
	query := prefixQuery(q)
	if query == "" {
		return Results{}, nil
	}
	hits, err := s.store.Search(ctx, userID, query, after, PageSize+1)
	if err != nil {
		return Results{}, err
	}
	var out Results
	if len(hits) > PageSize {
		hits = hits[:PageSize]
		out.Next = hits[PageSize-1].Cursor.String()
	}
	out.Results = make([]Result, len(hits))
	for i, h := range hits {
		r := Result{Day: h.Day, Notebook: h.Notebook, Snippet: segments(h.Headline)}
		if h.Notebook == "my-journal" {
			r.Title, _, _ = strings.Cut(h.Head, "\n")
		}
		out.Results[i] = r
	}
	return out, nil
}

// prefixQuery turns what the user typed into a to_tsquery string: every word quoted, made a
// prefix and joined with &, so "nov mum" finds November Pages about Mum. "" if q has no words.
func prefixQuery(q string) string {
	words := Words(q)
	for i, w := range words {
		words[i] = "'" + w + "':*"
	}
	return strings.Join(words, " & ")
}

// Words splits q into lowercase runs of letters and numbers, as web/src/search/match.ts does.
func Words(q string) []string {
	return strings.FieldsFunc(strings.ToLower(q), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
}

// segments splits a headline on its match markers.
func segments(headline string) []Segment {
	var out []Segment
	for headline != "" {
		before, rest, found := strings.Cut(headline, MatchStart)
		if before != "" {
			out = append(out, Segment{Text: before})
		}
		if !found {
			break
		}
		match, after, _ := strings.Cut(rest, MatchEnd)
		if match != "" {
			out = append(out, Segment{Text: match, Match: true})
		}
		headline = after
	}
	return out
}
