// Package web holds the built frontend (web/dist), embedded into the binary.
// Run `pnpm build` before `go build` so dist is current.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// Dist returns the built frontend, rooted so that index.html is at the top.
func Dist() fs.FS {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}
	return sub
}
