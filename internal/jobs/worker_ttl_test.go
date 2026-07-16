package jobs

import (
	"testing"
	"time"

	"repwire/internal/domain"
)

func TestAudioGetsLongerThanTheDefaultTTL(t *testing.T) {
	// This is the bug that kept the audio brief from ever finishing. A five-minute
	// edition is several narration chunks, each of which a rate-limited endpoint
	// may tell to wait 45 seconds before starting. The job cannot complete inside
	// five minutes, so every edition died on "context deadline exceeded" — a
	// deadline sized for fetching one article, applied to rendering an audio
	// programme.
	if got := jobTTLFor(domain.JobGenerateAudio); got <= defaultJobTTL {
		t.Fatalf("audio TTL %v must exceed the default %v", got, defaultJobTTL)
	}
}

func TestAnalysisGetsLongerThanTheDefaultTTL(t *testing.T) {
	// The daily piece translates up to six sources and then makes two more calls,
	// all on a paced endpoint. Same shape of problem.
	if got := jobTTLFor(domain.JobGenerateAnalysis); got <= defaultJobTTL {
		t.Fatalf("analysis TTL %v must exceed the default %v", got, defaultJobTTL)
	}
}

func TestLongJobsStayUnderTheReaper(t *testing.T) {
	// scheduler.reapStuck resets anything running past 15 minutes. A TTL at or
	// above that races the reaper: the job would be re-queued while still running,
	// and two workers would render the same edition.
	const reaperAfter = 15 * time.Minute
	if longJobTTL >= reaperAfter {
		t.Fatalf("longJobTTL %v must stay under the %v reaper", longJobTTL, reaperAfter)
	}
}

func TestOrdinaryJobsKeepTheShortTTL(t *testing.T) {
	// Fetching or scoring one item is seconds of work; a long deadline there
	// would let a wedged job hold a worker slot for twelve minutes.
	for _, kind := range []string{
		domain.JobFetchRSS, domain.JobTranslate, domain.JobSummarize,
		domain.JobScore, domain.JobProcessContent, domain.JobSendDaily,
	} {
		if got := jobTTLFor(kind); got != defaultJobTTL {
			t.Fatalf("kind %q got TTL %v, want the default %v", kind, got, defaultJobTTL)
		}
	}
}

func TestUnknownKindGetsTheDefaultTTL(t *testing.T) {
	if got := jobTTLFor("something_new"); got != defaultJobTTL {
		t.Fatalf("unknown kind got %v, want %v", got, defaultJobTTL)
	}
}
