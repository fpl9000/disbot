package parley

import "time"

// WindowLimiter allows at most Limit events per key within any sliding window
// of length Window. The engine uses one keyed by conversation (replies per
// minute in a channel) and one keyed by user (direct addresses per person).
//
// It is not safe for concurrent use; the engine guards it with its lock.
type WindowLimiter struct {
	Limit  int           // maximum events per window; <= 0 means unlimited
	Window time.Duration // length of the sliding window

	events map[string][]time.Time
}

// NewWindowLimiter returns a limiter allowing limit events per window.
func NewWindowLimiter(limit int, window time.Duration) *WindowLimiter {
	return &WindowLimiter{Limit: limit, Window: window, events: make(map[string][]time.Time)}
}

// prune forgets events for key that have slid out of the window.
func (l *WindowLimiter) prune(key string, now time.Time) []time.Time {
	ev := l.events[key]
	cutoff := now.Add(-l.Window)
	i := 0
	for i < len(ev) && !ev[i].After(cutoff) {
		i++
	}
	ev = ev[i:]
	if len(ev) == 0 {
		delete(l.events, key)
	} else {
		l.events[key] = ev
	}
	return ev
}

// Allow reports whether one more event for key would be within the limit,
// without recording anything.
func (l *WindowLimiter) Allow(key string, now time.Time) bool {
	if l.Limit <= 0 {
		return true
	}
	return len(l.prune(key, now)) < l.Limit
}

// Record notes that an event for key happened at now.
func (l *WindowLimiter) Record(key string, now time.Time) {
	if l.Limit <= 0 {
		return
	}
	l.events[key] = append(l.prune(key, now), now)
}

// Take is Allow followed, if allowed, by Record.
func (l *WindowLimiter) Take(key string, now time.Time) bool {
	if !l.Allow(key, now) {
		return false
	}
	l.Record(key, now)
	return true
}

// DailyBudget caps replies and estimated spending per calendar day, in a
// configured time zone. Silent turns cost money too, so cost is recorded for
// every Brain call, while the reply count only increases when something is
// actually posted.
type DailyBudget struct {
	MaxReplies int            // <= 0 means unlimited
	MaxUSD     float64        // <= 0 means unlimited
	Location   *time.Location // nil means UTC

	day     string
	replies int
	usd     float64
}

// roll resets the counters when the calendar day changes.
func (b *DailyBudget) roll(now time.Time) {
	loc := b.Location
	if loc == nil {
		loc = time.UTC
	}
	d := now.In(loc).Format("2006-01-02")
	if d != b.day {
		b.day, b.replies, b.usd = d, 0, 0
	}
}

// Exhausted reports whether today's budget is used up, and which part.
func (b *DailyBudget) Exhausted(now time.Time) (bool, string) {
	b.roll(now)
	if b.MaxReplies > 0 && b.replies >= b.MaxReplies {
		return true, "daily reply limit"
	}
	if b.MaxUSD > 0 && b.usd >= b.MaxUSD {
		return true, "daily spending limit"
	}
	return false, ""
}

// AddCost records the estimated cost of one Brain call.
func (b *DailyBudget) AddCost(now time.Time, usd float64) {
	b.roll(now)
	b.usd += usd
}

// AddReply records one posted reply.
func (b *DailyBudget) AddReply(now time.Time) {
	b.roll(now)
	b.replies++
}

// Today returns today's reply count and estimated spend.
func (b *DailyBudget) Today(now time.Time) (int, float64) {
	b.roll(now)
	return b.replies, b.usd
}
