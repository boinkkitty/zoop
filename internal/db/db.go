// Package db owns the schema: plain SQL files in migrations/, run by hand in
// filename order with psql. Nothing in the app applies them.
package db

import (
	"embed"
	"io/fs"
)

//go:embed migrations/*.sql
var embedded embed.FS

// Migrations returns the schema files, for the test harness to build throwaway databases.
func Migrations() fs.FS {
	sub, err := fs.Sub(embedded, "migrations")
	if err != nil {
		panic(err)
	}
	return sub
}
