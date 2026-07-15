package domain

import "time"

type Subscription struct {
	UserID           int64      `json:"user_id"`
	Plan             string     `json:"plan"`
	Status           string     `json:"status"`
	Provider         string     `json:"provider"`
	CurrentPeriodEnd *time.Time `json:"current_period_end,omitempty"`
}

func (s Subscription) Active(now time.Time) bool {
	return s.Status == "active" && s.CurrentPeriodEnd != nil && s.CurrentPeriodEnd.After(now)
}

type PaymentOrder struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"user_id"`
	InvoiceNumber string    `json:"invoice_number"`
	AmountVND     int64     `json:"amount_vnd"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

type AudioBrief struct {
	ID              int64     `json:"id"`
	BriefDate       time.Time `json:"brief_date"`
	Title           string    `json:"title"`
	Script          string    `json:"script,omitempty"`
	AudioURL        *string   `json:"audio_url,omitempty"`
	DurationSeconds *int      `json:"duration_seconds,omitempty"`
	ContentIDs      []int64   `json:"content_ids"`
	Status          string    `json:"status"`
	Error           *string   `json:"error,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type VideoBrief struct {
	ID              int64     `json:"id"`
	BriefDate       time.Time `json:"brief_date"`
	Title           string    `json:"title"`
	Script          string    `json:"script,omitempty"`
	VideoURL        *string   `json:"video_url,omitempty"`
	ThumbnailURL    *string   `json:"thumbnail_url,omitempty"`
	DurationSeconds *int      `json:"duration_seconds,omitempty"`
	ContentIDs      []int64   `json:"content_ids"`
	YouTubeVideoID  *string   `json:"youtube_video_id,omitempty"`
	Status          string    `json:"status"`
	Error           *string   `json:"error,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type PushSubscription struct {
	Endpoint  string `json:"endpoint"`
	P256DH    string `json:"p256dh"`
	Auth      string `json:"auth"`
	UserAgent string `json:"-"`
}
