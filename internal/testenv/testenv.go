// Package testenv gives each Go test package its own fresh Postgres database with
// every file in internal/db/migrations applied, in filename order.
//
// Use it from TestMain:
//
//	func TestMain(m *testing.M) { testenv.Run(m) }
//
// then call Pool(t) in tests. It connects to ZOOP_TEST_DATABASE_URL
// (default: the Docker Compose database on localhost:5433).
package testenv

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"sort"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/boinkkitty/zoop/internal/db"
)

const defaultURL = "postgres://zoop:zoop@localhost:5433/zoop?sslmode=disable"

var (
	pool    *pgxpool.Pool
	poolURL string
)

// Run creates a database for this test package, applies the schema, runs the tests, then drops it.
func Run(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	ctx := context.Background()
	adminURL := os.Getenv("ZOOP_TEST_DATABASE_URL")
	if adminURL == "" {
		adminURL = defaultURL
	}

	admin, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "testenv: connect to %s: %v (is `docker compose up -d` running?)\n", redact(adminURL), err)
		return 1
	}
	defer admin.Close(ctx)

	name := "zoop_test_" + randomHex(6)
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		fmt.Fprintf(os.Stderr, "testenv: create database: %v\n", err)
		return 1
	}
	defer func() {
		_, _ = admin.Exec(ctx, "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)")
	}()

	poolURL = withDatabase(adminURL, name)
	pool, err = pgxpool.New(ctx, poolURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "testenv: open pool: %v\n", err)
		return 1
	}
	defer pool.Close()

	if err := applySchema(ctx, pool); err != nil {
		fmt.Fprintf(os.Stderr, "testenv: apply schema: %v\n", err)
		return 1
	}
	return m.Run()
}

// applySchema runs each migration file in its own transaction, the way `psql -1 -f` would.
func applySchema(ctx context.Context, pool *pgxpool.Pool) error {
	files, err := fs.Glob(db.Migrations(), "*.sql")
	if err != nil {
		return err
	}
	sort.Strings(files)
	for _, name := range files {
		sql, err := fs.ReadFile(db.Migrations(), name)
		if err != nil {
			return err
		}
		if err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, string(sql))
			return err
		}); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	return nil
}

// Pool returns this package's database. It fails the test if Run wasn't used.
func Pool(t testing.TB) *pgxpool.Pool {
	t.Helper()
	if pool == nil {
		t.Fatal("testenv: Pool called without testenv.Run in TestMain")
	}
	return pool
}

// URL returns the connection string for this package's database.
func URL(t testing.TB) string {
	t.Helper()
	Pool(t)
	return poolURL
}

func withDatabase(raw, name string) string {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	u.Path = "/" + name
	return u.String()
}

func redact(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "<database url>"
	}
	return u.Redacted()
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
