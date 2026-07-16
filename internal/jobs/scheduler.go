package jobs

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"repwire/internal/domain"
	"repwire/internal/ingest"
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
	go s.loop(ctx, "translate", 10*time.Minute, s.enqueueTranslations)
	go s.loop(ctx, "cluster", 15*time.Minute, s.clusterStories)
	go s.loop(ctx, "media", time.Hour, s.backfillMedia)
	go s.loop(ctx, "reclean", time.Hour, s.recleanBodies)
	go s.loop(ctx, "audio", time.Hour, s.enqueueDailyAudio)
	go s.loop(ctx, "premium-audio-delivery", 15*time.Minute, s.enqueuePremiumAudioBriefs)
	go s.loop(ctx, "analysis-candidates", time.Hour, s.refreshAnalysisCandidates)
	go s.loop(ctx, "reaper", 5*time.Minute, s.reapStuck)
	go func() {
		if err := s.recleanBodies(ctx); err != nil {
			s.log.Error("initial body reclean failed", "err", err)
		}
	}()
	// Run the audio check explicitly at startup as well as inside its regular
	// loop. This makes a late-starting local worker catch up with the 06:00/20:00
	// edition instead of waiting for the first hourly tick.
	go func() {
		if err := s.enqueueDailyAudio(ctx); err != nil {
			s.log.Error("initial audio schedule failed", "err", err)
		}
	}()
	s.log.Info("scheduler started")
	<-ctx.Done()
	s.log.Info("scheduler stopped")
}

// enqueuePremiumAudioBriefs turns the two public audio editions into a daily
// appointment for connected Premium members. It runs in Vietnam time because
// the editorial schedule is fixed at 06:00 and 20:00 ICT.
func (s *Scheduler) enqueuePremiumAudioBriefs(ctx context.Context) error {
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		loc = time.FixedZone("ICT", 7*60*60)
	}
	now := time.Now().In(loc)
	editions := []struct {
		name string
		hour int
	}{{"morning", 6}, {"evening", 20}}
	for _, edition := range editions {
		if now.Hour() < edition.hour {
			continue
		}
		brief, err := s.db.Engagement.AudioBriefForDate(ctx, now, edition.name)
		if err != nil {
			if err == domain.ErrNotFound {
				continue
			}
			return err
		}
		if brief.DurationSeconds == nil || *brief.DurationSeconds < 180 {
			continue
		}
		users, err := s.db.Telegram.PremiumUsersForAudioBrief(ctx, edition.name)
		if err != nil {
			return err
		}
		for _, userID := range users {
			if err := s.enqueue.EnqueueSendPremiumBrief(ctx, userID, now, edition.name); err != nil {
				s.log.Error("enqueue premium audio failed", "user", userID, "edition", edition.name, "err", err)
			}
		}
	}
	return nil
}

func (s *Scheduler) enqueueDailyAudio(ctx context.Context) error {
	// The newsroom appointments are fixed to Vietnam time regardless of the
	// server's deployment timezone. Because scheduler loops run immediately on
	// startup, this also creates a missed edition when a worker starts late.
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		loc = time.FixedZone("ICT", 7*60*60)
	}
	now := time.Now().In(loc)
	editions := []struct {
		name string
		hour int
	}{{"morning", 5}, {"evening", 19}}
	for _, edition := range editions {
		if now.Hour() < edition.hour {
			s.log.Info("audio edition not due", "edition", edition.name, "date", now.Format("2006-01-02"), "hour", now.Hour())
			continue
		}
		exists, err := s.db.Engagement.HasAudioBriefDate(ctx, now, edition.name)
		if err != nil {
			return err
		}
		s.log.Info("audio edition check", "edition", edition.name, "date", now.Format("2006-01-02"), "ready", exists)
		if !exists {
			if err := s.enqueue.EnqueueGenerateAudio(ctx, now, edition.name); err != nil {
				return err
			}
			s.log.Info("audio generation enqueued", "edition", edition.name, "date", now.Format("2006-01-02"))
		}
	}
	return nil
}

func (s *Scheduler) refreshAnalysisCandidates(ctx context.Context) error {
	count, err := s.db.Analysis.RefreshCandidates(ctx, 5)
	if err != nil {
		return err
	}
	if count > 0 {
		s.log.Info("analysis desk refreshed", "candidates", count)
	}
	// Continuously draft the top proposed candidates so "Góc nhìn BaoTheX"
	// pieces are written automatically. Each draft still lands in needs_review
	// for an editor to approve before it is published — generation is automatic,
	// publishing stays gated. Daily LLM budget caps the real spend.
	return s.enqueueAnalysisDrafts(ctx, 3)
}

