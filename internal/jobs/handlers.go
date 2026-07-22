package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"repwire/internal/briefmedia"
	"repwire/internal/domain"
	"repwire/internal/ingest"
	"repwire/internal/postgres"
	"repwire/internal/process"
	"repwire/internal/telegram"
)

// Handlers bundles every dependency the job handlers need.
type Handlers struct {
	DB      *postgres.DB
	Enqueue *Enqueuer
	Log     *slog.Logger

	RSS     *ingest.RSSFetcher
	YouTube *ingest.YouTubeFetcher
	PMC     *ingest.EuropePMCFetcher
	Podcast *ingest.PodcastFetcher

	Summarizer *process.Summarizer
	Telegram   *telegram.Client
	Digest     *telegram.Digest
	// TTS is the Gemini narrator, kept as a fallback for the quota-free Edge one.
	TTS *briefmedia.TTS
	// Edge is the primary narrator: Microsoft's free read-aloud endpoint, tried
	// before TTS so the audio brief stops leaning on the limited Gemini quota.
	Edge *briefmedia.EdgeTTS
	// Google is the middle narrator: Google Translate's free voice, which works
	// from IPs where Edge's endpoint is blocked. Tried after Edge, before Gemini.
	Google        *briefmedia.GoogleTTS
	MediaDir      string
	PublicBaseURL string

	// ScoreThreshold: only items with base_score >= this get summarized.
	ScoreThreshold float64
	// TranslateMinScore: only foreign articles with base_score >= this are
	// translated on arrival. Everything below is parked, clustered but unspent.
	// Both thresholds are compared against process.BaseScore and must stay below
	// process.MaxArticleScore; config.Load enforces that.
	TranslateMinScore float64
}

// Register returns the kind→handler map for the worker.
func (h *Handlers) Register() map[string]HandlerFunc {
	return map[string]HandlerFunc{
		domain.JobFetchRSS:            h.handleFetch,
		domain.JobFetchYouTube:        h.handleFetch,
		domain.JobFetchPMC:            h.handleFetch,
		domain.JobFetchPodcast:        h.handleFetch,
		domain.JobProcessContent:      h.handleProcess,
		domain.JobSummarize:           h.handleSummarize,
		domain.JobTranslate:           h.handleTranslate,
		domain.JobScore:               h.handleScore,
		domain.JobSendDaily:           h.handleSendDaily,
		domain.JobFollowAlert:         h.handleFollowAlert,
		domain.JobGenerateAudio:       h.handleGenerateAudio,
		domain.JobGenerateAnalysis:    h.handleGenerateAnalysis,
		domain.JobGeneratePerspective: h.handleGeneratePerspective,
		domain.JobSendPremiumBrief:    h.handleSendPremiumBrief,
	}
}

