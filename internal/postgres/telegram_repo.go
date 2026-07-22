package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"repwire/internal/domain"
)

// TelegramRepo persists Telegram links, notification prefs and digest deliveries.
type TelegramRepo struct{ db *DB }

// CreateLinkCode stores a one-time link code for a user.
func (r *TelegramRepo) CreateLinkCode(ctx context.Context, code string, userID int64, expiresAt time.Time) error {
	_, err := r.db.Pool.Exec(ctx,
		`INSERT INTO telegram_link_codes (code, user_id, expires_at) VALUES ($1,$2,$3)`,
		code, userID, expiresAt)
	return err
}

// ConsumeLinkCode validates a code and links the chat, returning the user id.
// The code is deleted whether or not a prior connection existed.
func (r *TelegramRepo) ConsumeLinkCode(ctx context.Context, code string, chatID int64, username *string) (int64, error) {
	var userID int64
	err := r.db.WithTx(ctx, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`DELETE FROM telegram_link_codes WHERE code=$1 AND expires_at > now() RETURNING user_id`, code).
			Scan(&userID)
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO telegram_connections (user_id, chat_id, username) VALUES ($1,$2,$3)
			ON CONFLICT (user_id) DO UPDATE SET chat_id=EXCLUDED.chat_id, username=EXCLUDED.username, linked_at=now()`,
			userID, chatID, username)
		if err != nil {
			return err
		}
		// Following the deep link is the explicit opt-in. Re-linking also resumes
		// both fixed editions after a previous pause or blocked-bot recovery.
		_, err = tx.Exec(ctx, `INSERT INTO notification_preferences
			(user_id,daily_enabled,audio_enabled,evening_brief_enabled)
			VALUES ($1,true,true,true)
			ON CONFLICT (user_id) DO UPDATE SET daily_enabled=true,
				audio_enabled=true,evening_brief_enabled=true`, userID)
		return err
	})
	return userID, err
}

// ConnectionByChat returns the connection for a Telegram chat id.
func (r *TelegramRepo) ConnectionByChat(ctx context.Context, chatID int64) (*domain.TelegramConnection, error) {
	var c domain.TelegramConnection
	err := r.db.Pool.QueryRow(ctx,
		`SELECT user_id, chat_id, username, linked_at FROM telegram_connections WHERE chat_id=$1`, chatID).
		Scan(&c.UserID, &c.ChatID, &c.Username, &c.LinkedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &c, err
}

// ConnectionByUser returns the connection for a user id.
func (r *TelegramRepo) ConnectionByUser(ctx context.Context, userID int64) (*domain.TelegramConnection, error) {
	var c domain.TelegramConnection
	err := r.db.Pool.QueryRow(ctx,
		`SELECT user_id, chat_id, username, linked_at FROM telegram_connections WHERE user_id=$1`, userID).
		Scan(&c.UserID, &c.ChatID, &c.Username, &c.LinkedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &c, err
}

// Unlink removes a user's Telegram connection.
func (r *TelegramRepo) Unlink(ctx context.Context, userID int64) error {
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM telegram_connections WHERE user_id=$1`, userID)
	return err
}

// ---- Notification preferences ----

// GetPrefs returns a user's notification preferences (creating defaults if missing).
func (r *TelegramRepo) GetPrefs(ctx context.Context, userID int64) (*domain.NotificationPreferences, error) {
	p := domain.NotificationPreferences{UserID: userID}
	var days []int16
	var types []string
	err := r.db.Pool.QueryRow(ctx, `
		SELECT daily_enabled, audio_enabled, evening_brief_enabled, daily_hour, daily_days, daily_max_items, weekly_research, weekly_dow,
		       follow_alerts, highlights_only, quiet_start, quiet_end, content_types::text[], feed_following_only
		FROM notification_preferences WHERE user_id=$1`, userID).
		Scan(&p.DailyEnabled, &p.AudioEnabled, &p.EveningBriefEnabled, &p.DailyHour, &days, &p.DailyMaxItems, &p.WeeklyResearch, &p.WeeklyDOW,
			&p.FollowAlerts, &p.HighlightsOnly, &p.QuietStart, &p.QuietEnd, &types, &p.FeedFollowingOnly)
	if errors.Is(err, pgx.ErrNoRows) {
		def := domain.DefaultNotificationPreferences(userID)
		if _, e := r.db.Pool.Exec(ctx, `INSERT INTO notification_preferences (user_id) VALUES ($1) ON CONFLICT DO NOTHING`, userID); e != nil {
			return nil, e
		}
		return &def, nil
	}
	if err != nil {
		return nil, err
	}
	p.DailyDays = intsFromInt16(days)
	p.ContentTypes = make([]domain.ContentType, len(types))
	for i, t := range types {
		p.ContentTypes[i] = domain.ContentType(t)
	}
	return &p, nil
}