// enqueueAnalysisDrafts picks the highest-scoring proposed candidates and
// enqueues a draft-generation job for each (deduped per cluster).
func (s *Scheduler) enqueueAnalysisDrafts(ctx context.Context, max int) error {
	candidates, err := s.db.Analysis.ListCandidates(ctx, "proposed", max)
	if err != nil {
		return err
	}
	for _, c := range candidates {
		// Claim the candidate first; if another run already took it, skip.
		if err := s.db.Analysis.MarkDrafting(ctx, c.ClusterID); err != nil {
			if err == domain.ErrNotFound {
				continue
			}
			s.log.Error("analysis mark drafting failed", "cluster", c.ClusterID, "err", err)
			continue
		}
		if err := s.enqueue.EnqueueGenerateAnalysis(ctx, c.ClusterID); err != nil {
			_ = s.db.Analysis.MarkFailed(ctx, c.ClusterID, err)
			s.log.Error("enqueue analysis draft failed", "cluster", c.ClusterID, "err", err)
			continue
		}
		s.log.Info("analysis draft enqueued", "cluster", c.ClusterID)
	}
	return nil
}

// recleanBodies gradually re-cleans a batch of previously-ingested article
// bodies through the current boilerplate filter, stripping trailing tag lists,
// "đọc nhiều" rails and sponsored blocks. Pure text work — no LLM, no network.
func (s *Scheduler) recleanBodies(ctx context.Context) error {
	rows, err := s.db.Content.BodiesNeedingRecleanWide(ctx, 100)
	if err != nil {
		return err
	}
	cleaned := 0
	for _, row := range rows {
		vi := ingest.TrimTrailingBoilerplate(row.Vietnamese)
		orig := ingest.TrimTrailingBoilerplate(row.Original)
		if vi == row.Vietnamese && orig == row.Original {
			continue
		}
		// Safety: never blank out a body that previously had content.
		if strings.TrimSpace(orig) == "" {
			orig = row.Original
		}
		if row.Vietnamese != "" && strings.TrimSpace(vi) == "" {
			vi = row.Vietnamese
		}
		if err := s.db.Content.UpdateBodyText(ctx, row.ContentID, vi, orig); err != nil {
			s.log.Error("reclean body failed", "content", row.ContentID, "err", err)
			continue
		}
		cleaned++
	}
	if cleaned > 0 {
		s.log.Info("recleaned article bodies", "count", cleaned)
	}
	return nil
}

func (s *Scheduler) backfillMedia(ctx context.Context) error {
	targets, err := s.db.Content.MissingImageTargets(ctx, 6)
	if err != nil {
		return err
	}
	fetcher := ingest.NewRSSFetcher(nil)
	updated := 0
	for _, target := range targets {
		body, imageURL := fetcher.FetchArticleData(ctx, target.URL)
		if imageURL == "" {
			continue
		}
		if err := s.db.Content.BackfillMediaByID(ctx, target.ID, imageURL, body); err != nil {
			s.log.Error("media backfill failed", "content", target.ID, "err", err)
			continue
		}
		updated++
	}
	if updated > 0 {
		s.log.Info("backfilled article media", "count", updated)
	}
	return nil
}

func (s *Scheduler) clusterStories(ctx context.Context) error {
	ids, err := s.db.Content.IDsWithoutCluster(ctx, 100)
	if err != nil {
		return err
	}
	for _, id := range ids {
		item, err := s.db.Content.Get(ctx, id)
		if err != nil {
			continue
		}
		if err := s.db.Content.ClusterContent(ctx, id, item.Title); err != nil {
			s.log.Error("cluster story failed", "content", id, "err", err)
		}
	}
	if len(ids) > 0 {
		s.log.Info("clustered stories", "count", len(ids))
	}
	return nil
}

func (s *Scheduler) enqueueTranslations(ctx context.Context) error {
	ids, err := s.db.Content.IDsPendingTranslation(ctx, 50)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := s.enqueue.EnqueueTranslate(ctx, id); err != nil {
			s.log.Error("enqueue translation failed", "content", id, "err", err)
		}
	}
	if len(ids) > 0 {
		s.log.Info("enqueued translations", "count", len(ids))
	}
	return nil
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
	ids, err := s.db.Content.IDsToRescore(ctx, 50)
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
