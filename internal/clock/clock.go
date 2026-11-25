// Package clock is the source of time for everything that cares about Days.
package clock

import (
	"sync"
	"time"
)

// Clock tells the current moment. Production uses System; tests inject a Fake.
type Clock interface {
	Now() time.Time
}

// System is the real wall clock.
type System struct{}

func (System) Now() time.Time { return time.Now() }

// Fake is a Clock that only moves when told to.
type Fake struct {
	mu  sync.Mutex
	now time.Time
}

func NewFake(now time.Time) *Fake { return &Fake{now: now} }

func (f *Fake) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now
}

// Advance moves the clock forward by d.
func (f *Fake) Advance(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.now = f.now.Add(d)
}

// Set moves the clock to t.
func (f *Fake) Set(t time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.now = t
}
