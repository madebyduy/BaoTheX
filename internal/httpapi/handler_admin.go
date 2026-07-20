package httpapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"repwire/internal/domain"
	"repwire/internal/ingest"
	"repwire/internal/postgres"
)

// ---- sources ----

func (s *Server) handleListSources(w http.ResponseWriter, r *http.Request) {
	sources, err := s.db.Source.List(r.Context())
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, sources, nil)
}

func (s *Server) handleAdminListSources(w http.ResponseWriter, r *http.Request) {
	sources, err := s.db.Source.List(r.Context())
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, nonNilTopics(sources), nil)
}

type createSourceReq struct {
	Kind          string `json:"kind"`
	Name          string `json:"name"`
	HomepageURL   string `json:"homepage_url"`
	FeedURL       string `json:"feed_url"`
	Quality       int    `json:"quality"`
	FetchInterval string `json:"fetch_interval"` // Go duration string, e.g. "30m"
	DefaultLang   string `json:"default_lang"`
}

func (s *Server) handleAdminCreateSource(w http.ResponseWriter, r *http.Request) {
	var req createSourceReq
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Kind = strings.TrimSpace(req.Kind)
	req.DefaultLang = strings.TrimSpace(strings.ToLower(req.DefaultLang))
	if req.Name == "" || len([]rune(req.Name)) > 160 || !validSourceKind(req.Kind) {
		writeError(w, http.StatusBadRequest, "validation", "kind and name are required")
		return
	}
	if req.DefaultLang != "" && !validLanguageTag(req.DefaultLang) {
		writeError(w, http.StatusBadRequest, "validation", "default_lang is invalid")
		return
	}
	for field, raw := range map[string]string{"homepage_url": req.HomepageURL, "feed_url": req.FeedURL} {
		if raw != "" {
			if err := ingest.ValidatePublicHTTPURL(raw); err != nil {
				writeError(w, http.StatusBadRequest, "validation", field+": "+err.Error())
				return
			}
		}
	}
	interval := 30 * time.Minute
	if req.FetchInterval != "" {
		d, err := time.ParseDuration(req.FetchInterval)
		if err != nil || d < time.Minute || d > 7*24*time.Hour {
			writeError(w, http.StatusBadRequest, "validation", "fetch_interval must be between 1m and 168h")
			return
		}
		interval = d
	}
	quality := req.Quality
	if quality < 1 || quality > 5 {
		quality = 3
	}
	lang := req.DefaultLang
	if lang == "" {
		lang = "en"
	}
	src := &domain.Source{
		Kind:          domain.SourceKind(req.Kind),
		Name:          req.Name,
		Quality:       quality,
		DefaultLang:   lang,
		Enabled:       true,
		FetchInterval: interval,
	}
	if req.HomepageURL != "" {
		src.HomepageURL = &req.HomepageURL
	}
	if req.FeedURL != "" {
		src.FeedURL = &req.FeedURL
	}
	id, err := s.db.Source.Create(r.Context(), src)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	src.ID = id
	writeJSON(w, http.StatusCreated, src, nil)
}

func validSourceKind(value string) bool {
	switch domain.SourceKind(value) {
	case domain.SourceRSS, domain.SourceYouTube, domain.SourceEuropePMC,
		domain.SourcePodcastRSS, domain.SourceSitemap, domain.SourceManual:
		return true
	default:
		return false
	}
}

func validLanguageTag(value string) bool {
	if len(value) < 2 || len(value) > 15 {
		return false
	}
	for _, r := range value {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return false
		}
	}
	return true
}

type updateSourceReq struct {
	Enabled       *bool   `json:"enabled"`
	Quality       *int    `json:"quality"`
	FetchInterval *string `json:"fetch_interval"`
}

