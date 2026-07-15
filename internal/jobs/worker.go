package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"repwire/internal/domain"
	"repwire/internal/postgres"
	"repwire/internal/process"
)

// HandlerFunc processes a single job.
type HandlerFunc func(ctx context.Context, j *domain.Job) error

// Worker polls the queue and dispatches jobs to registered handlers.
type Worker struct {
	id       string
	queue    *postgres.JobRepo
	handlers map[string]HandlerFunc
	log      *slog.Logger
	jobTTL   time.Duration
}

// NewWorker constructs a Worker.
func NewWorker(id string, queue *postgres.JobRepo, handlers map[string]HandlerFunc, log *slog.Logger) *Worker {
	return &Worker{
		id:       id,
		queue:    queue,
		handlers: handlers,
		log:      log,
		jobTTL:   5 * time.Minute,
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

	jobCtx, cancel := context.WithTimeout(ctx, w.jobTTL)
	defer cancel()

	if err := handler(jobCtx, job); err != nil {
		w.log.Error("job failed", "id", job.ID, "kind", job.Kind,
			"attempt", job.Attempts, "duration_ms", time.Since(start).Milliseconds(), "err", err)
		w.fail(ctx, job, err)
		return
	}
	if err := w.queue.Done(ctx, job.ID); err != nil {
		w.log.Error("mark done failed", "id", job.ID, "err", err)
	}
	w.log.Info("job done", "id", job.ID, "kind", job.Kind,
		"attempt", job.Attempts, "duration_ms", time.Since(start).Milliseconds())
}

func (w *Worker) fail(ctx context.Context, job *domain.Job, cause error) {
	retryAfter := backoff(job.Attempts)
	if errors.Is(cause, process.ErrBudgetExceeded) {
		// The daily quota does not recover after a few minutes. A long delay
		// prevents a depleted key from starving fetch, scoring and media jobs.
		retryAfter = 6 * time.Hour
	}
	if err := w.queue.Fail(ctx, job, cause, retryAfter); err != nil {
		w.log.Error("mark fail failed", "id", job.ID, "err", err)
	}
}
