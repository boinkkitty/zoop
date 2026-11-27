//go:build !testclock

package main

import (
	"net/http"

	"github.com/boinkkitty/zoop/internal/clock"
)

// withTestClock is a no-op in production builds: the override is compiled out (see testclock.go).
func withTestClock(c clock.Clock) (clock.Clock, func(http.Handler) http.Handler) {
	return c, func(h http.Handler) http.Handler { return h }
}
