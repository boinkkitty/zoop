// Package days turns a moment and a user's timezone into Days.
//
// A Day is a calendar date, represented as a time.Time at midnight UTC.
package days

import "time"

// GraceHours is how long after midnight the previous Day can still be written.
const GraceHours = 2

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
