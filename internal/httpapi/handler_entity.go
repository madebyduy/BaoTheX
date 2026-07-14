package httpapi

import (
	"net/http"

	"repwire/internal/postgres"
)

func (s *Server) handleGetEntity(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	entity, err := s.db.Entity.BySlug(r.Context(), slug)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	recent, _, _ := s.db.Content.List(r.Context(), postgres.ContentFilter{
		EntitySlug: slug, Sort: "recent", Limit: 10, OnlyReady: true,
	})
	popular, _, _ := s.db.Content.List(r.Context(), postgres.ContentFilter{
		EntitySlug: slug, Sort: "top", Limit: 10, OnlyReady: true,
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"entity":  entity,
		"recent":  nonNilItems(recent),
		"popular": nonNilItems(popular),
	}, nil)
}
