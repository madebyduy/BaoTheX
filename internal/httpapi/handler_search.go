package httpapi

import (
	"net/http"
	"strings"

	"repwire/internal/domain"
)

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" || len([]rune(q)) > 100 {
		writeError(w, http.StatusBadRequest, "validation", "Query 'q' is required and must be at most 100 characters")
		return
	}
	if !s.searchLimiter.allow(clientIP(r, s.trustedProxy)) {
		writeError(w, http.StatusTooManyRequests, "rate_limited", "Too many searches, try again shortly")
		return
	}
	ctx := r.Context()

	topics, err := s.db.Search.MatchingTopics(ctx, q, 5)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	entities, err := s.db.Search.MatchingEntities(ctx, q, 5)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	research, err := s.db.Search.SearchByType(ctx, q, typePtr(domain.ContentResearch), 5)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	articles, err := s.db.Search.SearchByType(ctx, q, typePtr(domain.ContentArticle), 5)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	videos, err := s.db.Search.SearchByType(ctx, q, typePtr(domain.ContentVideo), 5)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}

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
	if len([]rune(q)) > 100 {
		writeError(w, http.StatusBadRequest, "validation", "Query must be at most 100 characters")
		return
	}
	if !s.searchLimiter.allow(clientIP(r, s.trustedProxy)) {
		writeError(w, http.StatusTooManyRequests, "rate_limited", "Too many suggestions, try again shortly")
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
