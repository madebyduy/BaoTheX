package domain

import (
	"encoding/json"
	"time"
)

// JobStatus is the lifecycle state of a queued job.
type JobStatus string

const (
	JobPending JobStatus = "pending"
	JobRunning JobStatus = "running"
	JobDone    JobStatus = "done"
	JobFailed  JobStatus = "failed"
	JobDead    JobStatus = "dead"
)

// Job kinds. Keep in sync with the worker handler registry.
const (
	JobFetchRSS         = "fetch_rss"
	JobFetchYouTube     = "fetch_youtube"
	JobFetchPMC         = "fetch_pmc"
	JobFetchPodcast     = "fetch_podcast"
	JobProcessContent   = "process_content"
	JobClassify         = "classify"
	JobSummarize        = "summarize"
	JobTranslate        = "translate"
	JobScore            = "score"
	JobSendDaily        = "send_daily"
	JobSendWeekly       = "send_weekly"
	JobFollowAlert      = "follow_alert"
	JobGenerateAudio    = "generate_audio_brief"
	JobGenerateVideo    = "generate_video_brief"
	JobGenerateAnalysis = "generate_cluster_analysis"
)

// Job is one unit of background work stored in the jobs table.
type Job struct {
	ID          int64           `json:"id"`
	Kind        string          `json:"kind"`
	Payload     json.RawMessage `json:"payload"`
	DedupKey    *string         `json:"dedup_key,omitempty"`
	Status      JobStatus       `json:"status"`
	Priority    int             `json:"priority"`
	RunAt       time.Time       `json:"run_at"`
	Attempts    int             `json:"attempts"`
	MaxAttempts int             `json:"max_attempts"`
	LockedBy    *string         `json:"locked_by,omitempty"`
	LockedAt    *time.Time      `json:"locked_at,omitempty"`
	LastError   *string         `json:"last_error,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	FinishedAt  *time.Time      `json:"finished_at,omitempty"`
}

// Unmarshal decodes the job payload into v.
func (j *Job) Unmarshal(v any) error {
	if len(j.Payload) == 0 {
		return nil
	}
	return json.Unmarshal(j.Payload, v)
}

// ---- Job payload shapes ----

// FetchPayload targets a single source for a fetch_* job.
type FetchPayload struct {
	SourceID int64 `json:"source_id"`
}

// ContentPayload targets a single content item for process/summarize/score jobs.
type ContentPayload struct {
	ContentID int64 `json:"content_id"`
}

// DigestPayload targets a single user for send_daily / send_weekly jobs.
type DigestPayload struct {
	UserID int64  `json:"user_id"`
	Date   string `json:"date,omitempty"` // YYYY-MM-DD, for dedup
}

type BriefPayload struct {
	Date    string `json:"date"`
	Edition string `json:"edition"`
}

type AnalysisPayload struct {
	ClusterID int64 `json:"cluster_id"`
}
