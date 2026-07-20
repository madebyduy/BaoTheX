package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"repwire/internal/domain"
	"repwire/internal/postgres"
)

func (s *Server) handleSports(w http.ResponseWriter, r *http.Request) {
	items, err := s.db.Sports.ListSports(r.Context())
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, items, nil)
}

func (s *Server) handleCompetitions(w http.ResponseWriter, r *http.Request) {
	items, err := s.db.Sports.ListCompetitions(r.Context(), r.URL.Query().Get("sport"))
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, items, nil)
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	status := q.Get("status")
	if status != "" && !validEventStatus(status) {
		writeError(w, http.StatusBadRequest, "validation", "Trạng thái sự kiện không hợp lệ")
		return
	}
	date := q.Get("date")
	if date != "" {
		if _, err := time.Parse("2006-01-02", date); err != nil {
			writeError(w, http.StatusBadRequest, "validation", "Ngày phải có dạng YYYY-MM-DD")
			return
		}
	}
	items, err := s.db.Sports.ListEvents(r.Context(), q.Get("sport"), q.Get("competition"), status, date, atoiDefault(q.Get("limit"), 50))
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, items, nil)
}

// handleCompetition powers the league hub: the competition plus its upcoming
// fixtures and most recent results. Standings are intentionally absent — the
// schema carries no table data — so the page shows fixtures and results only.
func (s *Server) handleCompetition(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	comp, err := s.db.Sports.GetCompetitionBySlug(r.Context(), slug)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	upcoming, _ := s.db.Sports.ListEvents(r.Context(), "", slug, "scheduled", "", 20)
	live, _ := s.db.Sports.ListEvents(r.Context(), "", slug, "live", "", 20)
	results, _ := s.db.Sports.ListEvents(r.Context(), "", slug, "finished", "", 20)
	writeJSON(w, http.StatusOK, map[string]any{
		"competition": comp,
		"live":        live,
		"upcoming":    upcoming,
		"results":     results,
	}, nil)
}

func (s *Server) handleEvent(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, 400, "bad_request", "Invalid id")
		return
	}
	var uid int64
	if u := userFrom(r.Context()); u != nil {
		uid = u.ID
	}
	item, err := s.db.Sports.GetEvent(r.Context(), id, uid)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, 200, item, nil)
}

func (s *Server) handleEventContent(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, 400, "bad_request", "Invalid id")
		return
	}
	items, err := s.db.Sports.EventContent(r.Context(), id, 30)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, 200, nonNilItems(items), nil)
}

func (s *Server) handleEventCalendar(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, 400, "bad_request", "Invalid id")
		return
	}
	item, err := s.db.Sports.GetEvent(r.Context(), id, 0)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	end := item.StartsAt.Add(2 * time.Hour)
	stamp := item.DataUpdatedAt.UTC().Format("20060102T150405Z")
	body := fmt.Sprintf("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//BaoTheX//Lich the thao//VI\r\nBEGIN:VEVENT\r\nUID:event-%d@baothex.vn\r\nDTSTAMP:%s\r\nDTSTART:%s\r\nDTEND:%s\r\nSUMMARY:%s\r\nDESCRIPTION:Nguon: %s. Cap nhat: %s\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n", item.ID, stamp, item.StartsAt.UTC().Format("20060102T150405Z"), end.UTC().Format("20060102T150405Z"), icsEscape(item.Title), icsEscape(item.DataSource), stamp)
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=baothex-event-%d.ics", item.ID))
	_, _ = w.Write([]byte(body))
}

func (s *Server) handleCatchUp(w http.ResponseWriter, r *http.Request) {
	duration := atoiDefault(r.URL.Query().Get("duration"), 3)
	limits := map[int]int{1: 3, 3: 5, 10: 15}
	limit, ok := limits[duration]
	if !ok {
		writeError(w, 400, "validation", "duration phải là 1, 3 hoặc 10")
		return
	}
	var uid int64
	if u := userFrom(r.Context()); u != nil {
		uid = u.ID
	}
	items, err := s.db.Sports.CatchUp(r.Context(), uid, limit)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, 200, map[string]any{"duration": duration, "items": nonNilItems(items)}, nil)
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	if r.Method == http.MethodGet {
		layout, err := s.db.Sports.Dashboard(r.Context(), u.ID)
		if err != nil {
			writeDomainError(w, s.log, err)
			return
		}
		writeJSON(w, 200, map[string]any{"layout": layout}, nil)
		return
	}
	var req struct {
		Layout json.RawMessage `json:"layout"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if !validDashboard(req.Layout) {
		writeError(w, 400, "validation", "Bố cục dashboard không hợp lệ")
		return
	}
	if err := s.db.Sports.SaveDashboard(r.Context(), u.ID, req.Layout); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"saved": true}, nil)
}

func (s *Server) handlePreferenceSync(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	var req struct {
		Layout    json.RawMessage `json:"layout"`
		Goals     []string        `json:"goals"`
		TopicIDs  []int64         `json:"topic_ids"`
		EntityIDs []int64         `json:"entity_ids"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if len(req.TopicIDs) > 100 || len(req.EntityIDs) > 100 || len(req.Goals) > 20 {
		writeError(w, 400, "validation", "Quá nhiều tuỳ chọn trong một lần đồng bộ")
		return
	}
	for _, goal := range req.Goals {
		if strings.TrimSpace(goal) == "" || len([]rune(goal)) > 80 {
			writeError(w, 400, "validation", "Mục tiêu không hợp lệ")
			return
		}
	}
	if !positiveIDs(req.TopicIDs) || !positiveIDs(req.EntityIDs) {
		writeError(w, 400, "validation", "Danh mục theo dõi không hợp lệ")
		return
	}
	if len(req.Layout) > 0 {
		if !validDashboard(req.Layout) {
			writeError(w, 400, "validation", "Bố cục dashboard không hợp lệ")
			return
		}
	}
	if err := s.db.Sports.SyncPreferences(r.Context(), u.ID, req.Layout, req.Goals, req.TopicIDs, req.EntityIDs); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"synced": true}, nil)
}

