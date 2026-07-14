package httpapi

import (
	"net/http"

	"repwire/internal/domain"
	"repwire/internal/postgres"
)

func (s *Server) handleListTopics(w http.ResponseWriter, r *http.Request) {
	topics, err := s.db.Topic.List(r.Context())
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	if topics == nil {
		topics = []domain.Topic{}
	}
	writeJSON(w, http.StatusOK, topics, nil)
}

func (s *Server) handleGetTopic(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	topic, err := s.db.Topic.BySlug(r.Context(), slug)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	featured, _, _ := s.db.Content.List(r.Context(), postgres.ContentFilter{
		TopicSlug: slug, Sort: "top", Limit: 10, OnlyReady: true,
	})
	research, _, _ := s.db.Content.List(r.Context(), postgres.ContentFilter{
		Type: string(domain.ContentResearch), TopicSlug: slug, Limit: 5, OnlyReady: true,
	})
	videos, _, _ := s.db.Content.List(r.Context(), postgres.ContentFilter{
		Type: string(domain.ContentVideo), TopicSlug: slug, Limit: 5, OnlyReady: true,
	})
	related, _ := s.db.Topic.Related(r.Context(), topic.ID)
	if related == nil {
		related = []domain.Topic{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"topic":    topic,
		"featured": nonNilItems(featured),
		"research": nonNilItems(research),
		"videos":   nonNilItems(videos),
		"related":  related,
	}, nil)
}
