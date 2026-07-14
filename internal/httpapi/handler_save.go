package httpapi

import (
	"net/http"
	"strconv"
	"strings"
)

type saveReq struct {
	CollectionID *int64  `json:"collection_id"`
	Note         *string `json:"note"`
}

func (s *Server) handleSave(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid id")
		return
	}
	var req saveReq
	// Body is optional for a bare save.
	if r.ContentLength > 0 {
		if !decodeJSON(w, r, &req) {
			return
		}
	}
	if err := s.db.User.Save(r.Context(), u.ID, id, req.CollectionID, req.Note); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"saved": true}, nil)
}

func (s *Server) handleUnsave(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid id")
		return
	}
	if err := s.db.User.Unsave(r.Context(), u.ID, id); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"saved": false}, nil)
}

func (s *Server) handleListSaved(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	page, perPage, offset := pagination(r)
	var collectionID *int64
	if c := strings.TrimSpace(r.URL.Query().Get("collection")); c != "" {
		if id, err := strconv.ParseInt(c, 10, 64); err == nil {
			collectionID = &id
		}
	}
	items, err := s.db.User.ListSaved(r.Context(), u.ID, collectionID, perPage, offset)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, nonNilItems(items), &Meta{Page: page, PerPage: perPage, Total: len(items)})
}

func (s *Server) handleListCollections(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	cols, err := s.db.User.ListCollections(r.Context(), u.ID)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, nonNilTopics(cols), nil)
}

type createCollectionReq struct {
	Name string `json:"name"`
}

func (s *Server) handleCreateCollection(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	var req createCollectionReq
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "validation", "Name is required")
		return
	}
	col, err := s.db.User.CreateCollection(r.Context(), u.ID, req.Name)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusCreated, col, nil)
}

func (s *Server) handleDeleteCollection(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid id")
		return
	}
	if err := s.db.User.DeleteCollection(r.Context(), u.ID, id); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true}, nil)
}

func (s *Server) handleHide(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid id")
		return
	}
	if err := s.db.User.Hide(r.Context(), u.ID, id); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"hidden": true}, nil)
}

func (s *Server) handleMarkRead(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid id")
		return
	}
	if err := s.db.User.MarkRead(r.Context(), u.ID, id); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true}, nil)
}
