package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"repwire/internal/domain"
)

// EntityRepo persists entities and content-entity links.
type EntityRepo struct{ db *DB }

const entityCols = `id, slug, name, kind, bio, avatar_url, expertise, official_links, aliases, follower_count`

func scanEntity(row pgx.Row) (*domain.Entity, error) {
	var e domain.Entity
	if err := row.Scan(&e.ID, &e.Slug, &e.Name, &e.Kind, &e.Bio, &e.AvatarURL, &e.Expertise,
		&e.OfficialLinks, &e.Aliases, &e.FollowerCount); err != nil {
		return nil, err
	}
	return &e, nil
}

// List returns all entities.
func (r *EntityRepo) List(ctx context.Context) ([]domain.Entity, error) {
	rows, err := r.db.Pool.Query(ctx, `SELECT `+entityCols+` FROM entities ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectEntities(rows)
}

// BySlug returns an entity by slug.
func (r *EntityRepo) BySlug(ctx context.Context, slug string) (*domain.Entity, error) {
	e, err := scanEntity(r.db.Pool.QueryRow(ctx, `SELECT `+entityCols+` FROM entities WHERE slug=$1`, slug))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return e, err
}

// WithAliases returns entities that have at least one alias (for extraction).
func (r *EntityRepo) WithAliases(ctx context.Context) ([]domain.Entity, error) {
	rows, err := r.db.Pool.Query(ctx,
		`SELECT `+entityCols+` FROM entities WHERE array_length(aliases,1) > 0`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectEntities(rows)
}

// ForContent returns entities linked to a content item.
func (r *EntityRepo) ForContent(ctx context.Context, contentID int64) ([]domain.Entity, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT `+prefixCols(entityCols, "e")+` FROM entities e
		JOIN content_entities ce ON ce.entity_id=e.id
		WHERE ce.content_id=$1 ORDER BY e.name`, contentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectEntities(rows)
}

// Create inserts an entity.
func (r *EntityRepo) Create(ctx context.Context, e *domain.Entity) (*domain.Entity, error) {
	if e.Aliases == nil {
		e.Aliases = []string{}
	}
	if e.Expertise == nil {
		e.Expertise = []string{}
	}
	if e.OfficialLinks == nil {
		e.OfficialLinks = []domain.OfficialLink{}
	}
	out, err := scanEntity(r.db.Pool.QueryRow(ctx, `
		INSERT INTO entities (slug, name, kind, bio, avatar_url, expertise, official_links, aliases)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING `+entityCols,
		e.Slug, e.Name, e.Kind, e.Bio, e.AvatarURL, e.Expertise, e.OfficialLinks, e.Aliases))
	if err != nil {
		var pgErr interface{ SQLState() string }
		if errors.As(err, &pgErr) && pgErr.SQLState() == "23505" {
			return nil, domain.ErrConflict
		}
		return nil, err
	}
	return out, nil
}

// UpdateAliases replaces an entity's alias list.
func (r *EntityRepo) UpdateAliases(ctx context.Context, id int64, aliases []string) error {
	if aliases == nil {
		aliases = []string{}
	}
	tag, err := r.db.Pool.Exec(ctx, `UPDATE entities SET aliases=$2 WHERE id=$1`, id, aliases)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// LinkEntity attaches an entity to a content item in a role.
func (r *EntityRepo) LinkEntity(ctx context.Context, tx pgx.Tx, contentID, entityID int64, role string) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO content_entities (content_id, entity_id, role) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`,
		contentID, entityID, role)
	return err
}

func collectEntities(rows pgx.Rows) ([]domain.Entity, error) {
	var out []domain.Entity
	for rows.Next() {
		e, err := scanEntity(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *e)
	}
	return out, rows.Err()
}
