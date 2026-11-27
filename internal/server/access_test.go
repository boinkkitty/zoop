package server_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/boinkkitty/zoop/internal/testenv"
)

// tokyo is a wall-clock moment in Tokyo (UTC+9, no DST).
func tokyo(month time.Month, day, hour, min, sec int) time.Time {
	return time.Date(2026, month, day, hour, min, sec, 0, time.FixedZone("JST", 9*3600))
}

func TestAFreeUserCanStillWriteYesterdayInTheGraceWindow(t *testing.T) {
	app := startApp(t, tokyoMorning) // 05:00 on 26 Nov in Tokyo
	b := app.browser()
	b.signup(uniqueEmail("grace-open"), "Asia/Tokyo")

	app.clock.Set(tokyo(11, 27, 1, 59, 59))
	got, res := b.page("2026-11-26", "my-journal")
	if res.status != http.StatusOK || !got.Editable {
		t.Fatalf("yesterday at 01:59:59 = %d editable %v, want 200 and editable", res.status, got.Editable)
	}
	if got.LocksAt == nil || *got.LocksAt != "2026-11-26T17:00:00Z" {
		t.Errorf("locksAt = %v, want 2026-11-26T17:00:00Z (02:00 Tokyo)", got.LocksAt)
	}
	if res := b.save("2026-11-26", "my-journal", helloDoc, 0); res.status != http.StatusOK {
		t.Errorf("PUT yesterday at 01:59:59 = %d %s, want 200", res.status, res.body)
	}
}

func TestYesterdayClosesAt0200WithAMinutesLeewayForSavesInFlight(t *testing.T) {
	app := startApp(t, tokyoMorning)
	b := app.browser()
	b.signup(uniqueEmail("grace-close"), "Asia/Tokyo")

	app.clock.Set(tokyo(11, 27, 2, 0, 30))
	got, _ := b.page("2026-11-26", "my-journal")
	if got.Editable || got.LocksAt != nil {
		t.Errorf("yesterday at 02:00:30 = editable %v locksAt %v, want read-only with no lock time", got.Editable, got.LocksAt)
	}
	// The editor flushed at 02:00 and the save arrived late: it still counts.
	if res := b.save("2026-11-26", "my-journal", helloDoc, 0); res.status != http.StatusOK {
		t.Errorf("PUT yesterday at 02:00:30 = %d %s, want 200 within the leeway", res.status, res.body)
	}

	app.clock.Set(tokyo(11, 27, 2, 1, 1))
	res := b.save("2026-11-26", "my-journal", emptyDoc, 1)
	if res.status != http.StatusForbidden || res.errorCode(t) != "page_locked" {
		t.Errorf("PUT yesterday at 02:01:01 = %d %s, want 403 page_locked", res.status, res.body)
	}
	if got, _ := b.page("2026-11-26", "my-journal"); got.Version != 1 || !sameJSON(t, got.Doc, helloDoc) {
		t.Errorf("after the refused save = version %d doc %s, want version 1 with what was written by 02:00", got.Version, got.Doc)
	}
}

// makePremium sets the plan flag by hand, the way it's done in v1.
func (a *testApp) makePremium(email string) {
	a.t.Helper()
	if _, err := testenv.Pool(a.t).Exec(context.Background(),
		`UPDATE users SET plan = 'premium' WHERE email = $1`, email); err != nil {
		a.t.Fatal(err)
	}
}

func TestAPremiumUserCanWriteEveryDaySinceTheirFirstButAFreeUserCant(t *testing.T) {
	app := startApp(t, tokyoMorning) // first Day: 26 Nov
	premium, free := app.browser(), app.browser()
	premiumEmail := uniqueEmail("premium")
	premium.signup(premiumEmail, "Asia/Tokyo")
	app.makePremium(premiumEmail)
	free.signup(uniqueEmail("free"), "Asia/Tokyo")

	app.clock.Set(tokyo(12, 6, 12, 0, 0))
	for _, day := range []string{"2026-11-26", "2026-12-01", "2026-12-06"} {
		got, _ := premium.page(day, "i-learned")
		if !got.Editable || got.LocksAt != nil {
			t.Errorf("premium %s = editable %v locksAt %v, want editable for good", day, got.Editable, got.LocksAt)
		}
		if res := premium.save(day, "i-learned", helloDoc, 0); res.status != http.StatusOK {
			t.Errorf("premium PUT %s = %d %s, want 200", day, res.status, res.body)
		}
	}
	for _, day := range []string{"2026-11-25", "2026-12-07"} {
		if res := premium.save(day, "i-learned", helloDoc, 0); res.status != http.StatusNotFound {
			t.Errorf("premium PUT %s = %d, want 404 (before the first Day, or the future)", day, res.status)
		}
	}

	got, _ := free.page("2026-11-26", "i-learned")
	if got.Editable {
		t.Error("a Free User's first Day is editable ten days later, want read-only")
	}
	if res := free.save("2026-11-26", "i-learned", helloDoc, 0); res.status != http.StatusForbidden || res.errorCode(t) != "page_locked" {
		t.Errorf("free PUT of a past Day = %d %s, want 403 page_locked", res.status, res.body)
	}
}
