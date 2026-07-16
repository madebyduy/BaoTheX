package domain

import "time"

// TelegramConnection links a user account to a Telegram chat.
type TelegramConnection struct {
	UserID   int64     `json:"user_id"`
	ChatID   int64     `json:"chat_id"`
	Username *string   `json:"username,omitempty"`
	LinkedAt time.Time `json:"linked_at"`
}

// DigestKind enumerates the kinds of Telegram digests.
type DigestKind string

const (
	DigestDaily          DigestKind = "daily"
	DigestWeeklyResearch DigestKind = "weekly_research"
	DigestFollowAlert    DigestKind = "follow_alert"
)

// NotificationPreferences controls what and when a user receives on Telegram.
type NotificationPreferences struct {
	UserID              int64         `json:"user_id"`
	DailyEnabled        bool          `json:"daily_enabled"`
	AudioEnabled        bool          `json:"audio_enabled"`
	EveningBriefEnabled bool          `json:"evening_brief_enabled"`
	DailyHour           int           `json:"daily_hour"`
	DailyDays           []int         `json:"daily_days"`
	DailyMaxItems       int           `json:"daily_max_items"`
	WeeklyResearch      bool          `json:"weekly_research"`
	WeeklyDOW           int           `json:"weekly_dow"`
	FollowAlerts        bool          `json:"follow_alerts"`
	HighlightsOnly      bool          `json:"highlights_only"`
	QuietStart          int           `json:"quiet_start"`
	QuietEnd            int           `json:"quiet_end"`
	ContentTypes        []ContentType `json:"content_types"`
	FeedFollowingOnly   bool          `json:"feed_following_only"`
}

// DefaultNotificationPreferences returns the prefs used for a brand-new user.
func DefaultNotificationPreferences(userID int64) NotificationPreferences {
	return NotificationPreferences{
		UserID:              userID,
		DailyEnabled:        true,
		AudioEnabled:        true,
		EveningBriefEnabled: true,
		DailyHour:           7,
		DailyDays:           []int{1, 2, 3, 4, 5, 6, 7},
		DailyMaxItems:       5,
		WeeklyResearch:      true,
		WeeklyDOW:           1,
		FollowAlerts:        true,
		HighlightsOnly:      false,
		QuietStart:          22,
		QuietEnd:            7,
		ContentTypes:        []ContentType{ContentArticle, ContentResearch, ContentVideo},
		FeedFollowingOnly:   false,
	}
}
