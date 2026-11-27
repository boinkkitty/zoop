package pages

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Store is the port Pages needs from persistence. internal/postgres implements it.
type Store interface {
	// Writer describes a live account's access to their journal, or returns ErrNotFound.
	Writer(ctx context.Context, userID uuid.UUID) (Writer, error)

	// Page finds the stored Page, or returns ErrNotFound if it was never saved.
	Page(ctx context.Context, userID uuid.UUID, notebook string, day time.Time) (StoredPage, error)

	// SavePage stores p as the next version if the Page is still at baseVersion (0: never saved),
	// or returns ErrStale.
	SavePage(ctx context.Context, userID uuid.UUID, notebook string, day time.Time, baseVersion int, p NewPage) (StoredPage, error)

	// PageTexts returns the start of every written Page from first to last (inclusive), newest first.
	PageTexts(ctx context.Context, userID uuid.UUID, first, last time.Time) ([]PageText, error)

	// WrittenDaysByMonth counts the Written Days in each month that has any, keyed by the month's 1st.
	WrittenDaysByMonth(ctx context.Context, userID uuid.UUID) (map[time.Time]int, error)
}

// PageText is the start of a written Page's plain text: enough for its Title and glimpse.
type PageText struct {
	Day      time.Time
	Notebook string
	Head     string
}

var (
	// ErrNotFound is how a Store says there's no such row.
	ErrNotFound = errors.New("pages: not found")
	// ErrStale is how a Store says the Page has moved past the base version.
	ErrStale = errors.New("pages: stale base version")
)

// NewPage is the content of a save.
type NewPage struct {
	Doc           json.RawMessage
	SchemaVersion int
	PlainText     string
	Now           time.Time
}

// Writer is what decides which Days a user may write.
type Writer struct {
	Plan         string
	FirstDay     time.Time
	Timezone     string
	NextTimezone *string    // a pending switch to this zone
	SwitchDay    *time.Time // from this Day
}

type StoredPage struct {
	Doc           json.RawMessage
	SchemaVersion int
	Version       int
	UpdatedAt     time.Time
}
