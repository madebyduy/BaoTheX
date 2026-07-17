package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"repwire/internal/domain"
)

// ContentRepo persists content items and their subtype rows.
type ContentRepo struct{ db *DB }

type MediaTarget struct {
	ID  int64
	URL string
}

// contentCols is the base column set (optionally joined with source name as source_name).
const contentCols = `c.id, c.source_id, c.type, c.status, COALESCE(c.translated_title, c.title), c.canonical_url, c.url_hash,
	c.title_hash, c.image_url, c.excerpt, c.summary, c.key_points, c.language, c.published_at,
	c.discovered_at, c.base_score, c.editorial_boost, c.final_score, c.view_count, c.save_count, c.updated_at,
	(SELECT sci.cluster_id FROM story_cluster_items sci WHERE sci.content_id=c.id LIMIT 1),
	COALESCE((SELECT sc.source_count FROM story_cluster_items sci JOIN story_clusters sc ON sc.id=sci.cluster_id WHERE sci.content_id=c.id LIMIT 1),1),
	COALESCE((SELECT sc.verification_status FROM story_cluster_items sci JOIN story_clusters sc ON sc.id=sci.cluster_id WHERE sci.content_id=c.id LIMIT 1),'rumor'),
	(SELECT quality FROM sources sq WHERE sq.id=c.source_id)`

func scanContent(row pgx.Row) (*domain.ContentItem, error) {
	var c domain.ContentItem
	if err := row.Scan(&c.ID, &c.SourceID, &c.Type, &c.Status, &c.Title, &c.CanonicalURL,
		&c.URLHash, &c.TitleHash, &c.ImageURL, &c.Excerpt, &c.Summary, &c.KeyPoints,
		&c.Language, &c.PublishedAt, &c.DiscoveredAt, &c.BaseScore, &c.EditorialBoost,
		&c.FinalScore, &c.ViewCount, &c.SaveCount, &c.UpdatedAt, &c.StoryClusterID,
		&c.ClusterSourceCount, &c.VerificationStatus, &c.SourceQuality); err != nil {
		return nil, err
	}
	return &c, nil
}

func scanContentWithSource(row pgx.Row) (*domain.ContentItem, error) {
	var c domain.ContentItem
	if err := row.Scan(&c.ID, &c.SourceID, &c.Type, &c.Status, &c.Title, &c.CanonicalURL,
		&c.URLHash, &c.TitleHash, &c.ImageURL, &c.Excerpt, &c.Summary, &c.KeyPoints,
		&c.Language, &c.PublishedAt, &c.DiscoveredAt, &c.BaseScore, &c.EditorialBoost,
		&c.FinalScore, &c.ViewCount, &c.SaveCount, &c.UpdatedAt, &c.StoryClusterID,
		&c.ClusterSourceCount, &c.VerificationStatus, &c.SourceQuality, &c.SourceName); err != nil {
		return nil, err
	}
	return &c, nil
}

// ---- Dedup lookups ----

// ExistsByURLHash reports whether an item with the given url_hash already exists.
func (r *ContentRepo) ExistsByURLHash(ctx context.Context, urlHash string) (bool, error) {
	var exists bool
	err := r.db.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM content_items WHERE url_hash=$1)`, urlHash).Scan(&exists)
	return exists, err
}

// ExistsByYouTubeID reports whether a video with the given youtube_id exists.
func (r *ContentRepo) ExistsByYouTubeID(ctx context.Context, ytID string) (bool, error) {
	var exists bool
	err := r.db.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM videos WHERE youtube_id=$1)`, ytID).Scan(&exists)
	return exists, err
}

// ExistsByDOIOrPMID reports whether a paper with the given doi or pmid exists.
func (r *ContentRepo) ExistsByDOIOrPMID(ctx context.Context, doi, pmid *string) (bool, error) {
	var exists bool
	err := r.db.Pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM research_papers WHERE (doi IS NOT NULL AND doi=$1) OR (pmid IS NOT NULL AND pmid=$2))`,
		doi, pmid).Scan(&exists)
	return exists, err
}

// ExistsByEpisodeGUID reports whether a podcast episode with the guid exists.
func (r *ContentRepo) ExistsByEpisodeGUID(ctx context.Context, guid string) (bool, error) {
	var exists bool
	err := r.db.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM podcast_episodes WHERE episode_guid=$1)`, guid).Scan(&exists)
	return exists, err
}

// SimilarTitle is a soft-dedup candidate (pg_trgm similarity).
type SimilarTitle struct {
	ID    int64
	Title string
	Sim   float64
}

