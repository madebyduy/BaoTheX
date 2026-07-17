package jobs

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"repwire/internal/domain"
	"repwire/internal/ingest"
	"repwire/internal/postgres"
	"repwire/internal/process"
	"repwire/internal/sportsdata"
)

// Scheduler enqueues periodic work: source fetches, digest sends, rescoring,
// and reaps stuck jobs.
type Scheduler struct {
	db      *postgres.DB
	enqueue *Enqueuer
	log     *slog.Logger

	// translateMinScore is the floor for spending a translation call on an
	// arriving foreign article; see postgres.IDsPendingTranslation.
	translateMinScore float64
	// translateMaxAge is the age past which a foreign article is abandoned
	// rather than queued, so the backlog can never outlive the news in it.
	translateMaxAge time.Duration
	// editorialStartHour is the hour, Vietnam time, from which the desk may
	// commit to a story. Before it, ranking a morning's headlines would mistake
	// noise for heat.
	editorialStartHour int
	// picksPerDay caps how many stories the desk commits to in a day.
	picksPerDay int
	sportsSync  *sportsdata.Syncer
	llmEnabled  bool
	ttsEnabled  bool
}

// NewScheduler constructs a Scheduler.
func NewScheduler(db *postgres.DB, enqueue *Enqueuer, log *slog.Logger, translateMinScore float64, translateMaxAge time.Duration, editorialStartHour, picksPerDay int, sportsSync *sportsdata.Syncer) *Scheduler {
	if picksPerDay < 1 {
		picksPerDay = 1
	}
	return &Scheduler{
		db:                 db,
		enqueue:            enqueue,
		log:                log,
		translateMinScore:  translateMinScore,
		translateMaxAge:    translateMaxAge,
		editorialStartHour: editorialStartHour,
		picksPerDay:        picksPerDay,
		sportsSync:         sportsSync,
		llmEnabled:         true,
		ttsEnabled:         true,
	}
}

// SetGenerationCapabilities prevents a zero-budget worker from even enqueueing
// jobs that another worker with paid credentials could accidentally consume.
func (s *Scheduler) SetGenerationCapabilities(llmEnabled, ttsEnabled bool) {
	s.llmEnabled = llmEnabled
	s.ttsEnabled = ttsEnabled
}

// Run starts all scheduling loops until ctx is done.
func (s *Scheduler) Run(ctx context.Context) {
	go s.loop(ctx, "fetch", time.Minute, s.enqueueDueFetches)
	go s.loop(ctx, "digest", 15*time.Minute, s.enqueueDigests)
	go s.loop(ctx, "rescore", time.Hour, s.enqueueRescore)
	go s.loop(ctx, "cluster", 15*time.Minute, s.clusterStories)
	go s.loop(ctx, "media", time.Hour, s.backfillMedia)
	go s.loop(ctx, "reclean", time.Hour, s.recleanBodies)
	go s.loop(ctx, "analysis-candidates", time.Hour, s.refreshAnalysisCandidates)
	go s.loop(ctx, "reaper", 5*time.Minute, s.reapStuck)
	if s.llmEnabled {
		go s.loop(ctx, "translate", 10*time.Minute, s.enqueueTranslations)
		go s.loop(ctx, "editorial-pick", time.Hour, s.pickHotTopics)
	}
	if s.ttsEnabled {
		go s.loop(ctx, "audio", time.Hour, s.enqueueDailyAudio)
		go s.loop(ctx, "premium-audio-delivery", 15*time.Minute, s.enqueuePremiumAudioBriefs)
	}
	if s.sportsSync != nil && s.sportsSync.Enabled() {
		go s.loop(ctx, "sports-data", 15*time.Minute, s.sportsSync.Sync)
	}
	go func() {
		if err := s.recleanBodies(ctx); err != nil {
			s.log.Error("initial body reclean failed", "err", err)
		}
	}()
	// Run the audio check explicitly at startup as well as inside its regular
	// loop. This makes a late-starting local worker catch up with the 06:00/20:00
	// edition instead of waiting for the first hourly tick.
	if s.ttsEnabled {
		go func() {
			if err := s.enqueueDailyAudio(ctx); err != nil {
				s.log.Error("initial audio schedule failed", "err", err)
			}
		}()
	}
	s.log.Info("scheduler started")
	<-ctx.Done()
	s.log.Info("scheduler stopped")
}

