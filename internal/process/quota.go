package process

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"repwire/internal/ratelimit"
)

var (
	ErrDailyQuotaExceeded = errors.New("llm daily request quota exceeded")
	ErrRateLimited        = errors.New("llm requests-per-minute limit exceeded")
)

// ProviderQuotaWindow follows Gemini's documented midnight-Pacific RPD reset.
func ProviderQuotaWindow(now time.Time) (time.Time, time.Time) {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		// Pacific is UTC-8 at worst; this fallback is conservative during DST.
		loc = time.FixedZone("Pacific", -8*60*60)
	}
	local := now.In(loc)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	return start, start.AddDate(0, 0, 1)
}

func isDailyQuotaMessage(msg string) bool {
	m := strings.ToLower(msg)
	for _, marker := range []string{"requests per day", "request per day", "perday", "per_day", "daily quota", " rpd"} {
		if strings.Contains(m, marker) {
			return true
		}
	}
	if hint, ok := ratelimit.RetryHint(m); ok && hint >= 10*time.Minute {
		return true
	}
	return false
}

func providerRequestError(status int, msg string) error {
	switch {
	case status == 429 && isDailyQuotaMessage(msg):
		return fmt.Errorf("%w: %s", ErrDailyQuotaExceeded, msg)
	case status == 429 || isQuotaErr(status, msg):
		return fmt.Errorf("%w: %s", ErrRateLimited, msg)
	default:
		return fmt.Errorf("summarizer: %s", msg)
	}
}
