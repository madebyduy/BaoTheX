package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"repwire/internal/domain"
)

type AnalysisRepo struct{ db *DB }

// UpsertCandidates records scored contenders for the admin desk without picking
// or drafting anything.
//
// There used to be a second scorer here — a SQL formula built around
// source_count*18 — while the daily pick scored with process.ClusterHeat. Both
// wrote analysis_candidates.score, so the desk's "heat" column mixed two scales:
// refreshed rows landed at 100-160, the day's actual pick at 59, and the chosen
// story looked like the weakest on the board. Worse, the old formula's answers
// were wrong in substance as well as scale: weighting raw source count above
// source quality ranked a fixture-list announcement top and a disputed semi-final
// fourth. One scorer now, in Go, shared by both paths.
//
// Rows already published, drafting or dismissed keep their score: re-ranking a
// decision that has been acted on only confuses the record.
func (r *AnalysisRepo) UpsertCandidates(ctx context.Context, picks []DailyPick) (int64, error) {
	var affected int64
	for _, p := range picks {
		terms, err := json.Marshal(p.Terms)
		if err != nil {
			return affected, err
		}
		tag, err := r.db.Pool.Exec(ctx, `
			INSERT INTO analysis_candidates
			  (cluster_id, score, source_count, high_quality_sources, velocity_24h,
			   velocity_6h, heat_score, follower_weight, controversy_score,
			   action_score, heat_terms, status, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$2,$7,$8,$9,$10,'proposed',now())
			ON CONFLICT (cluster_id) DO UPDATE SET
			  score=EXCLUDED.score, source_count=EXCLUDED.source_count,
			  high_quality_sources=EXCLUDED.high_quality_sources,
			  velocity_24h=EXCLUDED.velocity_24h, velocity_6h=EXCLUDED.velocity_6h,
			  heat_score=EXCLUDED.heat_score, follower_weight=EXCLUDED.follower_weight,
			  controversy_score=EXCLUDED.controversy_score,
			  action_score=EXCLUDED.action_score, heat_terms=EXCLUDED.heat_terms,
			  updated_at=now()
			WHERE analysis_candidates.status IN ('proposed','failed')`,
			p.ClusterID, p.Heat, p.Cluster.SourceCount, p.Cluster.QualitySources,
			p.Cluster.Velocity24h, p.Cluster.Velocity6h, p.Cluster.FollowerWeight,
			p.Controversy, p.Action, terms)
		if err != nil {
			return affected, err
		}
		affected += tag.RowsAffected()
	}
	return affected, nil
}

