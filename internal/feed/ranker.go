package feed

import (
	"context"

	"repwire/internal/domain"
	"repwire/internal/postgres"
)

// Ranker serves the paginated personalized feed. The heavy scoring lives in the
// SQL (postgres.personalFeedSQL); this wrapper exists so higher layers depend on
// a feed-domain type rather than the repo directly, and so the 50/30/20 balance
// logic has a home as it grows.
type Ranker struct {
	db *postgres.DB
}

// NewRanker constructs a Ranker.
func NewRanker(db *postgres.DB) *Ranker { return &Ranker{db: db} }

// PersonalFeed returns the paginated personalized feed for a user.
func (r *Ranker) PersonalFeed(ctx context.Context, userID int64, page, perPage int) ([]domain.ContentItem, error) {
	if page < 1 {
		page = 1
	}
	if perPage <= 0 || perPage > 50 {
		perPage = 20
	}
	return r.db.Content.PersonalFeed(ctx, userID, perPage, (page-1)*perPage)
}
