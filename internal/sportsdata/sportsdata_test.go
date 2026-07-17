package sportsdata

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func jsonClient(status int, body string) *http.Client {
	return &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}
}

func TestFootballDataNormalizesDelayedResult(t *testing.T) {
	p := NewFootballData(jsonClient(200, `{"matches":[{"id":7,"utcDate":"2026-07-17T12:00:00Z","status":"FINISHED","competition":{"id":1,"name":"Test League"},"homeTeam":{"name":"A"},"awayTeam":{"name":"B"},"score":{"fullTime":{"home":2,"away":1}}}]}`), "token")
	events, err := p.Fetch(context.Background(), time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC))
	if err != nil || len(events) != 1 {
		t.Fatalf("Fetch() = %d events, %v", len(events), err)
	}
	if events[0].Status != "finished" || events[0].Freshness != "delayed" || *events[0].HomeScore != "2" {
		t.Fatalf("bad normalization: %+v", events[0])
	}
}

func TestProvider429IsRecoverableError(t *testing.T) {
	p := NewFootballData(jsonClient(http.StatusTooManyRequests, `{}`), "token")
	_, err := p.Fetch(context.Background(), time.Now())
	if err == nil || !strings.Contains(err.Error(), "429") {
		t.Fatalf("expected 429 error, got %v", err)
	}
}

func TestTheSportsDBNeverClaimsLive(t *testing.T) {
	p := NewTheSportsDB(jsonClient(200, `{"events":[{"idEvent":"99","strLeague":"Test","strEvent":"A vs B","strHomeTeam":"A","strAwayTeam":"B","dateEvent":"2026-07-17","strTime":"12:00:00","intHomeScore":"3","intAwayScore":"2"}]}`), "123")
	events, err := p.Fetch(context.Background(), time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC))
	if err != nil || len(events) != 2 {
		t.Fatalf("Fetch() = %d events, %v", len(events), err)
	}
	for _, event := range events {
		if event.Freshness == "live" || event.Status == "live" {
			t.Fatalf("free metadata claimed live: %+v", event)
		}
	}
}
