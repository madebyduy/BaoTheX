package domain

import "time"

// SourceKind enumerates how a source is fetched.
type SourceKind string

const (
	SourceRSS        SourceKind = "rss"
	SourceYouTube    SourceKind = "youtube"
	SourceEuropePMC  SourceKind = "europepmc"
	SourcePodcastRSS SourceKind = "podcast_rss"
	SourceSitemap    SourceKind = "sitemap"
	SourceManual     SourceKind = "manual"
)

// Source is an approved content origin (an RSS feed, a YouTube channel, a PMC query...).
type Source struct {
	ID                  int64         `json:"id"`
	Kind                SourceKind    `json:"kind"`
	Name                string        `json:"name"`
	HomepageURL         *string       `json:"homepage_url,omitempty"`
	FeedURL             *string       `json:"feed_url,omitempty"`
	Quality             int           `json:"quality"`
	DefaultLang         string        `json:"default_lang"`
	Enabled             bool          `json:"enabled"`
	FetchInterval       time.Duration `json:"-"`
	ETag                *string       `json:"-"`
	LastModified        *string       `json:"-"`
	UploadsPlaylistID   *string       `json:"-"`
	LastFetchedAt       *time.Time    `json:"last_fetched_at,omitempty"`
	LastError           *string       `json:"last_error,omitempty"`
	ConsecutiveFailures int           `json:"consecutive_failures"`
	CreatedAt           time.Time     `json:"created_at"`
}

// FeedURLOrEmpty is a small helper so callers avoid nil-deref on FeedURL.
func (s *Source) FeedURLOrEmpty() string {
	if s.FeedURL == nil {
		return ""
	}
	return *s.FeedURL
}

// HomepageURLOrEmpty returns the configured public source page when present.
func (s *Source) HomepageURLOrEmpty() string {
	if s.HomepageURL == nil {
		return ""
	}
	return *s.HomepageURL
}
