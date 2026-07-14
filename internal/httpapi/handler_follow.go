package httpapi

import (
	"net/http"

	"repwire/internal/domain"
)

func (s *Server) handleListFollows(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	follows, err := s.db.Follow.ListFollows(r.Context(), u.ID)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, follows, nil)
}

// ---- topics ----

func (s *Server) handleFollowTopic(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid id")
		return
	}
	if err := s.db.Follow.FollowTopic(r.Context(), u.ID, id); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"following": true}, nil)
}

func (s *Server) handleUnfollowTopic(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid id")
		return
	}
	if err := s.db.Follow.UnfollowTopic(r.Context(), u.ID, id); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"following": false}, nil)
}

func (s *Server) handlePatchTopicFollow(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid id")
		return
	}
	var st domain.FollowSettings
	if !decodeJSON(w, r, &st) {
		return
	}
	if err := s.db.Follow.UpdateTopicFollow(r.Context(), u.ID, id, st); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, st, nil)
}

// ---- entities ----

func (s *Server) handleFollowEntity(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid id")
		return
	}
	if err := s.db.Follow.FollowEntity(r.Context(), u.ID, id); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"following": true}, nil)
}

func (s *Server) handleUnfollowEntity(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid id")
		return
	}
	if err := s.db.Follow.UnfollowEntity(r.Context(), u.ID, id); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"following": false}, nil)
}

func (s *Server) handlePatchEntityFollow(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid id")
		return
	}
	var st domain.FollowSettings
	if !decodeJSON(w, r, &st) {
		return
	}
	if err := s.db.Follow.UpdateEntityFollow(r.Context(), u.ID, id, st); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, st, nil)
}

// ---- sources ----

func (s *Server) handleFollowSource(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid id")
		return
	}
	if err := s.db.Follow.FollowSource(r.Context(), u.ID, id); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"following": true}, nil)
}

func (s *Server) handleUnfollowSource(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid id")
		return
	}
	if err := s.db.Follow.UnfollowSource(r.Context(), u.ID, id); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"following": false}, nil)
}

// ---- mutes ----

func (s *Server) handleMuteTopic(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid id")
		return
	}
	if err := s.db.Follow.MuteTopic(r.Context(), u.ID, id); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"muted": true}, nil)
}

func (s *Server) handleUnmuteTopic(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid id")
		return
	}
	if err := s.db.Follow.UnmuteTopic(r.Context(), u.ID, id); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"muted": false}, nil)
}
