package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"repwire/internal/domain"
)

type SportsRepo struct{ db *DB }

const eventSelect = `
SELECT e.id,e.sport_id,s.slug,s.name,e.competition_id,co.name,e.title,e.home_name,e.away_name,
       e.starts_at,e.status,e.home_score,e.away_score,e.result_details,e.data_source,e.external_id,
       e.data_updated_at,CASE WHEN NOT e.is_manual AND e.data_updated_at<now()-interval '2 hours' THEN 'stale' ELSE e.freshness END,e.is_manual,e.manual_locked,e.coverage`

func (r *SportsRepo) ListSports(ctx context.Context) ([]domain.Sport, error) {
	rows, err := r.db.Pool.Query(ctx, `SELECT id,slug,name,enabled FROM sports WHERE enabled ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Sport{}
	for rows.Next() {
		var v domain.Sport
		if err := rows.Scan(&v.ID, &v.Slug, &v.Name, &v.Enabled); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *SportsRepo) ListCompetitions(ctx context.Context, sport string) ([]domain.Competition, error) {
	rows, err := r.db.Pool.Query(ctx, `SELECT c.id,c.sport_id,s.slug,c.slug,c.name,c.country,c.data_source,c.external_id,c.coverage
		FROM sports_competitions c JOIN sports s ON s.id=c.sport_id
		WHERE ($1='' OR s.slug=$1) ORDER BY s.name,c.name`, sport)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Competition{}
	for rows.Next() {
		var v domain.Competition
		if err := rows.Scan(&v.ID, &v.SportID, &v.SportSlug, &v.Slug, &v.Name, &v.Country, &v.DataSource, &v.ExternalID, &v.Coverage); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// GetCompetitionBySlug returns a single competition for the league hub page.
func (r *SportsRepo) GetCompetitionBySlug(ctx context.Context, slug string) (*domain.Competition, error) {
	var v domain.Competition
	err := r.db.Pool.QueryRow(ctx, `SELECT c.id,c.sport_id,s.slug,c.slug,c.name,c.country,c.data_source,c.external_id,c.coverage
		FROM sports_competitions c JOIN sports s ON s.id=c.sport_id WHERE c.slug=$1`, slug).
		Scan(&v.ID, &v.SportID, &v.SportSlug, &v.Slug, &v.Name, &v.Country, &v.DataSource, &v.ExternalID, &v.Coverage)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &v, err
}

func scanSportsEvent(row pgx.Row) (*domain.SportsEvent, error) {
	var v domain.SportsEvent
	err := row.Scan(&v.ID, &v.SportID, &v.SportSlug, &v.SportName, &v.CompetitionID, &v.Competition,
		&v.Title, &v.HomeName, &v.AwayName, &v.StartsAt, &v.Status, &v.HomeScore, &v.AwayScore,
		&v.ResultDetails, &v.DataSource, &v.ExternalID, &v.DataUpdatedAt, &v.Freshness, &v.IsManual, &v.ManualLocked, &v.Coverage)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &v, err
}

func (r *SportsRepo) ListEvents(ctx context.Context, sport, competition, status, date string, limit int) ([]domain.SportsEvent, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	query := eventSelect + ` FROM sports_events e JOIN sports s ON s.id=e.sport_id
		LEFT JOIN sports_competitions co ON co.id=e.competition_id WHERE 1=1`
	args := []any{}
	add := func(clause string, v any) { args = append(args, v); query += fmt.Sprintf(clause, len(args)) }
	if sport != "" {
		add(` AND s.slug=$%d`, sport)
	}
	if competition != "" {
		add(` AND co.slug=$%d`, competition)
	}
	if status != "" {
		add(` AND e.status=$%d`, status)
	}
	if date != "" {
		add(` AND (e.starts_at AT TIME ZONE 'Asia/Ho_Chi_Minh')::date = $%d::date`, date)
	}
	args = append(args, limit)
	query += fmt.Sprintf(` ORDER BY e.starts_at ASC LIMIT $%d`, len(args))
	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.SportsEvent{}
	for rows.Next() {
		v, err := scanSportsEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}

func (r *SportsRepo) GetEvent(ctx context.Context, id int64, userID int64) (*domain.SportsEvent, error) {
	v, err := scanSportsEvent(r.db.Pool.QueryRow(ctx, eventSelect+` FROM sports_events e JOIN sports s ON s.id=e.sport_id
		LEFT JOIN sports_competitions co ON co.id=e.competition_id WHERE e.id=$1`, id))
	if err != nil {
		return nil, err
	}
	if userID > 0 {
		if err := r.db.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM user_event_follows WHERE user_id=$1 AND event_id=$2)`, userID, id).Scan(&v.Following); err != nil {
			return nil, err
		}
	}
	v.RelatedContent, err = r.EventContent(ctx, id, 20)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (r *SportsRepo) EventContent(ctx context.Context, eventID int64, limit int) ([]domain.ContentItem, error) {
	rows, err := r.db.Pool.Query(ctx, `SELECT `+contentCols+`,s.name FROM event_content_links l
		JOIN content_items c ON c.id=l.content_id JOIN sources s ON s.id=c.source_id
		WHERE l.event_id=$1 AND c.status='ready' ORDER BY c.final_score DESC,c.published_at DESC LIMIT $2`, eventID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectContent(rows)
}

type EventInput struct {
	ID            int64
	SportSlug     string
	CompetitionID *int64
	Title         string
	HomeName      *string
	AwayName      *string
	StartsAt      time.Time
	Status        string
	HomeScore     *string
	AwayScore     *string
	ManualLocked  bool
}

type ProviderEventInput struct {
	SportSlug      string
	CompetitionKey string
	Competition    string
	ExternalID     string
	Title          string
	HomeName       *string
	AwayName       *string
	StartsAt       time.Time
	Status         string
	HomeScore      *string
	AwayScore      *string
	DataSource     string
	Freshness      string
	Coverage       json.RawMessage
}

// UpsertProviderEvent caches normalized free-provider data. An editor-locked
// row always wins over later syncs, so manual corrections cannot be erased.
func (r *SportsRepo) UpsertProviderEvent(ctx context.Context, in ProviderEventInput) error {
	if len(in.Coverage) == 0 {
		in.Coverage = json.RawMessage(`{}`)
	}
	return r.db.WithTx(ctx, func(tx pgx.Tx) error {
		var sportID int64
		if err := tx.QueryRow(ctx, `SELECT id FROM sports WHERE slug=$1`, in.SportSlug).Scan(&sportID); err != nil {
			return err
		}
		var competitionID *int64
		if in.Competition != "" {
			key := in.CompetitionKey
			if key == "" {
				key = in.DataSource + "-" + strings.ToLower(strings.ReplaceAll(in.Competition, " ", "-"))
			}
			var id int64
			err := tx.QueryRow(ctx, `INSERT INTO sports_competitions(sport_id,slug,name,data_source,external_id,coverage)
				VALUES($1,$2,$3,$4,NULLIF($5,''),$6) ON CONFLICT(slug) DO UPDATE SET name=EXCLUDED.name,coverage=EXCLUDED.coverage,updated_at=now() RETURNING id`,
				sportID, key, in.Competition, in.DataSource, in.CompetitionKey, in.Coverage).Scan(&id)
			if err != nil {
				return err
			}
			competitionID = &id
		}
		_, err := tx.Exec(ctx, `INSERT INTO sports_events(sport_id,competition_id,title,home_name,away_name,starts_at,status,home_score,away_score,
			data_source,external_id,data_updated_at,freshness,is_manual,coverage)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,now(),$12,FALSE,$13)
			ON CONFLICT(data_source,external_id) WHERE external_id IS NOT NULL DO UPDATE SET
			competition_id=EXCLUDED.competition_id,title=EXCLUDED.title,home_name=EXCLUDED.home_name,away_name=EXCLUDED.away_name,
			starts_at=EXCLUDED.starts_at,status=EXCLUDED.status,home_score=EXCLUDED.home_score,away_score=EXCLUDED.away_score,
			data_updated_at=now(),freshness=EXCLUDED.freshness,coverage=EXCLUDED.coverage,updated_at=now()
			WHERE NOT sports_events.manual_locked`, sportID, competitionID, in.Title, in.HomeName, in.AwayName, in.StartsAt, in.Status, in.HomeScore, in.AwayScore, in.DataSource, in.ExternalID, in.Freshness, in.Coverage)
		return err
	})
}

func (r *SportsRepo) SaveManualEvent(ctx context.Context, in EventInput) (*domain.SportsEvent, error) {
	var id int64
	if in.ID == 0 {
		err := r.db.Pool.QueryRow(ctx, `INSERT INTO sports_events
			(sport_id,competition_id,title,home_name,away_name,starts_at,status,home_score,away_score,data_source,freshness,is_manual,manual_locked)
			SELECT s.id,$2,$3,$4,$5,$6,$7,$8,$9,'baothex','manual',TRUE,$10 FROM sports s WHERE s.slug=$1 RETURNING id`,
			in.SportSlug, in.CompetitionID, in.Title, in.HomeName, in.AwayName, in.StartsAt, in.Status, in.HomeScore, in.AwayScore, in.ManualLocked).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrValidation
		}
		if err != nil {
			return nil, err
		}
	} else {
		tag, err := r.db.Pool.Exec(ctx, `UPDATE sports_events SET sport_id=(SELECT id FROM sports WHERE slug=$2),competition_id=$3,
			title=$4,home_name=$5,away_name=$6,starts_at=$7,status=$8,home_score=$9,away_score=$10,
			manual_locked=$11,is_manual=TRUE,data_source='baothex',freshness='manual',data_updated_at=now(),updated_at=now() WHERE id=$1`,
			in.ID, in.SportSlug, in.CompetitionID, in.Title, in.HomeName, in.AwayName, in.StartsAt, in.Status, in.HomeScore, in.AwayScore, in.ManualLocked)
		if err != nil {
			return nil, err
		}
		if tag.RowsAffected() == 0 {
			return nil, domain.ErrNotFound
		}
		id = in.ID
	}
	return r.GetEvent(ctx, id, 0)
}

