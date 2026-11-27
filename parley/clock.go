package parley

import (
	"sync"
	"time"
)

// Clock abstracts the passage of time so that the engine's timing rules
// (debounce, maximum wait, cooldowns, rate-limit windows, daily budgets) can be
// tested deterministically.
type Clock interface {
	// Now returns the current time.
	Now() time.Time

	// AfterFunc arranges for f to be called once, after duration d.
	AfterFunc(d time.Duration, f func()) Timer
}

// Timer is the part of *time.Timer that the engine uses.
type Timer interface {
	// Stop prevents the timer from firing. It returns false if the timer had
	// already fired or been stopped.
	Stop() bool
}

// RealClock is the production Clock, backed by package time. Its AfterFunc
// callbacks run on their own goroutines, as with time.AfterFunc.
type RealClock struct{}

// Now returns time.Now().
func (RealClock) Now() time.Time { return time.Now() }

// AfterFunc wraps time.AfterFunc. *time.Timer already satisfies Timer.
func (RealClock) AfterFunc(d time.Duration, f func()) Timer { return time.AfterFunc(d, f) }

// FakeClock is a manually advanced Clock for tests. Time moves only when
// Advance is called, and due timers fire synchronously, in order of their due
// time, on the goroutine that called Advance. That makes every timing
// scenario in the tests exact and repeatable.
type FakeClock struct {
	mu     sync.Mutex
	now    time.Time
	timers []*fakeTimer
	seq    int // tie-breaker so timers due at the same instant fire in creation order
}

// fakeTimer is one pending callback registered with a FakeClock.
type fakeTimer struct {
	clock   *FakeClock
	when    time.Time
	seq     int
	f       func()
	stopped bool
	fired   bool
}

// Stop cancels the timer if it has not fired yet.
func (t *fakeTimer) Stop() bool {
	t.clock.mu.Lock()
	defer t.clock.mu.Unlock()
	if t.stopped || t.fired {
		return false
	}
	t.stopped = true
	return true
}

// NewFakeClock returns a FakeClock whose current time is start.
func NewFakeClock(start time.Time) *FakeClock {
	return &FakeClock{now: start}
}

// Now returns the fake current time.
func (c *FakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// AfterFunc registers f to run when the fake time reaches Now()+d.
func (c *FakeClock) AfterFunc(d time.Duration, f func()) Timer {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := &fakeTimer{clock: c, when: c.now.Add(d), seq: c.seq, f: f}
	c.seq++
	c.timers = append(c.timers, t)
	return t
}

// Advance moves the fake time forward by d, firing every timer that falls due
// on the way. Each callback runs with the clock's lock released, so callbacks
// may themselves register or stop timers; a timer registered by a callback
// fires during the same Advance if it falls due before the target time.
func (c *FakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	target := c.now.Add(d)
	c.mu.Unlock()

	for {
		c.mu.Lock()
		// Find the earliest live timer that is due at or before the target.
		var next *fakeTimer
		for _, t := range c.timers {
			if t.stopped || t.fired || t.when.After(target) {
				continue
			}
			if next == nil || t.when.Before(next.when) || (t.when.Equal(next.when) && t.seq < next.seq) {
				next = t
			}
		}
		if next == nil {
			// Nothing more is due: settle at the target time and drop the
			// timers that can never fire again.
			c.now = target
			live := c.timers[:0]
			for _, t := range c.timers {
				if !t.stopped && !t.fired {
					live = append(live, t)
				}
			}
			c.timers = live
			c.mu.Unlock()
			return
		}
		next.fired = true
		if next.when.After(c.now) {
			c.now = next.when
		}
		c.mu.Unlock()

		// Run the callback outside the lock.
		next.f()
	}
}
