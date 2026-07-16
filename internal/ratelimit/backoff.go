// Package ratelimit decides how long to wait before retrying a rejected API
// call.
//
// It exists because the same mistake was made independently on both paths that
// talk to Gemini: a 429 means "you are going too fast, wait N seconds", but both
// the summarizer and the TTS renderer treated it as a hard failure — one waited
// 1.5s against a 49s hint, the other rotated keys without waiting at all. In
// both cases healthy keys with untouched budget reported themselves exhausted.
// Free-tier Gemini allows only a handful of requests per minute, so this is the
// normal case, not an edge case, and it deserves one shared answer.
package ratelimit

import (
	"regexp"
	"strconv"
	"time"
)

// MaxWait caps a single wait. Providers can ask for very long delays, and
// honouring an unbounded one would pin a worker slot open until the job reaper
// kills it.
const MaxWait = 90 * time.Second

// hintRe matches the wait a rate-limited provider asks for, e.g. Gemini's
// "Please retry in 49.297410174s".
var hintRe = regexp.MustCompile(`retry in ([0-9]+(?:\.[0-9]+)?)s`)

// Backoff is the fallback delay when the provider offers no hint: 0.5s, 1s,
// 1.5s, ... Attempts are 1-based.
func Backoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	return time.Duration(attempt) * 500 * time.Millisecond
}

// Wait returns how long to sleep before retrying, preferring the provider's own
// number in msg over the exponential fallback. It never returns less than
// Backoff(attempt) and never more than MaxWait.
func Wait(attempt int, msg string) time.Duration {
	wait := Backoff(attempt)
	if m := hintRe.FindStringSubmatch(msg); m != nil {
		if secs, err := strconv.ParseFloat(m[1], 64); err == nil && secs > 0 {
			// A shade over the hint: returning at exactly the boundary just earns
			// another rejection and burns an attempt.
			if hinted := time.Duration(secs*float64(time.Second)) + time.Second; hinted > wait {
				wait = hinted
			}
		}
	}
	if wait > MaxWait {
		wait = MaxWait
	}
	return wait
}
