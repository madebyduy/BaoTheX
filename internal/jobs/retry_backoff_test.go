package jobs

import (
	"fmt"
	"testing"
	"time"

	"repwire/internal/briefmedia"
	"repwire/internal/process"
)

// Causes arrive wrapped by the time they reach the worker, so wrap them here
// too — testing the bare sentinels would pass even if retryDelay compared with
// == instead of errors.Is.
func wrapped(err error) error { return fmt.Errorf("job handler: %w", err) }

func TestRateLimitsRetrySoonerThanBudget(t *testing.T) {
	// Three times tonight the same conflation cost a feature: a throughput limit
	// ("retry in 45s") treated as a spend limit. Here it parked the 05:00 audio
	// brief until 11:00 — and because a pending job keeps its dedup key, the
	// scheduler's hourly retry was swallowed as a duplicate. One transient 429
	// cost the whole morning.
	rate := retryDelay(1, wrapped(briefmedia.ErrQuotaExceeded))
	budget := retryDelay(1, wrapped(process.ErrBudgetExceeded))
	if rate >= budget {
		t.Fatalf("rate-limit retry %v must be sooner than budget retry %v", rate, budget)
	}
}

func TestRateLimitRetryBeatsTheHourlySchedule(t *testing.T) {
	// The audio scheduler re-checks every hour. A rate-limit retry longer than
	// that lets the pending job's dedup key block the next scheduled attempt.
	if rateLimitRetryAfter >= time.Hour {
		t.Fatalf("rateLimitRetryAfter %v must be under the scheduler's hourly tick",
			rateLimitRetryAfter)
	}
}

func TestHourlyCapIsARateLimitNotABudget(t *testing.T) {
	// ErrHourlyCapReached clears within the hour on its own; treating it as a
	// spend problem would idle the job for six hours over nothing.
	if got := retryDelay(1, wrapped(process.ErrHourlyCapReached)); got != rateLimitRetryAfter {
		t.Fatalf("hourly cap retry = %v, want the rate-limit delay %v", got, rateLimitRetryAfter)
	}
}

func TestTTSQuotaIsTreatedAsARateLimit(t *testing.T) {
	if got := retryDelay(1, wrapped(briefmedia.ErrQuotaExceeded)); got != rateLimitRetryAfter {
		t.Fatalf("tts quota retry = %v, want %v", got, rateLimitRetryAfter)
	}
}

func TestBudgetStillWaitsOutTheDay(t *testing.T) {
	// The one case the long delay is right for: money does not return this hour.
	if got := retryDelay(1, wrapped(process.ErrBudgetExceeded)); got != budgetRetryAfter {
		t.Fatalf("budget retry = %v, want %v", got, budgetRetryAfter)
	}
}

func TestOrdinaryFailuresUseExponentialBackoff(t *testing.T) {
	// backoff() adds up to 60s of jitter so retries from a burst of failures do
	// not all return at once — hence a range rather than an equality. attempt=2
	// is 2^2 = 4 minutes, plus that jitter.
	got := retryDelay(2, fmt.Errorf("network unreachable"))
	if got < 4*time.Minute || got >= 5*time.Minute {
		t.Fatalf("ordinary failure retry = %v, want 4m plus jitter", got)
	}
}

func TestBackoffJitterStaysWithinItsStep(t *testing.T) {
	// If jitter could reach a full step, two attempts would become
	// indistinguishable and the exponential shape would blur.
	for attempt := 1; attempt <= 4; attempt++ {
		base := time.Duration(1<<attempt) * time.Minute
		got := backoff(attempt)
		if got < base || got >= base+time.Minute {
			t.Fatalf("backoff(%d) = %v, want [%v, %v)", attempt, got, base, base+time.Minute)
		}
	}
}
