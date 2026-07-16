// Package process implements classification, entity extraction, summarization
// and scoring of content items.
package process

import (
	"time"

	"repwire/internal/domain"
)

// MaxArticleScore is the highest score BaseScore can return for an article:
// 20 for a top-quality source, 10 for being fresh, 10 for a well-followed topic.
// Research papers and videos reach higher, but a plain article cannot.
//
// Beware the gap between this ceiling and the scores you will actually observe.
// The +10 requires a topic with over 50 followers, so on an instance whose
// topics are below that the real ceiling is 30, and a "reachable" threshold of
// 35 still matches nothing. Check the data before trusting the arithmetic:
//
//	SELECT type, max(base_score) FROM content_items GROUP BY type;
//
// Any threshold compared against an article's score must sit below this, or it
// disables the gate instead of tightening it. config.Load enforces that much.
const MaxArticleScore float64 = 40

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