func (r *SportsRepo) FollowEvent(ctx context.Context, userID, eventID int64, follow bool) error {
	if follow {
		_, err := r.db.Pool.Exec(ctx, `INSERT INTO user_event_follows(user_id,event_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, userID, eventID)
		return err
	}
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM user_event_follows WHERE user_id=$1 AND event_id=$2`, userID, eventID)
	return err
}

func (r *SportsRepo) FollowCluster(ctx context.Context, userID, clusterID int64, follow bool) error {
	if follow {
		_, err := r.db.Pool.Exec(ctx, `INSERT INTO story_cluster_follows(user_id,cluster_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, userID, clusterID)
		return err
	}
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM story_cluster_follows WHERE user_id=$1 AND cluster_id=$2`, userID, clusterID)
	return err
}

func (r *SportsRepo) MarkClusterRead(ctx context.Context, userID, clusterID int64) error {
	_, err := r.db.Pool.Exec(ctx, `INSERT INTO story_cluster_reads(user_id,cluster_id,last_read_at) VALUES($1,$2,now())
		ON CONFLICT(user_id,cluster_id) DO UPDATE SET last_read_at=now()`, userID, clusterID)
	return err
}

var defaultDashboard = json.RawMessage(`["today","catch_up","schedule","favorites","following","read_later","listen_later","predictions"]`)

func (r *SportsRepo) Dashboard(ctx context.Context, userID int64) (json.RawMessage, error) {
	var layout json.RawMessage
	err := r.db.Pool.QueryRow(ctx, `SELECT layout FROM user_dashboard_layouts WHERE user_id=$1`, userID).Scan(&layout)
	if errors.Is(err, pgx.ErrNoRows) {
		return defaultDashboard, nil
	}
	return layout, err
}

func (r *SportsRepo) SaveDashboard(ctx context.Context, userID int64, layout json.RawMessage) error {
	_, err := r.db.Pool.Exec(ctx, `INSERT INTO user_dashboard_layouts(user_id,layout) VALUES($1,$2)
		ON CONFLICT(user_id) DO UPDATE SET layout=EXCLUDED.layout,updated_at=now()`, userID, layout)
	return err
}

// SyncPreferences applies an offline client's preference bundle atomically.
// A missing topic/entity or a database error rolls back the dashboard, goals
// and every follow rather than reporting success for a partially synced state.
func (r *SportsRepo) SyncPreferences(ctx context.Context, userID int64, layout json.RawMessage, goals []string, topicIDs, entityIDs []int64) error {
	return r.db.WithTx(ctx, func(tx pgx.Tx) error {
		if len(layout) > 0 {
			if _, err := tx.Exec(ctx, `INSERT INTO user_dashboard_layouts(user_id,layout) VALUES($1,$2)
				ON CONFLICT(user_id) DO UPDATE SET layout=EXCLUDED.layout,updated_at=now()`, userID, layout); err != nil {
				return err
			}
		}
		// An empty list is meaningful: it clears goals and still completes
		// onboarding. Skipping this update made an empty selection impossible to
		// persist and left users stuck in onboarding.
		if _, err := tx.Exec(ctx, `UPDATE users SET goals=$2,onboarded_at=COALESCE(onboarded_at,now()) WHERE id=$1`, userID, goals); err != nil {
			return err
		}
		for _, topicID := range topicIDs {
			tag, err := tx.Exec(ctx, `INSERT INTO user_topic_follows(user_id,topic_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, userID, topicID)
			if err != nil {
				return err
			}
			if tag.RowsAffected() > 0 {
				if _, err := tx.Exec(ctx, `UPDATE topics SET follower_count=follower_count+1 WHERE id=$1`, topicID); err != nil {
					return err
				}
			}
		}
		for _, entityID := range entityIDs {
			tag, err := tx.Exec(ctx, `INSERT INTO user_entity_follows(user_id,entity_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, userID, entityID)
			if err != nil {
				return err
			}
			if tag.RowsAffected() > 0 {
				if _, err := tx.Exec(ctx, `UPDATE entities SET follower_count=follower_count+1 WHERE id=$1`, entityID); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (r *SportsRepo) CatchUp(ctx context.Context, userID int64, limit int) ([]domain.ContentItem, error) {
	if limit < 3 {
		limit = 3
	}
	if limit > 20 {
		limit = 20
	}
	rows, err := r.db.Pool.Query(ctx, `WITH ranked AS (
		SELECT c.id,
			row_number() OVER(
				PARTITION BY COALESCE(sci.cluster_id,-c.id)
				ORDER BY
					CASE WHEN $1::bigint > 0 AND (
						EXISTS(SELECT 1 FROM content_topics ct JOIN user_topic_follows utf ON utf.topic_id=ct.topic_id WHERE ct.content_id=c.id AND utf.user_id=$1 AND utf.in_feed AND (NOT utf.highlights_only OR c.final_score >= 40))
						OR EXISTS(SELECT 1 FROM content_entities ce JOIN user_entity_follows uef ON uef.entity_id=ce.entity_id WHERE ce.content_id=c.id AND uef.user_id=$1 AND uef.in_feed AND (NOT uef.highlights_only OR c.final_score >= 40))
						OR EXISTS(SELECT 1 FROM user_source_follows usf WHERE usf.user_id=$1 AND usf.source_id=c.source_id)
						OR EXISTS(SELECT 1 FROM story_cluster_follows scf WHERE scf.user_id=$1 AND scf.cluster_id=sci.cluster_id)
					) THEN 1 ELSE 0 END DESC,
					c.final_score DESC,c.published_at DESC
			) rn,
			CASE WHEN $1::bigint > 0 AND (
				EXISTS(SELECT 1 FROM content_topics ct JOIN user_topic_follows utf ON utf.topic_id=ct.topic_id WHERE ct.content_id=c.id AND utf.user_id=$1 AND utf.in_feed AND (NOT utf.highlights_only OR c.final_score >= 40))
				OR EXISTS(SELECT 1 FROM content_entities ce JOIN user_entity_follows uef ON uef.entity_id=ce.entity_id WHERE ce.content_id=c.id AND uef.user_id=$1 AND uef.in_feed AND (NOT uef.highlights_only OR c.final_score >= 40))
				OR EXISTS(SELECT 1 FROM user_source_follows usf WHERE usf.user_id=$1 AND usf.source_id=c.source_id)
				OR EXISTS(SELECT 1 FROM story_cluster_follows scf WHERE scf.user_id=$1 AND scf.cluster_id=sci.cluster_id)
			) THEN 1 ELSE 0 END AS personal_match
		FROM content_items c
		LEFT JOIN story_cluster_items sci ON sci.content_id=c.id
		LEFT JOIN content_bodies b ON b.content_id=c.id
		WHERE c.status='ready' AND c.published_at>now()-interval '14 days'
		AND (c.language='vi' OR (b.translation_status IN ('ready','digested') AND NULLIF(trim(c.translated_title),'') IS NOT NULL AND NULLIF(trim(c.summary),'') IS NOT NULL))
		AND ($1::bigint=0 OR NOT EXISTS(SELECT 1 FROM hidden_items h WHERE h.user_id=$1 AND h.content_id=c.id))
		AND ($1::bigint=0 OR NOT EXISTS(SELECT 1 FROM user_source_mutes sm WHERE sm.user_id=$1 AND sm.source_id=c.source_id))
		AND ($1::bigint=0 OR NOT EXISTS(SELECT 1 FROM content_topics ct JOIN user_topic_mutes m ON m.topic_id=ct.topic_id WHERE ct.content_id=c.id AND m.user_id=$1)))
		SELECT `+contentCols+`,s.name FROM ranked r JOIN content_items c ON c.id=r.id JOIN sources s ON s.id=c.source_id
		WHERE r.rn=1 ORDER BY r.personal_match DESC,c.final_score DESC,c.published_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectContent(rows)
}

func (r *SportsRepo) ListPredictions(ctx context.Context, userID int64, limit int) ([]domain.Prediction, error) {
	rows, err := r.db.Pool.Query(ctx, `SELECT p.id,p.event_id,p.kind,p.question,p.options,
		CASE WHEN p.status='settled' THEN p.correct_option ELSE NULL END,p.deadline,p.status,p.points,
		pa.answer,pa.is_correct,(SELECT count(*) FROM prediction_answers x WHERE x.prediction_id=p.id)
		FROM predictions p LEFT JOIN prediction_answers pa ON pa.prediction_id=p.id AND pa.user_id=$1
		WHERE p.status<>'cancelled' ORDER BY (p.status='open' AND p.deadline>now()) DESC,p.deadline DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Prediction{}
	for rows.Next() {
		var v domain.Prediction
		if err := rows.Scan(&v.ID, &v.EventID, &v.Kind, &v.Question, &v.Options, &v.CorrectOption, &v.Deadline, &v.Status, &v.Points, &v.UserAnswer, &v.IsCorrect, &v.AnswerCount); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *SportsRepo) AnswerPrediction(ctx context.Context, userID, predictionID int64, answer string) error {
	tag, err := r.db.Pool.Exec(ctx, `INSERT INTO prediction_answers(prediction_id,user_id,answer)
		SELECT id,$2,$3 FROM predictions WHERE id=$1 AND status='open' AND deadline>now() AND options ? $3
		ON CONFLICT(prediction_id,user_id) DO NOTHING`, predictionID, userID, answer)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrConflict
	}
	return nil
}

func (r *SportsRepo) SavePrediction(ctx context.Context, p domain.Prediction, userID int64) (*domain.Prediction, error) {
	var id int64
	if p.ID == 0 {
		err := r.db.Pool.QueryRow(ctx, `INSERT INTO predictions(event_id,kind,question,options,deadline,points,created_by)
		VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`, p.EventID, p.Kind, p.Question, p.Options, p.Deadline, p.Points, userID).Scan(&id)
		if err != nil {
			return nil, err
		}
	} else {
		tag, err := r.db.Pool.Exec(ctx, `UPDATE predictions SET event_id=$2,kind=$3,question=$4,options=$5,deadline=$6,points=$7
		WHERE id=$1 AND status='open'`, p.ID, p.EventID, p.Kind, p.Question, p.Options, p.Deadline, p.Points)
		if err != nil {
			return nil, err
		}
		if tag.RowsAffected() == 0 {
			return nil, domain.ErrConflict
		}
		id = p.ID
	}
	items, err := r.ListPredictions(ctx, 0, 100)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].ID == id {
			return &items[i], nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *SportsRepo) SettlePrediction(ctx context.Context, id int64, correct string) error {
	return r.db.WithTx(ctx, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE predictions SET status='settled',correct_option=$2,settled_at=now() WHERE id=$1 AND status='open' AND options ? $2`, id, correct)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrConflict
		}
		_, err = tx.Exec(ctx, `UPDATE prediction_answers a SET is_correct=(a.answer=$2),points_earned=CASE WHEN a.answer=$2 THEN p.points ELSE 0 END FROM predictions p WHERE p.id=$1 AND a.prediction_id=p.id`, id, correct)
		return err
	})
}

func (r *SportsRepo) Passport(ctx context.Context, userID int64) (*domain.FanPassport, error) {
	var p domain.FanPassport
	err := r.db.Pool.QueryRow(ctx, `SELECT
		(SELECT count(DISTINCT (occurred_at AT TIME ZONE 'Asia/Ho_Chi_Minh')::date) FROM product_events WHERE user_id=$1),
		(SELECT count(*) FROM reading_history WHERE user_id=$1),
		(SELECT count(*) FROM user_event_follows WHERE user_id=$1),
		(SELECT count(*) FROM prediction_answers WHERE user_id=$1),
		(SELECT count(*) FROM prediction_answers WHERE user_id=$1 AND is_correct),
		(SELECT COALESCE(sum(points_earned),0) FROM prediction_answers WHERE user_id=$1)`).Scan(&p.ActiveDays, &p.ArticlesRead, &p.EventsFollowed, &p.Predictions, &p.PredictionsGood, &p.Points)
	if err != nil {
		return nil, err
	}
	var days []time.Time
	rows, err := r.db.Pool.Query(ctx, `SELECT DISTINCT (occurred_at AT TIME ZONE 'Asia/Ho_Chi_Minh')::date FROM product_events WHERE user_id=$1 ORDER BY 1 DESC`, userID)
	if err == nil {
		for rows.Next() {
			var d time.Time
			if rows.Scan(&d) == nil {
				days = append(days, d)
			}
		}
		rows.Close()
	}
	p.CurrentStreak = dayStreak(days)
	if p.ArticlesRead >= 10 {
		p.Badges = append(p.Badges, "Độc giả bền bỉ")
	}
	if p.EventsFollowed >= 3 {
		p.Badges = append(p.Badges, "Theo sát đội nhà")
	}
	if p.PredictionsGood >= 5 {
		p.Badges = append(p.Badges, "Dự đoán sắc bén")
	}
	if p.ActiveDays >= 7 {
		p.Badges = append(p.Badges, "Bạn đồng hành 7 ngày")
	}
	if p.Badges == nil {
		p.Badges = []string{}
	}
	return &p, nil
}

func dayStreak(days []time.Time) int {
	return dayStreakAt(days, time.Now())
}

func dayStreakAt(days []time.Time, now time.Time) int {
	if len(days) == 0 {
		return 0
	}
	location, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		location = time.FixedZone("ICT", 7*60*60)
	}
	localNow := now.In(location)
	today := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, time.UTC)
	d := dateOnly(days[0])
	if today.Sub(d) > 24*time.Hour {
		return 0
	}
	streak := 1
	for i := 1; i < len(days); i++ {
		next := dateOnly(days[i])
		if d.Sub(next) != 24*time.Hour {
			break
		}
		streak++
		d = next
	}
	return streak
}

func dateOnly(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func (r *SportsRepo) RecordProductEvent(ctx context.Context, userID int64, clientID, name string, props json.RawMessage) error {
	if !allowedProductEvent(name) {
		return domain.ErrValidation
	}
	if len(props) == 0 {
		props = json.RawMessage(`{}`)
	}
	_, err := r.db.Pool.Exec(ctx, `INSERT INTO product_events(user_id,client_id,event_name,properties) VALUES(NULLIF($1,0),NULLIF($2,''),$3,$4)`, userID, clientID, name, props)
	return err
}

func allowedProductEvent(s string) bool {
	switch strings.TrimSpace(s) {
	case "visit", "onboarding_complete", "content_read", "content_saved", "event_followed", "catch_up_started", "catch_up_completed", "prediction_answered", "dashboard_updated":
		return true
	}
	return false
}
