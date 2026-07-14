package postgres

import (
	"context"

	"repwire/internal/domain"
)

// DailyCandidates returns ready items published recently that have not yet been
// sent to the user, personalized-ordered, filtered by content type and (when
// highlightsOnly) a minimum final_score. It over-fetches so the caller can apply
// diversity rules before trimming to the final count.
func (r *ContentRepo) DailyCandidates(ctx context.Context, userID int64, contentTypes []string, highlightsOnly bool, limit int) ([]domain.ContentItem, error) {
	minScore := float64(-1e9)
	if highlightsOnly {
		minScore = 40
	}
	rows, err := r.db.Pool.Query(ctx, `
		SELECT `+contentCols+`, s.name
		FROM content_items c
		JOIN sources s ON s.id=c.source_id
		WHERE c.status='ready'
		  AND c.published_at > now() - interval '36 hours'
		  AND c.final_score >= $2
		  AND ($3::text[] IS NULL OR c.type::text = ANY($3))
		  AND NOT EXISTS (
		      SELECT 1 FROM digest_deliveries d
		      WHERE d.user_id=$1 AND c.id = ANY(d.content_ids))
		  AND NOT EXISTS (SELECT 1 FROM hidden_items h WHERE h.user_id=$1 AND h.content_id=c.id)
		  AND NOT EXISTS (
		      SELECT 1 FROM content_topics ct
		      JOIN user_topic_mutes m ON m.topic_id=ct.topic_id
		      WHERE ct.content_id=c.id AND m.user_id=$1)
		ORDER BY (
		    c.final_score
		    + CASE WHEN EXISTS (
		        SELECT 1 FROM content_topics ct
		        JOIN user_topic_follows utf ON utf.topic_id=ct.topic_id
		        WHERE ct.content_id=c.id AND utf.user_id=$1 AND utf.in_telegram) THEN 10 ELSE 0 END
		    + CASE WHEN EXISTS (
		        SELECT 1 FROM content_entities ce
		        JOIN user_entity_follows uef ON uef.entity_id=ce.entity_id
		        WHERE ce.content_id=c.id AND uef.user_id=$1 AND uef.in_telegram) THEN 10 ELSE 0 END
		  ) DESC, c.published_at DESC
		LIMIT $4`, userID, minScore, nullableTextArray(contentTypes), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectContent(rows)
}

// WeeklyResearchCandidates returns notable research from the past week not yet
// sent to the user.
func (r *ContentRepo) WeeklyResearchCandidates(ctx context.Context, userID int64, limit int) ([]domain.ContentItem, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT `+contentCols+`, s.name
		FROM content_items c
		JOIN sources s ON s.id=c.source_id
		WHERE c.status='ready' AND c.type='research'
		  AND c.published_at > now() - interval '7 days'
		  AND NOT EXISTS (
		      SELECT 1 FROM digest_deliveries d
		      WHERE d.user_id=$1 AND d.kind='weekly_research' AND c.id = ANY(d.content_ids))
		ORDER BY c.final_score DESC, c.published_at DESC
		LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectContent(rows)
}

// PrimaryTopicID returns the primary topic id for a content item, or 0.
func (r *ContentRepo) PrimaryTopicID(ctx context.Context, contentID int64) (int64, error) {
	var id int64
	err := r.db.Pool.QueryRow(ctx,
		`SELECT topic_id FROM content_topics WHERE content_id=$1 ORDER BY is_primary DESC, confidence DESC LIMIT 1`,
		contentID).Scan(&id)
	if err != nil {
		return 0, nil // no topic assigned
	}
	return id, nil
}

// nullableTextArray returns nil for an empty slice so the SQL treats it as "no filter".
func nullableTextArray(s []string) any {
	if len(s) == 0 {
		return nil
	}
	return s
}
