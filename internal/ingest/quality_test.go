package ingest

import (
	"strings"
	"testing"
	"time"

	"repwire/internal/domain"
)

func TestAssessQualityPassesCompleteArticle(t *testing.T) {
	now := time.Date(2026, time.July, 22, 12, 0, 0, 0, time.UTC)
	image := "https://cdn.example.com/photo.jpg"
	item := &domain.ContentItem{
		Type:         domain.ContentArticle,
		Title:        "Vietnam win a close football final",
		CanonicalURL: "https://example.com/sport/final",
		ImageURL:     &image,
		PublishedAt:  ptr(now.Add(-time.Hour)),
	}

	got := AssessQuality(item, strings.Repeat("football report ", 70), now)
	if got.State != "passed" || len(got.Flags) != 0 {
		t.Fatalf("complete article should pass, got %#v", got)
	}
}

func TestAssessQualityRoutesShortArticleToReview(t *testing.T) {
	item := &domain.ContentItem{
		Type:         domain.ContentArticle,
		Title:        "A valid sports headline",
		CanonicalURL: "https://example.com/story",
	}

	got := AssessQuality(item, "Only a short excerpt is available.", time.Now())
	if got.State != "review" || !containsFlag(got.Flags, "body_too_short") {
		t.Fatalf("short article should need review, got %#v", got)
	}
	if !containsFlag(got.Flags, "missing_image") {
		t.Fatalf("missing image advisory flag not recorded: %#v", got)
	}
}

func TestAssessQualityRejectsInvalidURLAndFutureTimestamp(t *testing.T) {
	now := time.Date(2026, time.July, 22, 12, 0, 0, 0, time.UTC)
	future := now.Add(25 * time.Hour)
	item := &domain.ContentItem{
		Type:         domain.ContentVideo,
		Title:        "Match highlights",
		CanonicalURL: "javascript:alert(1)",
		PublishedAt:  &future,
	}

	got := AssessQuality(item, "", now)
	if got.State != "review" || !containsFlag(got.Flags, "invalid_url") || !containsFlag(got.Flags, "future_published_at") {
		t.Fatalf("unsafe item should need review, got %#v", got)
	}
}

func containsFlag(flags []string, want string) bool {
	for _, flag := range flags {
		if flag == want {
			return true
		}
	}
	return false
}
