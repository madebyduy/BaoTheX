package domain

import "time"

// StoryCluster groups coverage of the same event from independent sources.
type StoryCluster struct {
	ID                  int64         `json:"id"`
	RepresentativeTitle string        `json:"representative_title"`
	PrimaryContentID    *int64        `json:"primary_content_id,omitempty"`
	VerificationStatus  string        `json:"verification_status"`
	SourceCount         int           `json:"source_count"`
	CreatedAt           time.Time     `json:"created_at"`
	UpdatedAt           time.Time     `json:"updated_at"`
	Items               []ContentItem `json:"items"`
}
