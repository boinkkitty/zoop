// Package days turns a moment and a user's timezone into Days.
//
// A Day is a calendar date, represented as a time.Time at midnight UTC.
package days

import "time"

// GraceHours is how long after midnight the previous Day can still be written.
const GraceHours = 2

// SaveLeeway is how late a save may arrive after its Day locked and still count:
// the editor flushes at the lock, and the request takes time to land.
const SaveLeeway = 60 * time.Second

// Today is the Day it is at now in loc.
func Today(now time.Time, loc *time.Location) time.Time {
	y, m, d := now.In(loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// GraceUntil is when yesterday stops being writable, if now is inside the Grace Window; otherwise nil.
func GraceUntil(now time.Time, loc *time.Location) *time.Time {
	local := now.In(loc)
	y, m, d := local.Date()
	end := time.Date(y, m, d, GraceHours, 0, 0, 0, loc)
	if !local.Before(end) {
		return nil
	}
	end = end.UTC()
	return &end
}

// Format writes a Day as YYYY-MM-DD.
func Format(day time.Time) string { return day.Format(time.DateOnly) }

// Writer is who is asking to write: their plan, first Day and timezone.
type Writer struct {
	Premium  bool
	FirstDay time.Time
	Loc      *time.Location
}

// Access says whether w may write day at now, and when that ends (nil: it won't, or it already has).
// Nobody writes the future or before their first Day. Premium Users write every other Day.
// Free Users write today, and yesterday until the Grace Window ends.
func Access(w Writer, day, now time.Time) (editable bool, locksAt *time.Time) {
	today := Today(now, w.Loc)
	if day.Before(w.FirstDay) || day.After(today) {
		return false, nil
	}
	if w.Premium {
		return true, nil
	}
	if day.Before(today.AddDate(0, 0, -1)) {
		return false, nil
	}
	locks := freeLocksAt(day, w.Loc)
	if !now.Before(locks) {
		return false, nil
	}
	return true, &locks
}

// freeLocksAt is when a Free User's day stops being writable: the end of the next Day's Grace Window.
func freeLocksAt(day time.Time, loc *time.Location) time.Time {
	y, m, d := day.Date()
	return time.Date(y, m, d+1, GraceHours, 0, 0, 0, loc).UTC()
}
