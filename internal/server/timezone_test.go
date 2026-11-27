package server_test

import (
	"net/http"
	"slices"
	"testing"
	"time"
)

// london is a wall-clock moment in London (UTC+0 in November).
func london(month time.Month, day, hour, min int) time.Time {
	return time.Date(2026, month, day, hour, min, 0, 0, time.UTC)
}

func (b *browser) setTimezone(zone string) (me, response) {
	b.app.t.Helper()
	res := b.do(http.MethodPatch, "/api/v1/me", map[string]string{"timezone": zone})
	var m me
	if res.status == http.StatusOK {
		res.json(b.app.t, &m)
	}
	return m, res
}

func TestChoosingAnotherTimezoneSwitchesFromTomorrow(t *testing.T) {
	app := startApp(t, london(11, 26, 10, 0))
	b := app.browser()
	b.signup(uniqueEmail("tz-pending"), "Europe/London")

	got, res := b.setTimezone("Asia/Tokyo")
	if res.status != http.StatusOK {
		t.Fatalf("PATCH /me = %d %s, want 200", res.status, res.body)
	}
	if got.Timezone != "Europe/London" || got.Today != "2026-11-26" {
		t.Errorf("timezone %q today %q, want Europe/London and 2026-11-26 until the switch", got.Timezone, got.Today)
	}
	if got.PendingTimezone == nil || *got.PendingTimezone != "Asia/Tokyo" ||
		got.TimezoneSwitchDay == nil || *got.TimezoneSwitchDay != "2026-11-27" {
		t.Errorf("pending %v on %v, want Asia/Tokyo on 2026-11-27 (tomorrow in London)", got.PendingTimezone, got.TimezoneSwitchDay)
	}
	if again, _ := b.me(); again.PendingTimezone == nil || *again.PendingTimezone != "Asia/Tokyo" {
		t.Errorf("GET /me pending = %v, want Asia/Tokyo kept", again.PendingTimezone)
	}
}

func TestChoosingTheTimezoneInUseCancelsAPendingSwitch(t *testing.T) {
	app := startApp(t, london(11, 26, 10, 0))
	b := app.browser()
	b.signup(uniqueEmail("tz-cancel"), "Europe/London")
	b.setTimezone("Asia/Tokyo")

	got, res := b.setTimezone("Europe/London")
	if res.status != http.StatusOK {
		t.Fatalf("PATCH /me = %d %s, want 200", res.status, res.body)
	}
	if got.Timezone != "Europe/London" || got.PendingTimezone != nil || got.TimezoneSwitchDay != nil {
		t.Errorf("timezone %q pending %v on %v, want Europe/London with nothing pending", got.Timezone, got.PendingTimezone, got.TimezoneSwitchDay)
	}
}

func TestAnUnknownTimezoneIsRefusedAndChangesNothing(t *testing.T) {
	app := startApp(t, london(11, 26, 10, 0))
	b := app.browser()
	b.signup(uniqueEmail("tz-unknown"), "Europe/London")

	for _, zone := range []string{"", "Local", "Mars/Olympus_Mons"} {
		if _, res := b.setTimezone(zone); res.status != http.StatusBadRequest || res.errorCode(t) != "validation" {
			t.Errorf("PATCH /me %q = %d %s, want 400 validation", zone, res.status, res.body)
		}
	}
	if got, _ := b.me(); got.Timezone != "Europe/London" || got.PendingTimezone != nil {
		t.Errorf("timezone %q pending %v, want Europe/London unchanged", got.Timezone, got.PendingTimezone)
	}
}

// walkDays moves the clock from..to in steps, asking /me for today at each, and fails
// if today ever skips a Day, repeats one after leaving it, or moves backwards.
func walkDays(t *testing.T, app *testApp, b *browser, from, to time.Time, step time.Duration) []string {
	t.Helper()
	var seen []string
	for now := from; !now.After(to); now = now.Add(step) {
		app.clock.Set(now)
		got, _ := b.me()
		if n := len(seen); n > 0 && seen[n-1] == got.Today {
			continue
		}
		if n := len(seen); n > 0 {
			prev, _ := time.Parse(time.DateOnly, seen[n-1])
			if want := prev.AddDate(0, 0, 1).Format(time.DateOnly); got.Today != want {
				t.Fatalf("at %s today went %s → %s, want %s next", now.UTC().Format(time.RFC3339), seen[n-1], got.Today, want)
			}
		}
		seen = append(seen, got.Today)
	}
	return seen
}

