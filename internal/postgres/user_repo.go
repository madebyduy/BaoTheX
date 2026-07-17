package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"repwire/internal/domain"
)

// UserRepo persists users, sessions, saved items, collections and history.
type UserRepo struct{ db *DB }

const userCols = `id, email, password_hash, display_name, role, goals, timezone, onboarded_at, created_at`

func scanUser(row pgx.Row) (*domain.User, error) {
	var u domain.User
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.Role, &u.Goals,
		&u.Timezone, &u.OnboardedAt, &u.CreatedAt); err != nil {
		return nil, err
	}
	return &u, nil
}

// Create inserts a user and default notification preferences in one tx.
func (r *UserRepo) Create(ctx context.Context, email, passwordHash string, displayName *string) (*domain.User, error) {
	var u *domain.User
	err := r.db.WithTx(ctx, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx,
			`INSERT INTO users (email, password_hash, display_name) VALUES ($1,$2,$3) RETURNING `+userCols,
			email, passwordHash, displayName)
		var err error
		u, err = scanUser(row)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO notification_preferences (user_id) VALUES ($1)`, u.ID)
		return err
	})
	if err != nil {
		// unique_violation
		var pgErr interface{ SQLState() string }
		if errors.As(err, &pgErr) && pgErr.SQLState() == "23505" {
			return nil, domain.ErrConflict
		}
		return nil, err
	}
	return u, nil
}

// ByEmail looks a user up by email.
func (r *UserRepo) ByEmail(ctx context.Context, email string) (*domain.User, error) {
	u, err := scanUser(r.db.Pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE email=$1`, email))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return u, err
}

// ByID looks a user up by id.
func (r *UserRepo) ByID(ctx context.Context, id int64) (*domain.User, error) {
	u, err := scanUser(r.db.Pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE id=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return u, err
}

// SetOnboarding stores goals and marks the user onboarded.
func (r *UserRepo) SetOnboarding(ctx context.Context, id int64, goals []string) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE users SET goals=$2, onboarded_at=COALESCE(onboarded_at, now()) WHERE id=$1`, id, goals)
	return err
}

// ---- Sessions ----

// CreateSession stores a session hash.
func (r *UserRepo) CreateSession(ctx context.Context, tokenHash string, userID int64, expiresAt time.Time) error {
	_, err := r.db.Pool.Exec(ctx,
		`INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ($1,$2,$3)`,
		tokenHash, userID, expiresAt)
	return err
}

// UserBySession returns the user for a valid, unexpired session token hash and
// slides the expiry forward.
func (r *UserRepo) UserBySession(ctx context.Context, tokenHash string, slideTo time.Time) (*domain.User, error) {
	var uid int64
	err := r.db.Pool.QueryRow(ctx,
		`UPDATE sessions SET expires_at=$2 WHERE token_hash=$1 AND expires_at > now() RETURNING user_id`,
		tokenHash, slideTo).Scan(&uid)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUnauthorized
	}
	if err != nil {
		return nil, err
	}
	return r.ByID(ctx, uid)
}

// DeleteSession removes a session (logout).
func (r *UserRepo) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash=$1`, tokenHash)
	return err
}

// ---- Saved items / collections ----

// Save upserts a saved item.
func (r *UserRepo) Save(ctx context.Context, userID, contentID int64, collectionID *int64, note *string) error {
	return r.db.WithTx(ctx, func(tx pgx.Tx) error {
		var inserted bool
		err := tx.QueryRow(ctx, `
			INSERT INTO saved_items (user_id, content_id, collection_id, note)
			VALUES ($1,$2,$3,$4)
			ON CONFLICT (user_id, content_id) DO UPDATE SET collection_id=EXCLUDED.collection_id, note=EXCLUDED.note
			RETURNING (xmax=0)`, userID, contentID, collectionID, note).Scan(&inserted)
		if err != nil {
			return err
		}
		if inserted {
			_, err = tx.Exec(ctx, `UPDATE content_items SET save_count=save_count+1 WHERE id=$1`, contentID)
		}
		return err
	})
}

// Unsave removes a saved item.
func (r *UserRepo) Unsave(ctx context.Context, userID, contentID int64) error {
	return r.db.WithTx(ctx, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `DELETE FROM saved_items WHERE user_id=$1 AND content_id=$2`, userID, contentID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() > 0 {
			_, err = tx.Exec(ctx, `UPDATE content_items SET save_count=GREATEST(save_count-1,0) WHERE id=$1`, contentID)
		}
		return err
	})
}

// IsSaved returns the authoritative bookmark state for interactive readers.
func (r *UserRepo) IsSaved(ctx context.Context, userID, contentID int64) (bool, error) {
	var saved bool
	err := r.db.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM saved_items WHERE user_id=$1 AND content_id=$2)`, userID, contentID).Scan(&saved)
	return saved, err
}

// ListSaved returns a user's saved content items (optionally by collection).
func (r *UserRepo) ListSaved(ctx context.Context, userID int64, collectionID *int64, limit, offset int) ([]domain.ContentItem, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT `+contentCols+`, s.name
		FROM saved_items si
		JOIN content_items c ON c.id=si.content_id
		JOIN sources s ON s.id=c.source_id
		WHERE si.user_id=$1 AND ($2::bigint IS NULL OR si.collection_id=$2)
		ORDER BY si.saved_at DESC LIMIT $3 OFFSET $4`, userID, collectionID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectContent(rows)
}

// CreateCollection creates a named collection.
func (r *UserRepo) CreateCollection(ctx context.Context, userID int64, name string) (*domain.Collection, error) {
	var c domain.Collection
	err := r.db.Pool.QueryRow(ctx,
		`INSERT INTO collections (user_id, name) VALUES ($1,$2) RETURNING id, user_id, name, created_at`,
		userID, name).Scan(&c.ID, &c.UserID, &c.Name, &c.CreatedAt)
	if err != nil {
		var pgErr interface{ SQLState() string }
		if errors.As(err, &pgErr) && pgErr.SQLState() == "23505" {
			return nil, domain.ErrConflict
		}
		return nil, err
	}
	return &c, nil
}

// ListCollections returns a user's collections.
func (r *UserRepo) ListCollections(ctx context.Context, userID int64) ([]domain.Collection, error) {
	rows, err := r.db.Pool.Query(ctx,
		`SELECT id, user_id, name, created_at FROM collections WHERE user_id=$1 ORDER BY created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Collection
	for rows.Next() {
		var c domain.Collection
		if err := rows.Scan(&c.ID, &c.UserID, &c.Name, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// DeleteCollection removes a collection owned by the user.
func (r *UserRepo) DeleteCollection(ctx context.Context, userID, id int64) error {
	tag, err := r.db.Pool.Exec(ctx, `DELETE FROM collections WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ---- History / hidden ----

// MarkRead records a read event.
func (r *UserRepo) MarkRead(ctx context.Context, userID, contentID int64) error {
	_, err := r.db.Pool.Exec(ctx,
		`INSERT INTO reading_history (user_id, content_id) VALUES ($1,$2)
		 ON CONFLICT (user_id, content_id) DO UPDATE SET read_at=now()`, userID, contentID)
	return err
}

// Hide hides an item from a user's feed.
func (r *UserRepo) Hide(ctx context.Context, userID, contentID int64) error {
	_, err := r.db.Pool.Exec(ctx,
		`INSERT INTO hidden_items (user_id, content_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`,
		userID, contentID)
	return err
}
