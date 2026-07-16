package ratelimit

import (
	"testing"
	"time"
)

// The message free-tier Gemini actually returns, trimmed. Keeping the real
// wording here is the point: this parser exists to read this exact shape.
const geminiRateLimitMsg = `gemini: You exceeded your current quota, please check your plan and billing details. ` +
	`* Quota exceeded for metric: generativelanguage.googleapis.com/generate_content_free_tier_requests, ` +
	`limit: 5, model: gemini-3.5-flash. Please retry in 49.297410174s.`

func TestWaitHonoursProviderHint(t *testing.T) {
	// The bug this guards: waiting 1.5s against a 49s hint puts every attempt
	// inside the same rate-limit window, so the job dies reporting "all keys
	// failed" while the keys are healthy and the budget untouched.
	if got := Wait(1, geminiRateLimitMsg); got < 49*time.Second {
		t.Fatalf("Wait = %v, want at least the hinted 49s", got)
	}
}

func TestWaitOvershootsHintSlightly(t *testing.T) {
	// Returning at exactly the hint just earns another rejection.
	if got := Wait(1, "Please retry in 10s"); got <= 10*time.Second {
		t.Fatalf("Wait = %v, want strictly more than the hint", got)
	}
}

func TestWaitFallsBackWithoutHint(t *testing.T) {
	got := Wait(2, "gemini: This model is currently experiencing high demand.")
	if want := Backoff(2); got != want {
		t.Fatalf("Wait = %v, want the exponential fallback %v", got, want)
	}
}

func TestWaitIsCapped(t *testing.T) {
	// A worker slot held past the 15-minute reaper is worse than a failed job.
	if got := Wait(1, "Please retry in 3600s"); got != MaxWait {
		t.Fatalf("Wait = %v, want the cap %v", got, MaxWait)
	}
}

func TestWaitIgnoresGarbageHint(t *testing.T) {
	for _, msg := range []string{"retry in xs", "retry in -5s", "retry in 0s", "please retry soon", ""} {
		if got, want := Wait(1, msg), Backoff(1); got != want {
			t.Fatalf("Wait(%q) = %v, want fallback %v", msg, got, want)
		}
	}
}

func TestWaitNeverShortensTheFallback(t *testing.T) {
	// A hint smaller than the exponential backoff must not walk the wait back
	// down — later attempts should still spread out.
	if got, want := Wait(3, "Please retry in 0.2s"), Backoff(3); got < want {
		t.Fatalf("Wait = %v, want at least the fallback %v", got, want)
	}
}

func TestBackoffHandlesZeroAndNegativeAttempts(t *testing.T) {
	// Callers count attempts from 0 in some loops and from 1 in others; neither
	// may produce a zero or negative sleep that busy-loops the provider.
	for _, attempt := range []int{-1, 0, 1} {
		if got := Backoff(attempt); got <= 0 {
			t.Fatalf("Backoff(%d) = %v, want a positive delay", attempt, got)
		}
	}
}