func (h *Handlers) handleTranslate(ctx context.Context, j *domain.Job) error {
	var p domain.ContentPayload
	if err := j.Unmarshal(&p); err != nil {
		return err
	}
	if h.Summarizer == nil || !h.Summarizer.Enabled() {
		return fmt.Errorf("translator: LLM_API_KEY not configured")
	}
	body, err := h.DB.Content.GetBody(ctx, p.ContentID)
	if errors.Is(err, domain.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if ingest.BlockedArticleText(body.OriginalBody) {
		return h.DB.Content.QuarantineBlockedArticle(ctx, p.ContentID)
	}
	// Report which ceiling actually refused: "hourly cap" and "daily budget" are
	// fixed in different places, and conflating them sends an operator hunting
	// through the wrong setting.
	if refusal, err := h.Summarizer.BudgetStatus(ctx); err != nil {
		return err
	} else if refusal != nil {
		return refusal
	}
	item, err := h.DB.Content.Get(ctx, p.ContentID)
	if err != nil {
		return err
	}
	// Digest rather than translate. Readers get a Vietnamese headline, the key
	// points and a few sentences of context, plus a link to the original; we do
	// not republish a Vietnamese copy of someone else's article. That is roughly
	// seven times cheaper and it is the difference between aggregating and
	// reproducing. The full translation still happens for exactly one cluster a
	// day, in the editorial path, and is never shown to readers.
	out, err := h.Summarizer.DigestForeign(ctx, item.Title, body.OriginalBody)
	if err != nil {
		return err
	}
	if !looksVietnamese(out.VietnameseTitle) || !looksVietnamese(out.Summary) {
		return h.DB.Content.SetStatus(ctx, p.ContentID, domain.StatusNeedsReview)
	}
	if err := h.DB.Content.SetForeignDigest(ctx, p.ContentID, out.VietnameseTitle); err != nil {
		return err
	}
	// Re-cluster on the Vietnamese headline now that one exists: the trigram
	// match runs on COALESCE(translated_title, title), so this can pull the piece
	// together with Vietnamese coverage of the same event.
	if err := h.DB.Content.ClusterContent(ctx, p.ContentID, out.VietnameseTitle); err != nil {
		return err
	}
	summary := out.Summary
	return h.DB.Content.SetSummary(ctx, p.ContentID, &summary, out.KeyPoints, domain.StatusReady)
}

// ---- fetch_* ----

func (h *Handlers) handleFetch(ctx context.Context, j *domain.Job) error {
	var p domain.FetchPayload
	if err := j.Unmarshal(&p); err != nil {
		return err
	}
	src, err := h.DB.Source.Get(ctx, p.SourceID)
	if err != nil {
		return err
	}

	var result *ingest.FetchResult
	switch j.Kind {
	case domain.JobFetchRSS:
		result, err = h.RSS.Fetch(ctx, src)
	case domain.JobFetchYouTube:
		result, err = h.YouTube.Fetch(ctx, src)
	case domain.JobFetchPMC:
		result, err = h.PMC.Fetch(ctx, src)
	case domain.JobFetchPodcast:
		result, err = h.Podcast.Fetch(ctx, src)
	default:
		return fmt.Errorf("unknown fetch kind %q", j.Kind)
	}
	if err != nil {
		_ = h.DB.Source.MarkError(ctx, src.ID, err.Error())
		return err
	}
	if result.NotModified {
		return h.DB.Source.MarkFetched(ctx, src.ID, nil, nil)
	}

	var stored int
	for _, raw := range result.Items {
		id, err := ingest.Store(ctx, h.DB, src, raw)
		if err != nil {
			h.Log.Error("store item failed", "source", src.ID, "title", raw.Title, "err", err)
			continue
		}
		if id != 0 {
			stored++
			if err := h.Enqueue.EnqueueProcess(ctx, id); err != nil {
				h.Log.Error("enqueue process failed", "content", id, "err", err)
			}
		}
	}
	h.Log.Info("fetched source", "source", src.ID, "name", src.Name, "new_items", stored)
	return h.DB.Source.MarkFetched(ctx, src.ID, result.ETag, result.LastModified)
}

// ---- process_content ----

func (h *Handlers) handleProcess(ctx context.Context, j *domain.Job) error {
	var p domain.ContentPayload
	if err := j.Unmarshal(&p); err != nil {
		return err
	}
	item, err := h.DB.Content.Get(ctx, p.ContentID)
	if errors.Is(err, domain.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	// Run the cheap, deterministic quality gate before classification and LLM
	// work. Persisting the result makes review routing explainable in admin and
	// lets a later body backfill pass when this job is safely re-run.
	var articleBody *domain.ContentBody
	qualityBody := ""
	if item.Type == domain.ContentArticle {
		articleBody, err = h.DB.Content.GetBody(ctx, item.ID)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			return err
		}
		if articleBody != nil {
			qualityBody = articleBody.OriginalBody
		}
	}
	quality := ingest.AssessQuality(item, qualityBody, time.Now())
	if err := h.DB.Content.SetQuality(ctx, item.ID, quality.State, quality.Flags); err != nil {
		return err
	}
	if quality.State == "review" {
		if quality.Has("blocked_article_text") {
			return h.DB.Content.QuarantineBlockedArticle(ctx, item.ID)
		}
		return h.DB.Content.SetStatus(ctx, item.ID, domain.StatusNeedsReview)
	}

	if err := h.DB.Content.SetStatus(ctx, item.ID, domain.StatusProcessing); err != nil {
		return err
	}

	// Gather extra text (abstract / video description) to aid classification.
	extraText := h.extraText(ctx, item)

	// Classify topics.
	topics, err := h.DB.Topic.List(ctx)
	if err != nil {
		return err
	}
	assignments := process.Classify(item, extraText, topics)
	if len(assignments) == 0 {
		// Nothing in the text named a section. Fall back to the source's beat:
		// a Cyclingnews story is cycling even when its headline only says "Tour
		// de France", and sports headlines overwhelmingly name people and events
		// rather than sports. Only single-beat sources carry a default, so this
		// cannot file a basketball story under football — a general desk leaves
		// default_topic_id NULL and the article stays unsectioned, as before.
		assignments = h.sourceDefaultTopic(ctx, item)
	}
	if len(assignments) > 0 {
		if err := h.DB.Topic.AssignTopics(ctx, item.ID, process.ToContentTopics(item.ID, assignments)); err != nil {
			return err
		}
	}

	// Extract entities.
	entities, err := h.DB.Entity.WithAliases(ctx)
	if err != nil {
		return err
	}
	matches := process.ExtractEntities(item, extraText, entities)
	if len(matches) > 0 {
		if err := h.DB.WithTx(ctx, func(tx pgx.Tx) error {
			for _, m := range matches {
				if err := h.DB.Entity.LinkEntity(ctx, tx, item.ID, m.EntityID, m.Role); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}

	// Score first so the summarize gate can use base_score.
	if err := h.rescore(ctx, item); err != nil {
		return err
	}
	item, err = h.DB.Content.Get(ctx, item.ID)
	if err != nil {
		return err
	}

	// Articles are never public from a headline/excerpt alone. Foreign articles
	// additionally remain hidden until the Vietnamese edition has been stored.
	if item.Type == domain.ContentArticle {
		if item.Language != "vi" && !translationReady(articleBody) {
			if h.Summarizer == nil || !h.Summarizer.Enabled() {
				return h.DB.Content.SetStatus(ctx, item.ID, domain.StatusNeedsReview)
			}
			// Translating a full body is the single most expensive call we make,
			// so it is not something every arriving foreign article earns. Park
			// the low scorers instead: cluster them on their original headline so
			// they still count toward a story's heat, and leave them 'processing'.
			// The daily pick can still come back and translate a parked item when
			// its cluster wins the day, and IDsPendingTranslation will pick it up
			// if a later rescore lifts it over the bar.
			if item.BaseScore < h.TranslateMinScore {
				if err := h.DB.Content.ClusterContent(ctx, item.ID, item.Title); err != nil {
					return err
				}
				return h.DB.Content.SetStatus(ctx, item.ID, domain.StatusProcessing)
			}
			return h.Enqueue.EnqueueTranslate(ctx, item.ID)
		}
	}

	// Decide whether to summarize.
	if h.shouldSummarize(item) {
		return h.Enqueue.EnqueueSummarize(ctx, item.ID)
	}

	// No summary path: still make it visible. Items with no topic go to review.
	status := domain.StatusReady
	if len(assignments) == 0 {
		status = domain.StatusNeedsReview
	}
	if item.Language == "vi" {
		if err := h.DB.Content.ClusterContent(ctx, item.ID, item.Title); err != nil {
			return err
		}
	}
	return h.DB.Content.SetStatus(ctx, item.ID, status)
}

// translationReady reports whether a Vietnamese edition of this body is already
// stored, i.e. whether the expensive work has been done before.
//
// handleProcess resets an item to 'processing' on entry and re-derives its
// status from scratch, which is right for a freshly-ingested item and wrong for
// every other caller. Without this check, re-running it over a foreign article
// that is already live and already translated does one of two silent damages:
// under LLM_TRANSLATE_MIN_SCORE it parks the article back to 'processing' and
// takes a published story off the site, and over the threshold it queues a
// second translation of a body we already hold, spending free-tier quota to
// reproduce what is sitting in the row. Re-processing is a normal operation —
// a new classifier vocabulary is reason enough — so it has to be safe to repeat.
func translationReady(b *domain.ContentBody) bool {
	return b != nil && b.TranslationStatus == "ready" &&
		b.VietnameseBody != nil && strings.TrimSpace(*b.VietnameseBody) != ""
}

// sourceDefaultTopic returns the section configured for the item's source, or
// nothing when the source has no single beat.
//
// The confidence is deliberately below what a headline keyword match earns
// (0.5): the masthead is a reliable signal about the subject but a weaker claim
// than the article naming the sport itself, and a later rescore or a human
// should be able to outrank it. A lookup failure is not an error worth failing
// the job over — the article simply stays unsectioned, which is where it already
// was.
func (h *Handlers) sourceDefaultTopic(ctx context.Context, item *domain.ContentItem) []process.Assignment {
	src, err := h.DB.Source.Get(ctx, item.SourceID)
	if err != nil || src.DefaultTopicID == nil {
		return nil
	}
	return []process.Assignment{{TopicID: *src.DefaultTopicID, Confidence: 0.4, IsPrimary: true}}
}

// shouldSummarize gates the LLM by config threshold + availability.
func (h *Handlers) shouldSummarize(item *domain.ContentItem) bool {
	if h.Summarizer == nil || !h.Summarizer.Enabled() {
		return false
	}
	if item.BaseScore < h.ScoreThreshold {
		return false
	}
	// Videos are only summarized when a transcript exists (checked in handler).
	return true
}

// ---- summarize ----

func (h *Handlers) handleSummarize(ctx context.Context, j *domain.Job) error {
	var p domain.ContentPayload
	if err := j.Unmarshal(&p); err != nil {
		return err
	}
	item, err := h.DB.Content.Get(ctx, p.ContentID)
	if errors.Is(err, domain.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	// Respect both ceilings: defer (retry later) rather than fail hard, and name
	// the one that refused.
	if refusal, err := h.Summarizer.BudgetStatus(ctx); err != nil {
		return err
	} else if refusal != nil {
		return refusal
	}

	switch item.Type {
	case domain.ContentResearch:
		return h.summarizeResearch(ctx, item)
	case domain.ContentVideo:
		return h.summarizeVideo(ctx, item)
	default:
		return h.summarizeArticle(ctx, item)
	}
}

func (h *Handlers) summarizeArticle(ctx context.Context, item *domain.ContentItem) error {
	body := deref(item.Excerpt)
	if stored, err := h.DB.Content.GetBody(ctx, item.ID); err == nil && strings.TrimSpace(stored.OriginalBody) != "" {
		body = stored.OriginalBody
	}
	if ingest.BlockedArticleText(body) {
		return h.DB.Content.QuarantineBlockedArticle(ctx, item.ID)
	}
	if len(strings.Fields(body)) < 120 {
		return h.DB.Content.SetStatus(ctx, item.ID, domain.StatusNeedsReview)
	}
	out, err := h.Summarizer.SummarizeArticle(ctx, item.Title, body)
	if err != nil {
		return err
	}
	return h.DB.Content.SetSummary(ctx, item.ID, out.Summary, out.KeyPoints, domain.StatusReady)
}

func (h *Handlers) summarizeResearch(ctx context.Context, item *domain.ContentItem) error {
	rp, err := h.DB.Content.GetResearch(ctx, item.ID)
	if err != nil {
		return err
	}
	abstract := deref(rp.Abstract)
	if strings.TrimSpace(abstract) == "" {
		// Nothing to summarize; still show metadata.
		return h.DB.Content.SetStatus(ctx, item.ID, domain.StatusReady)
	}
	bd, err := h.Summarizer.SummarizeResearch(ctx, item.Title, abstract)
	if err != nil {
		return err
	}
	if err := h.DB.Content.UpdateResearchBreakdown(ctx, item.ID, *bd); err != nil {
		return err
	}
	// A short human-readable summary from the first finding.
	var summary *string
	if len(bd.Findings) > 0 {
		s := bd.Findings[0]
		summary = &s
	}
	return h.DB.Content.SetSummary(ctx, item.ID, summary, bd.Findings, domain.StatusReady)
}

func (h *Handlers) summarizeVideo(ctx context.Context, item *domain.ContentItem) error {
	v, err := h.DB.Content.GetVideo(ctx, item.ID)
	if err != nil {
		return err
	}
	// Only summarize with a valid transcript > 500 words (spec section 13).
	if !v.HasTranscript || v.Transcript == nil || len(strings.Fields(*v.Transcript)) < 500 {
		return h.DB.Content.SetStatus(ctx, item.ID, domain.StatusReady)
	}
	out, err := h.Summarizer.SummarizeArticle(ctx, item.Title, *v.Transcript)
	if err != nil {
		return err
	}
	return h.DB.Content.SetSummary(ctx, item.ID, out.Summary, out.KeyPoints, domain.StatusReady)
}

// ---- score ----

func (h *Handlers) handleScore(ctx context.Context, j *domain.Job) error {
	var p domain.ContentPayload
	if err := j.Unmarshal(&p); err != nil {
		return err
	}
	item, err := h.DB.Content.Get(ctx, p.ContentID)
	if errors.Is(err, domain.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return h.rescore(ctx, item)
}

func (h *Handlers) rescore(ctx context.Context, item *domain.ContentItem) error {
	src, err := h.DB.Source.Get(ctx, item.SourceID)
	if err != nil {
		return err
	}
	var rp *domain.ResearchPaper
	if item.Type == domain.ContentResearch {
		rp, _ = h.DB.Content.GetResearch(ctx, item.ID)
	}
	topics, _ := h.DB.Topic.ForContent(ctx, item.ID)
	score := process.BaseScore(item, src, rp, topics)
	return h.DB.Content.SetBaseScore(ctx, item.ID, score)
}

// ---- send_daily / send_weekly ----

func (h *Handlers) handleSendDaily(ctx context.Context, j *domain.Job) error {
	var p domain.DigestPayload
	if err := j.Unmarshal(&p); err != nil {
		return err
	}
	conn, err := h.DB.Telegram.ConnectionByUser(ctx, p.UserID)
	if errors.Is(err, domain.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	prefs, err := h.DB.Telegram.GetPrefs(ctx, p.UserID)
	if err != nil {
		return err
	}
	msg, ids, ok, err := h.Digest.BuildDaily(ctx, p.UserID, prefs)
	if err != nil {
		return err
	}
	if !ok {
		h.Log.Info("daily skipped: too few items", "user", p.UserID)
		return nil
	}
	if err := h.deliver(ctx, p.UserID, conn.ChatID, domain.DigestDaily, msg, ids); err != nil {
		return err
	}
	// Premium members receive the five-minute audio edition in the same chat.
	if sub, err := h.DB.Engagement.Subscription(ctx, p.UserID); err == nil && sub.Active(time.Now()) {
		if brief, err := h.DB.Engagement.LatestAudioBrief(ctx, "morning"); err == nil && brief.AudioURL != nil {
			_, err = h.Telegram.SendAudio(ctx, conn.ChatID, *brief.AudioURL, brief.Title+" · BaoTheX Premium")
			return err
		}
	}
	return nil
}

// handleFollowAlert nudges a user about new stories in the topics they follow.
//
// The switch for this — "Thông báo khi chủ đề theo dõi có bài mới" — has been in
// the settings panel, and the follow_alert value has been in the digest_kind
// enum, since the first migration. Everything existed except the part that
// sends: no enqueuer, no handler, no scheduler tick. A user could turn it on and
// nothing would ever arrive, which is worse than not offering it, because the
// setting is a promise and the reader has no way to see it is not kept.
func (h *Handlers) handleFollowAlert(ctx context.Context, j *domain.Job) error {
	var p domain.DigestPayload
	if err := j.Unmarshal(&p); err != nil {
		return err
	}
	conn, err := h.DB.Telegram.ConnectionByUser(ctx, p.UserID)
	if errors.Is(err, domain.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	prefs, err := h.DB.Telegram.GetPrefs(ctx, p.UserID)
	if err != nil {
		return err
	}
	// Re-checked here rather than trusted from the scheduler: the job may have
	// waited in the queue, and a user who switched the alert off in between is
	// entitled to have meant it.
	if !prefs.FollowAlerts {
		return nil
	}
	msg, ids, ok, err := h.Digest.BuildFollowAlert(ctx, p.UserID, prefs)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	return h.deliver(ctx, p.UserID, conn.ChatID, domain.DigestFollowAlert, msg, ids)
}

// deliver sends a message and records the delivery (handling a blocked bot).
func (h *Handlers) deliver(ctx context.Context, userID, chatID int64, kind domain.DigestKind, msg string, ids []int64) error {
	msgID, err := h.Telegram.SendMessage(ctx, chatID, msg, nil)
	if errors.Is(err, telegram.ErrBlocked) {
		// User blocked the bot: stop sending, log the delivery error.
		_ = h.DB.Telegram.SetDailyEnabled(ctx, userID, false)
		e := "blocked"
		_ = h.DB.Telegram.RecordDelivery(ctx, userID, kind, ids, nil, &e)
		return nil
	}
	if err != nil {
		e := err.Error()
		_ = h.DB.Telegram.RecordDelivery(ctx, userID, kind, ids, nil, &e)
		return err
	}
	return h.DB.Telegram.RecordDelivery(ctx, userID, kind, ids, &msgID, nil)
}

// extraText returns subtype text (abstract / description) for classification.
func (h *Handlers) extraText(ctx context.Context, item *domain.ContentItem) string {
	switch item.Type {
	case domain.ContentResearch:
		if rp, err := h.DB.Content.GetResearch(ctx, item.ID); err == nil {
			return deref(rp.Abstract)
		}
	case domain.ContentVideo:
		if v, err := h.DB.Content.GetVideo(ctx, item.ID); err == nil {
			return deref(v.Description)
		}
	}
	return ""
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