// enqueuePremiumAudioBriefs turns the two public audio editions into a daily
// appointment for connected Premium members. It runs in Vietnam time because
// the editorial schedule is fixed at 06:00 and 20:00 ICT.
func (s *Scheduler) enqueuePremiumAudioBriefs(ctx context.Context) error {
	now := time.Now().In(vietnamTime())
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
	now := time.Now().In(vietnamTime())
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

// refreshAnalysisCandidates keeps the admin desk's ranking current. It scores
// and stores candidates but never drafts: drafting is a bounded, deliberate
// decision taken by pickHotTopics — a few stories a day, spaced apart — not a
// standing order to write about whatever clears a bar this hour.
//
// It scores through the same scoreContenders as the pick. That is the point —
// the desk must rank stories the way the newsroom actually chooses them, or the
// number an editor reads is not the number that decided anything.
func (s *Scheduler) refreshAnalysisCandidates(ctx context.Context) error {
	contenders, err := s.db.Analysis.HotTopicContenders(ctx, 24*time.Hour, 60)
	if err != nil {
		return err
	}
	if len(contenders) == 0 {
		return nil
	}
	count, err := s.db.Analysis.UpsertCandidates(ctx, scoreContenders(contenders))
	if err != nil {
		return err
	}
	if count > 0 {
		s.log.Info("analysis desk refreshed", "candidates", count)
	}
	return nil
}

// scoreContenders runs every cluster through process.ClusterHeat. It is the one
// place a story's heat is decided; both the desk ranking and the daily pick read
// from it.
func scoreContenders(contenders []domain.HotTopicCluster) []postgres.DailyPick {
	out := make([]postgres.DailyPick, 0, len(contenders))
	for _, c := range contenders {
		heat, signals := process.ClusterHeat(process.ClusterHeatInput{
			Titles:         c.Titles,
			SourceCount:    c.SourceCount,
			QualitySources: c.QualitySources,
			Velocity6h:     c.Velocity6h,
			FollowerWeight: c.FollowerWeight,
		})
		out = append(out, postgres.DailyPick{
			ClusterID:   c.ClusterID,
			Heat:        heat,
			Controversy: signals.Controversy,
			Action:      signals.Action,
			Terms:       signals.Terms,
			Cluster:     c,
		})
	}
	return out
}

// editorialPickGap is the minimum time between two editorial commitments.
//
// It is what "spread across the day" actually means here. The scheduler ticks
// every minute, and picksPerDay alone would let it claim the whole day's quota
// in the first three ticks after editorialStartHour — three articles at 09:01
// and silence until tomorrow, which is the old once-a-day problem wearing a
// bigger number. The gap also lets the ranking see a changed world between
// picks: three hours of ingestion is long enough for a different story to
// become the hottest one.
const editorialPickGap = 3 * time.Hour

// pickHotTopics commits the newsroom to the day's stories worth an analysis.
//
// It does nothing before editorialStartHour, then claims at most picksPerDay
// stories, at most one per editorialPickGap, always the hottest cluster still
// unclaimed. Everything before a pick is free — clustering, counting sources,
// matching controversy words — and the LLM is spent only once a winner exists.
//
// This used to pick exactly one story, at 21:00, and the reasoning was explicit:
// it "commits its LLM budget to the single hottest story of the day". That was
// written while a bug billed Gemini calls at Anthropic's prices against a $0.50
// ceiling, so the budget really was down to its last few calls and one article
// really was all it could buy. The meter was wrong — the calls cost nothing —
// and the caution outlived the emergency that justified it. Meanwhile 18
// clusters a day carry the three translated sources an analysis needs, 22 sit
// proposed, and whole hours pass using six calls of a 240-call allowance.
//
// Waiting for the evening had a real argument behind it too: a story that broke
// at noon has had a day to be corroborated, and only by then can the ranking
// tell a genuine controversy from a headline that was merely loud at 9am. The
// gap is what preserves that argument — heat is still measured over 24 hours,
// and a story that fades between picks simply stops winning.
func (s *Scheduler) pickHotTopics(ctx context.Context) error {
	now := time.Now().In(vietnamTime())
	if now.Hour() < s.editorialStartHour {
		return nil
	}
	picked, lastPick, err := s.db.Analysis.PicksForDate(ctx, now)
	if err != nil {
		return err
	}
	if picked >= s.picksPerDay {
		return nil // the day's editorial quota is spent
	}
	if !lastPick.IsZero() && now.Sub(lastPick) < editorialPickGap {
		return nil // too soon after the last one
	}

	contenders, err := s.db.Analysis.HotTopicContenders(ctx, 24*time.Hour, 60)
	if err != nil {
		return err
	}
	if len(contenders) == 0 {
		s.log.Info("daily pick skipped: no contending clusters", "date", now.Format("2006-01-02"))
		return nil
	}

	best, ok := rankHotTopics(contenders)
	if !ok {
		s.log.Info("daily pick skipped: no cluster cleared the bar",
			"date", now.Format("2006-01-02"), "contenders", len(contenders))
		return nil
	}

	if err := s.db.Analysis.ClaimDailyPick(ctx, best, now); err != nil {
		if err == domain.ErrNotFound {
			return nil // another worker claimed it, or it is already published
		}
		return err
	}
	if err := s.enqueue.EnqueueGenerateAnalysis(ctx, best.ClusterID); err != nil {
		_ = s.db.Analysis.MarkFailed(ctx, best.ClusterID, err)
		return err
	}
	s.log.Info("hot topic picked",
		"date", now.Format("2006-01-02"),
		"pick", picked+1,
		"of", s.picksPerDay,
		"cluster", best.ClusterID,
		"title", best.Cluster.RepresentativeTitle,
		"heat", best.Heat,
		"controversy", best.Controversy,
		"terms", best.Terms,
		"quality_sources", best.Cluster.QualitySources,
		"velocity_6h", best.Cluster.Velocity6h)
	return nil
}

// minPickHeat is the floor below which a story is not worth an analysis.
//
// The principle behind it stands and is the reason there is a floor at all: a
// quiet day should produce no piece rather than a forced one. Two mid-quality
// outlets reporting the same routine result clears neither the corroboration nor
// the controversy bar, and writing about it anyway is how an opinion section
// loses its reason to exist.
//
// The number was 45, which enforced something stricter than that principle. On
// an ordinary day the hottest unclaimed cluster reaches about 42, so 45 did not
// mean "skip the quiet days" — it meant "skip every day but the extraordinary
// one". The three pieces ever written scored 122, 67 and 59; everything else
// bunched between 30 and 42 and was refused. Part of why it sat that high was
// the same fiction that shaped the rest of the desk: the floor guarded "the
// day's budget", and that budget was a pricing bug reading $0.91 against $0.50
// on a real spend of nothing.
//
// 40 is where the principle actually lands for a sports paper. It still refuses
// 21 of 27 current clusters. What it stops refusing is the story with three
// high-quality sources that simply is not a scandal — a Tour de France hat-trick,
// the World Cup final line-up — because sport mostly is not scandal, and a desk
// that only writes when someone is angry is not covering the sport.
const minPickHeat = 40

// rankHotTopics returns the highest-scoring contender, or ok=false when nothing
// clears minPickHeat. It shares scoreContenders with the desk refresh so
// the two can never drift onto different scales.
func rankHotTopics(contenders []domain.HotTopicCluster) (postgres.DailyPick, bool) {
	var best postgres.DailyPick
	found := false
	for _, p := range scoreContenders(contenders) {
		if p.Heat < minPickHeat {
			continue
		}
		if found && p.Heat <= best.Heat {
			continue
		}
		best = p
		found = true
	}
	return best, found
}

// vietnamTime returns the newsroom's timezone, falling back to a fixed offset on
// hosts without tzdata.
func vietnamTime() *time.Location {
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		return time.FixedZone("ICT", 7*60*60)
	}
	return loc
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
	// Ask for far fewer than the old 50: the hourly LLM allowance is the real
	// ceiling, and queueing more than it can drain only builds a backlog that
	// crowds out the editorial desk. The score and age floors keep this to
	// stories that are both worth translating and still current.
	ids, err := s.db.Content.IDsPendingTranslation(ctx, 12, s.translateMinScore, s.translateMaxAge)
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
	// The weekly research digest used to be enqueued here. Migration 0024
	// disabled every Europe PMC source, so from that day BuildWeekly could only
	// ever return "nothing to send" — the scheduler woke, queried, and found the
	// same empty week, forever. A digest with no possible content is not a
	// feature that is quiet; it is one that is over.
	//
	// Follow alerts are the one digest with no hour of its own: they answer to
	// the news, not to the clock. UsersDueForFollowAlert has already excluded
	// anyone inside quiet hours or inside the cooldown, and the handler sends
	// nothing when there is nothing worth sending, so the common outcome of this
	// loop is a cheap no-op.
	alerts, err := s.db.Telegram.UsersDueForFollowAlert(ctx)
	if err != nil {
		return err
	}
	now := time.Now()
	for _, uid := range alerts {
		if err := s.enqueue.EnqueueFollowAlert(ctx, uid, now); err != nil {
			s.log.Error("enqueue follow alert failed", "user", uid, "err", err)
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