func (s *Server) handleFollowEvent(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, 400, "bad_request", "Invalid id")
		return
	}
	follow := r.Method == http.MethodPost
	if err := s.db.Sports.FollowEvent(r.Context(), u.ID, id, follow); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"following": follow}, nil)
}

func (s *Server) handleFollowCluster(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, 400, "bad_request", "Invalid id")
		return
	}
	follow := r.Method == http.MethodPost
	if err := s.db.Sports.FollowCluster(r.Context(), u.ID, id, follow); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"following": follow}, nil)
}

func (s *Server) handleReadCluster(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, 400, "bad_request", "Invalid id")
		return
	}
	if err := s.db.Sports.MarkClusterRead(r.Context(), userFrom(r.Context()).ID, id); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handlePredictions(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	items, err := s.db.Sports.ListPredictions(r.Context(), u.ID, 50)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, 200, items, nil)
}

func (s *Server) handlePredictionAnswer(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, 400, "bad_request", "Invalid id")
		return
	}
	var req struct {
		Answer string `json:"answer"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Answer = strings.TrimSpace(req.Answer)
	if req.Answer == "" || len(req.Answer) > 200 {
		writeError(w, 400, "validation", "Câu trả lời không hợp lệ")
		return
	}
	if err := s.db.Sports.AnswerPrediction(r.Context(), u.ID, id, req.Answer); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, 201, map[string]bool{"answered": true}, nil)
}

func (s *Server) handleFanPassport(w http.ResponseWriter, r *http.Request) {
	p, err := s.db.Sports.Passport(r.Context(), userFrom(r.Context()).ID)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, 200, p, nil)
}

func (s *Server) handleProductEvent(w http.ResponseWriter, r *http.Request) {
	if !s.writeLimiter.allow(clientIP(r, s.trustedProxy) + "|product-event") {
		writeError(w, http.StatusTooManyRequests, "rate_limited", "Too many events")
		return
	}
	var req struct {
		ClientID   string          `json:"client_id"`
		EventName  string          `json:"event_name"`
		Properties json.RawMessage `json:"properties"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	req.ClientID = strings.TrimSpace(req.ClientID)
	req.EventName = strings.TrimSpace(req.EventName)
	if req.ClientID != "" && !validAnonymousClientID(req.ClientID) {
		writeError(w, http.StatusBadRequest, "validation", "client_id không hợp lệ")
		return
	}
	if len(req.Properties) > 16<<10 {
		writeError(w, http.StatusBadRequest, "validation", "properties quá lớn")
		return
	}
	var uid int64
	if u := userFrom(r.Context()); u != nil {
		uid = u.ID
	}
	if err := s.db.Sports.RecordProductEvent(r.Context(), uid, req.ClientID, req.EventName, req.Properties); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type eventRequest struct {
	SportSlug     string    `json:"sport_slug"`
	CompetitionID *int64    `json:"competition_id"`
	Title         string    `json:"title"`
	HomeName      *string   `json:"home_name"`
	AwayName      *string   `json:"away_name"`
	StartsAt      time.Time `json:"starts_at"`
	Status        string    `json:"status"`
	HomeScore     *string   `json:"home_score"`
	AwayScore     *string   `json:"away_score"`
	ManualLocked  bool      `json:"manual_locked"`
}

func (s *Server) handleAdminEvents(w http.ResponseWriter, r *http.Request) {
	var req eventRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" || req.SportSlug == "" || req.StartsAt.IsZero() || !validEventStatus(req.Status) {
		writeError(w, 400, "validation", "Môn, tiêu đề, thời gian và trạng thái hợp lệ là bắt buộc")
		return
	}
	item, err := s.db.Sports.SaveManualEvent(r.Context(), postgres.EventInput{SportSlug: req.SportSlug, CompetitionID: req.CompetitionID, Title: req.Title, HomeName: req.HomeName, AwayName: req.AwayName, StartsAt: req.StartsAt, Status: req.Status, HomeScore: req.HomeScore, AwayScore: req.AwayScore, ManualLocked: req.ManualLocked})
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, 201, item, nil)
}
func (s *Server) handleAdminEvent(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, 400, "bad_request", "Invalid id")
		return
	}
	var req eventRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if !validEventStatus(req.Status) {
		writeError(w, 400, "validation", "Trạng thái không hợp lệ")
		return
	}
	item, err := s.db.Sports.SaveManualEvent(r.Context(), postgres.EventInput{ID: id, SportSlug: req.SportSlug, CompetitionID: req.CompetitionID, Title: strings.TrimSpace(req.Title), HomeName: req.HomeName, AwayName: req.AwayName, StartsAt: req.StartsAt, Status: req.Status, HomeScore: req.HomeScore, AwayScore: req.AwayScore, ManualLocked: req.ManualLocked})
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, 200, item, nil)
}