// FindSimilarTitles returns recent items of the same type whose title is similar.
func (r *ContentRepo) FindSimilarTitles(ctx context.Context, title string, t domain.ContentType) ([]SimilarTitle, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT id, title, similarity(title, $1) AS sim
		FROM content_items
		WHERE published_at > now() - interval '7 days'
		  AND type = $2 AND title % $1
		ORDER BY sim DESC LIMIT 3`, title, t)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SimilarTitle
	for rows.Next() {
		var s SimilarTitle
		if err := rows.Scan(&s.ID, &s.Title, &s.Sim); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ---- Inserts ----

// InsertItem inserts the base content_items row and returns the new id. Uses
// ON CONFLICT (url_hash) DO NOTHING; returns (0, nil) on a hard-dedup collision.
func (r *ContentRepo) InsertItem(ctx context.Context, tx pgx.Tx, c *domain.ContentItem) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, `
		INSERT INTO content_items
			(source_id, type, status, title, canonical_url, url_hash, title_hash,
			 image_url, excerpt, summary, key_points, language, published_at, base_score)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		ON CONFLICT (url_hash) DO NOTHING
		RETURNING id`,
		c.SourceID, c.Type, c.Status, c.Title, c.CanonicalURL, c.URLHash, c.TitleHash,
		c.ImageURL, c.Excerpt, c.Summary, c.KeyPoints, c.Language, c.PublishedAt, c.BaseScore).
		Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil // conflict — already exists
	}
	return id, err
}

// InsertArticle inserts the article subtype row.
func (r *ContentRepo) InsertArticle(ctx context.Context, tx pgx.Tx, a *domain.Article) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO articles (content_id, author, word_count) VALUES ($1,$2,$3)
		 ON CONFLICT (content_id) DO NOTHING`,
		a.ContentID, a.Author, a.WordCount)
	return err
}

// InsertResearch inserts the research_papers subtype row.
func (r *ContentRepo) InsertResearch(ctx context.Context, tx pgx.Tx, p *domain.ResearchPaper) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO research_papers
			(content_id, doi, pmid, pmcid, journal, authors, abstract, study_type, is_human,
			 is_open_access, full_text_url, sample_size, population, duration_weeks, sex,
			 training_status, published_year)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
		ON CONFLICT (content_id) DO NOTHING`,
		p.ContentID, p.DOI, p.PMID, p.PMCID, p.Journal, p.Authors, p.Abstract, p.StudyType,
		p.IsHuman, p.IsOpenAccess, p.FullTextURL, p.SampleSize, p.Population, p.DurationWeeks,
		p.Sex, p.TrainingStatus, p.PublishedYear)
	return err
}

// InsertVideo inserts the videos subtype row.
func (r *ContentRepo) InsertVideo(ctx context.Context, tx pgx.Tx, v *domain.Video) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO videos
			(content_id, youtube_id, channel_id, channel_title, duration_sec, thumbnail_url,
			 description, has_transcript, yt_views, yt_likes)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (content_id) DO NOTHING`,
		v.ContentID, v.YouTubeID, v.ChannelID, v.ChannelTitle, v.DurationSec, v.ThumbnailURL,
		v.Description, v.HasTranscript, v.YTViews, v.YTLikes)
	return err
}

// InsertPodcast inserts the podcast_episodes subtype row.
func (r *ContentRepo) InsertPodcast(ctx context.Context, tx pgx.Tx, p *domain.PodcastEpisode) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO podcast_episodes
			(content_id, show_name, episode_guid, audio_url, duration_sec, show_notes)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (content_id) DO NOTHING`,
		p.ContentID, p.ShowName, p.EpisodeGUID, p.AudioURL, p.DurationSec, p.ShowNotes)
	return err
}

// ---- Reads ----

// Get returns a content item by id (with source name).
func (r *ContentRepo) Get(ctx context.Context, id int64) (*domain.ContentItem, error) {
	c, err := scanContentWithSource(r.db.Pool.QueryRow(ctx,
		`SELECT `+contentCols+`, s.name FROM content_items c JOIN sources s ON s.id=c.source_id WHERE c.id=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return c, err
}

