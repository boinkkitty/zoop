package pages

import (
	"context"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
)

// Island is one Day in the rail: whether it's a Written Day, its Title, and a
// glimpse of each Notebook's Page. All of it is derived from the Pages when asked, never stored.
type Island struct {
	Day      time.Time
	Written  bool
	Title    string
	Glimpses map[string]string // by Notebook slug; "" for an unwritten Page
}

// Month is one calendar month of a user's Islands, newest first.
type Month struct {
	Month   time.Time // the 1st of the month
	Islands []Island
	Newer   *time.Time // the next month with Days, if any
	Older   *time.Time // the previous month with Days, if any
}

// Month returns userID's Islands for the calendar month starting at month, within their first Day..today,
// or ErrNoPage if that month has none of their Days.
func (s *Service) Month(ctx context.Context, userID uuid.UUID, month time.Time) (Month, error) {
	w, err := s.writer(ctx, userID)
	if err != nil {
		return Month{}, err
	}
	month = startOfMonth(month)
	oldest, newest := startOfMonth(w.FirstDay), startOfMonth(w.today)
	if month.Before(oldest) || month.After(newest) {
		return Month{}, ErrNoPage
	}
	first, last := month, month.AddDate(0, 1, -1)
	if first.Before(w.FirstDay) {
		first = w.FirstDay
	}
	if last.After(w.today) {
		last = w.today
	}
	texts, err := s.store.PageTexts(ctx, userID, first, last)
	if err != nil {
		return Month{}, err
	}
	byDay := map[time.Time]map[string]string{}
	for _, t := range texts {
		if byDay[t.Day] == nil {
			byDay[t.Day] = map[string]string{}
		}
		byDay[t.Day][t.Notebook] = t.Head
	}
	m := Month{Month: month}
	if month.Before(newest) {
		next := month.AddDate(0, 1, 0)
		m.Newer = &next
	}
	if month.After(oldest) {
		prev := month.AddDate(0, -1, 0)
		m.Older = &prev
	}
	for day := last; !day.Before(first); day = day.AddDate(0, 0, -1) {
		heads := byDay[day]
		isl := Island{Day: day, Written: len(heads) > 0, Glimpses: map[string]string{}}
		for _, n := range Notebooks {
			isl.Glimpses[n] = glimpse(heads[n])
		}
		isl.Title, _, _ = strings.Cut(heads["my-journal"], "\n")
		m.Islands = append(m.Islands, isl)
	}
	return m, nil
}

func startOfMonth(day time.Time) time.Time {
	return time.Date(day.Year(), day.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// glimpseChars is about how much of a Page its glimpse shows.
const glimpseChars = 80

// glimpse is the start of a Page's text on one line: its non-blank lines joined
// with " · ", cut at glimpseChars characters with "…".
func glimpse(text string) string {
	var lines []string
	for line := range strings.SplitSeq(text, "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	g := []rune(strings.Join(lines, " · "))
	if len(g) <= glimpseChars {
		return string(g)
	}
	return strings.TrimRightFunc(string(g[:glimpseChars]), unicode.IsSpace) + "…"
}

// Summary is a user's whole history at a glance: what the rail's scrubber draws.
type Summary struct {
	FirstDay time.Time
	Today    time.Time
	Months   []MonthCount // every month from today's back to the first Day's, newest first
}

// MonthCount is how many Written Days a month holds.
type MonthCount struct {
	Month       time.Time // the 1st of the month
	WrittenDays int
}

// Summary counts userID's Written Days in each month of their history.
func (s *Service) Summary(ctx context.Context, userID uuid.UUID) (Summary, error) {
	w, err := s.writer(ctx, userID)
	if err != nil {
		return Summary{}, err
	}
	counts, err := s.store.WrittenDaysByMonth(ctx, userID)
	if err != nil {
		return Summary{}, err
	}
	sum := Summary{FirstDay: w.FirstDay, Today: w.today}
	for m := startOfMonth(w.today); !m.Before(startOfMonth(w.FirstDay)); m = m.AddDate(0, -1, 0) {
		sum.Months = append(sum.Months, MonthCount{Month: m, WrittenDays: counts[m]})
	}
	return sum, nil
}