func TestFlyingEastTheDaysFollowWithoutSkippingOrRepeating(t *testing.T) {
	app := startApp(t, london(11, 26, 10, 0))
	b := app.browser()
	b.signup(uniqueEmail("tz-east"), "Europe/London")
	b.setTimezone("Asia/Tokyo") // from 27 Nov

	// Tokyo reaches 27 Nov at 15:00 London time: Day 26 ends early.
	app.clock.Set(london(11, 26, 14, 59))
	if got, _ := b.me(); got.Today != "2026-11-26" || got.Timezone != "Europe/London" {
		t.Errorf("at 14:59 London today %q zone %q, want 2026-11-26 on London time", got.Today, got.Timezone)
	}
	app.clock.Set(london(11, 26, 15, 0))
	got, _ := b.me()
	if got.Today != "2026-11-27" || got.Timezone != "Asia/Tokyo" || got.PendingTimezone != nil {
		t.Errorf("at 00:00 Tokyo today %q zone %q pending %v, want 2026-11-27 on Tokyo time, nothing pending",
			got.Today, got.Timezone, got.PendingTimezone)
	}

	seen := walkDays(t, app, b, london(11, 26, 10, 0), london(11, 30, 10, 0), 15*time.Minute)
	if want := []string{"2026-11-26", "2026-11-27", "2026-11-28", "2026-11-29", "2026-11-30"}; !slices.Equal(seen, want) {
		t.Errorf("Days seen = %v, want %v", seen, want)
	}
}

func TestFlyingWestTheDaysFollowWithoutSkippingOrRepeating(t *testing.T) {
	app := startApp(t, tokyo(11, 27, 10, 0, 0))
	b := app.browser()
	b.signup(uniqueEmail("tz-west"), "Asia/Tokyo")
	b.setTimezone("Europe/London") // from 28 Nov

	// Tokyo reaches 28 Nov first, and Day 28 runs on until London reaches it too.
	app.clock.Set(tokyo(11, 28, 8, 59, 0)) // 23:59 on 27 Nov in London
	if got, _ := b.me(); got.Today != "2026-11-28" || got.Timezone != "Asia/Tokyo" {
		t.Errorf("at 08:59 Tokyo today %q zone %q, want 2026-11-28 still on Tokyo time", got.Today, got.Timezone)
	}
	app.clock.Set(london(11, 28, 0, 0))
	if got, _ := b.me(); got.Today != "2026-11-28" || got.Timezone != "Europe/London" || got.PendingTimezone != nil {
		t.Errorf("at 00:00 London today %q zone %q pending %v, want 2026-11-28 on London time, nothing pending",
			got.Today, got.Timezone, got.PendingTimezone)
	}

	seen := walkDays(t, app, b, tokyo(11, 27, 10, 0, 0), london(11, 30, 10, 0), 15*time.Minute)
	if want := []string{"2026-11-27", "2026-11-28", "2026-11-29", "2026-11-30"}; !slices.Equal(seen, want) {
		t.Errorf("Days seen = %v, want %v", seen, want)
	}
}

func TestFlyingWestYesterdayStaysClosedOnceItsGraceWindowEnded(t *testing.T) {
	app := startApp(t, tokyo(11, 27, 10, 0, 0))
	b := app.browser()
	b.signup(uniqueEmail("tz-west-grace"), "Asia/Tokyo")
	b.setTimezone("Europe/London")

	// Day 27 closed at 02:00 Tokyo on the 28th; it mustn't reopen at 02:00 London.
	app.clock.Set(london(11, 28, 0, 30))
	got, _ := b.page("2026-11-27", "my-journal")
	if got.Editable {
		t.Error("Day 27 is editable at 00:30 London, want read-only since 02:00 Tokyo")
	}
	if res := b.save("2026-11-27", "my-journal", helloDoc, 0); res.status != http.StatusForbidden {
		t.Errorf("PUT Day 27 at 00:30 London = %d %s, want 403 page_locked", res.status, res.body)
	}
	if me, _ := b.me(); me.GraceUntil != nil {
		t.Errorf("graceUntil = %q at 00:30 London, want null: Day 28 began at Tokyo's midnight", *me.GraceUntil)
	}
	// Day 28 closes at 02:00 London on the 29th, like any London Day.
	got, _ = b.page("2026-11-28", "my-journal")
	if !got.Editable || got.LocksAt == nil || *got.LocksAt != "2026-11-29T02:00:00Z" {
		t.Errorf("Day 28 = editable %v locksAt %v, want editable until 2026-11-29T02:00:00Z", got.Editable, got.LocksAt)
	}
}

