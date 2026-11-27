package parley

import (
	"testing"
	"time"
)

var t0 = time.Date(2026, 11, 27, 9, 0, 0, 0, time.UTC)

func TestWindowLimiter(t *testing.T) {
	l := NewWindowLimiter(2, time.Minute)
	if !l.Take("a", t0) || !l.Take("a", t0.Add(10*time.Second)) {
		t.Fatal("first two events refused")
	}
	if l.Take("a", t0.Add(20*time.Second)) {
		t.Fatal("third event inside the window allowed")
	}
	if !l.Take("b", t0.Add(20*time.Second)) {
		t.Fatal("keys are not independent")
	}
	// The first event slides out of the window after one minute.
	if !l.Take("a", t0.Add(61*time.Second)) {
		t.Fatal("event not allowed after the window slid")
	}
	if !NewWindowLimiter(0, time.Minute).Take("x", t0) {
		t.Fatal("zero limit should mean unlimited")
	}
}

func TestDailyBudget(t *testing.T) {
	b := DailyBudget{MaxReplies: 2, MaxUSD: 1.0}
	b.AddReply(t0)
	if out, _ := b.Exhausted(t0); out {
		t.Fatal("exhausted too early")
	}
	b.AddReply(t0)
	if out, why := b.Exhausted(t0); !out || why != "daily reply limit" {
		t.Fatalf("reply cap not enforced: %v %q", out, why)
	}
	// A new day resets the counters.
	if out, _ := b.Exhausted(t0.Add(24 * time.Hour)); out {
		t.Fatal("budget did not reset the next day")
	}
	b.AddCost(t0.Add(24*time.Hour), 1.5)
	if out, why := b.Exhausted(t0.Add(24 * time.Hour)); !out || why != "daily spending limit" {
		t.Fatalf("spend cap not enforced: %v %q", out, why)
	}
}

// "Daily" follows the configured time zone, not UTC.
func TestDailyBudgetTimeZone(t *testing.T) {
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("time zone data not available:", err)
	}
	b := DailyBudget{MaxReplies: 1, Location: ny}
	late := time.Date(2026, 11, 28, 3, 0, 0, 0, time.UTC) // 10 PM on the 27th in New York
	b.AddReply(late)
	if out, _ := b.Exhausted(late.Add(time.Hour)); !out {
		t.Fatal("11 PM New York time should still be the same day")
	}
	if out, _ := b.Exhausted(late.Add(3 * time.Hour)); out {
		t.Fatal("1 AM New York time should be a new day")
	}
}
