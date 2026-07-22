package ingest

import (
	"net/url"
	"strings"
	"time"

	"repwire/internal/domain"
)

const minimumArticleWords = 120

// QualityAssessment is the deterministic gate between ingestion and the more
// expensive enrichment/LLM stages. Flags are deliberately machine-readable so
// the admin surface can group failures without parsing log messages.
type QualityAssessment struct {
	State string
	Flags []string
}

// Has reports whether the assessment recorded a specific machine-readable
// flag. Callers use it when one review reason needs extra cleanup.
func (q QualityAssessment) Has(want string) bool {
	for _, flag := range q.Flags {
		if flag == want {
			return true
		}
	}
	return false
}

// AssessQuality validates the stored representation of an item. It is safe to
// run repeatedly: a later fetch may backfill an article body, after which the
// same item can pass without an operator having to clear stale quality state.
func AssessQuality(item *domain.ContentItem, body string, now time.Time) QualityAssessment {
	flags := make([]string, 0, 4)
	review := false

	if strings.TrimSpace(item.Title) == "" {
		flags = append(flags, "missing_title")
		review = true
	}
	if !validPublicArticleURL(item.CanonicalURL) {
		flags = append(flags, "invalid_url")
		review = true
	}
	if item.PublishedAt != nil && item.PublishedAt.After(now.Add(24*time.Hour)) {
		flags = append(flags, "future_published_at")
		review = true
	}

	if item.Type == domain.ContentArticle {
		trimmedBody := strings.TrimSpace(body)
		switch {
		case trimmedBody == "":
			flags = append(flags, "missing_body")
			review = true
		case BlockedArticleText(trimmedBody):
			flags = append(flags, "blocked_article_text")
			review = true
		case len(strings.Fields(trimmedBody)) < minimumArticleWords:
			flags = append(flags, "body_too_short")
			review = true
		}
	}

	// A missing image makes a card less useful but must not hold up otherwise
	// sound reporting. Keep it as an advisory flag for the media-repair job.
	if item.ImageURL == nil || strings.TrimSpace(*item.ImageURL) == "" {
		flags = append(flags, "missing_image")
	}

	state := "passed"
	if review {
		state = "review"
	}
	return QualityAssessment{State: state, Flags: flags}
}

func validPublicArticleURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Hostname() == "" || u.User != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}
