package httpapi

import (
	"net/http"

	"repwire/internal/domain"
	"repwire/internal/postgres"
)

func (s *Server) handleListVideos(w http.ResponseWriter, r *http.Request) {
	page, perPage, offset := pagination(r)
	q := r.URL.Query()
	f := postgres.ContentFilter{
		Type:      string(domain.ContentVideo),
		TopicSlug: q.Get("topic"),
		Sort:      q.Get("sort"),
		Limit:     perPage,
		Offset:    offset,
		OnlyReady: true,
	}
	items, total, err := s.db.Content.List(r.Context(), f)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, nonNilItems(items), &Meta{Page: page, PerPage: perPage, Total: total})
}

func (s *Server) handleGetVideo(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid id")
		return
	}
	item, err := s.db.Content.Get(r.Context(), id)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	v, err := s.db.Content.GetVideo(r.Context(), id)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	_ = s.db.Content.IncrementView(r.Context(), id)
	topics, _ := s.db.Topic.ForContent(r.Context(), id)
	entities, _ := s.db.Entity.ForContent(r.Context(), id)
	writeJSON(w, http.StatusOK, map[string]any{
		"item": item, "video": v, "topics": topics, "entities": entities,
	}, nil)
}
