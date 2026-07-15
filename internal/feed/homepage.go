// Package feed builds the personalized homepage and feed.
package feed

import (
	"context"

	"repwire/internal/domain"
	"repwire/internal/postgres"
)

// Homepage is the block-structured homepage payload (spec section 14).
type Homepage struct {
	Today       []domain.ContentItem `json:"today"`   // Nổi bật hôm nay (3-5)
	ForYou      []domain.ContentItem `json:"for_you"` // personalized
	NewResearch []domain.ContentItem `json:"new_research"`
	Videos      []domain.ContentItem `json:"videos"`
	Sports      []domain.ContentItem `json:"sports"`
	DeepReads   []domain.ContentItem `json:"deep_reads"`
	Discovery   []domain.ContentItem `json:"discovery"`
}

// Builder assembles the homepage from repository queries.
type Builder struct {
	db *postgres.DB
}

// NewBuilder constructs a homepage Builder.
func NewBuilder(db *postgres.DB) *Builder { return &Builder{db: db} }

// Build assembles the homepage. userID may be 0 for anonymous visitors, in
// which case only general/discovery-agnostic blocks are populated.
func (b *Builder) Build(ctx context.Context, userID int64) (*Homepage, error) {
	general, err := b.db.Content.TopGeneral(ctx, 10)
	if err != nil {
		return nil, err
	}

	var personal, discovery []domain.ContentItem
	if userID != 0 {
		if personal, err = b.db.Content.TopPersonal(ctx, userID, 6); err != nil {
			return nil, err
		}
		if discovery, err = b.db.Content.TopDiscovery(ctx, userID, 4); err != nil {
			return nil, err
		}
	}

	newResearch, err := b.db.Content.LatestByType(ctx, domain.ContentResearch, 5)
	if err != nil {
		return nil, err
	}
	videos, err := b.db.Content.LatestByType(ctx, domain.ContentVideo, 5)
	if err != nil {
		return nil, err
	}
	sports, err := b.db.Content.ByTopicSlugs(ctx, []string{"bong-da-viet-nam", "bong-da-quoc-te", "bong-ro", "tennis", "the-thao-viet-nam"}, 8)
	if err != nil {
		return nil, err
	}
	deepReads, err := b.db.Content.LongForm(ctx, 3)
	if err != nil {
		return nil, err
	}

	today := dedupMerge(general, personal)
	if len(today) > 5 {
		today = today[:5]
	}

	return &Homepage{
		Today:       nonNil(today),
		ForYou:      nonNil(personal),
		NewResearch: nonNil(newResearch),
		Videos:      nonNil(videos),
		Sports:      nonNil(sports),
		DeepReads:   nonNil(deepReads),
		Discovery:   nonNil(discovery),
	}, nil
}

// dedupMerge merges two slices preserving order, dropping duplicate ids.
func dedupMerge(a, b []domain.ContentItem) []domain.ContentItem {
	seen := map[int64]bool{}
	var out []domain.ContentItem
	for _, list := range [][]domain.ContentItem{a, b} {
		for _, it := range list {
			if seen[it.ID] {
				continue
			}
			seen[it.ID] = true
			out = append(out, it)
		}
	}
	return out
}

func nonNil(s []domain.ContentItem) []domain.ContentItem {
	if s == nil {
		return []domain.ContentItem{}
	}
	return s
}
