package postgres

import (
	"testing"
	"time"
)

func TestDayStreakUsesVietnamCalendarDay(t *testing.T) {
	ict := time.FixedZone("ICT", 7*60*60)
	now := time.Date(2026, 7, 20, 0, 30, 0, 0, ict)
	days := []time.Time{
		time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC),
	}
	if got := dayStreakAt(days, now); got != 3 {
		t.Fatalf("streak = %d, want 3", got)
	}
}

func TestDayStreakAllowsYesterdayButNotOlder(t *testing.T) {
	ict := time.FixedZone("ICT", 7*60*60)
	now := time.Date(2026, 7, 20, 12, 0, 0, 0, ict)
	if got := dayStreakAt([]time.Time{time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC)}, now); got != 1 {
		t.Fatalf("yesterday streak = %d, want 1", got)
	}
	if got := dayStreakAt([]time.Time{time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC)}, now); got != 0 {
		t.Fatalf("stale streak = %d, want 0", got)
	}
}
