package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"repwire/internal/domain"
)

// SourceRepo persists content sources.
type SourceRepo struct{ db *DB }

const sourceCols = `id, kind, name, homepage_url, feed_url, quality, default_lang, enabled,
	extract(epoch from fetch_interval)::bigint, etag, last_modified, uploads_playlist_id,
	last_fetched_at, last_error, consecutive_failures, created_at`

// sourceColsAliased returns the source columns qualified with a table alias,
// used when sources is joined with other tables. Scan order matches scanSource.
func sourceColsAliased(a string) string {
	return a + `.id, ` + a + `.kind, ` + a + `.name, ` + a + `.homepage_url, ` + a + `.feed_url, ` +
		a + `.quality, ` + a + `.default_lang, ` + a + `.enabled, ` +
		`extract(epoch from ` + a + `.fetch_interval)::bigint, ` + a + `.etag, ` + a + `.last_modified, ` +
		a + `.uploads_playlist_id, ` + a + `.last_fetched_at, ` + a + `.last_error, ` +
		a + `.consecutive_failures, ` + a + `.created_at`
}

func scanSource(row pgx.Row) (*domain.Source, error) {
	var s domain.Source
	var intervalSec int64
	if err := row.Scan(&s.ID, &s.Kind, &s.Name, &s.HomepageURL, &s.FeedURL, &s.Quality,
		&s.DefaultLang, &s.Enabled, &intervalSec, &s.ETag, &s.LastModified,
		&s.UploadsPlaylistID, &s.LastFetchedAt, &s.LastError, &s.ConsecutiveFailures,
		&s.CreatedAt); err != nil {
		return nil, err
	}
	s.FetchInterval = time.Duration(intervalSec) * time.Second
	return &s, nil
}

// Get returns a source by id.
func (r *SourceRepo) Get(ctx context.Context, id int64) (*domain.Source, error) {
	s, err := scanSource(r.db.Pool.QueryRow(ctx, `SELECT `+sourceCols+` FROM sources WHERE id=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return s, err
}

// List returns all sources ordered by name.
func (r *SourceRepo) List(ctx context.Context) ([]domain.Source, error) {
	rows, err := r.db.Pool.Query(ctx, `SELECT `+sourceCols+` FROM sources ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Source
	for rows.Next() {
		s, err := scanSource(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

// DueForFetch returns enabled sources whose fetch interval has elapsed.
func (r *SourceRepo) DueForFetch(ctx context.Context) ([]domain.Source, error) {
	rows, err := r.db.Pool.Query(ctx, `SELECT `+sourceCols+` FROM sources
		WHERE enabled = TRUE
		  AND (last_fetched_at IS NULL OR last_fetched_at + fetch_interval < now())
		ORDER BY last_fetched_at NULLS FIRST`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Source
	for rows.Next() {
		s, err := scanSource(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

// Create inserts a new source and returns its id.
func (r *SourceRepo) Create(ctx context.Context, s *domain.Source) (int64, error) {
	var id int64
	err := r.db.Pool.QueryRow(ctx, `
		INSERT INTO sources (kind, name, homepage_url, feed_url, quality, default_lang, enabled, fetch_interval)
		VALUES ($1,$2,$3,$4,$5,$6,$7, make_interval(secs => $8))
		RETURNING id`,
		s.Kind, s.Name, s.HomepageURL, s.FeedURL, s.Quality, s.DefaultLang, s.Enabled,
		s.FetchInterval.Seconds()).Scan(&id)
	return id, err
}

// Update patches editable fields on a source.
func (r *SourceRepo) Update(ctx context.Context, id int64, enabled *bool, quality *int, intervalSec *int64) error {
	tag, err := r.db.Pool.Exec(ctx, `
		UPDATE sources SET
			enabled        = COALESCE($2, enabled),
			quality        = COALESCE($3, quality),
			fetch_interval = COALESCE(make_interval(secs => $4), fetch_interval)
		WHERE id = $1`,
		id, enabled, quality, intervalSec)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// MarkFetched records a successful fetch, updating conditional-GET headers.
func (r *SourceRepo) MarkFetched(ctx context.Context, id int64, etag, lastModified *string) error {
	_, err := r.db.Pool.Exec(ctx, `
		UPDATE sources SET last_fetched_at=now(), last_error=NULL, consecutive_failures=0,
			etag=COALESCE($2, etag), last_modified=COALESCE($3, last_modified)
		WHERE id=$1`, id, etag, lastModified)
	return err
}

// MarkError records a failed fetch. After 3 consecutive failures the source is
// auto-disabled (spec section 21).
func (r *SourceRepo) MarkError(ctx context.Context, id int64, cause string) error {
	_, err := r.db.Pool.Exec(ctx, `
		UPDATE sources SET
			last_fetched_at = now(),
			last_error = $2,
			consecutive_failures = consecutive_failures + 1,
			enabled = CASE WHEN consecutive_failures + 1 >= 3 THEN FALSE ELSE enabled END
		WHERE id=$1`, id, cause)
	return err
}

// SetUploadsPlaylist caches a resolved YouTube uploads playlist id.
func (r *SourceRepo) SetUploadsPlaylist(ctx context.Context, id int64, playlistID string) error {
	_, err := r.db.Pool.Exec(ctx, `UPDATE sources SET uploads_playlist_id=$2 WHERE id=$1`, id, playlistID)
	return err
}
