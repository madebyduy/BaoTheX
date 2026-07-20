package postgres

import (
	"context"

	"repwire/internal/domain"
)

// SearchRepo runs full-text and trigram searches.
type SearchRepo struct{ db *DB }

// SearchByType returns ready items matching the query, optionally constrained to
// a content type, ranked by ts_rank blended with final_score.
func (r *SearchRepo) SearchByType(ctx context.Context, query string, t *domain.ContentType, limit int) ([]domain.ContentItem, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT `+contentCols+`, s.name,
		       ts_rank(c.search_tsv, q) * (1 + c.final_score/100) AS rank
		FROM content_items c
		JOIN sources s ON s.id=c.source_id,
		     websearch_to_tsquery('simple', unaccent($1)) q
		WHERE c.status='ready' AND c.search_tsv @@ q
		  AND ($2::content_type IS NULL OR c.type=$2)
		ORDER BY rank DESC, c.published_at DESC
		LIMIT $3`, query, t, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	// Same columns as scanContentWithSource plus a trailing rank we discard.
	var out []domain.ContentItem
	for rows.Next() {
		var c domain.ContentItem
		var rank float64
		if err := rows.Scan(&c.ID, &c.SourceID, &c.Type, &c.Status, &c.Title, &c.CanonicalURL,
			&c.URLHash, &c.TitleHash, &c.ImageURL, &c.Excerpt, &c.Summary, &c.KeyPoints,
			&c.Language, &c.PublishedAt, &c.DiscoveredAt, &c.BaseScore, &c.EditorialBoost,
			&c.FinalScore, &c.ViewCount, &c.SaveCount, &c.UpdatedAt, &c.SourceName, &rank); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// TopicHit / EntityHit are grouped search results.
type TopicHit struct {
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	ContentCount int    `json:"content_count"`
}

type EntityHit struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}

// MatchingTopics returns topics whose name matches the query (trigram).
func (r *SearchRepo) MatchingTopics(ctx context.Context, query string, limit int) ([]TopicHit, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT t.slug, t.name,
		       (SELECT count(*) FROM content_topics ct
		          JOIN content_items c ON c.id=ct.content_id
		          WHERE ct.topic_id=t.id AND c.status='ready') AS content_count
		FROM topics t
		WHERE t.name % $1 OR t.slug ILIKE '%'||$1||'%' OR unaccent(t.name) ILIKE '%'||unaccent($1)||'%'
		ORDER BY GREATEST(similarity(t.name,$1), similarity(unaccent(t.name),unaccent($1))) DESC LIMIT $2`, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TopicHit
	for rows.Next() {
		var h TopicHit
		if err := rows.Scan(&h.Slug, &h.Name, &h.ContentCount); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// MatchingEntities returns entities whose name matches the query (trigram).
func (r *SearchRepo) MatchingEntities(ctx context.Context, query string, limit int) ([]EntityHit, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT slug, name, kind::text FROM entities
		WHERE name % $1 OR slug ILIKE '%'||$1||'%' OR unaccent(name) ILIKE '%'||unaccent($1)||'%'
		   OR EXISTS(SELECT 1 FROM unnest(aliases) AS a(value) WHERE unaccent(a.value) ILIKE '%'||unaccent($1)||'%')
		ORDER BY GREATEST(similarity(name,$1), similarity(unaccent(name),unaccent($1))) DESC LIMIT $2`, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []EntityHit
	for rows.Next() {
		var h EntityHit
		if err := rows.Scan(&h.Slug, &h.Name, &h.Kind); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// Suggestion is one autocomplete row.
type Suggestion struct {
	Kind string  `json:"kind"` // topic | entity
	Slug string  `json:"slug"`
	Name string  `json:"name"`
	Sim  float64 `json:"-"`
}

// Suggest returns autocomplete suggestions across topics and entities.
func (r *SearchRepo) Suggest(ctx context.Context, query string, limit int) ([]Suggestion, error) {
	rows, err := r.db.Pool.Query(ctx, `
		(SELECT 'topic'::text AS kind, slug, name, GREATEST(similarity(name,$1),similarity(unaccent(name),unaccent($1))) AS sim FROM topics WHERE name % $1 OR unaccent(name) ILIKE '%'||unaccent($1)||'%')
		UNION ALL
		(SELECT 'entity'::text AS kind, slug, name, GREATEST(similarity(name,$1),similarity(unaccent(name),unaccent($1))) AS sim FROM entities WHERE name % $1 OR unaccent(name) ILIKE '%'||unaccent($1)||'%' OR EXISTS(SELECT 1 FROM unnest(aliases) AS a(value) WHERE unaccent(a.value) ILIKE '%'||unaccent($1)||'%'))
		ORDER BY sim DESC LIMIT $2`, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Suggestion
	for rows.Next() {
		var s Suggestion
		if err := rows.Scan(&s.Kind, &s.Slug, &s.Name, &s.Sim); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