// HotTopicContenders returns the day's clusters with the structural signals the
// database can compute for free, plus every headline so controversy can be
// scored in Go (see process.ClusterHeat).
//
// The bar here is deliberately lower than RefreshCandidates': it accepts
// 'verifying' clusters and two sources, because a story breaking at 8pm has not
// had time to be corroborated five times yet, and that is exactly the story an
// end-of-day pick should be allowed to catch.
func (r *AnalysisRepo) HotTopicContenders(ctx context.Context, window time.Duration, limit int) ([]domain.HotTopicCluster, error) {
	if limit <= 0 || limit > 200 {
		limit = 60
	}
	rows, err := r.db.Pool.Query(ctx, `
		SELECT sc.id, sc.representative_title,
		  array_agg(DISTINCT COALESCE(c.translated_title, c.title)) AS titles,
		  count(DISTINCT c.source_id)::int AS source_count,
		  count(DISTINCT c.source_id) FILTER (WHERE s.quality >= 4)::int AS quality_sources,
		  count(*) FILTER (WHERE c.published_at >= now()-interval '6 hours')::int AS velocity_6h,
		  count(*) FILTER (WHERE c.published_at >= now()-interval '24 hours')::int AS velocity_24h,
		  count(*) FILTER (WHERE b.translation_status='ready' OR c.language='vi')::int AS translated,
		  COALESCE((SELECT sum(t.follower_count) FROM (
		    SELECT DISTINCT t.id, t.follower_count
		    FROM story_cluster_items x
		    JOIN content_topics ct ON ct.content_id=x.content_id
		    JOIN topics t ON t.id=ct.topic_id WHERE x.cluster_id=sc.id
		  ) t), 0)::int AS followers
		FROM story_clusters sc
		JOIN story_cluster_items sci ON sci.cluster_id=sc.id
		JOIN content_items c ON c.id=sci.content_id
		JOIN sources s ON s.id=c.source_id
		LEFT JOIN content_bodies b ON b.content_id=c.id
		WHERE c.type='article'
		  AND c.status NOT IN ('failed','hidden')
		  AND c.published_at >= now()-$1::interval
		  -- Exclude stories the desk has already committed to. picked_for_date is
		  -- set only by ClaimDailyPick, never by UpsertCandidates, so this drops
		  -- the drafted/reviewed/published and keeps the merely proposed — which
		  -- is the pool worth ranking.
		  --
		  -- This mattered little while exactly one story was picked per day: the
		  -- pick happened once and the question never came up again. It is
		  -- load-bearing now that several are picked. Without it the ranker would
		  -- return the same hottest cluster on every pass, and ClaimDailyPick's
		  -- ON CONFLICT (cluster_id) DO UPDATE would re-draft that one story N
		  -- times over instead of covering N stories.
		  AND NOT EXISTS (
		      SELECT 1 FROM analysis_candidates ac
		      WHERE ac.cluster_id = sc.id AND ac.picked_for_date IS NOT NULL)
		GROUP BY sc.id
		HAVING count(DISTINCT c.source_id) >= 2
		ORDER BY count(DISTINCT c.source_id) FILTER (WHERE s.quality >= 4) DESC,
		         count(DISTINCT c.source_id) DESC
		LIMIT $2`, window.String(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.HotTopicCluster, 0)
	for rows.Next() {
		var c domain.HotTopicCluster
		if err := rows.Scan(&c.ClusterID, &c.RepresentativeTitle, &c.Titles,
			&c.SourceCount, &c.QualitySources, &c.Velocity6h, &c.Velocity24h,
			&c.TranslatedMaterials, &c.FollowerWeight); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// PickedForDate returns the cluster already chosen for the given day, if any.
// PicksForDate returns how many stories the desk has already committed to on
// this date, and when the most recent one was claimed. A zero time means none.
//
// It replaces a "has today been decided?" check that could only ever answer once
// per day, because the desk now commits to several stories and needs to know how
// many and how long ago rather than merely whether.
func (r *AnalysisRepo) PicksForDate(ctx context.Context, day time.Time) (count int, last time.Time, err error) {
	var lastAt *time.Time
	err = r.db.Pool.QueryRow(ctx, `
		SELECT count(*)::int, max(selected_at)
		FROM analysis_candidates
		WHERE picked_for_date = $1::date`, day.Format("2006-01-02")).Scan(&count, &lastAt)
	if err != nil {
		return 0, time.Time{}, err
	}
	if lastAt != nil {
		last = *lastAt
	}
	return count, last, nil
}

func (r *AnalysisRepo) PickedForDate(ctx context.Context, day time.Time) (*domain.AnalysisCandidate, error) {
	rows, err := r.db.Pool.Query(ctx, candidateSelect+`
		WHERE ac.picked_for_date=$1::date LIMIT 1`, day.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list, err := collectCandidates(rows)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, domain.ErrNotFound
	}
	return &list[0], nil
}

// DailyPick is the outcome of ranking, passed back down for storage. It carries
// plain values rather than a process.HeatSignals because process already imports
// this package; keeping the dependency one-way avoids an import cycle.
type DailyPick struct {
	ClusterID   int64
	Heat        float64
	Controversy float64
	Action      float64
	Terms       []string
	Cluster     domain.HotTopicCluster
}

// ClaimDailyPick records a cluster as the day's topic and moves it straight to
// 'drafting'. The partial unique index on picked_for_date is what makes this
// safe: if two workers pick concurrently, one insert loses and gets ErrNotFound
// rather than both spending the day's LLM budget on rival drafts.
func (r *AnalysisRepo) ClaimDailyPick(ctx context.Context, p DailyPick, day time.Time) error {
	terms, err := json.Marshal(p.Terms)
	if err != nil {
		return err
	}
	clusterID, heat, in := p.ClusterID, p.Heat, p.Cluster
	tag, err := r.db.Pool.Exec(ctx, `
		INSERT INTO analysis_candidates
		  (cluster_id, score, source_count, high_quality_sources, velocity_24h,
		   velocity_6h, heat_score, follower_weight, controversy_score,
		   action_score, heat_terms, picked_for_date, status, selected_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$2,$7,$8,$9,$10,$11::date,'drafting',now(),now())
		ON CONFLICT (cluster_id) DO UPDATE SET
		  score=EXCLUDED.score, source_count=EXCLUDED.source_count,
		  high_quality_sources=EXCLUDED.high_quality_sources,
		  velocity_24h=EXCLUDED.velocity_24h, velocity_6h=EXCLUDED.velocity_6h,
		  heat_score=EXCLUDED.heat_score, follower_weight=EXCLUDED.follower_weight,
		  controversy_score=EXCLUDED.controversy_score,
		  action_score=EXCLUDED.action_score, heat_terms=EXCLUDED.heat_terms,
		  picked_for_date=EXCLUDED.picked_for_date, status='drafting',
		  selected_at=now(), last_error=NULL, progress_stage='queued',
		  progress_current=0,progress_total=0,retry_at=NULL,updated_at=now()
		WHERE analysis_candidates.status <> 'published'`,
		clusterID, heat, in.SourceCount, in.QualitySources, in.Velocity24h,
		in.Velocity6h, in.FollowerWeight, p.Controversy, p.Action,
		terms, day.Format("2006-01-02"))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// candidateSelect is the shared projection for analysis_candidates. Callers
// append their own WHERE/ORDER/LIMIT. collectCandidates scans exactly these
// columns in this order — change one and you must change the other.
const candidateSelect = `
	SELECT ac.id,ac.cluster_id,sc.representative_title,ac.score,ac.source_count,
	 ac.high_quality_sources,ac.velocity_24h,ac.velocity_6h,ac.heat_score,
	 ac.controversy_score,ac.action_score,ac.heat_terms,ac.follower_weight,
	 ac.picked_for_date,ac.status,ac.consensus,ac.conflicts,ac.unique_claims,
	 ac.open_questions,ac.draft_content_id,ac.last_error,ac.proposed_at,
	 ac.progress_stage,ac.progress_current,ac.progress_total,ac.retry_at,
	 ac.selected_at,ac.generated_at,ac.updated_at
	FROM analysis_candidates ac JOIN story_clusters sc ON sc.id=ac.cluster_id`

func collectCandidates(rows pgx.Rows) ([]domain.AnalysisCandidate, error) {
	result := make([]domain.AnalysisCandidate, 0)
	for rows.Next() {
		var c domain.AnalysisCandidate
		if err := rows.Scan(&c.ID, &c.ClusterID, &c.RepresentativeTitle, &c.Score, &c.SourceCount,
			&c.HighQualitySources, &c.Velocity24h, &c.Velocity6h, &c.HeatScore,
			&c.ControversyScore, &c.ActionScore, &c.HeatTerms, &c.FollowerWeight,
			&c.PickedForDate, &c.Status, &c.Consensus, &c.Conflicts, &c.UniqueClaims,
			&c.OpenQuestions, &c.DraftContentID, &c.LastError, &c.ProposedAt,
			&c.ProgressStage, &c.ProgressCurrent, &c.ProgressTotal, &c.RetryAt,
			&c.SelectedAt, &c.GeneratedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

// ListCandidates returns the analysis desk's queue.
//
// Anything awaiting a human decision sorts first, ahead of heat score. That
// ordering is the whole point of the list and it used to be missing: the desk
// fetches this endpoint unfiltered and picks out the drafts client-side, so with
// a pure score ordering and a limit of thirty, eighty-odd 'proposed' candidates
// filled the response and pushed the finished drafts out of it. The desk then
// reported "no drafts waiting" while two sat in the database — the one number an
// editor actually opens the page for, wrong, because the queue was sorted by
// what was hottest rather than by what needed answering.
func (r *AnalysisRepo) ListCandidates(ctx context.Context, status string, limit int) ([]domain.AnalysisCandidate, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	rows, err := r.db.Pool.Query(ctx, candidateSelect+`
		WHERE ($1='' OR ac.status=$1)
		ORDER BY CASE ac.status
		           WHEN 'needs_review' THEN 0
		           WHEN 'drafting' THEN 1
		           WHEN 'failed' THEN 2
		           ELSE 3
		         END,
		         ac.score DESC, ac.updated_at DESC
		LIMIT $2`, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectCandidates(rows)
}

func (r *AnalysisRepo) GetMaterials(ctx context.Context, clusterID int64) ([]domain.AnalysisMaterial, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT c.id,COALESCE(c.translated_title,c.title),s.name,s.quality,c.published_at,
		 COALESCE(c.summary,''),c.key_points,
		 COALESCE(NULLIF(b.vietnamese_body,''),CASE WHEN c.language='vi' THEN b.original_body END,'')
		FROM story_cluster_items sci
		-- A human can explicitly send a newly ingested item from the review queue
		-- to the desk. Keep it in the source packet, but never include failed or
		-- hidden content.
		-- (SQL comments start with --; a // here is a syntax error that killed
		-- every cluster analysis job before it could read its own sources.)
		JOIN content_items c ON c.id=sci.content_id AND c.status IN ('ready','needs_review')
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

// QueueContentForAnalysis lets an editor deliberately nominate a reviewed
// article's story cluster for a BaoTheX analysis. The minimum independent-source
// check happens before a job is created: a witty point of view is still not a
// licence to manufacture corroboration from one report.
func (r *AnalysisRepo) QueueContentForAnalysis(ctx context.Context, contentID int64) (int64, error) {
	var clusterID int64
	err := r.db.WithTx(ctx, func(tx pgx.Tx) error {
		var contentType domain.ContentType
		err := tx.QueryRow(ctx, `SELECT c.type,sci.cluster_id
			FROM content_items c
			JOIN story_cluster_items sci ON sci.content_id=c.id
			WHERE c.id=$1`, contentID).Scan(&contentType, &clusterID)
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: bai viet chua duoc gom vao cum su kien", domain.ErrValidation)
		}
		if err != nil {
			return err
		}
		if contentType != domain.ContentArticle {
			return fmt.Errorf("%w: chi bai viet moi co the dua vao hang cho Goc nhin", domain.ErrValidation)
		}

		var sources, highQuality, velocity24, velocity6 int
		err = tx.QueryRow(ctx, `SELECT count(DISTINCT c.source_id)::int,
			count(DISTINCT c.source_id) FILTER (WHERE s.quality >= 4)::int,
			count(*) FILTER (WHERE c.published_at >= now()-interval '24 hours')::int,
			count(*) FILTER (WHERE c.published_at >= now()-interval '6 hours')::int
			FROM story_cluster_items sci
			JOIN content_items c ON c.id=sci.content_id
			JOIN sources s ON s.id=c.source_id
			LEFT JOIN content_bodies b ON b.content_id=c.id
			WHERE sci.cluster_id=$1
			  AND c.type='article'
			  AND c.status IN ('ready','needs_review')
			  AND (c.language='vi' OR (b.translation_status='ready' AND length(trim(COALESCE(b.vietnamese_body,'')))>=400))`, clusterID).
			Scan(&sources, &highQuality, &velocity24, &velocity6)
		if err != nil {
			return err
		}
		if sources < 3 {
			return fmt.Errorf("%w: can it nhat 3 nguon doc lap de viet Goc nhin (hien co %d)", domain.ErrValidation, sources)
		}

		tag, err := tx.Exec(ctx, `INSERT INTO analysis_candidates
			(cluster_id,score,source_count,high_quality_sources,velocity_24h,velocity_6h,heat_score,status,selected_at,updated_at)
			VALUES ($1,0,$2,$3,$4,$5,0,'drafting',now(),now())
			ON CONFLICT (cluster_id) DO UPDATE SET
			 source_count=EXCLUDED.source_count,high_quality_sources=EXCLUDED.high_quality_sources,
			 velocity_24h=EXCLUDED.velocity_24h,velocity_6h=EXCLUDED.velocity_6h,
			 status='drafting',selected_at=now(),last_error=NULL,progress_stage='queued',
			 progress_current=0,progress_total=0,retry_at=NULL,updated_at=now()
			WHERE analysis_candidates.status IN ('proposed','failed','dismissed')`,
			clusterID, sources, highQuality, velocity24, velocity6)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("%w: chu de nay dang duoc viet, cho duyet, hoac da xuat ban", domain.ErrConflict)
		}
		return nil
	})
	return clusterID, err
}

// QueueContentForPerspective is the single-article sibling of
// QueueContentForAnalysis. It deliberately SKIPS the three-source gate — a Góc
// nhìn written on one article is an editorial choice, not a cross-source
// synthesis — but reuses the same candidate row so the draft flows through the
// exact same review-and-publish desk. It returns the article's cluster id.
func (r *AnalysisRepo) QueueContentForPerspective(ctx context.Context, contentID int64) (int64, error) {
	var clusterID int64
	err := r.db.WithTx(ctx, func(tx pgx.Tx) error {
		var contentType domain.ContentType
		err := tx.QueryRow(ctx, `SELECT c.type,sci.cluster_id
			FROM content_items c
			JOIN story_cluster_items sci ON sci.content_id=c.id
			WHERE c.id=$1`, contentID).Scan(&contentType, &clusterID)
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: bai viet chua duoc gom vao cum su kien", domain.ErrValidation)
		}
		if err != nil {
			return err
		}
		if contentType != domain.ContentArticle {
			return fmt.Errorf("%w: chi bai viet moi co the tao Goc nhin", domain.ErrValidation)
		}

		// Metrics are for the desk's display only here; unlike the analysis path
		// they do not gate whether the draft may be written.
		var sources, highQuality, velocity24, velocity6 int
		err = tx.QueryRow(ctx, `SELECT count(DISTINCT c.source_id)::int,
			count(DISTINCT c.source_id) FILTER (WHERE s.quality >= 4)::int,
			count(*) FILTER (WHERE c.published_at >= now()-interval '24 hours')::int,
			count(*) FILTER (WHERE c.published_at >= now()-interval '6 hours')::int
			FROM story_cluster_items sci
			JOIN content_items c ON c.id=sci.content_id
			JOIN sources s ON s.id=c.source_id
			WHERE sci.cluster_id=$1 AND c.type='article'`, clusterID).
			Scan(&sources, &highQuality, &velocity24, &velocity6)
		if err != nil {
			return err
		}

		tag, err := tx.Exec(ctx, `INSERT INTO analysis_candidates
			(cluster_id,score,source_count,high_quality_sources,velocity_24h,velocity_6h,heat_score,status,selected_at,updated_at)
			VALUES ($1,0,$2,$3,$4,$5,0,'drafting',now(),now())
			ON CONFLICT (cluster_id) DO UPDATE SET
			 source_count=EXCLUDED.source_count,high_quality_sources=EXCLUDED.high_quality_sources,
			 velocity_24h=EXCLUDED.velocity_24h,velocity_6h=EXCLUDED.velocity_6h,
			 status='drafting',selected_at=now(),last_error=NULL,progress_stage='queued',
			 progress_current=0,progress_total=0,retry_at=NULL,updated_at=now()
			WHERE analysis_candidates.status IN ('proposed','failed','dismissed')`,
			clusterID, sources, highQuality, velocity24, velocity6)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("%w: chu de nay dang duoc viet, cho duyet, hoac da xuat ban", domain.ErrConflict)
		}
		return nil
	})
	return clusterID, err
}

func (r *AnalysisRepo) MarkDrafting(ctx context.Context, clusterID int64) error {
	tag, err := r.db.Pool.Exec(ctx, `UPDATE analysis_candidates SET status='drafting',selected_at=now(),last_error=NULL,
		progress_stage='queued',progress_current=0,progress_total=0,retry_at=NULL,updated_at=now()
		WHERE cluster_id=$1 AND status IN ('proposed','failed','needs_review')`, clusterID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *AnalysisRepo) MarkFailed(ctx context.Context, clusterID int64, cause error) error {
	_, err := r.db.Pool.Exec(ctx, `UPDATE analysis_candidates SET status='failed',progress_stage='failed',
		retry_at=NULL,last_error=$2,updated_at=now() WHERE cluster_id=$1`, clusterID, cause.Error())
	return err
}

func (r *AnalysisRepo) UpdateProgress(ctx context.Context, clusterID int64, stage string, current, total int, retryAt *time.Time) error {
	_, err := r.db.Pool.Exec(ctx, `UPDATE analysis_candidates SET progress_stage=$2,
		progress_current=$3,progress_total=$4,retry_at=$5,updated_at=now() WHERE cluster_id=$1`,
		clusterID, stage, current, total, retryAt)
	return err
}

func (r *AnalysisRepo) SaveClaimsCheckpoint(ctx context.Context, clusterID int64, claims domain.AnalysisClaims) error {
	raw, err := json.Marshal(claims)
	if err != nil {
		return err
	}
	_, err = r.db.Pool.Exec(ctx, `UPDATE analysis_candidates SET checkpoint_claims=$2,
		progress_stage='writing_draft',updated_at=now() WHERE cluster_id=$1`, clusterID, raw)
	return err
}

func (r *AnalysisRepo) LoadClaimsCheckpoint(ctx context.Context, clusterID int64) (*domain.AnalysisClaims, bool, error) {
	var raw []byte
	err := r.db.Pool.QueryRow(ctx, `SELECT checkpoint_claims FROM analysis_candidates WHERE cluster_id=$1`, clusterID).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) || len(raw) == 0 {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var claims domain.AnalysisClaims
	if err := json.Unmarshal(raw, &claims); err != nil {
		return nil, false, err
	}
	return &claims, true, nil
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
		// $1::bigint, not bare $1. A parameter in a SELECT list has no column to
		// infer its type from, so Postgres assumes text and the insert fails with
		// "column content_id is of type bigint but expression is of type text".
		// This sat here unreached for as long as the analysis job died earlier;
		// it only surfaced once the desk got far enough to write a draft.
		if _, err := tx.Exec(ctx, `INSERT INTO content_topics(content_id,topic_id,confidence,is_primary)
			SELECT DISTINCT $1::bigint,ct.topic_id,0.9,false FROM story_cluster_items sci
			JOIN content_topics ct ON ct.content_id=sci.content_id WHERE sci.cluster_id=$2::bigint
			ON CONFLICT(content_id,topic_id) DO NOTHING`, contentID, clusterID); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE analysis_candidates SET status='needs_review',consensus=$2,
			conflicts=$3,unique_claims=$4,open_questions=$5,draft_content_id=$6,
			generated_at=now(),last_error=NULL,progress_stage='completed',progress_current=0,
			progress_total=0,retry_at=NULL,checkpoint_claims=NULL,updated_at=now() WHERE cluster_id=$1`,
			clusterID, claims.Consensus, claims.Conflicts, claims.UniqueClaims, claims.OpenQuestions, contentID)
		return err
	})
	return contentID, err
}

// Publish approves a reviewed draft: it flips the draft content to 'ready' so it
// appears publicly, and marks the candidate 'published'. It only works while the
// candidate is awaiting review, so a piece cannot be published twice or skip the
// editorial gate.
func (r *AnalysisRepo) Publish(ctx context.Context, clusterID int64) (int64, error) {
	var contentID int64
	err := r.db.WithTx(ctx, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `SELECT draft_content_id FROM analysis_candidates
			WHERE cluster_id=$1 AND status='needs_review' AND draft_content_id IS NOT NULL`, clusterID).Scan(&contentID)
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE content_items
			SET status='ready',published_at=COALESCE(published_at,now()),updated_at=now()
			WHERE id=$1`, contentID); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE analysis_candidates
			SET status='published',updated_at=now() WHERE cluster_id=$1`, clusterID)
		return err
	})
	return contentID, err
}

// Published lists the analyses readers can see.
//
// The corroboration trio is taken from the candidate rather than from
// contentCols' usual cluster lookup. A published analysis is a freshly written
// content_item that belongs to no story cluster of its own, so the standard
// lookup found nothing and fell back to "1 source, unverified" — which the
// front page then printed directly underneath the words "phân tích đa nguồn",
// on a piece the desk only commissions when at least three outlets have covered
// the story. The candidate row remembers the cluster the piece was drafted from,
// and that is the number the reader is owed.
func (r *AnalysisRepo) Published(ctx context.Context, limit int) ([]domain.ContentItem, error) {
	rows, err := r.db.Pool.Query(ctx, `SELECT `+contentBaseCols+`,
		ac.cluster_id,
		GREATEST(COALESCE(ac.source_count,1),1),
		COALESCE((SELECT sc.verification_status FROM story_clusters sc WHERE sc.id=ac.cluster_id),'verifying'),
		`+contentSourceQualityCol+`,s.name FROM analysis_candidates ac
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
