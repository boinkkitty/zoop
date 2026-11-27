package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/boinkkitty/zoop/internal/db/dbgen"
	"github.com/boinkkitty/zoop/internal/pages"
)

// PageStore implements pages.Store.
type PageStore struct {
	pool *pgxpool.Pool
	q    *dbgen.Queries
}

var _ pages.Store = (*PageStore)(nil)

func NewPageStore(pool *pgxpool.Pool) *PageStore {
	return &PageStore{pool: pool, q: dbgen.New(pool)}
}

func (s *PageStore) Writer(ctx context.Context, userID uuid.UUID) (pages.Writer, error) {
	w, err := s.q.GetWriter(ctx, userID)
	if err != nil {
		return pages.Writer{}, pageNotFound(err)
	}
	return pages.Writer{
		Plan: w.Plan, FirstDay: w.FirstDay,
		Timezone: w.Timezone, NextTimezone: w.NextTimezone, SwitchDay: w.TimezoneSwitchDay,
	}, nil
}

func (s *PageStore) Page(ctx context.Context, userID uuid.UUID, notebook string, day time.Time) (pages.StoredPage, error) {
	p, err := s.q.GetPage(ctx, dbgen.GetPageParams{UserID: userID, Notebook: notebook, Day: day})
	if err != nil {
		return pages.StoredPage{}, pageNotFound(err)
	}
	return pages.StoredPage{
		Doc: p.Doc, SchemaVersion: int(p.SchemaVersion), Version: int(p.Version), UpdatedAt: p.UpdatedAt,
	}, nil
}

func (s *PageStore) SavePage(ctx context.Context, userID uuid.UUID, notebook string, day time.Time, baseVersion int, p pages.NewPage) (pages.StoredPage, error) {
	var (
		row dbgen.UpdatePageRow
		err error
	)
	if baseVersion == 0 {
		var ins dbgen.InsertPageRow
		ins, err = s.q.InsertPage(ctx, dbgen.InsertPageParams{
			UserID: userID, Notebook: notebook, Day: day,
			Doc: p.Doc, SchemaVersion: int16(p.SchemaVersion), PlainText: p.PlainText, Now: p.Now,
		})
		row = dbgen.UpdatePageRow(ins)
	} else {
		row, err = s.q.UpdatePage(ctx, dbgen.UpdatePageParams{
			UserID: userID, Notebook: notebook, Day: day, BaseVersion: int32(baseVersion),
			Doc: p.Doc, SchemaVersion: int16(p.SchemaVersion), PlainText: p.PlainText, Now: p.Now,
		})
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return pages.StoredPage{}, pages.ErrStale
	}
	if err != nil {
		return pages.StoredPage{}, err
	}
	return pages.StoredPage{
		Doc: row.Doc, SchemaVersion: int(row.SchemaVersion), Version: int(row.Version), UpdatedAt: row.UpdatedAt,
	}, nil
}

func pageNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return pages.ErrNotFound
	}
	return err
}

func (s *PageStore) PageTexts(ctx context.Context, userID uuid.UUID, first, last time.Time) ([]pages.PageText, error) {
	rows, err := s.q.ListPageTexts(ctx, dbgen.ListPageTextsParams{UserID: userID, FromDay: first, ToDay: last})
	if err != nil {
		return nil, err
	}
	texts := make([]pages.PageText, len(rows))
	for i, r := range rows {
		texts[i] = pages.PageText{Day: r.Day, Notebook: r.Notebook, Head: r.Head}
	}
	return texts, nil
}

func (s *PageStore) WrittenDaysByMonth(ctx context.Context, userID uuid.UUID) (map[time.Time]int, error) {
	rows, err := s.q.CountWrittenDaysByMonth(ctx, userID)
	if err != nil {
		return nil, err
	}
	counts := make(map[time.Time]int, len(rows))
	for _, r := range rows {
		counts[r.Month] = int(r.WrittenDays)
	}
	return counts, nil
}
