package process

import (
	"errors"
	"testing"
	"time"
)

func TestProviderQuotaWindowResetsAtPacificMidnight(t *testing.T) {
	now := time.Date(2026, 7, 21, 10, 0, 0, 0, time.FixedZone("ICT", 7*60*60))
	start, reset := ProviderQuotaWindow(now)
	if got := reset.In(now.Location()).Hour(); got != 14 {
		t.Fatalf("Vietnam reset hour = %d, want 14 during Pacific daylight time", got)
	}
	if !start.Before(now) || !reset.After(now) {
		t.Fatalf("window %v..%v does not contain %v", start, reset, now)
	}
}

func TestProviderRequestErrorDistinguishesRPD(t *testing.T) {
	err := providerRequestError(429, "quota requests per day exhausted; retry in 14400s")
	if !errors.Is(err, ErrDailyQuotaExceeded) {
		t.Fatalf("error = %v, want daily quota", err)
	}
}

func TestProviderRequestErrorTreatsShortHintAsRPM(t *testing.T) {
	err := providerRequestError(429, "resource exhausted; retry in 45s")
	if !errors.Is(err, ErrRateLimited) || errors.Is(err, ErrDailyQuotaExceeded) {
		t.Fatalf("error = %v, want RPM limit", err)
	}
}
