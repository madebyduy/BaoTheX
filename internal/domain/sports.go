package domain

import (
	"encoding/json"
	"time"
)

type Sport struct {
	ID      int64  `json:"id"`
	Slug    string `json:"slug"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

type Competition struct {
	ID         int64           `json:"id"`
	SportID    int64           `json:"sport_id"`
	SportSlug  string          `json:"sport_slug"`
	Slug       string          `json:"slug"`
	Name       string          `json:"name"`
	Country    *string         `json:"country,omitempty"`
	DataSource string          `json:"data_source"`
	ExternalID *string         `json:"external_id,omitempty"`
	Coverage   json.RawMessage `json:"coverage"`
}

type SportsEvent struct {
	ID             int64           `json:"id"`
	SportID        int64           `json:"sport_id"`
	SportSlug      string          `json:"sport_slug"`
	SportName      string          `json:"sport_name"`
	CompetitionID  *int64          `json:"competition_id,omitempty"`
	Competition    *string         `json:"competition,omitempty"`
	Title          string          `json:"title"`
	HomeName       *string         `json:"home_name,omitempty"`
	AwayName       *string         `json:"away_name,omitempty"`
	StartsAt       time.Time       `json:"starts_at"`
	Status         string          `json:"status"`
	HomeScore      *string         `json:"home_score,omitempty"`
	AwayScore      *string         `json:"away_score,omitempty"`
	ResultDetails  json.RawMessage `json:"result_details"`
	DataSource     string          `json:"data_source"`
	ExternalID     *string         `json:"external_id,omitempty"`
	DataUpdatedAt  time.Time       `json:"data_updated_at"`
	Freshness      string          `json:"freshness"`
	IsManual       bool            `json:"is_manual"`
	ManualLocked   bool            `json:"manual_locked"`
	Coverage       json.RawMessage `json:"coverage"`
	Following      bool            `json:"following,omitempty"`
	RelatedContent []ContentItem   `json:"related_content,omitempty"`
}

type Prediction struct {
	ID            int64           `json:"id"`
	EventID       *int64          `json:"event_id,omitempty"`
	Kind          string          `json:"kind"`
	Question      string          `json:"question"`
	Options       json.RawMessage `json:"options"`
	CorrectOption *string         `json:"correct_option,omitempty"`
	Deadline      time.Time       `json:"deadline"`
	Status        string          `json:"status"`
	Points        int             `json:"points"`
	UserAnswer    *string         `json:"user_answer,omitempty"`
	IsCorrect     *bool           `json:"is_correct,omitempty"`
	AnswerCount   int             `json:"answer_count"`
}

type FanPassport struct {
	ActiveDays      int      `json:"active_days"`
	CurrentStreak   int      `json:"current_streak"`
	ArticlesRead    int      `json:"articles_read"`
	EventsFollowed  int      `json:"events_followed"`
	Predictions     int      `json:"predictions"`
	PredictionsGood int      `json:"predictions_correct"`
	Points          int      `json:"points"`
	Badges          []string `json:"badges"`
}
