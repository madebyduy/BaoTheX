package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"repwire/internal/domain"
)

// JobRepo persists and dequeues background jobs.
type JobRepo struct{ db *DB }

// ErrNoJob is returned by Dequeue when the queue is empty.
var ErrNoJob = errors.New("no job available")

// ErrJobLeaseLost means another worker/reaper already changed the claimed job.
// The stale worker must not overwrite the newer owner's state.
var ErrJobLeaseLost = errors.New("job lease lost")

// EnqueueOpts tunes an Enqueue call.
type EnqueueOpts struct {
	DedupKey    string
	Priority    int
	RunAt       time.Time
	MaxAttempts int
}

// Enqueue inserts a job. If DedupKey collides with a pending/running job the
// insert is skipped (ON CONFLICT DO NOTHING) and no error is returned.
func (r *JobRepo) Enqueue(ctx context.Context, kind string, payload any, opts EnqueueOpts) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if payload == nil {
		raw = []byte("{}")
	}
	var dedup *string
	if opts.DedupKey != "" {
		dedup = &opts.DedupKey
	}
	runAt := opts.RunAt
	if runAt.IsZero() {
		runAt = time.Now()
	}
	maxAttempts := opts.MaxAttempts
	if maxAttempts == 0 {
		maxAttempts = 5
	}
	_, err = r.db.Pool.Exec(ctx, `
		INSERT INTO jobs (kind, payload, dedup_key, priority, run_at, max_attempts)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (dedup_key) WHERE dedup_key IS NOT NULL AND status IN ('pending','running')
		DO NOTHING`,
		kind, raw, dedup, opts.Priority, runAt, maxAttempts)
	return err
}

// WakePendingByDedup makes an explicitly retried queued job runnable now.
// It never touches a running job, so an editor cannot start two copies of the
// same analysis concurrently.
func (r *JobRepo) WakePendingByDedup(ctx context.Context, dedupKey string) (bool, error) {
	tag, err := r.db.Pool.Exec(ctx, `
		UPDATE jobs SET run_at=now(), attempts=0, last_error=NULL,
		       locked_by=NULL, locked_at=NULL, finished_at=NULL
		WHERE dedup_key=$1 AND status='pending'`, dedupKey)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

const dequeueSQL = `
UPDATE jobs SET
    status    = 'running',
    locked_by = $1,
    locked_at = now(),
    attempts  = attempts + 1
WHERE id = (
    SELECT id FROM jobs
    WHERE status = 'pending' AND run_at <= now()
    ORDER BY priority DESC, run_at
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
RETURNING id, kind, payload, attempts, max_attempts;`

// Dequeue atomically claims the next runnable job for workerID.
func (r *JobRepo) Dequeue(ctx context.Context, workerID string) (*domain.Job, error) {
	var j domain.Job
	err := r.db.Pool.QueryRow(ctx, dequeueSQL, workerID).
		Scan(&j.ID, &j.Kind, &j.Payload, &j.Attempts, &j.MaxAttempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoJob
	}
	if err != nil {
		return nil, err
	}
	return &j, nil
}

// Done marks a job as successfully completed.
func (r *JobRepo) Done(ctx context.Context, id int64, workerID string) error {
	tag, err := r.db.Pool.Exec(ctx, `
		UPDATE jobs SET status='done', finished_at=now(), last_error=NULL
		WHERE id=$1 AND status='running' AND locked_by=$2`, id, workerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrJobLeaseLost
	}
	return nil
}

// Fail records an error and either reschedules the job with backoff or, once
// attempts >= max_attempts, marks it dead.
func (r *JobRepo) Fail(ctx context.Context, j *domain.Job, workerID string, cause error, backoff time.Duration) error {
	msg := cause.Error()
	var tag pgconn.CommandTag
	var err error
	if j.Attempts >= j.MaxAttempts {
		tag, err = r.db.Pool.Exec(ctx, `
			UPDATE jobs SET status='dead', finished_at=now(), last_error=$2
			WHERE id=$1 AND status='running' AND locked_by=$3`, j.ID, msg, workerID)
	} else {
		tag, err = r.db.Pool.Exec(ctx, `
			UPDATE jobs SET status='pending', run_at=now()+$2::interval, last_error=$3,
			       locked_by=NULL, locked_at=NULL
			WHERE id=$1 AND status='running' AND locked_by=$4`,
			j.ID, backoff.String(), msg, workerID)
	}
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrJobLeaseLost
	}
	return nil
}

// ReapStuck resets jobs stuck in 'running' beyond timeout back to 'pending'
// (their worker likely crashed). Returns the number reset.
func (r *JobRepo) ReapStuck(ctx context.Context, timeout time.Duration) (int64, error) {
	tag, err := r.db.Pool.Exec(ctx, `
		UPDATE jobs SET status='pending', locked_by=NULL, locked_at=NULL
		WHERE status='running' AND locked_at < now() - $1::interval`,
		timeout.String())
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// ---- Admin views ----

// List returns jobs filtered by status (empty = all), newest first.
func (r *JobRepo) List(ctx context.Context, status string, limit, offset int) ([]domain.Job, error) {
	q := `SELECT id, kind, payload, dedup_key, status, priority, run_at, attempts,
	             max_attempts, locked_by, locked_at, last_error, created_at, finished_at
	      FROM jobs`
	args := []any{}
	if status != "" {
		q += ` WHERE status = $1`
		args = append(args, status)
	}
	q += ` ORDER BY created_at DESC LIMIT ` + placeholder(len(args)+1) + ` OFFSET ` + placeholder(len(args)+2)
	args = append(args, limit, offset)

	rows, err := r.db.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Job
	for rows.Next() {
		var j domain.Job
		if err := rows.Scan(&j.ID, &j.Kind, &j.Payload, &j.DedupKey, &j.Status, &j.Priority,
			&j.RunAt, &j.Attempts, &j.MaxAttempts, &j.LockedBy, &j.LockedAt, &j.LastError,
			&j.CreatedAt, &j.FinishedAt); err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

// Retry resets a failed/dead job so it runs again immediately.
func (r *JobRepo) Retry(ctx context.Context, id int64) error {
	tag, err := r.db.Pool.Exec(ctx,
		`UPDATE jobs SET status='pending', run_at=now(), attempts=0, last_error=NULL,
		        locked_by=NULL, locked_at=NULL, finished_at=NULL
		 WHERE id=$1 AND status IN ('failed','dead')`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// StatCount is one row of the jobs-by-kind-and-status stats.
type StatCount struct {
	Kind   string `json:"kind"`
	Status string `json:"status"`
	Count  int    `json:"count"`
}

// Stats returns pending/running/failed/dead counts grouped by kind.
func (r *JobRepo) Stats(ctx context.Context) ([]StatCount, error) {
	rows, err := r.db.Pool.Query(ctx,
		`SELECT kind, status::text, count(*) FROM jobs GROUP BY kind, status ORDER BY kind, status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StatCount
	for rows.Next() {
		var s StatCount
		if err := rows.Scan(&s.Kind, &s.Status, &s.Count); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// CountByStatus returns the number of jobs in the given status.
func (r *JobRepo) CountByStatus(ctx context.Context, status domain.JobStatus) (int, error) {
	var n int
	err := r.db.Pool.QueryRow(ctx, `SELECT count(*) FROM jobs WHERE status=$1`, status).Scan(&n)
	return n, err
}
