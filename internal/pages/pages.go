// Package pages reads and writes Pages: one Notebook's writing for one Day.
package pages

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/boinkkitty/zoop/internal/clock"
	"github.com/boinkkitty/zoop/internal/days"
)

// Content is a Page's document as stored (TipTap JSON, ADR 0001). It's the
// most private thing Zoop holds, so it never appears in logs.
type Content string

func (Content) LogValue() slog.Value { return slog.StringValue("[redacted]") }

// Notebooks are the v1 Notebooks, by slug.
var Notebooks = []string{"my-journal", "i-learned"}

// MaxChars is the most plain-text characters one Page may hold.
const MaxChars = 100_000

var (
	// ErrNoPage means there's no such Page for this user: an unknown Notebook, or a Day outside their journal.
	ErrNoPage = errors.New("pages: no such page")
	// ErrTooLong means the Page holds more than MaxChars characters.
	ErrTooLong = errors.New("pages: page too long")
	// ErrLocked means the Page's Day is read only for this user now.
	ErrLocked = errors.New("pages: page is locked")
	// ErrClientOutdated means the save came from an editor on an older SchemaVersion.
	ErrClientOutdated = errors.New("pages: client schema is outdated")
)

// ConflictError means the save was based on an older version than the stored one.
type ConflictError struct{ Current Page }

func (e *ConflictError) Error() string {
	return fmt.Sprintf("pages: page is at version %d", e.Current.Version)
}

// Service is the Pages module. It persists through its Store and tells time by its Clock.
type Service struct {
	store Store
	clock clock.Clock
}

func New(store Store, c clock.Clock) *Service {
	return &Service{store: store, clock: c}
}

// Page is a Page as its writer sees it now.
type Page struct {
	Day           time.Time
	Notebook      string
	Doc           json.RawMessage
	SchemaVersion int
	Version       int // 0 until the first save
	UpdatedAt     *time.Time
	Editable      bool
	LocksAt       *time.Time // when it stops being editable, if it will
}

// Get returns userID's Page for day in notebook. An unwritten Page is an empty doc at version 0.
func (s *Service) Get(ctx context.Context, userID uuid.UUID, day time.Time, notebook string) (Page, error) {
	w, err := s.writer(ctx, userID)
	if err != nil {
		return Page{}, err
	}
	if !isNotebook(notebook) || day.Before(w.FirstDay) || day.After(w.today) {
		return Page{}, ErrNoPage
	}
	p := Page{Day: day, Notebook: notebook, Doc: EmptyDoc, SchemaVersion: SchemaVersion}
	stored, err := s.store.Page(ctx, userID, notebook, day)
	switch {
	case errors.Is(err, ErrNotFound):
	case err != nil:
		return Page{}, err
	default:
		p.Doc, p.SchemaVersion, p.Version, p.UpdatedAt = stored.Doc, stored.SchemaVersion, stored.Version, &stored.UpdatedAt
	}
	p.Editable, p.LocksAt = w.access(day)
	return p, nil
}

// Save is a whole Page document sent by the editor, written over baseVersion.
type Save struct {
	Doc           json.RawMessage
	SchemaVersion int
	BaseVersion   int
}

// Save stores a new version of userID's Page and returns it.
func (s *Service) Save(ctx context.Context, userID uuid.UUID, day time.Time, notebook string, in Save) (Page, error) {
	w, err := s.writer(ctx, userID)
	if err != nil {
		return Page{}, err
	}
	if !isNotebook(notebook) || day.Before(w.FirstDay) || day.After(w.today) {
		return Page{}, ErrNoPage
	}
	if !w.writable(day) {
		return Page{}, ErrLocked
	}
	if in.SchemaVersion < SchemaVersion {
		return Page{}, ErrClientOutdated
	}
	if in.SchemaVersion > SchemaVersion {
		return Page{}, invalid("schemaVersion %d is newer than this server's %d", in.SchemaVersion, SchemaVersion)
	}
	plain, err := ParseDoc(notebook, in.Doc)
	if err != nil {
		return Page{}, err
	}
	if utf8.RuneCountInString(plain) > MaxChars {
		return Page{}, ErrTooLong
	}
	if strings.TrimSpace(plain) == "" {
		plain = "" // an emptied Page: only blank lines don't make a Written Day
	}
	stored, err := s.store.SavePage(ctx, userID, notebook, day, in.BaseVersion, NewPage{
		Doc: in.Doc, SchemaVersion: in.SchemaVersion, PlainText: plain, Now: w.now,
	})
	if errors.Is(err, ErrStale) {
		current, err := s.Get(ctx, userID, day, notebook)
		if err != nil {
			return Page{}, err
		}
		return Page{}, &ConflictError{Current: current}
	}
	if err != nil {
		return Page{}, err
	}
	p := Page{
		Day: day, Notebook: notebook,
		Doc: stored.Doc, SchemaVersion: stored.SchemaVersion, Version: stored.Version, UpdatedAt: &stored.UpdatedAt,
	}
	p.Editable, p.LocksAt = w.access(day)
	return p, nil
}

type writer struct {
	Writer
	cal   days.Calendar
	now   time.Time
	today time.Time
}

func (s *Service) writer(ctx context.Context, userID uuid.UUID) (writer, error) {
	w, err := s.store.Writer(ctx, userID)
	if err != nil {
		return writer{}, err
	}
	cal, err := days.LoadCalendar(w.Timezone, w.NextTimezone, w.SwitchDay)
	if err != nil {
		return writer{}, err
	}
	now := s.clock.Now(ctx)
	return writer{Writer: w, cal: cal, now: now, today: cal.Today(now)}, nil
}

// access says whether day is editable now, and when it stops being.
func (w writer) access(day time.Time) (bool, *time.Time) { return w.accessAt(day, w.now) }

func (w writer) accessAt(day, now time.Time) (bool, *time.Time) {
	return days.Access(days.Writer{Premium: w.Plan == "premium", FirstDay: w.FirstDay, Calendar: w.cal}, day, now)
}

// writable says whether a save of day arriving now counts: a save the editor sent
// as the Day locked may arrive up to days.SaveLeeway late.
func (w writer) writable(day time.Time) bool {
	ok, _ := w.accessAt(day, w.now.Add(-days.SaveLeeway))
	return ok
}

func isNotebook(slug string) bool {
	for _, n := range Notebooks {
		if n == slug {
			return true
		}
	}
	return false
}