func TestFlyingEastYesterdayGetsItsGraceWindowOnTheNewClock(t *testing.T) {
	app := startApp(t, london(11, 26, 10, 0))
	b := app.browser()
	b.signup(uniqueEmail("tz-east-grace"), "Europe/London")
	b.setTimezone("Asia/Tokyo")

	// Day 26 ended at 00:00 Tokyo (15:00 London), and stays open until 02:00 Tokyo.
	app.clock.Set(tokyo(11, 27, 1, 30, 0))
	got, _ := b.page("2026-11-26", "my-journal")
	if !got.Editable || got.LocksAt == nil || *got.LocksAt != "2026-11-26T17:00:00Z" {
		t.Errorf("Day 26 at 01:30 Tokyo = editable %v locksAt %v, want editable until 2026-11-26T17:00:00Z (02:00 Tokyo)",
			got.Editable, got.LocksAt)
	}
}

// Pago Pago (UTC−11) and Kiritimati (UTC+14) are 25 hours apart: two dates apart for an hour a day.
func TestAcrossZonesMoreThanADayApartNoDayIsSkippedAndTodayNeverGoesBack(t *testing.T) {
	pago := time.FixedZone("SST", -11*3600)
	cases := map[string]struct {
		from, to string
		want     []string
	}{
		// Kiritimati is already past the switch Day, so Day 26 ends at once; Day 27 then
		// lasts the half hour until Pago Pago's midnight, rather than being skipped.
		"east": {"Pacific/Pago_Pago", "Pacific/Kiritimati",
			[]string{"2026-11-27", "2026-11-28", "2026-11-29", "2026-11-30", "2026-12-01", "2026-12-02"}},
		// Day 29 holds from Kiritimati's midnight until Pago Pago's, rather than going back to 27.
		"west": {"Pacific/Kiritimati", "Pacific/Pago_Pago",
			[]string{"2026-11-28", "2026-11-29", "2026-11-30"}},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			// 23:30 on 26 Nov in Pago Pago is already 00:30 on 28 Nov in Kiritimati.
			start := time.Date(2026, 11, 26, 23, 30, 0, 0, pago)
			app := startApp(t, start)
			b := app.browser()
			b.signup(uniqueEmail("tz-far-"+name), c.from)
			b.setTimezone(c.to)

			seen := walkDays(t, app, b, start, start.Add(96*time.Hour), 15*time.Minute)
			if !slices.Equal(seen, c.want) {
				t.Errorf("Days seen = %v, want %v", seen, c.want)
			}
			if got, _ := b.me(); got.Timezone != c.to {
				t.Errorf("timezone = %q after four days, want %q", got.Timezone, c.to)
			}
		})
	}
}

func TestOnceSwitchedTheNewZoneIsTheOneInUseForTheNextSwitch(t *testing.T) {
	app := startApp(t, london(11, 26, 10, 0))
	b := app.browser()
	b.signup(uniqueEmail("tz-again"), "Europe/London")
	b.setTimezone("Asia/Tokyo")

	app.clock.Set(tokyo(11, 28, 10, 0, 0))
	if got, _ := b.setTimezone("Asia/Tokyo"); got.Timezone != "Asia/Tokyo" || got.PendingTimezone != nil {
		t.Errorf("choosing Tokyo on Tokyo time = zone %q pending %v, want Asia/Tokyo, nothing pending", got.Timezone, got.PendingTimezone)
	}
	got, _ := b.setTimezone("Europe/London")
	if got.Timezone != "Asia/Tokyo" || got.Today != "2026-11-28" ||
		got.PendingTimezone == nil || *got.PendingTimezone != "Europe/London" ||
		got.TimezoneSwitchDay == nil || *got.TimezoneSwitchDay != "2026-11-29" {
		t.Errorf("flying home = zone %q today %q pending %v on %v, want Asia/Tokyo today 2026-11-28, London from 2026-11-29",
			got.Timezone, got.Today, got.PendingTimezone, got.TimezoneSwitchDay)
	}
}
