package domain

// Topic is a subject users can follow (goal / discipline / knowledge area).
type Topic struct {
	ID            int64    `json:"id"`
	Slug          string   `json:"slug"`
	Name          string   `json:"name"`
	Description   *string  `json:"description,omitempty"`
	Category      *string  `json:"category,omitempty"`
	Keywords      []string `json:"-"`
	FollowerCount int      `json:"follower_count"`
}

// ContentTopic links a content item to a topic with a classification confidence.
type ContentTopic struct {
	ContentID  int64   `json:"content_id"`
	TopicID    int64   `json:"topic_id"`
	Confidence float64 `json:"confidence"`
	IsPrimary  bool    `json:"is_primary"`
}

// EntityKind enumerates the type of a person or organisation.
type EntityKind string

const (
	EntityAthlete     EntityKind = "athlete"
	EntityCoach       EntityKind = "coach"
	EntityResearcher  EntityKind = "researcher"
	EntityCreator     EntityKind = "creator"
	EntityChannel     EntityKind = "channel"
	EntityPodcast     EntityKind = "podcast"
	EntityCompetition EntityKind = "competition"
	EntityFederation  EntityKind = "federation"
	EntityBrand       EntityKind = "brand"
	EntityPublication EntityKind = "publication"
)

// OfficialLink is one verified link on an entity profile.
type OfficialLink struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

// Entity is a person or organisation (athlete, coach, channel, federation, ...).
type Entity struct {
	ID            int64          `json:"id"`
	Slug          string         `json:"slug"`
	Name          string         `json:"name"`
	Kind          EntityKind     `json:"kind"`
	Bio           *string        `json:"bio,omitempty"`
	AvatarURL     *string        `json:"avatar_url,omitempty"`
	Expertise     []string       `json:"expertise"`
	OfficialLinks []OfficialLink `json:"official_links"`
	Aliases       []string       `json:"-"`
	FollowerCount int            `json:"follower_count"`
}

// ContentEntity links a content item to an entity in a given role.
type ContentEntity struct {
	ContentID int64  `json:"content_id"`
	EntityID  int64  `json:"entity_id"`
	Role      string `json:"role"` // author | mentioned | subject
}
