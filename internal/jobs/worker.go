package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"repwire/internal/briefmedia"
	"repwire/internal/domain"
	"repwire/internal/postgres"
	"repwire/internal/process"
)

// HandlerFunc processes a single job.
type HandlerFunc func(ctx context.Context, j *domain.Job) error

// defaultJobTTL bounds a normal job. Fetching, scoring or digesting one article
// is a handful of seconds; five minutes means something is wedged.
const defaultJobTTL = 5 * time.Minute

// longJobTTL covers work that is slow by design rather than by fault.
//
// Rendering an audio brief is several narration requests against a rate-limited
// endpoint: a five-minute edition is three or more chunks, each of which may be
// told to wait 45 seconds before it is allowed to start. That job cannot finish
// inside defaultJobTTL, and it never did — every edition was killed at five
// minutes with "context deadline exceeded", which read like a hang and was
// actually a deadline set for a different kind of work. It stays under the
// 15-minute reaper so a genuinely stuck render is still collected.
const longJobTTL = 12 * time.Minute

// jobTTLFor returns how long a job of the given kind may run.
func jobTTLFor(kind string) time.Duration {
	switch kind {
	case domain.JobGenerateAudio, domain.JobGenerateAnalysis:
		return longJobTTL
	default:
		return defaultJobTTL
	}
}

// Worker polls the queue and dispatches jobs to registered handlers.
type Worker struct {
	id       string
	queue    *postgres.JobRepo
	handlers map[string]HandlerFunc
	log      *slog.Logger
}

// NewWorker constructs a Worker.
func NewWorker(id string, queue *postgres.JobRepo, handlers map[string]HandlerFunc, log *slog.Logger) *Worker {
	return &Worker{
		id:       id,
		queue:    queue,
		handlers: handlers,
		log:      log,
	}
}

// Run polls for jobs and runs up to `concurrency` at once until ctx is done.
func (w *Worker) Run(ctx context.Context, concurrency int) {
	if concurrency < 1 {
		concurrency = 1
	}
	sem := make(chan struct{}, concurrency)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	w.log.Info("worker started", "id", w.id, "concurrency", concurrency)
	for {
		select {
		case <-ctx.Done():
			// Graceful shutdown: wait for in-flight jobs to finish.
			for i := 0; i < concurrency; i++ {
				sem <- struct{}{}
			}
			w.log.Info("worker stopped", "id", w.id)
			return
		case <-ticker.C:
			// Drain as many jobs as we have free slots for this tick.
			for {
				select {
				case sem <- struct{}{}:
				default:
					goto nextTick
				}
				job, err := w.queue.Dequeue(ctx, w.id)
				if errors.Is(err, postgres.ErrNoJob) {
					<-sem
					goto nextTick
				}
				if err != nil {
					<-sem
					w.log.Error("dequeue failed", "err", err)
					goto nextTick
				}
				go w.run(ctx, job, sem)
			}
		nextTick:
		}
	}
}

func (w *Worker) run(ctx context.Context, job *domain.Job, sem chan struct{}) {
	start := time.Now()
	defer func() { <-sem }()
	defer func() {
		if r := recover(); r != nil {
			w.fail(ctx, job, fmt.Errorf("panic: %v", r))
		}
	}()

	handler, ok := w.handlers[job.Kind]
	if !ok {
		w.fail(ctx, job, fmt.Errorf("no handler for kind %q", job.Kind))
		return
	}

	jobCtx, cancel := context.WithTimeout(ctx, jobTTLFor(job.Kind))
	defer cancel()

	if err := handler(jobCtx, job); err != nil {
		w.log.Error("job failed", "id", job.ID, "kind", job.Kind,
			"attempt", job.Attempts, "duration_ms", time.Since(start).Milliseconds(), "err", err)
		w.fail(ctx, job, err)
		return
	}
	stateCtx, stateCancel := jobStateContext(ctx)
	defer stateCancel()
	if err := w.queue.Done(stateCtx, job.ID, w.id); err != nil {
		level := slog.LevelError
		if errors.Is(err, postgres.ErrJobLeaseLost) {
			level = slog.LevelWarn
		}
		w.log.Log(stateCtx, level, "mark done failed", "id", job.ID, "err", err)
	}
	w.log.Info("job done", "id", job.ID, "kind", job.Kind,
		"attempt", job.Attempts, "duration_ms", time.Since(start).Milliseconds())
}

// Retry delays, sized to how long the thing that refused actually stays refusing.
const (
	// budgetRetryAfter waits out a spend ceiling. Money does not come back within
	// the hour, so there is no point asking again soon.
	budgetRetryAfter = 6 * time.Hour
	// rateLimitRetryAfter waits out a throughput limit. An hourly cap rolls
	// within the hour and a per-minute limit clears in seconds.
	//
	// This used to be 6 hours as well, on the theory that a rejection meant the
	// day's quota was gone. It rarely did: free-tier Gemini answers "retry in
	// 45s" far more often than it answers "come back tomorrow", and the audio
	// brief paid for the confusion. A 05:00 edition that hit one rate limit was
	// parked until 11:00, and because the job kept its dedup key while pending,
	// the scheduler's hourly attempt to re-enqueue was swallowed as a duplicate.
	// One transient 429 cost the whole morning.
	rateLimitRetryAfter = 15 * time.Minute
)

// retryDelay decides when a failed job may run again, based on what refused it.
func retryDelay(attempts int, cause error) time.Duration {
	switch {
	case errors.Is(cause, process.ErrBudgetExceeded):
		return budgetRetryAfter
	case errors.Is(cause, process.ErrHourlyCapReached), errors.Is(cause, briefmedia.ErrQuotaExceeded):
		return rateLimitRetryAfter
	default:
		return backoff(attempts)
	}
}

func (w *Worker) fail(ctx context.Context, job *domain.Job, cause error) {
	stateCtx, cancel := jobStateContext(ctx)
	defer cancel()
	if err := w.queue.Fail(stateCtx, job, w.id, cause, retryDelay(job.Attempts, cause)); err != nil {
		level := slog.LevelError
		if errors.Is(err, postgres.ErrJobLeaseLost) {
			level = slog.LevelWarn
		}
		w.log.Log(stateCtx, level, "mark fail failed", "id", job.ID, "err", err)
	}
}

// State transitions must survive cancellation of the job context during a
// graceful shutdown; otherwise the row remains running until the next reaper.
func jobStateContext(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(parent), 10*time.Second)
}
