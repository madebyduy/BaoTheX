package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
)

// reactionClientID reads the anonymous per-device id from the query string or
// JSON body. It is a random token the browser stores; no login required.
func reactionClientID(r *http.Request) string {
	if q := strings.TrimSpace(r.URL.Query().Get("client_id")); q != "" {
		return q
	}
	var body struct {
		ClientID string `json:"client_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	return strings.TrimSpace(body.ClientID)
}

func (s *Server) handleReactions(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid content id")
		return
	}
	count, liked, err := s.db.Engagement.Reactions(r.Context(), id, r.URL.Query().Get("client_id"))
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"count": count, "liked": liked}, nil)
}

func (s *Server) handleLike(w http.ResponseWriter, r *http.Request)   { s.reactionMutate(w, r, true) }
func (s *Server) handleUnlike(w http.ResponseWriter, r *http.Request) { s.reactionMutate(w, r, false) }

func (s *Server) reactionMutate(w http.ResponseWriter, r *http.Request, like bool) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid content id")
		return
	}
	clientID := reactionClientID(r)
	if clientID == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "client_id là bắt buộc")
		return
	}
	var err error
	if like {
		err = s.db.Engagement.AddReaction(r.Context(), id, clientID)
	} else {
		err = s.db.Engagement.RemoveReaction(r.Context(), id, clientID)
	}
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	count, liked, err := s.db.Engagement.Reactions(r.Context(), id, clientID)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"count": count, "liked": liked}, nil)
}
