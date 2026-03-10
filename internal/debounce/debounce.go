// debounce package provides a generic debouncer with a configurable interval and a callback.
package debounce

import (
	"sync"
	"time"
)

// Debouncer invokes a callback at most once per quiet period after triggers stop.
// Safe for concurrent use from multiple goroutines (e.g. netlink handler + poll ticker).
type Debouncer struct {
	interval time.Duration
	mu       sync.Mutex
	timer    *time.Timer
	fn       func()
}

// New returns a debouncer that calls fn after interval of no triggers.
func New(interval time.Duration, fn func()) *Debouncer {
	return &Debouncer{interval: interval, fn: fn}
}

// Trigger schedules or resets the debounced call.
func (d *Debouncer) Trigger() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil {
		d.timer.Stop()
	}
	d.timer = time.AfterFunc(d.interval, d.run)
}

// run is a helper to run the debounced callback.
func (d *Debouncer) run() {
	d.mu.Lock()
	d.timer = nil
	d.mu.Unlock()
	d.fn()
}

// Stop cancels any pending debounced call.
func (d *Debouncer) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}
}
