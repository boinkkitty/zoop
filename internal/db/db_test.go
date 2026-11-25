package db_test

import (
	"context"
	"testing"

	"github.com/boinkkitty/zoop/internal/testenv"
)

func TestMain(m *testing.M) { testenv.Run(m) }

// No Notebook endpoint exists yet, so the seed is read straight from the
// migrated database. Replace with an API-level test once GET /api/v1/notebooks lands (#22).
func TestMigrationsSeedTheTwoNotebooks(t *testing.T) {
	rows, err := testenv.Pool(t).Query(context.Background(),
		"SELECT id, slug FROM notebooks ORDER BY id")
	if err != nil {
		t.Fatalf("query notebooks: %v", err)
	}
	type nb struct {
		id   int16
		slug string
	}
	var got []nb
	for rows.Next() {
		var n nb
		if err := rows.Scan(&n.id, &n.slug); err != nil {
			t.Fatal(err)
		}
		got = append(got, n)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	want := []nb{{1, "my-journal"}, {2, "i-learned"}}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("notebooks = %v, want %v", got, want)
	}
}
