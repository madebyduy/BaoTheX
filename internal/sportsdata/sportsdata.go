// Package sportsdata imports only documented free-tier/open sports feeds.
package sportsdata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"repwire/internal/postgres"
)

type Event = postgres.ProviderEventInput

type Provider interface {
	Name() string
	Enabled() bool
	Fetch(context.Context, time.Time) ([]Event, error)
}

type Syncer struct {
	repo      *postgres.SportsRepo
	providers []Provider
	log       *slog.Logger
}

func NewSyncer(repo *postgres.SportsRepo, log *slog.Logger, providers ...Provider) *Syncer {
	return &Syncer{repo: repo, providers: providers, log: log}
}
func (s *Syncer) Enabled() bool {
	for _, p := range s.providers {
		if p.Enabled() {
			return true
		}
	}
	return false
}
func (s *Syncer) Sync(ctx context.Context) error {
	var first error
	for _, p := range s.providers {
		if !p.Enabled() {
			continue
		}
		events, err := p.Fetch(ctx, time.Now().UTC())
		if err != nil {
			if first == nil {
				first = err
			}
			s.log.Warn("sports provider sync failed", "provider", p.Name(), "err", err)
			continue
		}
		stored := 0
		for _, event := range events {
			if err := s.repo.UpsertProviderEvent(ctx, event); err != nil {
				s.log.Warn("sports event cache failed", "provider", p.Name(), "external_id", event.ExternalID, "err", err)
				continue
			}
			stored++
		}
		s.log.Info("sports provider synced", "provider", p.Name(), "events", stored)
	}
	return first
}

type FootballData struct {
	client *http.Client
	token  string
}

func NewFootballData(c *http.Client, token string) Provider {
	return &FootballData{client: c, token: strings.TrimSpace(token)}
}
func (p *FootballData) Name() string  { return "football-data.org" }
func (p *FootballData) Enabled() bool { return p.token != "" }
func (p *FootballData) Fetch(ctx context.Context, now time.Time) ([]Event, error) {
	from := now.Add(-48 * time.Hour).Format("2006-01-02")
	to := now.Add(14 * 24 * time.Hour).Format("2006-01-02")
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.football-data.org/v4/matches?dateFrom="+from+"&dateTo="+to, nil)
	req.Header.Set("X-Auth-Token", p.token)
	var body struct {
		Matches []struct {
			ID          int       `json:"id"`
			UTCDate     time.Time `json:"utcDate"`
			Status      string    `json:"status"`
			Competition struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			} `json:"competition"`
			HomeTeam struct {
				Name string `json:"name"`
			} `json:"homeTeam"`
			AwayTeam struct {
				Name string `json:"name"`
			} `json:"awayTeam"`
			Score struct {
				FullTime struct {
					Home *int `json:"home"`
					Away *int `json:"away"`
				} `json:"fullTime"`
			} `json:"score"`
		} `json:"matches"`
	}
	if err := getJSON(p.client, req, &body); err != nil {
		return nil, err
	}
	out := make([]Event, 0, len(body.Matches))
	for _, m := range body.Matches {
		home, away := m.HomeTeam.Name, m.AwayTeam.Name
		status := footballStatus(m.Status)
		out = append(out, Event{SportSlug: "football", CompetitionKey: "football-data-" + strconv.Itoa(m.Competition.ID), Competition: m.Competition.Name, ExternalID: strconv.Itoa(m.ID), Title: home + " — " + away, HomeName: &home, AwayName: &away, StartsAt: m.UTCDate, Status: status, HomeScore: intString(m.Score.FullTime.Home), AwayScore: intString(m.Score.FullTime.Away), DataSource: p.Name(), Freshness: "delayed", Coverage: json.RawMessage(`{"scores":"delayed","standings":true}`)})
	}
	return out, nil
}

type PandaScore struct {
	client *http.Client
	token  string
}