func (s *Server) handleAdminUpdateSource(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid id")
		return
	}
	var req updateSourceReq
	if !decodeJSON(w, r, &req) {
		return
	}
	var intervalSec *int64
	if req.FetchInterval != nil {
		d, err := time.ParseDuration(*req.FetchInterval)
		if err != nil || d < time.Minute || d > 7*24*time.Hour {
			writeError(w, http.StatusBadRequest, "validation", "fetch_interval must be between 1m and 168h")
			return
		}
		secs := int64(d.Seconds())
		intervalSec = &secs
	}
	if req.Quality != nil && (*req.Quality < 1 || *req.Quality > 5) {
		writeError(w, http.StatusBadRequest, "validation", "quality must be between 1 and 5")
		return
	}
	if err := s.db.Source.Update(r.Context(), id, req.Enabled, req.Quality, intervalSec); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"updated": true}, nil)
}

func (s *Server) handleAdminFetchSource(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid id")
		return
	}
	src, err := s.db.Source.Get(r.Context(), id)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	kind := fetchJobKind(src.Kind)
	if kind == "" {
		writeError(w, http.StatusBadRequest, "validation", "source kind cannot be fetched")
		return
	}
	if err := s.enqueue.EnqueueFetch(r.Context(), kind, id); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]bool{"queued": true}, nil)
}

// ---- content ----

func (s *Server) handleAdminListContent(w http.ResponseWriter, r *http.Request) {
	page, perPage, offset := pagination(r)
	q := r.URL.Query()
	f := postgres.ContentFilter{
		Type:   q.Get("type"),
		Limit:  perPage,
		Offset: offset,
	}
	// Admin can view any status; needs_review is the common filter.
	status := q.Get("status")
	if q.Get("needs_review") == "true" {
		status = string(domain.StatusNeedsReview)
	}
	var minScore float64
	if raw := q.Get("min_score"); raw != "" {
		if parsed, err := strconv.ParseFloat(raw, 64); err == nil {
			minScore = parsed
		}
	}
	items, total, err := s.adminListContent(r, f, status, minScore)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, nonNilItems(items), &Meta{Page: page, PerPage: perPage, Total: total})
}

// adminListContent lists content by explicit status (not limited to 'ready'),
// ordered by notability and optionally filtered to a minimum score.
func (s *Server) adminListContent(r *http.Request, f postgres.ContentFilter, status string, minScore float64) ([]domain.ContentItem, int, error) {
	return s.db.Content.AdminList(r.Context(), status, f.Type, minScore, f.Limit, f.Offset)
}

type adminUpdateContentReq struct {
	Title          *string  `json:"title"`
	Body           *string  `json:"body"`
	Status         *string  `json:"status"`
	Summary        *string  `json:"summary"`
	KeyPoints      []string `json:"key_points"`
	EditorialBoost *float64 `json:"editorial_boost"`
}

func (s *Server) handleAdminUpdateContent(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid id")
		return
	}
	var req adminUpdateContentReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.db.Content.AdminUpdate(r.Context(), id, req.Title, req.Body, req.Status, req.Summary, req.KeyPoints, req.EditorialBoost); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"updated": true}, nil)
}

func (s *Server) handleAdminGetContent(w http.ResponseWriter, r *http.Request) {
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
	body, _ := s.db.Content.GetBody(r.Context(), id)
	writeJSON(w, http.StatusOK, map[string]any{"item": item, "body": body}, nil)
}

type setTopicsReq struct {
	TopicIDs []int64 `json:"topic_ids"`
}

func (s *Server) handleAdminSetTopics(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid id")
		return
	}
	var req setTopicsReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.db.Topic.SetTopicsByID(r.Context(), id, req.TopicIDs); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"updated": true}, nil)
}

type highlightReq struct {
	Boost float64 `json:"boost"`
}

func (s *Server) handleAdminHighlight(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid id")
		return
	}
	var req highlightReq
	if r.ContentLength > 0 {
		if !decodeJSON(w, r, &req) {
			return
		}
	}
	if req.Boost == 0 {
		req.Boost = 30
	}
	if err := s.db.Content.SetEditorialBoost(r.Context(), id, req.Boost); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"highlighted": true}, nil)
}

func (s *Server) handleAdminHideContent(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid id")
		return
	}
	if err := s.db.Content.SetStatus(r.Context(), id, domain.StatusHidden); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"hidden": true}, nil)
}

