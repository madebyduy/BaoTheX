package ratelimit

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestPacerSpacesCallsApart(t *testing.T) {
	// 600/min = 100ms apart: fast enough for a test, slow enough to measure.
	p := NewPacer(600)
	ctx := context.Background()

	start := time.Now()
	for i := 0; i < 4; i++ {
		if err := p.Wait(ctx); err != nil {
			t.Fatal(err)
		}
	}
	// Four calls means three gaps; the first is free.
	if elapsed := time.Since(start); elapsed < 250*time.Millisecond {
		t.Fatalf("four calls took %v, want at least ~300ms of spacing", elapsed)
	}
}

func TestPacerFirstCallIsImmediate(t *testing.T) {
	p := NewPacer(5)
	start := time.Now()
	if err := p.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Fatalf("first call waited %v, want none", elapsed)
	}
}

func TestPacerSerialisesConcurrentCallers(t *testing.T) {
	// The case that matters: four worker slots hitting a shared limit at once
	// must queue, not stampede. Without this they all pass their own budget
	// check and fire simultaneously.
	p := NewPacer(600) // 100ms apart
	ctx := context.Background()

	start := time.Now()
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := p.Wait(ctx); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if elapsed := time.Since(start); elapsed < 250*time.Millisecond {
		t.Fatalf("four concurrent callers took %v — they stampeded", elapsed)
	}
}

func TestPacerZeroRateDisablesPacing(t *testing.T) {
	// A paid tier with headroom should pay nothing for this.
	p := NewPacer(0)
	start := time.Now()
	for i := 0; i < 100; i++ {
		if err := p.Wait(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Fatalf("disabled pacer still waited %v", elapsed)
	}
}

func TestPacerNilIsSafe(t *testing.T) {
	// A Summarizer built without a pacer must not panic.
	var p *Pacer
	if err := p.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestPacerHonoursContextCancellation(t *testing.T) {
	// A slot 12s away must not hold a worker past a shutdown signal.
	p := NewPacer(1) // one per minute
	if err := p.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	start := time.Now()
	if err := p.Wait(ctx); err == nil {
		t.Fatal("expected the cancelled context to abort the wait")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("cancellation took %v to take effect", elapsed)
	}
}
