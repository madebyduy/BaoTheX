// Package process implements classification, entity extraction, summarization
// and scoring of content items.
package process

import (
	"time"

	"repwire/internal/domain"
)

// BaseScore computes the algorithmic score for an item (spec section 14).
// rp and topics may be nil/empty when not applicable.
func BaseScore(c *domain.ContentItem, src *domain.Source, rp *domain.ResearchPaper, topics []domain.Topic) float64 {
	s := 0.0

	// Source quality contributes 4..20.
	s += float64(src.Quality) * 4

	if rp != nil {
		if rp.Abstract != nil && *rp.Abstract != "" {
			s += 20
		}
		switch rp.StudyType {
		case domain.MetaAnalysis, domain.SystematicReview, domain.RCT:
			s += 15
		}
		if rp.IsOpenAccess {
			s += 5
		}
	}

	// Approved YouTube channels (quality >= 4) get a video boost.
	if c.Type == domain.ContentVideo && src.Quality >= 4 {
		s += 15
	}

	// Freshness.
	if c.PublishedAt != nil {
		age := time.Since(*c.PublishedAt)
		switch {
		case age < 24*time.Hour:
			s += 10
		case age < 72*time.Hour:
			s += 5
		}
	}

	// Belongs to a well-followed topic.
	for _, t := range topics {
		if t.FollowerCount > 50 {
			s += 10
			break
		}
	}

	return s
}
