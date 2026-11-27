// Command devseed makes a local account with months of past journal entries, so the
// Island rail and scrubber have history to show. Dev only: it refuses any database
// that isn't on localhost. Running it again rewrites the same account's Pages.
//
//	DATABASE_URL=postgres://zoop:zoop@localhost:5433/zoop?sslmode=disable go run ./cmd/devseed
//
// Then sign in as dev@zoop.local / correct horse battery staple.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math/rand/v2"
	"net/url"
	"os"
	"time"
	_ "time/tzdata"

	"github.com/jackc/pgx/v5"

	"github.com/boinkkitty/zoop/internal/auth/password"
	"github.com/boinkkitty/zoop/internal/pages"
)

func main() {
	email := flag.String("email", "dev@zoop.local", "account email")
	pass := flag.String("password", "correct horse battery staple", "account password")
	tz := flag.String("tz", "Asia/Singapore", "account timezone (IANA)")
	days := flag.Int("days", 420, "how many Days back the account starts")
	written := flag.Float64("written", 0.6, "share of past Days that have writing")
	flag.Parse()
	if err := run(*email, *pass, *tz, *days, *written); err != nil {
		fmt.Fprintln(os.Stderr, "devseed:", err)
		os.Exit(1)
	}
}

func run(email, pass, tz string, days int, written float64) error {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return errors.New("DATABASE_URL is not set")
	}
	u, err := url.Parse(dbURL)
	if err != nil {
		return err
	}
	if h := u.Hostname(); h != "localhost" && h != "127.0.0.1" && h != "::1" {
		return fmt.Errorf("refusing to seed %q: devseed only writes to a localhost database", h)
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return err
	}
	hash, err := password.Hash(pass)
	if err != nil {
		return err
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	now := time.Now()
	y, m, d := now.In(loc).Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	first := today.AddDate(0, 0, -days)

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var userID string
	err = tx.QueryRow(ctx, `
		INSERT INTO users (email, email_verified_at, first_day, timezone, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $2, $2)
		ON CONFLICT (email) DO UPDATE SET first_day = EXCLUDED.first_day, timezone = EXCLUDED.timezone,
			next_timezone = NULL, timezone_switch_day = NULL, deleted_at = NULL, updated_at = EXCLUDED.updated_at
		RETURNING id`, email, now, first, tz).Scan(&userID)
	if err != nil {
		return fmt.Errorf("user: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO auth_identities (user_id, provider, subject, password_hash, created_at)
		VALUES ($1, 'password', $2, $3, $4)
		ON CONFLICT (user_id, provider) DO UPDATE SET password_hash = EXCLUDED.password_hash`,
		userID, email, hash, now); err != nil {
		return fmt.Errorf("identity: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM pages WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("clear pages: %w", err)
	}

	rng := rand.New(rand.NewPCG(42, uint64(days)))
	n := 0
	for day := first; day.Before(today); day = day.AddDate(0, 0, 1) {
		if rng.Float64() >= written {
			continue
		}
		title := titles[rng.IntN(len(titles))]
		body := bodies[rng.IntN(len(bodies))]
		if err := putPage(ctx, tx, userID, 1, "my-journal", day, now, title, body); err != nil {
			return err
		}
		n++
		if rng.Float64() < 0.35 {
			if err := putPage(ctx, tx, userID, 2, "i-learned", day, now, learned[rng.IntN(len(learned))]); err != nil {
				return err
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	fmt.Printf("seeded %s: first Day %s, %d written Days before %s (%s)\n",
		email, first.Format(time.DateOnly), n, today.Format(time.DateOnly), tz)
	return nil
}

func putPage(ctx context.Context, tx pgx.Tx, userID string, notebookID int, notebook string, day, now time.Time, lines ...string) error {
	content := make([]map[string]any, len(lines))
	for i, l := range lines {
		content[i] = map[string]any{"type": "paragraph", "content": []map[string]any{{"type": "text", "text": l}}}
	}
	doc, err := json.Marshal(map[string]any{"type": "doc", "content": content})
	if err != nil {
		return err
	}
	plain, err := pages.ParseDoc(notebook, doc)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO pages (user_id, notebook_id, day, doc, schema_version, plain_text, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, 1, $7, $7)`,
		userID, notebookID, day, doc, pages.SchemaVersion, plain, now)
	if err != nil {
		return fmt.Errorf("page %s %s: %w", notebook, day.Format(time.DateOnly), err)
	}
	return nil
}

var titles = []string{
	"Finally fixed the bike chain", "Rain all afternoon", "Long walk by the reservoir",
	"Dinner with old friends", "A slow Sunday", "First swim of the year",
	"Shipped the thing", "Couldn't sleep", "Market morning", "New coffee place",
	"Read on the train the whole way", "Moved the desk by the window",
	"Too many meetings", "Cooked laksa from scratch", "Quiet day, good day",
}

var bodies = []string{
	"Didn't plan much and it turned out fine.",
	"Spent most of it outside. My legs are tired in the good way.",
	"Talked for hours. Should do this more often.",
	"Wrote down three things to try next week and actually meant it.",
	"The light at six was ridiculous.",
}

var learned = []string{
	"Derailleurs bend more easily than you'd think.",
	"Postgres partial indexes can make a WHERE clause free.",
	"Laksa paste keeps for a week in the fridge.",
	"The Hokkien word for this is hard to spell.",
	"You can pin a clock in Playwright per browser context.",
}
