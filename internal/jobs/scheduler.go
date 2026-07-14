package jobs

import (
	"context"
	"log/slog"
	"time"

	"repwire/internal/domain"
	"repwire/internal/postgres"
)

// Scheduler enqueues periodic work: source fetches, digest sends, rescoring,
// and reaps stuck jobs.
type Scheduler struct {
	db      *postgres.DB
	enqueue *Enqueuer
	log     *slog.Logger
}

// NewScheduler constructs a Scheduler.
func NewScheduler(db *postgres.DB, enqueue *Enqueuer, log *slog.Logger) *Scheduler {
	return &Scheduler{db: db, enqueue: enqueue, log: log}
}

// Run starts all scheduling loops until ctx is done.
func (s *Scheduler) Run(ctx context.Context) {
	go s.loop(ctx, "fetch", time.Minute, s.enqueueDueFetches)
	go s.loop(ctx, "digest", 15*time.Minute, s.enqueueDigests)
	go s.loop(ctx, "rescore", time.Hour, s.enqueueRescore)
	go s.loop(ctx, "reaper", 5*time.Minute, s.reapStuck)
	s.log.Info("scheduler started")
	<-ctx.Done()
	s.log.Info("scheduler stopped")
}

// loop runs fn immediately (after a short delay) and then every interval.
func (s *Scheduler) loop(ctx context.Context, name string, interval time.Duration, fn func(context.Context) error) {
	// Small initial stagger so all loops don't fire at once on boot.
	select {
	case <-ctx.Done():
		return
	case <-time.After(5 * time.Second):
	}
	run := func() {
		if err := fn(ctx); err != nil {
			s.log.Error("scheduler task failed", "task", name, "err", err)
		}
	}
	run()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}

// enqueueDueFetches enqueues a fetch job for every source past its interval.
func (s *Scheduler) enqueueDueFetches(ctx context.Context) error {
	sources, err := s.db.Source.DueForFetch(ctx)
	if err != nil {
		return err
	}
	for _, src := range sources {
		kind := fetchKindForSource(src.Kind)
		if kind == "" {
			continue
		}
		if err := s.enqueue.EnqueueFetch(ctx, kind, src.ID); err != nil {
			s.log.Error("enqueue fetch failed", "source", src.ID, "err", err)
		}
	}
	if len(sources) > 0 {
		s.log.Info("enqueued fetches", "count", len(sources))
	}
	return nil
}

// enqueueDigests enqueues daily/weekly digests for users whose local time matches.
func (s *Scheduler) enqueueDigests(ctx context.Context) error {
	daily, err := s.db.Telegram.UsersDueForDaily(ctx)
	if err != nil {
		return err
	}
	for _, uid := range daily {
		if err := s.enqueue.EnqueueSendDaily(ctx, uid, false); err != nil {
			s.log.Error("enqueue daily failed", "user", uid, "err", err)
		}
	}
	weekly, err := s.db.Telegram.UsersDueForWeekly(ctx)
	if err != nil {
		return err
	}
	for _, uid := range weekly {
		if err := s.enqueue.EnqueueSendWeekly(ctx, uid); err != nil {
			s.log.Error("enqueue weekly failed", "user", uid, "err", err)
		}
	}
	return nil
}

// enqueueRescore re-scores a batch of items so freshness decay is applied.
func (s *Scheduler) enqueueRescore(ctx context.Context) error {
	ids, err := s.db.Content.IDsToRescore(ctx, 200)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := s.enqueue.EnqueueScore(ctx, id); err != nil {
			s.log.Error("enqueue score failed", "content", id, "err", err)
		}
	}
	return nil
}

// reapStuck resets jobs whose worker crashed mid-run.
func (s *Scheduler) reapStuck(ctx context.Context) error {
	n, err := s.db.Job.ReapStuck(ctx, 15*time.Minute)
	if err != nil {
		return err
	}
	if n > 0 {
		s.log.Warn("reaped stuck jobs", "count", n)
	}
	return nil
}

// fetchKindForSource maps a source kind to its fetch job kind.
func fetchKindForSource(k domain.SourceKind) string {
	switch k {
	case domain.SourceRSS:
		return domain.JobFetchRSS
	case domain.SourceYouTube:
		return domain.JobFetchYouTube
	case domain.SourceEuropePMC:
		return domain.JobFetchPMC
	case domain.SourcePodcastRSS:
		return domain.JobFetchPodcast
	default:
		return ""
	}
}
