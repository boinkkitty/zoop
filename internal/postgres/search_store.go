package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/boinkkitty/zoop/internal/db/dbgen"
	"github.com/boinkkitty/zoop/internal/search"
)

// SearchStore implements search.Store.
type SearchStore struct{ q *dbgen.Queries }

var _ search.Store = (*SearchStore)(nil)

func NewSearchStore(pool *pgxpool.Pool) *SearchStore { return &SearchStore{q: dbgen.New(pool)} }

// headlineOptions marks matches with search's markers and keeps snippets to about three lines.
const headlineOptions = "StartSel=" + search.MatchStart + ", StopSel=" + search.MatchEnd + ", MinWords=20, MaxWords=40"

func (s *SearchStore) Search(ctx context.Context, userID uuid.UUID, query string, after *search.Cursor, limit int) ([]search.Hit, error) {
	p := dbgen.SearchPagesParams{Query: query, Options: headlineOptions, UserID: userID, MaxResults: int32(limit)}
	if after != nil {
		pos := int16(after.Position)
		p.AfterDay, p.AfterPosition = &after.Day, &pos
	}
	rows, err := s.q.SearchPages(ctx, p)
	if err != nil {
		return nil, err
	}
	hits := make([]search.Hit, len(rows))
	for i, r := range rows {
		hits[i] = search.Hit{
			Cursor:   search.Cursor{Day: r.Day, Position: int(r.Position)},
			Notebook: r.Notebook, Head: r.Head, Headline: r.Headline,
		}
	}
	return hits, nil
}