// UpdatePrefs replaces a user's notification preferences.
func (r *TelegramRepo) UpdatePrefs(ctx context.Context, p *domain.NotificationPreferences) error {
	types := make([]string, len(p.ContentTypes))
	for i, t := range p.ContentTypes {
		types[i] = string(t)
	}
	_, err := r.db.Pool.Exec(ctx, `
		INSERT INTO notification_preferences
			(user_id, daily_enabled, audio_enabled, evening_brief_enabled, daily_hour, daily_days, daily_max_items, weekly_research,
			 weekly_dow, follow_alerts, highlights_only, quiet_start, quiet_end, content_types, feed_following_only)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14::content_type[],$15)
		ON CONFLICT (user_id) DO UPDATE SET
			daily_enabled=EXCLUDED.daily_enabled, daily_hour=EXCLUDED.daily_hour,
			audio_enabled=EXCLUDED.audio_enabled,
			evening_brief_enabled=EXCLUDED.evening_brief_enabled,
			daily_days=EXCLUDED.daily_days, daily_max_items=EXCLUDED.daily_max_items,
			weekly_research=EXCLUDED.weekly_research, weekly_dow=EXCLUDED.weekly_dow,
			follow_alerts=EXCLUDED.follow_alerts, highlights_only=EXCLUDED.highlights_only,
			quiet_start=EXCLUDED.quiet_start, quiet_end=EXCLUDED.quiet_end,
			content_types=EXCLUDED.content_types,
			feed_following_only=EXCLUDED.feed_following_only`,
		p.UserID, p.DailyEnabled, p.AudioEnabled, p.EveningBriefEnabled, p.DailyHour, int16sFromInts(p.DailyDays), p.DailyMaxItems,
		p.WeeklyResearch, p.WeeklyDOW, p.FollowAlerts, p.HighlightsOnly, p.QuietStart, p.QuietEnd, types, p.FeedFollowingOnly)
	return err
}

// UsersForAudioBrief returns every linked user who opted into the requested
// fixed edition. Morning follows DailyEnabled; evening has its own switch.
func (r *TelegramRepo) UsersForAudioBrief(ctx context.Context, edition string) ([]int64, error) {
	condition := "p.daily_enabled"
	if edition == "evening" {
		condition = "p.evening_brief_enabled"
	}
	rows, err := r.db.Pool.Query(ctx, `
		SELECT u.id FROM users u
		JOIN telegram_connections tc ON tc.user_id=u.id
		JOIN notification_preferences p ON p.user_id=u.id
		WHERE p.audio_enabled AND `+condition)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanIDs(rows)
}

// SetBriefDeliveryEnabled toggles the morning text/audio and evening audio
// together. It is used by /pause, /resume, and blocked-bot recovery.
func (r *TelegramRepo) SetBriefDeliveryEnabled(ctx context.Context, userID int64, enabled bool) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE notification_preferences SET daily_enabled=$2,audio_enabled=$2,
			evening_brief_enabled=$2 WHERE user_id=$1`, userID, enabled)
	return err
}

// ---- Digest scheduling ----

// UsersDueForDaily returns user ids whose local time matches their daily_hour
// and who have not yet received today's daily digest.
func (r *TelegramRepo) UsersDueForDaily(ctx context.Context) ([]int64, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT u.id FROM users u
		JOIN notification_preferences p ON p.user_id=u.id
		JOIN telegram_connections t ON t.user_id=u.id
		WHERE p.daily_enabled
		  AND EXTRACT(hour FROM now() AT TIME ZONE u.timezone) = p.daily_hour
		  AND EXTRACT(isodow FROM now() AT TIME ZONE u.timezone) = ANY(p.daily_days)
		  AND NOT EXISTS (
		      SELECT 1 FROM digest_deliveries d
		      WHERE d.user_id=u.id AND d.kind='daily' AND d.error IS NULL
		        AND (d.sent_at AT TIME ZONE u.timezone)::date = (now() AT TIME ZONE u.timezone)::date)`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanIDs(rows)
}