func NewPandaScore(c *http.Client, token string) Provider {
	return &PandaScore{client: c, token: strings.TrimSpace(token)}
}
func (p *PandaScore) Name() string  { return "pandascore" }
func (p *PandaScore) Enabled() bool { return p.token != "" }
func (p *PandaScore) Fetch(ctx context.Context, now time.Time) ([]Event, error) {
	endpoint := "https://api.pandascore.co/matches/upcoming?per_page=50&sort=begin_at"
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	req.Header.Set("Authorization", "Bearer "+p.token)
	var body []struct {
		ID      int64      `json:"id"`
		Name    string     `json:"name"`
		BeginAt *time.Time `json:"begin_at"`
		League  struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"league"`
		Opponents []struct {
			Opponent struct {
				Name string `json:"name"`
			} `json:"opponent"`
		} `json:"opponents"`
	}
	if err := getJSON(p.client, req, &body); err != nil {
		return nil, err
	}
	out := []Event{}
	for _, m := range body {
		if m.BeginAt == nil {
			continue
		}
		var home, away *string
		if len(m.Opponents) > 0 {
			x := m.Opponents[0].Opponent.Name
			home = &x
		}
		if len(m.Opponents) > 1 {
			x := m.Opponents[1].Opponent.Name
			away = &x
		}
		out = append(out, Event{SportSlug: "esports", CompetitionKey: "pandascore-" + strconv.FormatInt(m.League.ID, 10), Competition: m.League.Name, ExternalID: strconv.FormatInt(m.ID, 10), Title: m.Name, HomeName: home, AwayName: away, StartsAt: *m.BeginAt, Status: "scheduled", DataSource: p.Name(), Freshness: "delayed", Coverage: json.RawMessage(`{"fixtures":true,"live":false}`)})
	}
	return out, nil
}

type OpenF1 struct {
	client  *http.Client
	enabled bool
}

// TheSportsDB uses only the documented v1 day-events endpoint. It is treated
// as delayed metadata coverage, never as a live feed.
type TheSportsDB struct {
	client *http.Client
	key    string
}

func NewTheSportsDB(c *http.Client, key string) Provider {
	return &TheSportsDB{client: c, key: strings.TrimSpace(key)}
}
func (p *TheSportsDB) Name() string  { return "thesportsdb" }
func (p *TheSportsDB) Enabled() bool { return p.key != "" }
func (p *TheSportsDB) Fetch(ctx context.Context, now time.Time) ([]Event, error) {
	type rawEvent struct {
		ID        string  `json:"idEvent"`
		League    string  `json:"strLeague"`
		Title     string  `json:"strEvent"`
		Home      string  `json:"strHomeTeam"`
		Away      string  `json:"strAwayTeam"`
		Date      string  `json:"dateEvent"`
		Time      string  `json:"strTime"`
		HomeScore *string `json:"intHomeScore"`
		AwayScore *string `json:"intAwayScore"`
		Status    string  `json:"strStatus"`
	}
	var out []Event
	for _, spec := range []struct{ providerSport, slug string }{{"Basketball", "basketball"}, {"Tennis", "tennis"}} {
		endpoint := "https://www.thesportsdb.com/api/v1/json/" + url.PathEscape(p.key) + "/eventsday.php?" + query(map[string]string{"d": now.Format("2006-01-02"), "s": spec.providerSport})
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		var body struct {
			Events []rawEvent `json:"events"`
		}
		if err := getJSON(p.client, req, &body); err != nil {
			return nil, err
		}
		for _, raw := range body.Events {
			starts, err := time.Parse("2006-01-02 15:04:05", raw.Date+" "+strings.TrimSuffix(raw.Time, "Z"))
			if err != nil {
				starts, err = time.Parse("2006-01-02", raw.Date)
			}
			if err != nil || raw.ID == "" {
				continue
			}
			starts = starts.UTC()
			status := "scheduled"
			if raw.HomeScore != nil || raw.AwayScore != nil || strings.EqualFold(raw.Status, "Match Finished") {
				status = "finished"
			}
			home, away := raw.Home, raw.Away
			out = append(out, Event{SportSlug: spec.slug, CompetitionKey: "thesportsdb-" + strings.ToLower(strings.ReplaceAll(raw.League, " ", "-")), Competition: raw.League, ExternalID: raw.ID, Title: raw.Title, HomeName: stringPtr(home), AwayName: stringPtr(away), StartsAt: starts, Status: status, HomeScore: raw.HomeScore, AwayScore: raw.AwayScore, DataSource: p.Name(), Freshness: "delayed", Coverage: json.RawMessage(`{"metadata":true,"live":false}`)})
		}
	}
	return out, nil
}

func NewOpenF1(c *http.Client, enabled bool) Provider { return &OpenF1{client: c, enabled: enabled} }
func (p *OpenF1) Name() string                        { return "openf1" }
func (p *OpenF1) Enabled() bool                       { return p.enabled }
func (p *OpenF1) Fetch(ctx context.Context, now time.Time) ([]Event, error) {
	endpoint := "https://api.openf1.org/v1/sessions?year=" + strconv.Itoa(now.Year())
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	var body []struct {
		SessionKey  int       `json:"session_key"`
		SessionName string    `json:"session_name"`
		CountryName string    `json:"country_name"`
		Location    string    `json:"location"`
		DateStart   time.Time `json:"date_start"`
		DateEnd     time.Time `json:"date_end"`
	}
	if err := getJSON(p.client, req, &body); err != nil {
		return nil, err
	}
	out := []Event{}
	for _, m := range body {
		if m.DateEnd.Before(now.Add(-48*time.Hour)) || m.DateStart.After(now.Add(60*24*time.Hour)) {
			continue
		}
		status := "scheduled"
		fresh := "delayed"
		if now.After(m.DateStart) && now.Before(m.DateEnd) {
			status = "live"
			fresh = "live"
		} else if now.After(m.DateEnd) {
			status = "finished"
		}
		title := m.SessionName + " — " + m.Location
		out = append(out, Event{SportSlug: "formula-1", CompetitionKey: "openf1-" + strconv.Itoa(now.Year()), Competition: "Formula 1 " + strconv.Itoa(now.Year()), ExternalID: strconv.Itoa(m.SessionKey), Title: title, StartsAt: m.DateStart, Status: status, DataSource: p.Name(), Freshness: fresh, Coverage: json.RawMessage(`{"sessions":true,"telemetry":false}`)})
	}
	return out, nil
}

func getJSON(client *http.Client, req *http.Request, out any) error {
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("rate limited (429)")
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return fmt.Errorf("provider returned %d", res.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(res.Body, 4<<20)).Decode(out)
}
func footballStatus(v string) string {
	switch v {
	case "IN_PLAY", "PAUSED":
		return "live"
	case "FINISHED", "AWARDED":
		return "finished"
	case "POSTPONED", "SUSPENDED":
		return "postponed"
	case "CANCELLED":
		return "cancelled"
	}
	return "scheduled"
}
func intString(v *int) *string {
	if v == nil {
		return nil
	}
	s := strconv.Itoa(*v)
	return &s
}
func stringPtr(v string) *string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return &v
}
func query(values map[string]string) string {
	v := url.Values{}
	for k, x := range values {
		v.Set(k, x)
	}
	return v.Encode()
}