func (s *Server) handleAdminUpdateResearch(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid id")
		return
	}
	var bd domain.ResearchBreakdown
	if !decodeJSON(w, r, &bd) {
		return
	}
	// not_proven must not be empty before publishing (spec section 20).
	if bd.NotProven == nil || strings.TrimSpace(*bd.NotProven) == "" {
		writeError(w, http.StatusBadRequest, "validation", "not_proven is required")
		return
	}
	if err := s.db.Content.UpdateResearchBreakdown(r.Context(), id, bd); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"updated": true}, nil)
}

// ---- jobs ----

func (s *Server) handleAdminListJobs(w http.ResponseWriter, r *http.Request) {
	page, perPage, offset := pagination(r)
	status := r.URL.Query().Get("status")
	jobs, err := s.db.Job.List(r.Context(), status, perPage, offset)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, nonNilTopics(jobs), &Meta{Page: page, PerPage: perPage, Total: len(jobs)})
}

func (s *Server) handleAdminRetryJob(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid id")
		return
	}
	if err := s.db.Job.Retry(r.Context(), id); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"retrying": true}, nil)
}

func (s *Server) handleAdminJobStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.db.Job.Stats(r.Context())
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, nonNilTopics(stats), nil)
}

// ---- entities ----

func (s *Server) handleAdminListEntities(w http.ResponseWriter, r *http.Request) {
	entities, err := s.db.Entity.List(r.Context())
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, nonNilTopics(entities), nil)
}

type createEntityReq struct {
	Slug          string                `json:"slug"`
	Name          string                `json:"name"`
	Kind          string                `json:"kind"`
	Bio           *string               `json:"bio"`
	Aliases       []string              `json:"aliases"`
	Expertise     []string              `json:"expertise"`
	OfficialLinks []domain.OfficialLink `json:"official_links"`
}

func (s *Server) handleAdminCreateEntity(w http.ResponseWriter, r *http.Request) {
	var req createEntityReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Slug == "" || req.Name == "" || req.Kind == "" {
		writeError(w, http.StatusBadRequest, "validation", "slug, name and kind are required")
		return
	}
	e := &domain.Entity{
		Slug:          req.Slug,
		Name:          req.Name,
		Kind:          domain.EntityKind(req.Kind),
		Bio:           req.Bio,
		Aliases:       req.Aliases,
		Expertise:     req.Expertise,
		OfficialLinks: req.OfficialLinks,
	}
	out, err := s.db.Entity.Create(r.Context(), e)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusCreated, out, nil)
}

type updateEntityReq struct {
	Aliases []string `json:"aliases"`
}

func (s *Server) handleAdminUpdateEntity(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid id")
		return
	}
	var req updateEntityReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.db.Entity.UpdateAliases(r.Context(), id, req.Aliases); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"updated": true}, nil)
}

// ---- topics ----

func (s *Server) handleAdminListTopics(w http.ResponseWriter, r *http.Request) {
	topics, err := s.db.Topic.List(r.Context())
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, nonNilTopics(topics), nil)
}

type createTopicReq struct {
	Slug     string   `json:"slug"`
	Name     string   `json:"name"`
	Keywords []string `json:"keywords"`
}

func (s *Server) handleAdminCreateTopic(w http.ResponseWriter, r *http.Request) {
	var req createTopicReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Slug == "" || req.Name == "" {
		writeError(w, http.StatusBadRequest, "validation", "slug and name are required")
		return
	}
	t, err := s.db.Topic.Create(r.Context(), req.Slug, req.Name, req.Keywords)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusCreated, t, nil)
}

type updateTopicReq struct {
	Keywords []string `json:"keywords"`
}

func (s *Server) handleAdminUpdateTopic(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid id")
		return
	}
	var req updateTopicReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.db.Topic.UpdateKeywords(r.Context(), id, req.Keywords); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"updated": true}, nil)
}

// fetchJobKind maps a source kind to its fetch job kind (admin fetch-now).
func fetchJobKind(k domain.SourceKind) string {
	switch k {
	case domain.SourceRSS:
		return domain.JobFetchRSS
	case domain.SourceYouTube:
		return domain.JobFetchYouTube
	case domain.SourceEuropePMC:
		return domain.JobFetchPMC
	case domain.SourcePodcastRSS:
		return domain.JobFetchPodcast
	default:
		return ""
	}
}
