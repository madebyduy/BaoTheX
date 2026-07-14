package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"repwire/internal/domain"
)

// TopicRepo persists topics and content-topic assignments.
type TopicRepo struct{ db *DB }

const topicCols = `id, slug, name, description, category, keywords, follower_count`

func scanTopic(row pgx.Row) (*domain.Topic, error) {
	var t domain.Topic
	if err := row.Scan(&t.ID, &t.Slug, &t.Name, &t.Description, &t.Category, &t.Keywords, &t.FollowerCount); err != nil {
		return nil, err
	}
	return &t, nil
}

// List returns all topics.
func (r *TopicRepo) List(ctx context.Context) ([]domain.Topic, error) {
	rows, err := r.db.Pool.Query(ctx, `SELECT `+topicCols+` FROM topics ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectTopics(rows)
}

// BySlug returns a topic by slug.
func (r *TopicRepo) BySlug(ctx context.Context, slug string) (*domain.Topic, error) {
	t, err := scanTopic(r.db.Pool.QueryRow(ctx, `SELECT `+topicCols+` FROM topics WHERE slug=$1`, slug))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return t, err
}

// ForContent returns the topics assigned to a content item.
func (r *TopicRepo) ForContent(ctx context.Context, contentID int64) ([]domain.Topic, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT `+prefixCols(topicCols, "t")+` FROM topics t
		JOIN content_topics ct ON ct.topic_id=t.id
		WHERE ct.content_id=$1 ORDER BY ct.is_primary DESC, ct.confidence DESC`, contentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectTopics(rows)
}

// Related returns topics related to the given one.
func (r *TopicRepo) Related(ctx context.Context, topicID int64) ([]domain.Topic, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT `+prefixCols(topicCols, "t")+` FROM topics t
		JOIN topic_relations tr ON tr.related_id=t.id
		WHERE tr.topic_id=$1 ORDER BY t.name`, topicID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectTopics(rows)
}

// Create inserts a topic.
func (r *TopicRepo) Create(ctx context.Context, slug, name string, keywords []string) (*domain.Topic, error) {
	if keywords == nil {
		keywords = []string{}
	}
	t, err := scanTopic(r.db.Pool.QueryRow(ctx,
		`INSERT INTO topics (slug, name, keywords) VALUES ($1,$2,$3) RETURNING `+topicCols,
		slug, name, keywords))
	if err != nil {
		var pgErr interface{ SQLState() string }
		if errors.As(err, &pgErr) && pgErr.SQLState() == "23505" {
			return nil, domain.ErrConflict
		}
		return nil, err
	}
	return t, nil
}

// UpdateKeywords replaces a topic's keyword list.
func (r *TopicRepo) UpdateKeywords(ctx context.Context, id int64, keywords []string) error {
	if keywords == nil {
		keywords = []string{}
	}
	tag, err := r.db.Pool.Exec(ctx, `UPDATE topics SET keywords=$2 WHERE id=$1`, id, keywords)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// AssignTopics replaces the topic assignments for a content item.
func (r *TopicRepo) AssignTopics(ctx context.Context, contentID int64, assignments []domain.ContentTopic) error {
	return r.db.WithTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM content_topics WHERE content_id=$1`, contentID); err != nil {
			return err
		}
		for _, a := range assignments {
			if _, err := tx.Exec(ctx,
				`INSERT INTO content_topics (content_id, topic_id, confidence, is_primary)
				 VALUES ($1,$2,$3,$4) ON CONFLICT DO NOTHING`,
				contentID, a.TopicID, a.Confidence, a.IsPrimary); err != nil {
				return err
			}
		}
		return nil
	})
}

// SetTopicsByID assigns a bare list of topic ids (admin manual override).
func (r *TopicRepo) SetTopicsByID(ctx context.Context, contentID int64, topicIDs []int64) error {
	assignments := make([]domain.ContentTopic, len(topicIDs))
	for i, id := range topicIDs {
		assignments[i] = domain.ContentTopic{ContentID: contentID, TopicID: id, Confidence: 1.0, IsPrimary: i == 0}
	}
	return r.AssignTopics(ctx, contentID, assignments)
}

func collectTopics(rows pgx.Rows) ([]domain.Topic, error) {
	var out []domain.Topic
	for rows.Next() {
		t, err := scanTopic(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}
