package httpapi

import (
	"net/http"
	"strconv"
)

func (s *Server) handlePublishedAnalyses(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 50 {
			limit = parsed
		}
	}
	items, err := s.db.Analysis.Published(r.Context(), limit)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, nonNilItems(items), nil)
}

func (s *Server) handleAdminAnalysisCandidates(w http.ResponseWriter, r *http.Request) {
	items, err := s.db.Analysis.ListCandidates(r.Context(), r.URL.Query().Get("status"), 30)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, items, nil)
}

func (s *Server) handleAdminGenerateAnalysis(w http.ResponseWriter, r *http.Request) {
	clusterID, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid cluster id")
		return
	}
	if err := s.db.Analysis.MarkDrafting(r.Context(), clusterID); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	if err := s.enqueue.EnqueueGenerateAnalysis(r.Context(), clusterID); err != nil {
		_ = s.db.Analysis.MarkFailed(r.Context(), clusterID, err)
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"queued": true, "cluster_id": clusterID}, nil)
}

func (s *Server) handleAdminQueueContentAnalysis(w http.ResponseWriter, r *http.Request) {
	contentID, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid content id")
		return
	}
	clusterID, err := s.db.Analysis.QueueContentForAnalysis(r.Context(), contentID)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	if err := s.enqueue.EnqueueGenerateAnalysis(r.Context(), clusterID); err != nil {
		_ = s.db.Analysis.MarkFailed(r.Context(), clusterID, err)
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"queued": true, "content_id": contentID, "cluster_id": clusterID}, nil)
}

func (s *Server) handleAdminPublishAnalysis(w http.ResponseWriter, r *http.Request) {
	clusterID, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid cluster id")
		return
	}
	contentID, err := s.db.Analysis.Publish(r.Context(), clusterID)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"published": true, "cluster_id": clusterID, "content_id": contentID}, nil)
}

func (s *Server) handleAdminDismissAnalysis(w http.ResponseWriter, r *http.Request) {
	clusterID, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid cluster id")
		return
	}
	if err := s.db.Analysis.Dismiss(r.Context(), clusterID); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"dismissed": true}, nil)
}
