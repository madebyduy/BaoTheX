package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"repwire/internal/domain"
)

type EngagementRepo struct{ db *DB }

func (r *EngagementRepo) Subscription(ctx context.Context, userID int64) (*domain.Subscription, error) {
	var s domain.Subscription
	err := r.db.Pool.QueryRow(ctx, `
		SELECT user_id, plan, status, provider, current_period_end
		FROM user_subscriptions WHERE user_id=$1`, userID).
		Scan(&s.UserID, &s.Plan, &s.Status, &s.Provider, &s.CurrentPeriodEnd)
	if errors.Is(err, pgx.ErrNoRows) {
		return &domain.Subscription{UserID: userID, Plan: "free", Status: "inactive"}, nil
	}
	return &s, err
}

func (r *EngagementRepo) CreatePaymentOrder(ctx context.Context, userID int64, invoice string, amount int64) (*domain.PaymentOrder, error) {
	var p domain.PaymentOrder
	err := r.db.Pool.QueryRow(ctx, `
		INSERT INTO payment_orders (user_id, invoice_number, amount_vnd)
		VALUES ($1,$2,$3)
		RETURNING id,user_id,invoice_number,amount_vnd,status,created_at`, userID, invoice, amount).
		Scan(&p.ID, &p.UserID, &p.InvoiceNumber, &p.AmountVND, &p.Status, &p.CreatedAt)
	return &p, err
}

// MarkPaymentPaid is idempotent. newlyPaid is true only for the first valid IPN.
func (r *EngagementRepo) MarkPaymentPaid(ctx context.Context, invoice, transactionID, providerOrderID string, amount int64) (userID int64, newlyPaid bool, err error) {
	err = r.db.WithTx(ctx, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			UPDATE payment_orders SET status='paid', provider_transaction=$2,
			provider_order_id=$3, paid_at=now(), updated_at=now()
			WHERE invoice_number=$1 AND status='pending' AND amount_vnd=$4
			RETURNING user_id`, invoice, transactionID, providerOrderID, amount).Scan(&userID)
		if errors.Is(err, pgx.ErrNoRows) {
			var status string
			var expected int64
			if e := tx.QueryRow(ctx, `SELECT user_id,status,amount_vnd FROM payment_orders WHERE invoice_number=$1`, invoice).Scan(&userID, &status, &expected); e != nil {
				return e
			}
			if status == "pending" && expected != amount {
				return domain.ErrValidation
			}
			return nil
		}
		if err != nil {
			return err
		}
		newlyPaid = true
		_, err = tx.Exec(ctx, `
			INSERT INTO user_subscriptions (user_id,plan,status,provider,current_period_end)
			VALUES ($1,'premium','active','sepay',now()+interval '30 days')
			ON CONFLICT (user_id) DO UPDATE SET
			  plan='premium', status='active', provider='sepay',
			  current_period_end=GREATEST(COALESCE(user_subscriptions.current_period_end,now()),now())+interval '30 days',
			  updated_at=now()`, userID)
		return err
	})
	return
}

func (r *EngagementRepo) LatestAudioBrief(ctx context.Context, edition string) (*domain.AudioBrief, error) {
	var b domain.AudioBrief
	err := r.db.Pool.QueryRow(ctx, `
		SELECT id,brief_date,edition,title,script,audio_url,duration_seconds,content_ids,status,error,created_at,updated_at
		FROM audio_briefs WHERE status='ready' AND ($1='' OR edition=$1)
		ORDER BY brief_date DESC, updated_at DESC LIMIT 1`, edition).
		Scan(&b.ID, &b.BriefDate, &b.Edition, &b.Title, &b.Script, &b.AudioURL, &b.DurationSeconds, &b.ContentIDs, &b.Status, &b.Error, &b.CreatedAt, &b.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &b, err
}

// AudioBriefForDate gets exactly one scheduled edition. It is used for
// delivery so a delayed queue item can never accidentally send a newer issue.
func (r *EngagementRepo) AudioBriefForDate(ctx context.Context, date time.Time, edition string) (*domain.AudioBrief, error) {
	var b domain.AudioBrief
	err := r.db.Pool.QueryRow(ctx, `
		SELECT id,brief_date,edition,title,script,audio_url,duration_seconds,content_ids,status,error,created_at,updated_at
		FROM audio_briefs WHERE brief_date=$1 AND edition=$2 AND status='ready' AND audio_url IS NOT NULL`,
		date.Format("2006-01-02"), edition).
		Scan(&b.ID, &b.BriefDate, &b.Edition, &b.Title, &b.Script, &b.AudioURL, &b.DurationSeconds, &b.ContentIDs, &b.Status, &b.Error, &b.CreatedAt, &b.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &b, err
}

// RecordAudioBriefDelivery claims a send slot before Telegram is called.
// The unique key makes scheduled retries idempotent.
func (r *EngagementRepo) RecordAudioBriefDelivery(ctx context.Context, briefID, userID int64, messageID *int64, deliveryErr *string) (bool, error) {
	tag, err := r.db.Pool.Exec(ctx, `
		INSERT INTO audio_brief_deliveries (audio_brief_id,user_id,telegram_message_id,error)
		VALUES ($1,$2,$3,$4) ON CONFLICT (audio_brief_id,user_id) DO NOTHING`, briefID, userID, messageID, deliveryErr)
	return tag.RowsAffected() == 1, err
}

func (r *EngagementRepo) CompleteAudioBriefDelivery(ctx context.Context, briefID, userID, messageID int64) error {
	_, err := r.db.Pool.Exec(ctx, `
		UPDATE audio_brief_deliveries SET telegram_message_id=$3,error=NULL,sent_at=now()
		WHERE audio_brief_id=$1 AND user_id=$2`, briefID, userID, messageID)
	return err
}

// ReleaseAudioBriefDelivery makes a transient Telegram failure retryable.
func (r *EngagementRepo) ReleaseAudioBriefDelivery(ctx context.Context, briefID, userID int64) error {
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM audio_brief_deliveries WHERE audio_brief_id=$1 AND user_id=$2`, briefID, userID)
	return err
}

