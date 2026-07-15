package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"

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

	// ScoreThreshold: only items with base_score >= this get summarized.
	ScoreThreshold float64
}

// Register returns the kind→handler map for the worker.
func (h *Handlers) Register() map[string]HandlerFunc {
	return map[string]HandlerFunc{
		domain.JobFetchRSS:       h.handleFetch,
		domain.JobFetchYouTube:   h.handleFetch,
		domain.JobFetchPMC:       h.handleFetch,
		domain.JobFetchPodcast:   h.handleFetch,
		domain.JobProcessContent: h.handleProcess,
		domain.JobSummarize:      h.handleSummarize,
		domain.JobTranslate:      h.handleTranslate,
		domain.JobScore:          h.handleScore,
		domain.JobSendDaily:      h.handleSendDaily,
		domain.JobSendWeekly:     h.handleSendWeekly,
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
	if ok, err := h.Summarizer.BudgetOK(ctx); err != nil {
		return err
	} else if !ok {
		return process.ErrBudgetExceeded
	}
	body, err := h.DB.Content.GetBody(ctx, p.ContentID)
	if errors.Is(err, domain.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	item, err := h.DB.Content.Get(ctx, p.ContentID)
	if err != nil {
		return err
	}
	out, err := h.Summarizer.TranslateAndSummarize(ctx, item.Title, body.OriginalBody)
	if err != nil {
		return err
	}
	if err := h.DB.Content.SetVietnameseContent(ctx, p.ContentID, out.VietnameseTitle, out.VietnameseBody); err != nil {
		return err
	}
	if err := h.DB.Content.ClusterContent(ctx, p.ContentID, out.VietnameseTitle); err != nil {
		return err
	}
	if out.Summary == nil {
		fallback, err := h.Summarizer.SummarizeArticle(ctx, item.Title, body.OriginalBody)
		if err != nil {
			return err
		}
		out.Summary = fallback.Summary
		out.KeyPoints = fallback.KeyPoints
	}
	return h.DB.Content.SetSummary(ctx, p.ContentID, out.Summary, out.KeyPoints, domain.StatusReady)
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
	_ = h.DB.Content.SetStatus(ctx, item.ID, domain.StatusProcessing)

	// Gather extra text (abstract / video description) to aid classification.
	extraText := h.extraText(ctx, item)

	// Classify topics.
	topics, err := h.DB.Topic.List(ctx)
	if err != nil {
		return err
	}
	assignments := process.Classify(item, extraText, topics)
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
	item, _ = h.DB.Content.Get(ctx, item.ID)

	// Vietnamese sources are displayed as-is. For foreign articles with a
	// captured body, one combined translate+summarize call is cheaper than
	// running separate translation and summarization jobs.
	if item.Language != "vi" && h.Summarizer != nil && h.Summarizer.Enabled() {
		if body, bodyErr := h.DB.Content.GetBody(ctx, item.ID); bodyErr == nil && strings.TrimSpace(body.OriginalBody) != "" {
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

	// Respect the daily budget: defer (retry later) rather than fail hard.
	if ok, err := h.Summarizer.BudgetOK(ctx); err != nil {
		return err
	} else if !ok {
		return process.ErrBudgetExceeded
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
	return h.deliver(ctx, p.UserID, conn.ChatID, domain.DigestDaily, msg, ids)
}

func (h *Handlers) handleSendWeekly(ctx context.Context, j *domain.Job) error {
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
	msg, ids, ok, err := h.Digest.BuildWeekly(ctx, p.UserID)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	return h.deliver(ctx, p.UserID, conn.ChatID, domain.DigestWeeklyResearch, msg, ids)
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
