// Package jobs implements the PostgreSQL-backed job queue, worker pool,
// scheduler and per-kind handlers.
package jobs

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"repwire/internal/domain"
	"repwire/internal/postgres"
)

// Enqueuer wraps the job repository with typed enqueue helpers. It also
// satisfies telegram.Enqueuer.
type Enqueuer struct {
	repo *postgres.JobRepo
}

// NewEnqueuer constructs an Enqueuer.
func NewEnqueuer(repo *postgres.JobRepo) *Enqueuer { return &Enqueuer{repo: repo} }

// EnqueueFetch schedules a fetch job for a source, deduped per source+kind.
func (e *Enqueuer) EnqueueFetch(ctx context.Context, kind string, sourceID int64) error {
	return e.repo.Enqueue(ctx, kind, domain.FetchPayload{SourceID: sourceID}, postgres.EnqueueOpts{
		DedupKey: fmt.Sprintf("%s:source:%d", kind, sourceID),
	})
}

// EnqueueProcess schedules processing of a freshly-ingested content item.
func (e *Enqueuer) EnqueueProcess(ctx context.Context, contentID int64) error {
	return e.repo.Enqueue(ctx, domain.JobProcessContent, domain.ContentPayload{ContentID: contentID}, postgres.EnqueueOpts{
		DedupKey: fmt.Sprintf("process:%d", contentID),
		Priority: 1,
	})
}

// EnqueueSummarize schedules summarization of a content item.
func (e *Enqueuer) EnqueueSummarize(ctx context.Context, contentID int64) error {
	return e.repo.Enqueue(ctx, domain.JobSummarize, domain.ContentPayload{ContentID: contentID}, postgres.EnqueueOpts{
		DedupKey: fmt.Sprintf("summarize:%d", contentID),
	})
}

// EnqueueTranslate schedules Vietnamese translation for a content body.
func (e *Enqueuer) EnqueueTranslate(ctx context.Context, contentID int64) error {
	return e.repo.Enqueue(ctx, domain.JobTranslate, domain.ContentPayload{ContentID: contentID}, postgres.EnqueueOpts{
		DedupKey: fmt.Sprintf("translate:%d", contentID),
	})
}

// EnqueueScore schedules (re)scoring of a content item.
func (e *Enqueuer) EnqueueScore(ctx context.Context, contentID int64) error {
	return e.repo.Enqueue(ctx, domain.JobScore, domain.ContentPayload{ContentID: contentID}, postgres.EnqueueOpts{
		DedupKey: fmt.Sprintf("score:%d", contentID),
	})
}

// EnqueueSendDaily schedules a daily digest for a user. When immediate is false
// it is deduped per user+day so the scheduler cannot double-send.
func (e *Enqueuer) EnqueueSendDaily(ctx context.Context, userID int64, immediate bool) error {
	day := time.Now().UTC().Format("2006-01-02")
	opts := postgres.EnqueueOpts{Priority: 2}
	if !immediate {
		opts.DedupKey = fmt.Sprintf("daily:%d:%s", userID, day)
	}
	return e.repo.Enqueue(ctx, domain.JobSendDaily, domain.DigestPayload{UserID: userID, Date: day}, opts)
}

// EnqueueSendWeekly schedules a weekly research digest for a user.
func (e *Enqueuer) EnqueueSendWeekly(ctx context.Context, userID int64) error {
	week := time.Now().UTC().Format("2006-W02")
	return e.repo.Enqueue(ctx, domain.JobSendWeekly, domain.DigestPayload{UserID: userID}, postgres.EnqueueOpts{
		DedupKey: fmt.Sprintf("weekly:%d:%s", userID, week),
		Priority: 2,
	})
}

func (e *Enqueuer) EnqueueGenerateAudio(ctx context.Context, day time.Time, edition string) error {
	date := day.Format("2006-01-02")
	return e.repo.Enqueue(ctx, domain.JobGenerateAudio, domain.BriefPayload{Date: date, Edition: edition}, postgres.EnqueueOpts{
		DedupKey:    "audio-brief:" + edition + ":" + date,
		Priority:    2,
		MaxAttempts: 3,
	})
}

func (e *Enqueuer) EnqueueSendPremiumBrief(ctx context.Context, userID int64, day time.Time, edition string) error {
	date := day.Format("2006-01-02")
	return e.repo.Enqueue(ctx, domain.JobSendPremiumBrief, domain.PremiumBriefPayload{UserID: userID, Date: date, Edition: edition}, postgres.EnqueueOpts{
		DedupKey:    fmt.Sprintf("premium-audio:%d:%s:%s", userID, edition, date),
		Priority:    3,
		MaxAttempts: 3,
	})
}

func (e *Enqueuer) EnqueueGenerateAnalysis(ctx context.Context, clusterID int64) error {
	dedupKey := fmt.Sprintf("cluster-analysis:%d", clusterID)
	if woke, err := e.repo.WakePendingByDedup(ctx, dedupKey); err != nil {
		return err
	} else if woke {
		return nil
	}
	return e.repo.Enqueue(ctx, domain.JobGenerateAnalysis, domain.AnalysisPayload{ClusterID: clusterID}, postgres.EnqueueOpts{
		DedupKey:    dedupKey,
		Priority:    3,
		MaxAttempts: 3,
	})
}

// backoff returns an exponential backoff duration with jitter, capped at 2h.
func backoff(attempt int) time.Duration {
	d := time.Duration(math.Pow(2, float64(attempt))) * time.Minute // 2,4,8,16,32...
	if d > 2*time.Hour {
		d = 2 * time.Hour
	}
	return d + time.Duration(rand.Intn(60))*time.Second
}
