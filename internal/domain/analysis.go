package domain

import "time"

type AnalysisCandidate struct {
	ID                  int64   `json:"id"`
	ClusterID           int64   `json:"cluster_id"`
	RepresentativeTitle string  `json:"representative_title"`
	Score               float64 `json:"score"`
	SourceCount         int     `json:"source_count"`
	HighQualitySources  int     `json:"high_quality_sources"`
	Velocity24h         int     `json:"velocity_24h"`
	Velocity6h          int     `json:"velocity_6h"`
	HeatScore           float64 `json:"heat_score"`
	ControversyScore    float64 `json:"controversy_score"`
	ActionScore         float64 `json:"action_score"`
	// HeatTerms are the controversy/action words found in the cluster's
	// headlines. They are surfaced so an editor can see why a topic ranked where
	// it did rather than being handed an unexplained number.
	HeatTerms      []string `json:"heat_terms"`
	FollowerWeight int      `json:"follower_weight"`
	// PickedForDate is set on the one cluster chosen as a given day's topic.
	PickedForDate  *time.Time `json:"picked_for_date,omitempty"`
	Status         string     `json:"status"`
	Consensus      []string   `json:"consensus"`
	Conflicts      []string   `json:"conflicts"`
	UniqueClaims   []string   `json:"unique_claims"`
	OpenQuestions  []string   `json:"open_questions"`
	DraftContentID *int64     `json:"draft_content_id,omitempty"`
	LastError      *string    `json:"last_error,omitempty"`
	ProposedAt     time.Time  `json:"proposed_at"`
	SelectedAt     *time.Time `json:"selected_at,omitempty"`
	GeneratedAt    *time.Time `json:"generated_at,omitempty"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// HotTopicCluster is a day's contender, carrying the free structural signals
// plus the headlines needed to score controversy in Go.
type HotTopicCluster struct {
	ClusterID           int64
	RepresentativeTitle string
	Titles              []string
	SourceCount         int
	QualitySources      int
	Velocity6h          int
	Velocity24h         int
	FollowerWeight      int
	// TranslatedMaterials is how many pieces already have a Vietnamese body.
	// The pick prefers a cluster that needs less translation, all else equal.
	TranslatedMaterials int
}

type AnalysisMaterial struct {
	ContentID     int64      `json:"content_id"`
	Title         string     `json:"title"`
	SourceName    string     `json:"source_name"`
	SourceQuality int        `json:"source_quality"`
	PublishedAt   *time.Time `json:"published_at,omitempty"`
	Summary       string     `json:"summary"`
	KeyPoints     []string   `json:"key_points"`
	Body          string     `json:"body"`
}

type AnalysisClaims struct {
	Consensus     []string `json:"consensus"`
	Conflicts     []string `json:"conflicts"`
	UniqueClaims  []string `json:"unique_claims"`
	OpenQuestions []string `json:"open_questions"`
}

type AnalysisDraft struct {
	Title     string   `json:"title"`
	Summary   string   `json:"summary"`
	Body      string   `json:"body"`
	KeyPoints []string `json:"key_points"`
}