// GetPublic returns only content that has completed the editorial pipeline.
// Admin and worker paths continue to use Get so they can inspect drafts.
func (r *ContentRepo) GetPublic(ctx context.Context, id int64) (*domain.ContentItem, error) {
	c, err := scanContentWithSource(r.db.Pool.QueryRow(ctx,
		`SELECT `+contentCols+`, s.name FROM content_items c JOIN sources s ON s.id=c.source_id
		 WHERE c.id=$1 AND c.status='ready'`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return c, err
}

// InsertBody stores the readable source body captured during ingestion.
func (r *ContentRepo) InsertBody(ctx context.Context, tx pgx.Tx, contentID int64, language, body string) error {
	_, err := tx.Exec(ctx, `INSERT INTO content_bodies
		(content_id, original_language, original_body)
		VALUES ($1,$2,$3) ON CONFLICT (content_id) DO NOTHING`, contentID, language, body)
	return err
}

// UpsertBodyByURLHash backfills the body when a source is re-fetched after the
// content item already exists. This lets newly-enabled full-text ingestion
// upgrade items that were previously stored with only an excerpt.
func (r *ContentRepo) UpsertBodyByURLHash(ctx context.Context, urlHash, language, body string) (int64, bool, error) {
	var contentID int64
	err := r.db.Pool.QueryRow(ctx, `INSERT INTO content_bodies
		(content_id, original_language, original_body)
		SELECT id, $2, $3 FROM content_items WHERE url_hash=$1
		ON CONFLICT (content_id) DO UPDATE SET
		original_language=EXCLUDED.original_language,
		original_body=EXCLUDED.original_body,
		translation_status='pending',
		updated_at=now()
		WHERE length(EXCLUDED.original_body) > length(content_bodies.original_body)
		RETURNING content_id`, urlHash, language, body).Scan(&contentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	return contentID, err == nil, err
}

// BackfillMediaByURLHash upgrades an existing RSS item when a later crawl
// discovers its social preview image or a better excerpt.
func (r *ContentRepo) BackfillMediaByURLHash(ctx context.Context, urlHash string, imageURL, excerpt *string) error {
	_, err := r.db.Pool.Exec(ctx, `UPDATE content_items SET
		image_url=CASE WHEN COALESCE(image_url,'')='' AND COALESCE($2::text,'')<>'' THEN $2 ELSE image_url END,
		excerpt=CASE WHEN COALESCE(excerpt,'')='' AND COALESCE($3::text,'')<>'' THEN $3 ELSE excerpt END
		WHERE url_hash=$1`, urlHash, imageURL, excerpt)
	return err
}

// MissingImageTargets returns a small repair batch, prioritising translated
// international coverage where RSS feeds frequently omit media metadata.
func (r *ContentRepo) MissingImageTargets(ctx context.Context, limit int) ([]MediaTarget, error) {
	rows, err := r.db.Pool.Query(ctx, `SELECT id,canonical_url FROM content_items
		WHERE status='ready' AND COALESCE(image_url,'')='' AND COALESCE(canonical_url,'')<>''
		ORDER BY (language<>'vi') DESC,published_at DESC NULLS LAST LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var targets []MediaTarget
	for rows.Next() {
		var target MediaTarget
		if err := rows.Scan(&target.ID, &target.URL); err != nil {
			return nil, err
		}
		targets = append(targets, target)
	}
	return targets, rows.Err()
}

func (r *ContentRepo) BackfillMediaByID(ctx context.Context, id int64, imageURL, excerpt string) error {
	if imageURL == "" && excerpt == "" {
		return nil
	}
	_, err := r.db.Pool.Exec(ctx, `UPDATE content_items SET
		image_url=CASE WHEN COALESCE(image_url,'')='' AND $2<>'' THEN $2 ELSE image_url END,
		excerpt=CASE WHEN COALESCE(excerpt,'')='' AND $3<>'' THEN LEFT($3,1200) ELSE excerpt END,
		updated_at=now() WHERE id=$1`, id, imageURL, excerpt)
	return err
}

// GetBody returns the source body and optional Vietnamese translation.
// BodyReclean is a stored body flagged for boilerplate re-cleaning.
type BodyReclean struct {
	ContentID  int64
	Vietnamese string
	Original   string
}

// BodiesNeedingReclean returns bodies whose stored text still begins a line with
// an end-of-article publisher block, so a re-clean is guaranteed to make
// progress (once the marker line is cut, the row stops matching).
func (r *ContentRepo) BodiesNeedingReclean(ctx context.Context, limit int) ([]BodyReclean, error) {
	const marker = `(?n)^[[:space:]]*(tags?[:：]|đọc nhiều|thông tin doanh nghiệp|tin liên quan|bài liên quan)`
	rows, err := r.db.Pool.Query(ctx, `
		SELECT content_id, COALESCE(vietnamese_body,''), COALESCE(original_body,'')
		FROM content_bodies
		WHERE original_body ~* $1 OR vietnamese_body ~* $1
		ORDER BY content_id DESC
		LIMIT $2`, marker, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BodyReclean
	for rows.Next() {
		var b BodyReclean
		if err := rows.Scan(&b.ContentID, &b.Vietnamese, &b.Original); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// BodiesNeedingRecleanWide also finds legacy bodies where publisher widgets
// were concatenated onto the same line as the article text.
func (r *ContentRepo) BodiesNeedingRecleanWide(ctx context.Context, limit int) ([]BodyReclean, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT content_id, COALESCE(vietnamese_body,''), COALESCE(original_body,'')
		FROM content_bodies
		WHERE strpos(lower(original_body), 'tags:') > 400
		   OR strpos(lower(vietnamese_body), 'tags:') > 400
		   OR strpos(lower(original_body), 'thông tin doanh nghiệp') > 400
		   OR strpos(lower(vietnamese_body), 'thông tin doanh nghiệp') > 400
		   OR strpos(lower(original_body), 'trở lại chủ đề') > 400
		   OR strpos(lower(vietnamese_body), 'trở lại chủ đề') > 400
		   OR strpos(lower(original_body), 'tặng sao') > 400
		   OR strpos(lower(vietnamese_body), 'tặng sao') > 400
		ORDER BY content_id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BodyReclean
	for rows.Next() {
		var b BodyReclean
		if err := rows.Scan(&b.ContentID, &b.Vietnamese, &b.Original); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// UpdateBodyText overwrites a stored body's cleaned text.
func (r *ContentRepo) UpdateBodyText(ctx context.Context, contentID int64, vietnamese, original string) error {
	_, err := r.db.Pool.Exec(ctx, `UPDATE content_bodies
		SET vietnamese_body = NULLIF($2,''), original_body = $3, updated_at = now()
		WHERE content_id = $1`, contentID, vietnamese, original)
	return err
}

func (r *ContentRepo) GetBody(ctx context.Context, contentID int64) (*domain.ContentBody, error) {
	var b domain.ContentBody
	err := r.db.Pool.QueryRow(ctx, `SELECT content_id, original_language, original_body,
		vietnamese_body, translation_status, translated_at, updated_at
		FROM content_bodies WHERE content_id=$1`, contentID).
		Scan(&b.ContentID, &b.OriginalLanguage, &b.OriginalBody, &b.VietnameseBody,
			&b.TranslationStatus, &b.TranslatedAt, &b.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &b, err
}

// SetVietnameseContent saves the translated title and body atomically enough
// for public read paths: the item only becomes ready after this returns.
func (r *ContentRepo) SetVietnameseContent(ctx context.Context, contentID int64, title, body string) error {
	_, err := r.db.Pool.Exec(ctx, `WITH saved_body AS (
		UPDATE content_bodies SET vietnamese_body=$3, translation_status='ready',
			translated_at=now(), updated_at=now() WHERE content_id=$1 RETURNING content_id)
		UPDATE content_items SET translated_title=$2
		WHERE id=$1 AND EXISTS (SELECT 1 FROM saved_body)`, contentID, title, body)
	return err
}

// TranslationDigested marks a foreign article that has a reader-facing
// Vietnamese headline, summary and key points but deliberately no translated
// body.
//
// It is a distinct state from 'ready' for one reason: a digested article may
// still earn a full translation later, if its cluster wins the day and the
// analysis desk needs it as source material. Marking it 'ready' would tell
// IDsPendingTranslationForCluster the work was already done and quietly starve
// the piece it was gathered for.
const TranslationDigested = "digested"

// SetForeignDigest stores the reader-facing Vietnamese form of a foreign
// article: a headline on content_items, and the digested marker on the body row.
//
// original_body stays — it is the input for a later full translation if this
// story wins the day — but vietnamese_body is cleared, not merely left alone.
// Clearing matters: a stale job from before this pipeline changed can arrive
// after an article was already translated in full, and leaving that copy behind
// would keep a Vietnamese reproduction of someone else's article in the
// database while every comment here claims we don't hold one. The daily pick
// regenerates what it needs.
func (r *ContentRepo) SetForeignDigest(ctx context.Context, contentID int64, title string) error {
	_, err := r.db.Pool.Exec(ctx, `WITH marked AS (
		UPDATE content_bodies SET translation_status=$3, vietnamese_body=NULL,
			translated_at=now(), updated_at=now()
		WHERE content_id=$1 RETURNING content_id)
		UPDATE content_items SET translated_title=$2
		WHERE id=$1 AND EXISTS (SELECT 1 FROM marked)`, contentID, title, TranslationDigested)
	return err
}

// MarkTranslationPending marks an item while a translation job is queued.
func (r *ContentRepo) MarkTranslationPending(ctx context.Context, contentID int64) error {
	_, err := r.db.Pool.Exec(ctx, `UPDATE content_bodies SET translation_status='processing', updated_at=now() WHERE content_id=$1`, contentID)
	return err
}

// QuarantineBlockedArticle removes derived AI text and returns an access-wall
// item to editorial review. The canonical source link and metadata are kept.
func (r *ContentRepo) QuarantineBlockedArticle(ctx context.Context, contentID int64) error {
	return r.db.WithTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `UPDATE content_bodies SET
			original_body='', vietnamese_body=NULL, translation_status='blocked', translated_at=NULL, updated_at=now()
			WHERE content_id=$1`, contentID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `UPDATE content_items SET
			status='needs_review', translated_title=NULL, summary=NULL, key_points='{}', updated_at=now()
			WHERE id=$1`, contentID)
		return err
	})
}

// GetResearch returns the research subtype row for a content id.
func (r *ContentRepo) GetResearch(ctx context.Context, contentID int64) (*domain.ResearchPaper, error) {
	var p domain.ResearchPaper
	err := r.db.Pool.QueryRow(ctx, `
		SELECT content_id, doi, pmid, pmcid, journal, authors, abstract, study_type, is_human,
		       is_open_access, full_text_url, sample_size, population, duration_weeks, sex,
		       training_status, bd_question, bd_participants, bd_intervention, bd_findings,
		       bd_not_proven, bd_limitations, bd_practical, funding_note, published_year
		FROM research_papers WHERE content_id=$1`, contentID).
		Scan(&p.ContentID, &p.DOI, &p.PMID, &p.PMCID, &p.Journal, &p.Authors, &p.Abstract,
			&p.StudyType, &p.IsHuman, &p.IsOpenAccess, &p.FullTextURL, &p.SampleSize,
			&p.Population, &p.DurationWeeks, &p.Sex, &p.TrainingStatus,
			&p.Breakdown.Question, &p.Breakdown.Participants, &p.Breakdown.Intervention,
			&p.Breakdown.Findings, &p.Breakdown.NotProven, &p.Breakdown.Limitations,
			&p.Breakdown.Practical, &p.Breakdown.FundingNote, &p.PublishedYear)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &p, err
}

// GetVideo returns the video subtype row for a content id.
func (r *ContentRepo) GetVideo(ctx context.Context, contentID int64) (*domain.Video, error) {
	var v domain.Video
	err := r.db.Pool.QueryRow(ctx, `
		SELECT content_id, youtube_id, channel_id, channel_title, duration_sec, thumbnail_url,
		       description, has_transcript, timeline, yt_views, yt_likes
		FROM videos WHERE content_id=$1`, contentID).
		Scan(&v.ContentID, &v.YouTubeID, &v.ChannelID, &v.ChannelTitle, &v.DurationSec,
			&v.ThumbnailURL, &v.Description, &v.HasTranscript, &v.Timeline, &v.YTViews, &v.YTLikes)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &v, err
}

// GetArticle returns the article subtype row for a content id.
func (r *ContentRepo) GetArticle(ctx context.Context, contentID int64) (*domain.Article, error) {
	var a domain.Article
	err := r.db.Pool.QueryRow(ctx,
		`SELECT content_id, author, word_count FROM articles WHERE content_id=$1`, contentID).
		Scan(&a.ContentID, &a.Author, &a.WordCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &a, err
}

// ContentFilter captures the /content list query parameters.
type ContentFilter struct {
	Type       string
	TopicSlug  string
	SourceID   *int64
	EntitySlug string
	Language   string
	HasSummary *bool
	OpenAccess *bool
	Sort       string // recent | top
	Limit      int
	Offset     int
	OnlyReady  bool
}

// List returns content items matching the filter, plus the total count.
func (r *ContentRepo) List(ctx context.Context, f ContentFilter) ([]domain.ContentItem, int, error) {
	var where []string
	var args []any
	add := func(cond string, val any) {
		args = append(args, val)
		where = append(where, fmt.Sprintf(cond, len(args)))
	}
	if f.OnlyReady {
		where = append(where, "c.status = 'ready'")
	}
	if f.Type != "" {
		add("c.type = $%d", f.Type)
	}
	if f.SourceID != nil {
		add("c.source_id = $%d", *f.SourceID)
	}
	if f.Language != "" {
		add("c.language = $%d", f.Language)
	}
	if f.HasSummary != nil {
		if *f.HasSummary {
			where = append(where, "c.summary IS NOT NULL")
		} else {
			where = append(where, "c.summary IS NULL")
		}
	}
	if f.TopicSlug != "" {
		add("EXISTS (SELECT 1 FROM content_topics ct JOIN topics t ON t.id=ct.topic_id WHERE ct.content_id=c.id AND t.slug=$%d)", f.TopicSlug)
	}
	if f.EntitySlug != "" {
		add("EXISTS (SELECT 1 FROM content_entities ce JOIN entities e ON e.id=ce.entity_id WHERE ce.content_id=c.id AND e.slug=$%d)", f.EntitySlug)
	}
	if f.OpenAccess != nil {
		add("EXISTS (SELECT 1 FROM research_papers rp WHERE rp.content_id=c.id AND rp.is_open_access=$%d)", *f.OpenAccess)
	}

	whereSQL := ""
	if len(where) > 0 {
		whereSQL = " WHERE " + strings.Join(where, " AND ")
	}

	var total int
	if err := r.db.Pool.QueryRow(ctx, `SELECT count(*) FROM content_items c`+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	order := "c.published_at DESC NULLS LAST"
	if f.Sort == "top" {
		order = "c.final_score DESC, c.published_at DESC NULLS LAST"
	}
	args = append(args, f.Limit, f.Offset)
	q := `SELECT ` + contentCols + `, s.name FROM content_items c JOIN sources s ON s.id=c.source_id` +
		whereSQL + ` ORDER BY ` + order + fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := r.db.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []domain.ContentItem
	for rows.Next() {
		c, err := scanContentWithSource(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *c)
	}
	return out, total, rows.Err()
}

// AdminList lists content of any status (optionally filtered by status and
// type), newest first, plus a total count. Used by the admin content queue.
func (r *ContentRepo) AdminList(ctx context.Context, status, typ string, minScore float64, limit, offset int) ([]domain.ContentItem, int, error) {
	var where []string
	var args []any
	add := func(cond string, val any) {
		args = append(args, val)
		where = append(where, fmt.Sprintf(cond, len(args)))
	}
	if status != "" {
		add("c.status = $%d", status)
	}
	if typ != "" {
		add("c.type = $%d", typ)
	}
	if minScore > 0 {
		add("c.final_score >= $%d", minScore)
	}
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = " WHERE " + strings.Join(where, " AND ")
	}

	var total int
	if err := r.db.Pool.QueryRow(ctx, `SELECT count(*) FROM content_items c`+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	// Order by notability so the most important items to decide on surface first
	// and never scroll off the bottom of the queue.
	q := `SELECT ` + contentCols + `, s.name FROM content_items c JOIN sources s ON s.id=c.source_id` +
		whereSQL + fmt.Sprintf(" ORDER BY c.final_score DESC NULLS LAST, c.editorial_boost DESC, c.discovered_at DESC, c.id DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args))
	rows, err := r.db.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items, err := collectContent(rows)
	return items, total, err
}

// Related returns items sharing a topic with the given item.
func (r *ContentRepo) Related(ctx context.Context, id int64, limit int) ([]domain.ContentItem, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT `+contentCols+`, s.name
		FROM content_items c JOIN sources s ON s.id=c.source_id
		WHERE c.status='ready' AND c.id <> $1
		  AND EXISTS (
		    SELECT 1 FROM content_topics a JOIN content_topics b ON a.topic_id=b.topic_id
		    WHERE a.content_id=$1 AND b.content_id=c.id)
		ORDER BY c.final_score DESC, c.published_at DESC
		LIMIT $2`, id, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectContent(rows)
}

// ---- Feed / homepage building blocks ----

// TopGeneral returns the highest-scoring recent items regardless of follows.
func (r *ContentRepo) TopGeneral(ctx context.Context, limit int) ([]domain.ContentItem, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT `+contentCols+`, s.name FROM content_items c JOIN sources s ON s.id=c.source_id
		WHERE c.status='ready' AND c.published_at > now() - interval '48 hours'
		ORDER BY c.final_score DESC, c.published_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectContent(rows)
}

// LatestByType returns the newest ready items of a given type.
func (r *ContentRepo) LatestByType(ctx context.Context, t domain.ContentType, limit int) ([]domain.ContentItem, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT `+contentCols+`, s.name FROM content_items c JOIN sources s ON s.id=c.source_id
		WHERE c.status='ready' AND c.type=$1
		ORDER BY c.published_at DESC NULLS LAST LIMIT $2`, t, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectContent(rows)
}

// ByTopicSlugs returns top items across the given topics.
func (r *ContentRepo) ByTopicSlugs(ctx context.Context, slugs []string, limit int) ([]domain.ContentItem, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT `+contentCols+`, s.name FROM content_items c JOIN sources s ON s.id=c.source_id
		WHERE c.status='ready' AND EXISTS (
		    SELECT 1 FROM content_topics ct JOIN topics t ON t.id=ct.topic_id
		    WHERE ct.content_id=c.id AND t.slug = ANY($1))
		ORDER BY c.final_score DESC, c.published_at DESC LIMIT $2`, slugs, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectContent(rows)
}

// LongForm returns long articles (high word count) for the "deep reads" block.
func (r *ContentRepo) LongForm(ctx context.Context, limit int) ([]domain.ContentItem, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT `+contentCols+`, s.name FROM content_items c
		JOIN sources s ON s.id=c.source_id
		JOIN articles a ON a.content_id=c.id
		WHERE c.status='ready' AND coalesce(a.word_count,0) > 1500
		ORDER BY c.published_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectContent(rows)
}

// TopPersonal returns items boosted by the user's topic/entity follows.
func (r *ContentRepo) TopPersonal(ctx context.Context, userID int64, limit int) ([]domain.ContentItem, error) {
	rows, err := r.db.Pool.Query(ctx, personalFeedSQL+` LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectContent(rows)
}

// TopDiscovery returns high-scoring items in topics the user does NOT follow.
func (r *ContentRepo) TopDiscovery(ctx context.Context, userID int64, limit int) ([]domain.ContentItem, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT `+contentCols+`, s.name FROM content_items c JOIN sources s ON s.id=c.source_id
		WHERE c.status='ready' AND c.published_at > now() - interval '30 days'
		  AND EXISTS (SELECT 1 FROM content_topics ct WHERE ct.content_id=c.id)
		  AND NOT EXISTS (
		      SELECT 1 FROM content_topics ct
		      JOIN user_topic_follows utf ON utf.topic_id=ct.topic_id
		      WHERE ct.content_id=c.id AND utf.user_id=$1 AND utf.in_feed)
		  AND NOT EXISTS (SELECT 1 FROM hidden_items h WHERE h.user_id=$1 AND h.content_id=c.id)
		ORDER BY c.final_score DESC, c.published_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectContent(rows)
}

// personalFeedSQL implements the personalized scoring from spec section 14.
// $1 = user id. Caller appends LIMIT/OFFSET.
const personalFeedSQL = `
SELECT ` + contentCols + `, s.name
FROM content_items c
JOIN sources s ON s.id=c.source_id
WHERE c.status='ready'
  AND c.published_at > now() - interval '30 days'
  AND NOT EXISTS (SELECT 1 FROM hidden_items h WHERE h.user_id=$1 AND h.content_id=c.id)
  AND NOT EXISTS (SELECT 1 FROM user_source_mutes sm WHERE sm.user_id=$1 AND sm.source_id=c.source_id)
  AND NOT EXISTS (
      SELECT 1 FROM content_topics ct
      JOIN user_topic_mutes m ON m.topic_id=ct.topic_id
      WHERE ct.content_id=c.id AND m.user_id=$1)
ORDER BY
  CASE WHEN
    EXISTS (
        SELECT 1 FROM content_topics ct
        JOIN user_topic_follows utf ON utf.topic_id=ct.topic_id
        WHERE ct.content_id=c.id AND utf.user_id=$1 AND utf.in_feed
          AND (NOT utf.highlights_only OR c.final_score >= 40))
    OR EXISTS (
        SELECT 1 FROM content_entities ce
        JOIN user_entity_follows uef ON uef.entity_id=ce.entity_id
        WHERE ce.content_id=c.id AND uef.user_id=$1 AND uef.in_feed
          AND (NOT uef.highlights_only OR c.final_score >= 40))
    OR EXISTS (
        SELECT 1 FROM user_source_follows usf
        WHERE usf.source_id=c.source_id AND usf.user_id=$1)
    THEN 1 ELSE 0 END DESC,
  (
    c.final_score
    + COALESCE((SELECT MAX(22 + LEAST(GREATEST(utf.priority, 0), 3) * 5)
        FROM content_topics ct
        JOIN user_topic_follows utf ON utf.topic_id=ct.topic_id
        WHERE ct.content_id=c.id AND utf.user_id=$1 AND utf.in_feed
          AND (NOT utf.highlights_only OR c.final_score >= 40)), 0)
    + COALESCE((SELECT MAX(18 + LEAST(GREATEST(uef.priority, 0), 3) * 5)
        FROM content_entities ce
        JOIN user_entity_follows uef ON uef.entity_id=ce.entity_id
        WHERE ce.content_id=c.id AND uef.user_id=$1 AND uef.in_feed
          AND (NOT uef.highlights_only OR c.final_score >= 40)), 0)
    + CASE WHEN EXISTS (
        SELECT 1 FROM user_source_follows usf
        WHERE usf.source_id=c.source_id AND usf.user_id=$1) THEN 8 ELSE 0 END
    - CASE WHEN EXISTS (
        SELECT 1 FROM reading_history rh
        WHERE rh.content_id=c.id AND rh.user_id=$1) THEN 5 ELSE 0 END
  ) DESC, c.published_at DESC`

// PersonalFeed returns the paginated personalized feed for a user.
func (r *ContentRepo) PersonalFeed(ctx context.Context, userID int64, limit, offset int) ([]domain.ContentItem, error) {
	rows, err := r.db.Pool.Query(ctx, personalFeedSQL+` LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectContent(rows)
}

// FollowingFeed is the strict personalized stream: every item must match at
// least one topic the user explicitly keeps in their feed.
func (r *ContentRepo) FollowingFeed(ctx context.Context, userID int64, limit, offset int) ([]domain.ContentItem, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT `+contentCols+`, s.name
		FROM content_items c
		JOIN sources s ON s.id=c.source_id
		WHERE c.status='ready'
		  AND c.published_at > now() - interval '30 days'
		  AND EXISTS (
		      SELECT 1 FROM content_topics ct
		      JOIN user_topic_follows utf ON utf.topic_id=ct.topic_id
		      WHERE ct.content_id=c.id AND utf.user_id=$1 AND utf.in_feed
		        AND (NOT utf.highlights_only OR c.final_score >= 40))
		  AND NOT EXISTS (SELECT 1 FROM hidden_items h WHERE h.user_id=$1 AND h.content_id=c.id)
		  AND NOT EXISTS (SELECT 1 FROM user_source_mutes sm WHERE sm.user_id=$1 AND sm.source_id=c.source_id)
		  AND NOT EXISTS (
		      SELECT 1 FROM content_topics ct
		      JOIN user_topic_mutes m ON m.topic_id=ct.topic_id
		      WHERE ct.content_id=c.id AND m.user_id=$1)
		ORDER BY (
		    c.final_score + COALESCE((
		        SELECT MAX(LEAST(GREATEST(utf.priority, 0), 3) * 4)
		        FROM content_topics ct
		        JOIN user_topic_follows utf ON utf.topic_id=ct.topic_id
		        WHERE ct.content_id=c.id AND utf.user_id=$1 AND utf.in_feed
		    ), 0)
		) DESC, c.published_at DESC
		LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectContent(rows)
}

// ---- Mutations ----

// SetStatus updates the status of an item.
func (r *ContentRepo) SetStatus(ctx context.Context, id int64, status domain.ContentStatus) error {
	_, err := r.db.Pool.Exec(ctx, `UPDATE content_items SET status=$2 WHERE id=$1`, id, status)
	return err
}

// SetBaseScore updates the algorithmic score.
func (r *ContentRepo) SetBaseScore(ctx context.Context, id int64, score float64) error {
	_, err := r.db.Pool.Exec(ctx, `UPDATE content_items SET base_score=$2 WHERE id=$1`, id, score)
	return err
}

// SetSummary stores an AI-generated summary and key points, marking the item ready.
func (r *ContentRepo) SetSummary(ctx context.Context, id int64, summary *string, keyPoints []string, status domain.ContentStatus) error {
	if keyPoints == nil {
		keyPoints = []string{}
	}
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE content_items SET summary=$2, key_points=$3, status=$4 WHERE id=$1`,
		id, summary, keyPoints, status)
	return err
}

// AdminUpdate patches admin-editable fields; nil args are left unchanged.
func (r *ContentRepo) AdminUpdate(ctx context.Context, id int64, title, body, status, summary *string, keyPoints []string, editorialBoost *float64) error {
	var kp any
	if keyPoints != nil {
		kp = keyPoints
	}
	tag, err := r.db.Pool.Exec(ctx, `
		UPDATE content_items SET
			title           = COALESCE($2, title),
			status          = COALESCE($3::content_status, status),
			summary         = COALESCE($4, summary),
			key_points      = COALESCE($5::jsonb, key_points),
			editorial_boost = COALESCE($6, editorial_boost)
		WHERE id=$1`, id, title, status, summary, kp, editorialBoost)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	if body != nil {
		_, err = r.db.Pool.Exec(ctx, `UPDATE content_bodies SET original_body=$2,vietnamese_body=$2,updated_at=now() WHERE content_id=$1`, id, body)
	}
	return nil
}

// SetEditorialBoost sets the editorial boost (used by /highlight).
func (r *ContentRepo) SetEditorialBoost(ctx context.Context, id int64, boost float64) error {
	_, err := r.db.Pool.Exec(ctx, `UPDATE content_items SET editorial_boost=$2 WHERE id=$1`, id, boost)
	return err
}

// UpdateResearchBreakdown patches the 8-section research breakdown.
func (r *ContentRepo) UpdateResearchBreakdown(ctx context.Context, contentID int64, b domain.ResearchBreakdown) error {
	findings := b.Findings
	if findings == nil {
		findings = []string{}
	}
	limitations := b.Limitations
	if limitations == nil {
		limitations = []string{}
	}
	tag, err := r.db.Pool.Exec(ctx, `
		UPDATE research_papers SET
			bd_question=$2, bd_participants=$3, bd_intervention=$4, bd_findings=$5,
			bd_not_proven=$6, bd_limitations=$7, bd_practical=$8, funding_note=$9
		WHERE content_id=$1`,
		contentID, b.Question, b.Participants, b.Intervention, findings, b.NotProven,
		limitations, b.Practical, b.FundingNote)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// SetVideoTranscript stores a video transcript and flips has_transcript.
func (r *ContentRepo) SetVideoTranscript(ctx context.Context, contentID int64, transcript string) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE videos SET transcript=$2, has_transcript=TRUE WHERE content_id=$1`, contentID, transcript)
	return err
}

// IncrementView bumps the view counter.
func (r *ContentRepo) IncrementView(ctx context.Context, id int64) error {
	_, err := r.db.Pool.Exec(ctx, `UPDATE content_items SET view_count=view_count+1 WHERE id=$1`, id)
	return err
}

// ReadyForScoring returns ids of items that need (re)scoring.
func (r *ContentRepo) IDsToRescore(ctx context.Context, limit int) ([]int64, error) {
	rows, err := r.db.Pool.Query(ctx,
		`SELECT id FROM content_items WHERE status IN ('processing','ready') ORDER BY updated_at ASC LIMIT $1`, limit)
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

// IDsPendingTranslation returns foreign articles worth spending a translation
// call on, best first.
//
// Two floors, and both matter:
//
// minScore parks the low scorers so routine wire copy cannot drain the hourly
// LLM allowance out from under the editorial desk.
//
// maxAge is the one that keeps the pipeline honest. A queue is the wrong shape
// for news, because news expires: without a cutoff, a backlog that outgrows the
// hourly allowance means the worker spends today translating last week, forever
// behind and publishing nothing anyone wants. So anything older than maxAge is
// abandoned rather than queued — not dropped by accident when the backlog wins,
// but on purpose, while it is still our decision to make. If we did not get to a
// story while it was news, it is not news any more.
func (r *ContentRepo) IDsPendingTranslation(ctx context.Context, limit int, minScore float64, maxAge time.Duration) ([]int64, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT c.id FROM content_items c
		JOIN content_bodies b ON b.content_id=c.id
		WHERE c.language <> 'vi' AND c.status='processing'
		  AND b.translation_status NOT IN ('ready', 'digested')
		  AND length(trim(b.original_body)) > 0
		  AND c.base_score >= $2
		  AND c.published_at >= now() - $3::interval
		ORDER BY c.final_score DESC, c.published_at DESC NULLS LAST
		LIMIT $1`, limit, minScore, maxAge.String())
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

// IDsPendingTranslationForCluster returns the untranslated foreign articles in
// one cluster, best source first, ignoring the score floor.
//
// This is the deliberate exception to parking: once a cluster has won the day,
// its materials are worth translating regardless of what each individual
// headline scored, because the value is in the story, not the article.
//
// Readiness is judged on translation_status alone, never on content status. A
// digested foreign article is deliberately 'ready' — the digest is what readers
// get — while still holding no Vietnamese body for the desk to quote. Filtering
// on c.status IN ('processing','needs_review') therefore skipped every digested
// piece, so a cluster of foreign coverage was translated never, gathered zero
// materials, and failed with "need 3 publishable sources, have 0". That is the
// exact starvation the 'digested' status was introduced to prevent; it simply
// arrived through the other column.
func (r *ContentRepo) IDsPendingTranslationForCluster(ctx context.Context, clusterID int64, limit int) ([]int64, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT c.id FROM content_items c
		JOIN story_cluster_items sci ON sci.content_id=c.id
		JOIN content_bodies b ON b.content_id=c.id
		JOIN sources s ON s.id=c.source_id
		WHERE sci.cluster_id=$1
		  AND c.language <> 'vi'
		  AND c.status NOT IN ('failed','hidden')
		  AND b.translation_status <> 'ready'
		  AND length(trim(b.original_body)) > 0
		ORDER BY s.quality DESC, c.final_score DESC
		LIMIT $2`, clusterID, limit)
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

// collectContent scans a full result set of content-with-source rows.
func collectContent(rows pgx.Rows) ([]domain.ContentItem, error) {
	var out []domain.ContentItem
	for rows.Next() {
		c, err := scanContentWithSource(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}
