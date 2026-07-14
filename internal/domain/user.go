package domain

import "time"

// Role values for User.Role.
const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

// User is a registered account.
type User struct {
	ID           int64      `json:"id"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	DisplayName  *string    `json:"display_name,omitempty"`
	Role         string     `json:"role"`
	Goals        []string   `json:"goals"`
	Timezone     string     `json:"timezone"`
	OnboardedAt  *time.Time `json:"onboarded_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// IsAdmin reports whether the user has the admin role.
func (u *User) IsAdmin() bool { return u.Role == RoleAdmin }

// Session is a server-side login session; the raw token is never stored.
type Session struct {
	TokenHash string
	UserID    int64
	ExpiresAt time.Time
	CreatedAt time.Time
}

// Collection is a named bucket of saved items.
type Collection struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// SavedItem is one item a user has bookmarked.
type SavedItem struct {
	UserID       int64     `json:"user_id"`
	ContentID    int64     `json:"content_id"`
	CollectionID *int64    `json:"collection_id,omitempty"`
	Note         *string   `json:"note,omitempty"`
	SavedAt      time.Time `json:"saved_at"`
}

// FollowSettings are the shared knobs on topic/entity follows.
type FollowSettings struct {
	InFeed         bool `json:"in_feed"`
	InTelegram     bool `json:"in_telegram"`
	HighlightsOnly bool `json:"highlights_only"`
	Priority       int  `json:"priority"`
}
