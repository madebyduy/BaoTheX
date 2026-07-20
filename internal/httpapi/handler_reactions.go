package httpapi

import (
	"encoding/json"
	"io"
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
	_ = json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&body)
	return strings.TrimSpace(body.ClientID)
}

func (s *Server) handleReactions(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid content id")
		return
	}
	clientID := strings.TrimSpace(r.URL.Query().Get("client_id"))
	if clientID != "" && !validAnonymousClientID(clientID) {
		writeError(w, http.StatusBadRequest, "bad_request", "client_id không hợp lệ")
		return
	}
	count, liked, err := s.db.Engagement.Reactions(r.Context(), id, clientID)
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
	if !validAnonymousClientID(clientID) {
		writeError(w, http.StatusBadRequest, "bad_request", "client_id không hợp lệ")
		return
	}
	if !s.writeLimiter.allow(clientIP(r, s.trustedProxy) + "|reaction") {
		writeError(w, http.StatusTooManyRequests, "rate_limited", "Thao tác quá nhanh")
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

func validAnonymousClientID(value string) bool {
	if len(value) < 8 || len(value) > 64 {
		return false
	}
	for _, r := range value {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' && r != '_' {
			return false
		}
	}
	return true
}
