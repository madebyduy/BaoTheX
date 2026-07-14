package ingest

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"

	"repwire/internal/domain"
)

// PodcastFetcher parses podcast RSS feeds (RSS + itunes namespace).
type PodcastFetcher struct {
	client *http.Client
	parser *gofeed.Parser
}

// NewPodcastFetcher constructs a PodcastFetcher.
func NewPodcastFetcher(client *http.Client) *PodcastFetcher {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &PodcastFetcher{client: client, parser: gofeed.NewParser()}
}

// Fetch retrieves and parses a podcast feed.
func (f *PodcastFetcher) Fetch(ctx context.Context, src *domain.Source) (*FetchResult, error) {
	feedURL := src.FeedURLOrEmpty()
	if feedURL == "" {
		return &FetchResult{}, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	if src.ETag != nil && *src.ETag != "" {
		req.Header.Set("If-None-Match", *src.ETag)
	}
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotModified {
		return &FetchResult{NotModified: true}, nil
	}
	if resp.StatusCode >= 400 {
		return nil, &httpError{status: resp.StatusCode, url: feedURL}
	}

	feed, err := f.parser.Parse(resp.Body)
	if err != nil {
		return nil, err
	}
	showName := src.Name
	if feed.Title != "" {
		showName = feed.Title
	}

	res := &FetchResult{}
	if etag := resp.Header.Get("ETag"); etag != "" {
		res.ETag = &etag
	}
	for _, it := range feed.Items {
		if it.Title == "" {
			continue
		}
		guid := it.GUID
		if guid == "" {
			guid = it.Link
		}
		if guid == "" {
			continue
		}
		link := it.Link
		if link == "" {
			link = guid
		}
		ep := &domain.PodcastEpisode{
			ShowName:    showName,
			EpisodeGUID: guid,
		}
		if len(it.Enclosures) > 0 {
			ep.AudioURL = ptr(it.Enclosures[0].URL)
		}
		if it.ITunesExt != nil && it.ITunesExt.Duration != "" {
			if secs := parseDurationField(it.ITunesExt.Duration); secs > 0 {
				ep.DurationSec = ptr(secs)
			}
		}
		if it.Description != "" {
			ep.ShowNotes = ptr(stripHTML(it.Description))
		}
		raw := RawItem{
			Type:      domain.ContentPodcast,
			Title:     strings.TrimSpace(it.Title),
			URL:       link,
			Published: it.PublishedParsed,
			Language:  src.DefaultLang,
			Podcast:   ep,
		}
		if notes := cleanReadableText(it.Description); notes != "" {
			raw.Body = ptr(notes)
			raw.Excerpt = ptr(truncate(strings.Join(strings.Fields(notes), " "), 200))
		}
		if img := imageFrom(it); img != "" {
			raw.ImageURL = ptr(img)
		}
		res.Items = append(res.Items, raw)
	}
	return res, nil
}

// parseDurationField parses an itunes:duration that is either seconds ("3600")
// or HH:MM:SS / MM:SS.
func parseDurationField(s string) int {
	s = strings.TrimSpace(s)
	if !strings.Contains(s, ":") {
		n := 0
		for _, r := range s {
			if r < '0' || r > '9' {
				return 0
			}
			n = n*10 + int(r-'0')
		}
		return n
	}
	parts := strings.Split(s, ":")
	total := 0
	for _, p := range parts {
		total *= 60
		v := 0
		for _, r := range strings.TrimSpace(p) {
			if r < '0' || r > '9' {
				continue
			}
			v = v*10 + int(r-'0')
		}
		total += v
	}
	return total
}
