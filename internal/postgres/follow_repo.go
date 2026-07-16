package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"

	"repwire/internal/domain"
)

// FollowRepo persists topic/entity/source follows and topic mutes.
type FollowRepo struct{ db *DB }

// HasFeedTopics reports whether strict feed mode has at least one usable topic.
func (r *FollowRepo) HasFeedTopics(ctx context.Context, userID int64) (bool, error) {
	var ok bool
	err := r.db.Pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM user_topic_follows
			WHERE user_id=$1 AND in_feed
		)`, userID).Scan(&ok)
	return ok, err
}

// FollowTopic follows a topic (idempotent) and bumps the follower count.
func (r *FollowRepo) FollowTopic(ctx context.Context, userID, topicID int64) error {
	return r.db.WithTx(ctx, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`INSERT INTO user_topic_follows (user_id, topic_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`,
			userID, topicID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() > 0 {
			_, err = tx.Exec(ctx, `UPDATE topics SET follower_count=follower_count+1 WHERE id=$1`, topicID)
		}
		return err
	})
}

// UnfollowTopic removes a topic follow and decrements the count.
func (r *FollowRepo) UnfollowTopic(ctx context.Context, userID, topicID int64) error {
	return r.db.WithTx(ctx, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `DELETE FROM user_topic_follows WHERE user_id=$1 AND topic_id=$2`, userID, topicID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() > 0 {
			_, err = tx.Exec(ctx, `UPDATE topics SET follower_count=GREATEST(follower_count-1,0) WHERE id=$1`, topicID)
		}
		return err
	})
}

// UpdateTopicFollow patches a topic follow's settings.
func (r *FollowRepo) UpdateTopicFollow(ctx context.Context, userID, topicID int64, s domain.FollowSettings) error {
	tag, err := r.db.Pool.Exec(ctx, `
		UPDATE user_topic_follows SET in_feed=$3, in_telegram=$4, highlights_only=$5, priority=$6
		WHERE user_id=$1 AND topic_id=$2`,
		userID, topicID, s.InFeed, s.InTelegram, s.HighlightsOnly, s.Priority)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// FollowEntity follows an entity (idempotent) and bumps the follower count.
func (r *FollowRepo) FollowEntity(ctx context.Context, userID, entityID int64) error {
	return r.db.WithTx(ctx, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`INSERT INTO user_entity_follows (user_id, entity_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`,
			userID, entityID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() > 0 {
			_, err = tx.Exec(ctx, `UPDATE entities SET follower_count=follower_count+1 WHERE id=$1`, entityID)
		}
		return err
	})
}

// UnfollowEntity removes an entity follow and decrements the count.
func (r *FollowRepo) UnfollowEntity(ctx context.Context, userID, entityID int64) error {
	return r.db.WithTx(ctx, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `DELETE FROM user_entity_follows WHERE user_id=$1 AND entity_id=$2`, userID, entityID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() > 0 {
			_, err = tx.Exec(ctx, `UPDATE entities SET follower_count=GREATEST(follower_count-1,0) WHERE id=$1`, entityID)
		}
		return err
	})
}

// UpdateEntityFollow patches an entity follow's settings.
func (r *FollowRepo) UpdateEntityFollow(ctx context.Context, userID, entityID int64, s domain.FollowSettings) error {
	tag, err := r.db.Pool.Exec(ctx, `
		UPDATE user_entity_follows SET in_feed=$3, in_telegram=$4, highlights_only=$5, priority=$6
		WHERE user_id=$1 AND entity_id=$2`,
		userID, entityID, s.InFeed, s.InTelegram, s.HighlightsOnly, s.Priority)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// FollowSource / UnfollowSource manage source follows.
func (r *FollowRepo) FollowSource(ctx context.Context, userID, sourceID int64) error {
	_, err := r.db.Pool.Exec(ctx,
		`INSERT INTO user_source_follows (user_id, source_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`,
		userID, sourceID)
	return err
}

func (r *FollowRepo) UnfollowSource(ctx context.Context, userID, sourceID int64) error {
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM user_source_follows WHERE user_id=$1 AND source_id=$2`, userID, sourceID)
	return err
}

// MuteTopic / UnmuteTopic manage topic mutes.
func (r *FollowRepo) MuteTopic(ctx context.Context, userID, topicID int64) error {
	_, err := r.db.Pool.Exec(ctx,
		`INSERT INTO user_topic_mutes (user_id, topic_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`,
		userID, topicID)
	return err
}

func (r *FollowRepo) UnmuteTopic(ctx context.Context, userID, topicID int64) error {
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM user_topic_mutes WHERE user_id=$1 AND topic_id=$2`, userID, topicID)
	return err
}

// Follows is the aggregate of everything a user follows.
type Follows struct {
	Topics   []domain.Topic  `json:"topics"`
	Entities []domain.Entity `json:"entities"`
	Sources  []domain.Source `json:"sources"`
}

// ListFollows returns a user's followed topics, entities and sources.
func (r *FollowRepo) ListFollows(ctx context.Context, userID int64) (*Follows, error) {
	f := &Follows{Topics: []domain.Topic{}, Entities: []domain.Entity{}, Sources: []domain.Source{}}

	tRows, err := r.db.Pool.Query(ctx, `
		SELECT `+prefixCols(topicCols, "t")+` FROM topics t
		JOIN user_topic_follows utf ON utf.topic_id=t.id WHERE utf.user_id=$1 ORDER BY t.name`, userID)
	if err != nil {
		return nil, err
	}
	f.Topics, err = collectTopics(tRows)
	tRows.Close()
	if err != nil {
		return nil, err
	}

	eRows, err := r.db.Pool.Query(ctx, `
		SELECT `+prefixCols(entityCols, "e")+` FROM entities e
		JOIN user_entity_follows uef ON uef.entity_id=e.id WHERE uef.user_id=$1 ORDER BY e.name`, userID)
	if err != nil {
		return nil, err
	}
	f.Entities, err = collectEntities(eRows)
	eRows.Close()
	if err != nil {
		return nil, err
	}

	sRows, err := r.db.Pool.Query(ctx, `
		SELECT `+sourceColsAliased("s")+` FROM sources s
		JOIN user_source_follows usf ON usf.source_id=s.id WHERE usf.user_id=$1 ORDER BY s.name`, userID)
	if err != nil {
		return nil, err
	}
	defer sRows.Close()
	for sRows.Next() {
		src, err := scanSource(sRows)
		if err != nil {
			return nil, err
		}
		f.Sources = append(f.Sources, *src)
	}
	return f, sRows.Err()
}
