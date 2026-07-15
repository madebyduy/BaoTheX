package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"repwire/internal/domain"
)

type AnalysisRepo struct{ db *DB }

// RefreshCandidates scores confirmed, multi-source clusters without drafting
// anything. This is the automated filter before an editor spends LLM quota.
func (r *AnalysisRepo) RefreshCandidates(ctx context.Context, limit int) (int64, error) {
	if limit <= 0 {
		limit = 5
	}
	tag, err := r.db.Pool.Exec(ctx, `
		WITH ranked AS (
		  SELECT sc.id AS cluster_id,
		         sc.source_count,
		         count(DISTINCT c.source_id) FILTER (WHERE s.quality >= 4)::int AS quality_sources,
		         count(*) FILTER (WHERE c.published_at >= now()-interval '24 hours')::int AS velocity,
		         COALESCE(sum(c.final_score),0)::real AS heat,
		         COALESCE((SELECT sum(t.follower_count) FROM (
		           SELECT DISTINCT t.id,t.follower_count
		           FROM story_cluster_items x
		           JOIN content_topics ct ON ct.content_id=x.content_id
		           JOIN topics t ON t.id=ct.topic_id WHERE x.cluster_id=sc.id
		         ) t),0)::int AS followers,
		         (sc.source_count*18
		          + count(DISTINCT c.source_id) FILTER (WHERE s.quality >= 4)*10
		          + LEAST(COALESCE(sum(c.final_score),0),80)*0.35
		          + count(*) FILTER (WHERE c.published_at >= now()-interval '24 hours')*8
		          + LEAST(COALESCE((SELECT sum(t.follower_count) FROM (
		              SELECT DISTINCT t.id,t.follower_count
		              FROM story_cluster_items x
		              JOIN content_topics ct ON ct.content_id=x.content_id
		              JOIN topics t ON t.id=ct.topic_id WHERE x.cluster_id=sc.id
		            ) t),0),1000)*0.02)::real AS score
		  FROM story_clusters sc
		  JOIN story_cluster_items sci ON sci.cluster_id=sc.id
		  JOIN content_items c ON c.id=sci.content_id AND c.status='ready'
		  JOIN sources s ON s.id=c.source_id
		  WHERE sc.verification_status='confirmed' AND sc.source_count >= 3
		    AND sc.updated_at >= now()-interval '48 hours'
		  GROUP BY sc.id
		  HAVING count(DISTINCT c.source_id) FILTER (WHERE s.quality >= 4) >= 2
		  ORDER BY score DESC LIMIT $1
		)
		INSERT INTO analysis_candidates
		  (cluster_id,score,source_count,high_quality_sources,velocity_24h,heat_score,follower_weight)
		SELECT cluster_id,score,source_count,quality_sources,velocity,heat,followers FROM ranked
		ON CONFLICT (cluster_id) DO UPDATE SET
		  score=EXCLUDED.score,source_count=EXCLUDED.source_count,
		  high_quality_sources=EXCLUDED.high_quality_sources,
		  velocity_24h=EXCLUDED.velocity_24h,heat_score=EXCLUDED.heat_score,
		  follower_weight=EXCLUDED.follower_weight,updated_at=now()
		WHERE analysis_candidates.status IN ('proposed','failed')`, limit)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (r *AnalysisRepo) ListCandidates(ctx context.Context, status string, limit int) ([]domain.AnalysisCandidate, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	rows, err := r.db.Pool.Query(ctx, `
		SELECT ac.id,ac.cluster_id,sc.representative_title,ac.score,ac.source_count,
		 ac.high_quality_sources,ac.velocity_24h,ac.heat_score,ac.follower_weight,
		 ac.status,ac.consensus,ac.conflicts,ac.unique_claims,ac.open_questions,
		 ac.draft_content_id,ac.last_error,ac.proposed_at,ac.selected_at,ac.generated_at,ac.updated_at
		FROM analysis_candidates ac JOIN story_clusters sc ON sc.id=ac.cluster_id
		WHERE ($1='' OR ac.status=$1) ORDER BY ac.score DESC,ac.updated_at DESC LIMIT $2`, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]domain.AnalysisCandidate, 0)
	for rows.Next() {
		var c domain.AnalysisCandidate
		if err := rows.Scan(&c.ID, &c.ClusterID, &c.RepresentativeTitle, &c.Score, &c.SourceCount,
			&c.HighQualitySources, &c.Velocity24h, &c.HeatScore, &c.FollowerWeight,
			&c.Status, &c.Consensus, &c.Conflicts, &c.UniqueClaims, &c.OpenQuestions,
			&c.DraftContentID, &c.LastError, &c.ProposedAt, &c.SelectedAt, &c.GeneratedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

func (r *AnalysisRepo) GetMaterials(ctx context.Context, clusterID int64) ([]domain.AnalysisMaterial, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT c.id,COALESCE(c.translated_title,c.title),s.name,s.quality,c.published_at,
		 COALESCE(c.summary,''),c.key_points,
		 COALESCE(NULLIF(b.vietnamese_body,''),CASE WHEN c.language='vi' THEN b.original_body END,'')
		FROM story_cluster_items sci
		JOIN content_items c ON c.id=sci.content_id AND c.status='ready'
		JOIN sources s ON s.id=c.source_id
		LEFT JOIN content_bodies b ON b.content_id=c.id
		WHERE sci.cluster_id=$1
		  AND (c.language='vi' OR (b.translation_status='ready' AND length(trim(COALESCE(b.vietnamese_body,'')))>=400))
		ORDER BY s.quality DESC,c.published_at DESC`, clusterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	materials := make([]domain.AnalysisMaterial, 0)
	for rows.Next() {
		var m domain.AnalysisMaterial
		if err := rows.Scan(&m.ContentID, &m.Title, &m.SourceName, &m.SourceQuality, &m.PublishedAt,
			&m.Summary, &m.KeyPoints, &m.Body); err != nil {
			return nil, err
		}
		if strings.TrimSpace(m.Body) == "" {
			m.Body = m.Summary
		}
		materials = append(materials, m)
	}
	return materials, rows.Err()
}

func (r *AnalysisRepo) MarkDrafting(ctx context.Context, clusterID int64) error {
	tag, err := r.db.Pool.Exec(ctx, `UPDATE analysis_candidates SET status='drafting',selected_at=now(),last_error=NULL,updated_at=now() WHERE cluster_id=$1 AND status IN ('proposed','failed')`, clusterID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *AnalysisRepo) MarkFailed(ctx context.Context, clusterID int64, cause error) error {
	_, err := r.db.Pool.Exec(ctx, `UPDATE analysis_candidates SET status='failed',last_error=$2,updated_at=now() WHERE cluster_id=$1`, clusterID, cause.Error())
	return err
}

func (r *AnalysisRepo) Dismiss(ctx context.Context, clusterID int64) error {
	tag, err := r.db.Pool.Exec(ctx, `UPDATE analysis_candidates SET status='dismissed',updated_at=now() WHERE cluster_id=$1 AND status <> 'published'`, clusterID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *AnalysisRepo) CreateDraft(ctx context.Context, clusterID int64, claims domain.AnalysisClaims, draft domain.AnalysisDraft) (int64, error) {
	var contentID int64
	err := r.db.WithTx(ctx, func(tx pgx.Tx) error {
		var sourceID int64
		if err := tx.QueryRow(ctx, `SELECT id FROM sources WHERE kind='manual' AND name='Góc nhìn BaoTheX' LIMIT 1`).Scan(&sourceID); err != nil {
			return err
		}
		canonical := fmt.Sprintf("/goc-nhin/cluster-%d", clusterID)
		hash := sha256.Sum256([]byte(canonical))
		urlHash := hex.EncodeToString(hash[:])
		err := tx.QueryRow(ctx, `INSERT INTO content_items
			(source_id,type,status,title,canonical_url,url_hash,excerpt,summary,key_points,language,published_at,base_score)
			VALUES ($1,'article','needs_review',$2,$3,$4,$5,$5,$6,'vi',now(),20)
			ON CONFLICT (url_hash) DO UPDATE SET title=$2,excerpt=$5,summary=$5,key_points=$6,status='needs_review',updated_at=now()
			RETURNING id`, sourceID, draft.Title, canonical, urlHash, draft.Summary, draft.KeyPoints).Scan(&contentID)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO articles(content_id,author,word_count)
			VALUES ($1,'Tòa soạn BaoTheX',array_length(regexp_split_to_array(trim($2),'\s+'),1))
			ON CONFLICT(content_id) DO UPDATE SET author=EXCLUDED.author,word_count=EXCLUDED.word_count`, contentID, draft.Body); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO content_bodies(content_id,original_language,original_body,vietnamese_body,translation_status,translated_at)
			VALUES ($1,'vi',$2,$2,'ready',now()) ON CONFLICT(content_id) DO UPDATE SET
			original_body=$2,vietnamese_body=$2,translation_status='ready',translated_at=now(),updated_at=now()`, contentID, draft.Body); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO content_topics(content_id,topic_id,confidence,is_primary)
			SELECT DISTINCT $1,ct.topic_id,0.9,false FROM story_cluster_items sci
			JOIN content_topics ct ON ct.content_id=sci.content_id WHERE sci.cluster_id=$2
			ON CONFLICT(content_id,topic_id) DO NOTHING`, contentID, clusterID); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE analysis_candidates SET status='needs_review',consensus=$2,
			conflicts=$3,unique_claims=$4,open_questions=$5,draft_content_id=$6,
			generated_at=now(),last_error=NULL,updated_at=now() WHERE cluster_id=$1`,
			clusterID, claims.Consensus, claims.Conflicts, claims.UniqueClaims, claims.OpenQuestions, contentID)
		return err
	})
	return contentID, err
}

func (r *AnalysisRepo) Published(ctx context.Context, limit int) ([]domain.ContentItem, error) {
	rows, err := r.db.Pool.Query(ctx, `SELECT `+contentCols+`,s.name FROM analysis_candidates ac
		JOIN content_items c ON c.id=ac.draft_content_id
		JOIN sources s ON s.id=c.source_id
		WHERE ac.status='published' AND c.status='ready'
		ORDER BY c.published_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectContent(rows)
}