func (s *Server) handleAdminEventResult(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, 400, "bad_request", "Invalid id")
		return
	}
	var req struct {
		Status       string  `json:"status"`
		HomeScore    *string `json:"home_score"`
		AwayScore    *string `json:"away_score"`
		ManualLocked bool    `json:"manual_locked"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if !validEventStatus(req.Status) {
		writeError(w, 400, "validation", "Trạng thái không hợp lệ")
		return
	}
	current, err := s.db.Sports.GetEvent(r.Context(), id, 0)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	item, err := s.db.Sports.SaveManualEvent(r.Context(), postgres.EventInput{ID: id, SportSlug: current.SportSlug, CompetitionID: current.CompetitionID, Title: current.Title, HomeName: current.HomeName, AwayName: current.AwayName, StartsAt: current.StartsAt, Status: req.Status, HomeScore: req.HomeScore, AwayScore: req.AwayScore, ManualLocked: req.ManualLocked})
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, 200, item, nil)
}

func (s *Server) handleAdminPredictions(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	var req domain.Prediction
	if !decodeJSON(w, r, &req) {
		return
	}
	if !validPrediction(&req) {
		writeError(w, 400, "validation", "Dự đoán không hợp lệ")
		return
	}
	item, err := s.db.Sports.SavePrediction(r.Context(), req, u.ID)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, 201, item, nil)
}
func (s *Server) handleAdminPrediction(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, 400, "bad_request", "Invalid id")
		return
	}
	var req domain.Prediction
	if !decodeJSON(w, r, &req) {
		return
	}
	req.ID = id
	if !validPrediction(&req) {
		writeError(w, 400, "validation", "Dự đoán không hợp lệ")
		return
	}
	item, err := s.db.Sports.SavePrediction(r.Context(), req, u.ID)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, 200, item, nil)
}
func (s *Server) handleAdminSettlePrediction(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, 400, "bad_request", "Invalid id")
		return
	}
	var req struct {
		CorrectOption string `json:"correct_option"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.CorrectOption) == "" {
		writeError(w, 400, "validation", "Đáp án đúng là bắt buộc")
		return
	}
	if err := s.db.Sports.SettlePrediction(r.Context(), id, req.CorrectOption); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"settled": true}, nil)
}

func validEventStatus(v string) bool {
	switch v {
	case "scheduled", "live", "finished", "postponed", "cancelled":
		return true
	}
	return false
}
func validDashboard(raw json.RawMessage) bool {
	var ids []string
	if json.Unmarshal(raw, &ids) != nil || len(ids) == 0 || len(ids) > 12 {
		return false
	}
	allowed := map[string]bool{"today": true, "schedule": true, "favorites": true, "following": true, "read_later": true, "listen_later": true, "catch_up": true, "predictions": true}
	seen := map[string]bool{}
	for _, id := range ids {
		if !allowed[id] || seen[id] {
			return false
		}
		seen[id] = true
	}
	return true
}
func validPrediction(p *domain.Prediction) bool {
	if strings.TrimSpace(p.Question) == "" || p.Deadline.IsZero() || p.Points < 0 || p.Points > 1000 {
		return false
	}
	switch p.Kind {
	case "winner", "score", "player", "quiz", "poll":
	default:
		return false
	}
	var options []string
	return json.Unmarshal(p.Options, &options) == nil && len(options) >= 2 && len(options) <= 12
}
func icsEscape(s string) string {
	return strings.NewReplacer("\\", "\\\\", ";", "\\;", ",", "\\,", "\n", "\\n", "\r", "").Replace(s)
}
