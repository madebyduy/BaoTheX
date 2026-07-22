package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"repwire/internal/domain"
	"repwire/internal/textutil"
)

// Clustering thresholds. titleSimilarityFloor is pg_trgm similarity on the
// reader-facing title; sharedEntityFloor is how many entities two pieces must
// have in common before they are called the same story on that evidence alone.
//
// Two shared entities, not one: every Manchester United article in a week
// shares "Manchester United", so a floor of one would collapse a club's entire
// news cycle into a single cluster. "Man Utd + Ten Hag" appearing together is a
// specific event; "Man Utd" alone is a beat.
const (
	titleSimilarityFloor = 0.32
	sharedEntityFloor    = 2
	// entityMatchWindow is tighter than the title window because entity overlap
	// is the weaker evidence: coverage of one event lands within a couple of
	// days, while a rewritten headline stays recognisable for longer.
	entityMatchWindow = "2 days"
	titleMatchWindow  = "4 days"
)

// ClusterContent attaches an item to the closest recent story, or creates a new
// one.
//
// Two pieces are the same story if their titles look alike OR they share enough
// entities. The second path is what makes this work at all across languages:
// trigram similarity between "Man Utd sack their manager" and "MU sa thải HLV"
// is essentially zero, so title matching alone had merged English with
// Vietnamese coverage exactly once across the entire database, and left 88% of
// clusters holding a single article. Entities do not care which language spelled
// them — "Messi" is "Messi" either way — so they carry the join that text
// similarity cannot.
//
// This only pays off with a populated entity table: see migration 0023. With
// seven strength-training entities it degrades silently to title matching,
// which is exactly the state it was written to escape.
func (r *ContentRepo) ClusterContent(ctx context.Context, contentID int64, title string) error {
	return r.db.WithTx(ctx, func(tx pgx.Tx) error {
		var existing int64
		if err := tx.QueryRow(ctx, `SELECT cluster_id FROM story_cluster_items WHERE content_id=$1`, contentID).Scan(&existing); err == nil {
			return r.refreshCluster(ctx, tx, existing)
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}

		var clusterID int64
		var similarity float64
		err := tx.QueryRow(ctx, `
			WITH me AS (
			  SELECT id, type, COALESCE(published_at, now()) AS pub
			  FROM content_items WHERE id=$1
			),
			mine AS (
			  SELECT entity_id FROM content_entities WHERE content_id=$1
			),
			candidates AS (
			  SELECT sci.cluster_id,
			         max(similarity(COALESCE(c.translated_title,c.title), $2)) AS sim,
			         max(shared.n) AS shared,
			         min(abs(extract(epoch FROM c.published_at - me.pub))) AS gap
			  FROM content_items c
			  JOIN story_cluster_items sci ON sci.content_id=c.id
			  CROSS JOIN me
			  CROSS JOIN LATERAL (
			    SELECT count(*)::int AS n FROM content_entities ce
			    WHERE ce.content_id=c.id AND ce.entity_id IN (SELECT entity_id FROM mine)
			  ) shared
			  WHERE c.id <> me.id
			    AND c.type = me.type
			    AND c.published_at > me.pub - $4::interval
			    AND c.published_at < me.pub + $4::interval
			  GROUP BY sci.cluster_id
			)
			SELECT cluster_id, sim FROM candidates
			WHERE sim >= $3
			   OR (shared >= $5 AND gap <= extract(epoch FROM $6::interval))
			ORDER BY (shared * 0.25 + sim) DESC, sim DESC
			LIMIT 1`,
			contentID, title, titleSimilarityFloor, titleMatchWindow,
			sharedEntityFloor, entityMatchWindow).Scan(&clusterID, &similarity)
		if errors.Is(err, pgx.ErrNoRows) {
			err = tx.QueryRow(ctx, `INSERT INTO story_clusters
				(representative_title,primary_content_id) VALUES ($1,$2) RETURNING id`, title, contentID).Scan(&clusterID)
			similarity = 1
		}
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO story_cluster_items (cluster_id,content_id,similarity)
			VALUES ($1,$2,$3) ON CONFLICT (content_id) DO NOTHING`, clusterID, contentID, similarity); err != nil {
			return err
		}
		return r.refreshCluster(ctx, tx, clusterID)
	})
}

func (r *ContentRepo) refreshCluster(ctx context.Context, tx pgx.Tx, clusterID int64) error {
	// Verification is weighted by source quality, not a raw source count: three
	// low-quality outlets copying each other must NOT read as "confirmed", while
	// two independent high-quality sources should. quality_sources counts
	// distinct sources with quality >= 4 (tier-1 / reliable).
	_, err := tx.Exec(ctx, `
		WITH stats AS (
			SELECT count(DISTINCT c.source_id)::int AS sources,
			       count(DISTINCT c.source_id) FILTER (WHERE s.quality >= 4)::int AS quality_sources,
			       (array_agg(c.id ORDER BY c.final_score DESC,c.published_at DESC))[1] AS primary_id
			FROM story_cluster_items sci
			JOIN content_items c ON c.id=sci.content_id
			JOIN sources s ON s.id=c.source_id
			WHERE sci.cluster_id=$1)
		UPDATE story_clusters sc SET
			source_count=stats.sources,
			primary_content_id=stats.primary_id,
			verification_status=CASE
				WHEN (stats.sources>=3 AND stats.quality_sources>=1) OR stats.quality_sources>=2 THEN 'confirmed'
				WHEN stats.sources>=2 OR stats.quality_sources>=1 THEN 'verifying'
				ELSE 'rumor' END,
			updated_at=now()
		FROM stats WHERE sc.id=$1`, clusterID)
	return err
}

// IDsWithoutCluster returns publishable stories that still need grouping.
func (r *ContentRepo) IDsWithoutCluster(ctx context.Context, limit int) ([]int64, error) {
	rows, err := r.db.Pool.Query(ctx, `SELECT c.id FROM content_items c
		WHERE c.status='ready' AND NOT EXISTS
		(SELECT 1 FROM story_cluster_items sci WHERE sci.content_id=c.id)
		ORDER BY c.published_at DESC NULLS LAST LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// GetStoryCluster returns the cluster and its independent source coverage.
func (r *ContentRepo) GetStoryCluster(ctx context.Context, id int64) (*domain.StoryCluster, error) {
	var cluster domain.StoryCluster
	err := r.db.Pool.QueryRow(ctx, `SELECT id,representative_title,primary_content_id,
		verification_status,source_count,created_at,updated_at FROM story_clusters WHERE id=$1`, id).
		Scan(&cluster.ID, &cluster.RepresentativeTitle, &cluster.PrimaryContentID,
			&cluster.VerificationStatus, &cluster.SourceCount, &cluster.CreatedAt, &cluster.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	cluster.RepresentativeTitle = textutil.DecodeHTMLEntities(cluster.RepresentativeTitle)
	rows, err := r.db.Pool.Query(ctx, `SELECT `+contentCols+`,s.name
		FROM story_cluster_items sci
		JOIN content_items c ON c.id=sci.content_id
		JOIN sources s ON s.id=c.source_id
		WHERE sci.cluster_id=$1 AND c.status='ready'
		ORDER BY c.final_score DESC,c.published_at DESC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cluster.Items, err = collectContent(rows)
	return &cluster, err
}
