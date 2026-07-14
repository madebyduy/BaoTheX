package domain

import "time"

// ContentType is the discriminator for the ContentItem subtype tables.
type ContentType string

const (
	ContentArticle      ContentType = "article"
	ContentResearch     ContentType = "research"
	ContentVideo        ContentType = "video"
	ContentPodcast      ContentType = "podcast"
	ContentAnnouncement ContentType = "announcement"
	ContentEvent        ContentType = "event"
)

// ContentStatus tracks an item through the ingestion pipeline.
type ContentStatus string

const (
	StatusDiscovered  ContentStatus = "discovered"
	StatusFetching    ContentStatus = "fetching"
	StatusProcessing  ContentStatus = "processing"
	StatusReady       ContentStatus = "ready"
	StatusFailed      ContentStatus = "failed"
	StatusHidden      ContentStatus = "hidden"
	StatusNeedsReview ContentStatus = "needs_review"
)

// ContentItem is the central concept: every piece of content is one of these,
// with an optional row in a subtype table (articles, research_papers, ...).
type ContentItem struct {
	ID             int64         `json:"id"`
	SourceID       int64         `json:"source_id"`
	Type           ContentType   `json:"type"`
	Status         ContentStatus `json:"status"`
	Title          string        `json:"title"`
	CanonicalURL   string        `json:"canonical_url"`
	URLHash        string        `json:"-"`
	TitleHash      *string       `json:"-"`
	ImageURL       *string       `json:"image_url,omitempty"`
	Excerpt        *string       `json:"excerpt,omitempty"`
	Summary        *string       `json:"summary,omitempty"`
	KeyPoints      []string      `json:"key_points"`
	Language       string        `json:"language"`
	PublishedAt    *time.Time    `json:"published_at,omitempty"`
	DiscoveredAt   time.Time     `json:"discovered_at"`
	BaseScore      float64       `json:"base_score"`
	EditorialBoost float64       `json:"editorial_boost"`
	FinalScore     float64       `json:"final_score"`
	ViewCount      int           `json:"view_count"`
	SaveCount      int           `json:"save_count"`
	UpdatedAt      time.Time     `json:"updated_at"`

	// SourceName is denormalised for read paths; empty unless the query joined sources.
	SourceName string `json:"source_name,omitempty"`
}

// ContentBody stores the readable source body and its optional Vietnamese translation.
type ContentBody struct {
	ContentID         int64      `json:"content_id"`
	OriginalLanguage  string     `json:"original_language"`
	OriginalBody      string     `json:"original_body"`
	VietnameseBody    *string    `json:"vietnamese_body,omitempty"`
	TranslationStatus string     `json:"translation_status"`
	TranslatedAt      *time.Time `json:"translated_at,omitempty"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// Article is the subtype row for ContentArticle.
type Article struct {
	ContentID int64   `json:"content_id"`
	Author    *string `json:"author,omitempty"`
	WordCount *int    `json:"word_count,omitempty"`
}

// StudyType classifies research papers by evidence strength.
type StudyType string

const (
	MetaAnalysis     StudyType = "meta_analysis"
	SystematicReview StudyType = "systematic_review"
	RCT              StudyType = "rct"
	Cohort           StudyType = "cohort"
	CrossSectional   StudyType = "cross_sectional"
	CaseStudy        StudyType = "case_study"
	NarrativeReview  StudyType = "narrative_review"
	StudyOther       StudyType = "other"
)

// ResearchBreakdown is the fixed 8-section summary of a paper (AI + admin edited).
type ResearchBreakdown struct {
	Question     *string  `json:"question"`
	Participants *string  `json:"participants"`
	Intervention *string  `json:"intervention"`
	Findings     []string `json:"findings"`
	NotProven    *string  `json:"not_proven"`
	Limitations  []string `json:"limitations"`
	Practical    *string  `json:"practical"`
	FundingNote  *string  `json:"funding_note"`
}

// ResearchPaper is the subtype row for ContentResearch.
type ResearchPaper struct {
	ContentID      int64             `json:"content_id"`
	DOI            *string           `json:"doi,omitempty"`
	PMID           *string           `json:"pmid,omitempty"`
	PMCID          *string           `json:"pmcid,omitempty"`
	Journal        *string           `json:"journal,omitempty"`
	Authors        []string          `json:"authors"`
	Abstract       *string           `json:"abstract,omitempty"`
	StudyType      StudyType         `json:"study_type"`
	IsHuman        *bool             `json:"is_human,omitempty"`
	IsOpenAccess   bool              `json:"is_open_access"`
	FullTextURL    *string           `json:"full_text_url,omitempty"`
	SampleSize     *int              `json:"sample_size,omitempty"`
	Population     *string           `json:"population,omitempty"`
	DurationWeeks  *int              `json:"duration_weeks,omitempty"`
	Sex            *string           `json:"sex,omitempty"`
	TrainingStatus *string           `json:"training_status,omitempty"`
	Breakdown      ResearchBreakdown `json:"breakdown"`
	PublishedYear  *int              `json:"published_year,omitempty"`
}

// Video is the subtype row for ContentVideo.
type Video struct {
	ContentID     int64           `json:"content_id"`
	YouTubeID     string          `json:"youtube_id"`
	ChannelID     string          `json:"channel_id"`
	ChannelTitle  string          `json:"channel_title"`
	DurationSec   *int            `json:"duration_sec,omitempty"`
	ThumbnailURL  *string         `json:"thumbnail_url,omitempty"`
	Description   *string         `json:"description,omitempty"`
	HasTranscript bool            `json:"has_transcript"`
	Transcript    *string         `json:"-"`
	Timeline      []TimelineEntry `json:"timeline"`
	YTViews       *int64          `json:"yt_views,omitempty"`
	YTLikes       *int64          `json:"yt_likes,omitempty"`
}

// TimelineEntry is one chapter marker in a video timeline.
type TimelineEntry struct {
	T     int    `json:"t"`
	Label string `json:"label"`
}

// PodcastEpisode is the subtype row for ContentPodcast.
type PodcastEpisode struct {
	ContentID   int64   `json:"content_id"`
	ShowName    string  `json:"show_name"`
	EpisodeGUID string  `json:"episode_guid"`
	AudioURL    *string `json:"audio_url,omitempty"`
	DurationSec *int    `json:"duration_sec,omitempty"`
	ShowNotes   *string `json:"show_notes,omitempty"`
	Transcript  *string `json:"-"`
}
