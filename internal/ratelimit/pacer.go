package ratelimit

import (
	"context"
	"sync"
	"time"
)

// Pacer enforces a minimum interval between calls across every goroutine that
// shares it.
//
// It exists because an hourly cap is not a rate limit. LLM_MAX_CALLS_PER_HOUR=60
// permits all sixty calls in the first minute, and with four worker slots each
// retrying three times across two keys, a burst of twenty-four requests can hit
// a provider that allows five per minute. Everything past the fifth is rejected,
// every rejection burns an attempt, and jobs die reporting "all keys failed"
// while both keys are healthy — the failure looks like exhausted quota and is
// actually self-inflicted congestion.
//
// Waiting our turn before the request is strictly cheaper than being rejected
// and retrying after it: the provider's own backoff hint is ~49s, while pacing
// to five per minute costs 12s.
type Pacer struct {
	mu       sync.Mutex
	interval time.Duration
	next     time.Time
}

// NewPacer returns a Pacer allowing at most perMinute calls per minute. A
// non-positive rate disables pacing entirely (Wait returns immediately), which
// is the right behaviour for a paid tier with headroom.
func NewPacer(perMinute int) *Pacer {
	if perMinute <= 0 {
		return &Pacer{}
	}
	return &Pacer{interval: time.Minute / time.Duration(perMinute)}
}

// Wait blocks until this caller may issue its request, or until ctx is done.
//
// Slots are reserved under the lock and the sleep happens outside it, so callers
// queue in arrival order without serialising on the lock itself.
func (p *Pacer) Wait(ctx context.Context) error {
	if p == nil || p.interval <= 0 {
		return nil
	}
	p.mu.Lock()
	now := time.Now()
	slot := p.next
	if slot.Before(now) {
		slot = now
	}
	p.next = slot.Add(p.interval)
	p.mu.Unlock()

	delay := time.Until(slot)
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