func (r *EngagementRepo) HasAudioBriefDate(ctx context.Context, date time.Time, edition string) (bool, error) {
	var exists bool
	err := r.db.Pool.QueryRow(ctx, `SELECT EXISTS(
		SELECT 1 FROM audio_briefs
		WHERE brief_date=$1 AND edition=$2 AND status='ready' AND duration_seconds >= 300)`,
		date.Format("2006-01-02"), edition).Scan(&exists)
	return exists, err
}

func (r *EngagementRepo) SaveAudioBrief(ctx context.Context, date time.Time, edition, title, script, audioURL string, duration int, ids []int64) error {
	_, err := r.db.Pool.Exec(ctx, `
		INSERT INTO audio_briefs (brief_date,edition,title,script,audio_url,duration_seconds,content_ids,status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,'ready')
		ON CONFLICT (brief_date,edition) DO UPDATE SET title=$3,script=$4,audio_url=$5,
		  duration_seconds=$6,content_ids=$7,status='ready',error=NULL,updated_at=now()`,
		date.Format("2006-01-02"), edition, title, script, audioURL, duration, ids)
	return err
}

// AddReaction records a device's like for a content item (idempotent per device).
func (r *EngagementRepo) AddReaction(ctx context.Context, contentID int64, clientID string) error {
	_, err := r.db.Pool.Exec(ctx,
		`INSERT INTO content_reactions (content_id, client_id) VALUES ($1,$2)
		 ON CONFLICT (content_id, client_id) DO NOTHING`, contentID, clientID)
	return err
}

// RemoveReaction removes a device's like.
func (r *EngagementRepo) RemoveReaction(ctx context.Context, contentID int64, clientID string) error {
	_, err := r.db.Pool.Exec(ctx,
		`DELETE FROM content_reactions WHERE content_id=$1 AND client_id=$2`, contentID, clientID)
	return err
}

// Reactions returns the total like count and whether the given device liked it.
func (r *EngagementRepo) Reactions(ctx context.Context, contentID int64, clientID string) (int, bool, error) {
	var count int
	var liked bool
	err := r.db.Pool.QueryRow(ctx, `
		SELECT count(*)::int, COALESCE(bool_or(client_id = $2), false)
		FROM content_reactions WHERE content_id=$1`, contentID, clientID).Scan(&count, &liked)
	if err != nil {
		return 0, false, err
	}
	return count, liked, nil
}

func (r *EngagementRepo) UpsertPushSubscription(ctx context.Context, userID int64, sub domain.PushSubscription) error {
	_, err := r.db.Pool.Exec(ctx, `
		INSERT INTO push_subscriptions (endpoint,user_id,p256dh,auth,user_agent)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (endpoint) DO UPDATE SET user_id=$2,p256dh=$3,auth=$4,user_agent=$5,last_used_at=now()`,
		sub.Endpoint, userID, sub.P256DH, sub.Auth, sub.UserAgent)
	return err
}

func (r *EngagementRepo) DeletePushSubscription(ctx context.Context, userID int64, endpoint string) error {
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM push_subscriptions WHERE user_id=$1 AND endpoint=$2`, userID, endpoint)
	return err
}

func (r *EngagementRepo) PushSubscriptions(ctx context.Context, userID int64) ([]domain.PushSubscription, error) {
	rows, err := r.db.Pool.Query(ctx, `SELECT endpoint,p256dh,auth,COALESCE(user_agent,'') FROM push_subscriptions WHERE user_id=$1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.PushSubscription
	for rows.Next() {
		var sub domain.PushSubscription
		if err := rows.Scan(&sub.Endpoint, &sub.P256DH, &sub.Auth, &sub.UserAgent); err != nil {
			return nil, err
		}
		out = append(out, sub)
	}
	return out, rows.Err()
}