// FollowAlertCooldown is the minimum gap between two follow alerts to one user.
//
// A follow alert fires on news rather than on a clock, which is exactly what
// makes it dangerous: a user following "Bóng đá quốc tế" during a World Cup
// would otherwise be pinged every few minutes by an aggregator that ingests
// dozens of articles an hour. The daily brief is safe because it is rationed by
// definition; this is not, so the ration is explicit. Six hours means at most
// four a day, and in practice far fewer, since quiet hours cover the night.
const FollowAlertCooldown = 6 * time.Hour

// UsersDueForFollowAlert returns users who have follow alerts switched on, are
// connected to the bot, are not inside their quiet hours, and have not been sent
// an alert within FollowAlertCooldown.
//
// The quiet window is a pair of local hours and it usually wraps midnight
// (22→07 by default), so it cannot be expressed as a simple BETWEEN: when start
// is greater than end the quiet period is "at or after start OR before end".
// Equal values mean no quiet hours at all rather than a 24-hour blackout.
func (r *TelegramRepo) UsersDueForFollowAlert(ctx context.Context) ([]int64, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT u.id FROM users u
		JOIN notification_preferences p ON p.user_id=u.id
		JOIN telegram_connections t ON t.user_id=u.id
		WHERE p.follow_alerts
		  AND NOT (
		      CASE
		        WHEN p.quiet_start = p.quiet_end THEN FALSE
		        WHEN p.quiet_start < p.quiet_end THEN
		          EXTRACT(hour FROM now() AT TIME ZONE u.timezone) >= p.quiet_start
		          AND EXTRACT(hour FROM now() AT TIME ZONE u.timezone) < p.quiet_end
		        ELSE
		          EXTRACT(hour FROM now() AT TIME ZONE u.timezone) >= p.quiet_start
		          OR EXTRACT(hour FROM now() AT TIME ZONE u.timezone) < p.quiet_end
		      END)
		  AND NOT EXISTS (
		      SELECT 1 FROM digest_deliveries d
		      WHERE d.user_id=u.id AND d.kind='follow_alert' AND d.error IS NULL
		        AND d.sent_at > now() - $1::interval)`, FollowAlertCooldown)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanIDs(rows)
}

// RecordDelivery logs a digest delivery (with optional error).
func (r *TelegramRepo) RecordDelivery(ctx context.Context, userID int64, kind domain.DigestKind, contentIDs []int64, messageID *int64, deliveryErr *string) error {
	if contentIDs == nil {
		contentIDs = []int64{}
	}
	_, err := r.db.Pool.Exec(ctx,
		`INSERT INTO digest_deliveries (user_id, kind, content_ids, message_id, error) VALUES ($1,$2,$3,$4,$5)
		 ON CONFLICT DO NOTHING`,
		userID, kind, contentIDs, messageID, deliveryErr)
	return err
}

// AlertsSentToday counts follow alerts sent to a user today (rate limiting).
func (r *TelegramRepo) AlertsSentToday(ctx context.Context, userID int64) (int, error) {
	var n int
	err := r.db.Pool.QueryRow(ctx, `
		SELECT count(*) FROM digest_deliveries
		WHERE user_id=$1 AND kind='follow_alert' AND error IS NULL AND sent_at::date = now()::date`, userID).Scan(&n)
	return n, err
}

func scanIDs(rows pgx.Rows) ([]int64, error) {
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
