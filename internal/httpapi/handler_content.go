package httpapi

import (
	"net/http"
	"strconv"

	"repwire/internal/postgres"
)

func (s *Server) handleListEntities(w http.ResponseWriter, r *http.Request) {
	entities, err := s.db.Entity.List(r.Context())
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, entities, nil)
}

func (s *Server) handleListContent(w http.ResponseWriter, r *http.Request) {
	page, perPage, offset := pagination(r)
	q := r.URL.Query()
	f := postgres.ContentFilter{
		Type:       q.Get("type"),
		TopicSlug:  q.Get("topic"),
		EntitySlug: q.Get("entity"),
		Language:   q.Get("lang"),
		HasSummary: boolPtr(q.Get("has_summary")),
		OpenAccess: boolPtr(q.Get("open_access")),
		Sort:       q.Get("sort"),
		Limit:      perPage,
		Offset:     offset,
		OnlyReady:  true,
	}
	if sid := q.Get("source"); sid != "" {
		if id, err := strconv.ParseInt(sid, 10, 64); err == nil {
			f.SourceID = &id
		}
	}
	items, total, err := s.db.Content.List(r.Context(), f)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, nonNilItems(items), &Meta{Page: page, PerPage: perPage, Total: total})
}

func (s *Server) handleGetContent(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid id")
		return
	}
	item, err := s.db.Content.GetPublic(r.Context(), id)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	// Fire-and-forget view increment; ignore errors.
	_ = s.db.Content.IncrementView(r.Context(), id)

	topics, _ := s.db.Topic.ForContent(r.Context(), id)
	entities, _ := s.db.Entity.ForContent(r.Context(), id)

	out := map[string]any{
		"item":     item,
		"topics":   topics,
		"entities": entities,
	}
	if body, err := s.db.Content.GetBody(r.Context(), id); err == nil {
		out["body"] = body
	}
	// Attach subtype detail.
	switch item.Type {
	case "research":
		if rp, err := s.db.Content.GetResearch(r.Context(), id); err == nil {
			out["research"] = rp
		}
	case "video":
		if v, err := s.db.Content.GetVideo(r.Context(), id); err == nil {
			out["video"] = v
		}
	case "article", "announcement", "event":
		if a, err := s.db.Content.GetArticle(r.Context(), id); err == nil {
			out["article"] = a
		}
	}
	writeJSON(w, http.StatusOK, out, nil)
}

func (s *Server) handleTranslate(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid id")
		return
	}
	if _, err := s.db.Content.Get(r.Context(), id); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	if _, err := s.db.Content.GetBody(r.Context(), id); err != nil {
		writeError(w, http.StatusConflict, "body_unavailable", "Bài này chưa có nội dung để dịch")
		return
	}
	if err := s.db.Content.MarkTranslationPending(r.Context(), id); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	if err := s.enqueue.EnqueueTranslate(r.Context(), id); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "processing"}, nil)
}

func (s *Server) handleRelated(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid id")
		return
	}
	items, err := s.db.Content.Related(r.Context(), id, 8)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, nonNilItems(items), nil)
}

func (s *Server) handleGetStoryCluster(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid cluster id")
		return
	}
	cluster, err := s.db.Content.GetStoryCluster(r.Context(), id)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, cluster, nil)
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	var userID int64
	if u := userFrom(r.Context()); u != nil {
		userID = u.ID
	}
	home, err := s.homepage.Build(r.Context(), userID)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, home, nil)
}
