// Package days turns a moment and a user's timezone into Days.
//
// A Day is a calendar date, represented as a time.Time at midnight UTC.
package days

import (
	"fmt"
	"time"
)

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

// Calendar is a user's timezone and the switch to another one, if they asked for one.
//
// A switch takes effect once Next's date reaches SwitchDay. Days stay whole across it:
// flying east the Day before SwitchDay ends early, flying west SwitchDay runs long,
// and today never skips, repeats or moves backwards, even between zones more than 24h apart.
type Calendar struct {
	Zone      *time.Location
	Next      *time.Location // the zone being switched to; nil when no switch is pending
	SwitchDay time.Time      // the first Day on Next's time
}

// LoadCalendar reads a Calendar as stored: a zone name, and the pending switch's zone and Day, if any.
func LoadCalendar(zone string, next *string, switchDay *time.Time) (Calendar, error) {
	loc, err := time.LoadLocation(zone)
	if err != nil {
		return Calendar{}, fmt.Errorf("stored timezone %q: %w", zone, err)
	}
	c := Calendar{Zone: loc}
	if next != nil && switchDay != nil {
		if c.Next, err = time.LoadLocation(*next); err != nil {
			return Calendar{}, fmt.Errorf("stored next timezone %q: %w", *next, err)
		}
		c.SwitchDay = *switchDay
	}
	return c, nil
}

// Today is the Day it is at now.
func (c Calendar) Today(now time.Time) time.Time {
	old := Today(now, c.Zone)
	if c.Next == nil {
		return old
	}
	switch next := Today(now, c.Next); {
	case next.Before(c.SwitchDay):
		return earlier(old, c.SwitchDay) // Next is more than 24h behind: hold SwitchDay until it arrives
	case old.Before(c.SwitchDay):
		return c.SwitchDay // Next is more than 24h ahead: don't skip SwitchDay
	default:
		return next
	}
}

// Switched says whether the pending switch has taken effect at now.
func (c Calendar) Switched(now time.Time) bool {
	return c.Next != nil && !Today(now, c.Next).Before(c.SwitchDay)
}

// ZoneAt is the zone the user's Days are on at now.
func (c Calendar) ZoneAt(now time.Time) *time.Location {
	if c.Switched(now) {
		return c.Next
	}
	return c.Zone
}

// GraceUntil is when yesterday stops being writable, if now is inside the Grace Window; otherwise nil.
func (c Calendar) GraceUntil(now time.Time) *time.Time {
	end := c.graceEnd(c.Today(now))
	if !now.Before(end) {
		return nil
	}
	return &end
}

// Switch asks for Days on to's time from tomorrow. Asking for the zone already in use cancels a pending switch.
// A switch that has already taken effect is kept, and any Day it already started is lost track of.
func (c Calendar) Switch(to *time.Location, now time.Time) Calendar {
	base := Calendar{Zone: c.ZoneAt(now)}
	if to.String() == base.Zone.String() {
		return base
	}
	return Calendar{Zone: base.Zone, Next: to, SwitchDay: c.Today(now).AddDate(0, 0, 1)}
}

// Settle drops a switch once it makes no difference to today or any Day still writable,
// so the calendar can be stored as Next alone.
func (c Calendar) Settle(now time.Time) Calendar {
	if c.Next == nil {
		return c
	}
	today := c.Today(now)
	if date, loc := c.dayStart(today); today.After(c.SwitchDay) && date.Equal(today) && loc == c.Next {
		return Calendar{Zone: c.Next}
	}
	return c
}

// dayStart says when day begins: at midnight of date in loc.
func (c Calendar) dayStart(day time.Time) (date time.Time, loc *time.Location) {
	if c.Next == nil {
		return day, c.Zone
	}
	if !day.After(c.SwitchDay) {
		// Up to SwitchDay, Days begin at the old zone's midnight, unless the switch to Next begins SwitchDay first.
		if !midnight(c.SwitchDay, c.Next).Before(midnight(day, c.Zone)) {
			return day, c.Zone
		}
		return c.SwitchDay, c.Next
	}
	// After it, at Next's midnight, though never before the old zone has reached SwitchDay.
	if midnight(c.SwitchDay, c.Zone).After(midnight(day, c.Next)) {
		return c.SwitchDay, c.Zone
	}
	return day, c.Next
}

// graceEnd is when day's Grace Window closes: 02:00 on the clock its Day began by.
func (c Calendar) graceEnd(day time.Time) time.Time {
	date, loc := c.dayStart(day)
	y, m, d := date.Date()
	return time.Date(y, m, d, GraceHours, 0, 0, 0, loc).UTC()
}

func midnight(day time.Time, loc *time.Location) time.Time {
	y, m, d := day.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, loc)
}

func earlier(a, b time.Time) time.Time {
	if b.Before(a) {
		return b
	}
	return a
}

// Format writes a Day as YYYY-MM-DD.
func Format(day time.Time) string { return day.Format(time.DateOnly) }

// Writer is who is asking to write: their plan, first Day and Calendar.
type Writer struct {
	Premium  bool
	FirstDay time.Time
	Calendar Calendar
}

// Access says whether w may write day at now, and when that ends (nil: it won't, or it already has).
// Nobody writes the future or before their first Day. Premium Users write every other Day.
// Free Users write today, and yesterday until the Grace Window ends.
func Access(w Writer, day, now time.Time) (editable bool, locksAt *time.Time) {
	today := w.Calendar.Today(now)
	if day.Before(w.FirstDay) || day.After(today) {
		return false, nil
	}
	if w.Premium {
		return true, nil
	}
	if day.Before(today.AddDate(0, 0, -1)) {
		return false, nil
	}
	// A Free User's Day stops being writable when the next Day's Grace Window closes.
	locks := w.Calendar.graceEnd(day.AddDate(0, 0, 1))
	if !now.Before(locks) {
		return false, nil
	}
	return true, &locks
}
