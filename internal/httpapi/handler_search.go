package httpapi

import (
	"net/http"
	"strings"

	"repwire/internal/domain"
)

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeError(w, http.StatusBadRequest, "validation", "Query 'q' is required")
		return
	}
	ctx := r.Context()

	topics, _ := s.db.Search.MatchingTopics(ctx, q, 5)
	entities, _ := s.db.Search.MatchingEntities(ctx, q, 5)

	research, _ := s.db.Search.SearchByType(ctx, q, typePtr(domain.ContentResearch), 5)
	articles, _ := s.db.Search.SearchByType(ctx, q, typePtr(domain.ContentArticle), 5)
	videos, _ := s.db.Search.SearchByType(ctx, q, typePtr(domain.ContentVideo), 5)

	writeJSON(w, http.StatusOK, map[string]any{
		"topics":   nonNilTopics(topics),
		"entities": nonNilEntities(entities),
		"research": nonNilItems(research),
		"articles": nonNilItems(articles),
		"videos":   nonNilItems(videos),
	}, nil)
}

func (s *Server) handleSuggest(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeJSON(w, http.StatusOK, []any{}, nil)
		return
	}
	suggestions, err := s.db.Search.Suggest(r.Context(), q, 8)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	if suggestions == nil {
		writeJSON(w, http.StatusOK, []any{}, nil)
		return
	}
	writeJSON(w, http.StatusOK, suggestions, nil)
}

func typePtr(t domain.ContentType) *domain.ContentType { return &t }
