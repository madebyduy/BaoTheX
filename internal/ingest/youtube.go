package ingest

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"repwire/internal/domain"
	"repwire/internal/postgres"
)

// YouTubeFetcher polls approved channels via the Data API v3, using the cheap
// playlistItems endpoint (1 quota unit) instead of search.list (100 units).
type YouTubeFetcher struct {
	client *http.Client
	apiKey string
	db     *postgres.DB
}

// NewYouTubeFetcher constructs a YouTubeFetcher. db is used to cache the
// resolved uploads playlist id on the source.
func NewYouTubeFetcher(client *http.Client, apiKey string, db *postgres.DB) *YouTubeFetcher {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &YouTubeFetcher{client: client, apiKey: apiKey, db: db}
}

// Fetch returns the latest uploads for a channel source as RawItems.
func (f *YouTubeFetcher) Fetch(ctx context.Context, src *domain.Source) (*FetchResult, error) {
	if f.apiKey == "" {
		return f.fetchPublicFeed(ctx, src)
	}

	playlistID, err := f.resolveUploadsPlaylist(ctx, src)
	if err != nil {
		return nil, err
	}

	// 1. List recent uploads (id + published) — 1 unit.
	items, err := f.playlistItems(ctx, playlistID)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return &FetchResult{}, nil
	}

	// 2. Hydrate with contentDetails/statistics/snippet — 1 unit.
	videoIDs := make([]string, 0, len(items))
	for _, it := range items {
		videoIDs = append(videoIDs, it.videoID)
	}
	details, err := f.videoDetails(ctx, videoIDs)
	if err != nil {
		return nil, err
	}

	res := &FetchResult{}
	for _, d := range details {
		published := d.published
		raw := RawItem{
			Type:      domain.ContentVideo,
			Title:     d.title,
			URL:       "https://www.youtube.com/watch?v=" + d.id,
			Published: published,
			Language:  src.DefaultLang,
		}
		if d.description != "" {
			raw.Excerpt = ptr(truncate(d.description, 200))
			if body := cleanReadableText(d.description); body != "" {
				raw.Body = ptr(body)
			}
		}
		if d.thumbnail != "" {
			raw.ImageURL = ptr(d.thumbnail)
		}
		raw.Video = &domain.Video{
			YouTubeID:    d.id,
			ChannelID:    d.channelID,
			ChannelTitle: d.channelTitle,
			DurationSec:  ptr(d.durationSec),
			ThumbnailURL: ptr(d.thumbnail),
			Description:  ptr(d.description),
			YTViews:      d.views,
			YTLikes:      d.likes,
		}
		res.Items = append(res.Items, raw)
	}
	return res, nil
}

var youtubeChannelIDRE = regexp.MustCompile(`(?:"channelId"|"externalId")\s*:\s*"(UC[A-Za-z0-9_-]{20,})"`)

// fetchPublicFeed keeps approved YouTube channels working without consuming
// Data API quota. It resolves the channel id from the public channel page and
// then reads YouTube's Atom feed. Statistics and duration are hydrated only
// when an API key is configured, but the playable video and thumbnail remain.
func (f *YouTubeFetcher) fetchPublicFeed(ctx context.Context, src *domain.Source) (*FetchResult, error) {
	channelID, err := f.resolvePublicChannelID(ctx, src)
	if err != nil {
		return nil, err
	}
	feedURL := "https://www.youtube.com/feeds/videos.xml?channel_id=" + url.QueryEscape(channelID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, &httpError{status: resp.StatusCode, url: "youtube/feed"}
	}
	var feed struct {
		Entries []struct {
			VideoID   string `xml:"videoId"`
			ChannelID string `xml:"channelId"`
			Title     string `xml:"title"`
			Published string `xml:"published"`
			Link      struct {
				Href string `xml:"href,attr"`
			} `xml:"link"`
			Media struct {
				Description string `xml:"description"`
				Thumbnail   struct {
					URL string `xml:"url,attr"`
				} `xml:"thumbnail"`
			} `xml:"group"`
		} `xml:"entry"`
	}
	if err := xml.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&feed); err != nil {
		return nil, fmt.Errorf("youtube: decode public feed: %w", err)
	}
	result := &FetchResult{}
	for _, entry := range feed.Entries {
		if entry.VideoID == "" || entry.Title == "" {
			continue
		}
		published, _ := time.Parse(time.RFC3339, entry.Published)
		videoURL := entry.Link.Href
		if videoURL == "" {
			videoURL = "https://www.youtube.com/watch?v=" + entry.VideoID
		}
		raw := RawItem{
			Type: domain.ContentVideo, Title: entry.Title, URL: videoURL,
			Language: src.DefaultLang,
		}
		if !published.IsZero() {
			raw.Published = &published
		}
		if entry.Media.Description != "" {
			raw.Excerpt = ptr(truncate(entry.Media.Description, 200))
		}
		if entry.Media.Thumbnail.URL != "" {
			raw.ImageURL = ptr(entry.Media.Thumbnail.URL)
		}
		raw.Video = &domain.Video{
			YouTubeID: entry.VideoID, ChannelID: channelID, ChannelTitle: src.Name,
			ThumbnailURL: ptr(entry.Media.Thumbnail.URL), Description: ptr(entry.Media.Description),
		}
		result.Items = append(result.Items, raw)
	}
	return result, nil
}

func (f *YouTubeFetcher) resolvePublicChannelID(ctx context.Context, src *domain.Source) (string, error) {
	if src.UploadsPlaylistID != nil && strings.HasPrefix(*src.UploadsPlaylistID, "UU") {
		return "UC" + strings.TrimPrefix(*src.UploadsPlaylistID, "UU"), nil
	}
	pageURL := src.HomepageURLOrEmpty()
	if pageURL == "" {
		ref := src.FeedURLOrEmpty()
		if strings.HasPrefix(ref, "UC") {
			return ref, nil
		}
		pageURL = "https://www.youtube.com/" + strings.TrimPrefix(ref, "/")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := f.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", &httpError{status: resp.StatusCode, url: "youtube/channel"}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 3<<20))
	if err != nil {
		return "", err
	}
	match := youtubeChannelIDRE.FindSubmatch(body)
	if len(match) < 2 {
		return "", fmt.Errorf("youtube: cannot resolve public channel id for %q", pageURL)
	}
	return string(match[1]), nil
}

// resolveUploadsPlaylist returns the channel's uploads playlist id, resolving
// and caching it via channels.list when not already stored on the source.
func (f *YouTubeFetcher) resolveUploadsPlaylist(ctx context.Context, src *domain.Source) (string, error) {
	if src.UploadsPlaylistID != nil && *src.UploadsPlaylistID != "" {
		return *src.UploadsPlaylistID, nil
	}
	ref := src.FeedURLOrEmpty()
	if ref == "" {
		return "", fmt.Errorf("youtube: source %d has no channel reference", src.ID)
	}

	q := url.Values{"part": {"contentDetails"}, "key": {f.apiKey}}
	if strings.HasPrefix(ref, "UC") && !strings.HasPrefix(ref, "@") {
		q.Set("id", ref)
	} else {
		q.Set("forHandle", strings.TrimPrefix(ref, "@"))
	}

	var out struct {
		Items []struct {
			ContentDetails struct {
				RelatedPlaylists struct {
					Uploads string `json:"uploads"`
				} `json:"relatedPlaylists"`
			} `json:"contentDetails"`
		} `json:"items"`
	}
	if err := f.getJSON(ctx, "channels", q, &out); err != nil {
		return "", err
	}
	if len(out.Items) == 0 {
		return "", fmt.Errorf("youtube: channel %q not found", ref)
	}
	playlist := out.Items[0].ContentDetails.RelatedPlaylists.Uploads
	if playlist == "" {
		return "", fmt.Errorf("youtube: no uploads playlist for %q", ref)
	}
	if f.db != nil {
		_ = f.db.Source.SetUploadsPlaylist(ctx, src.ID, playlist)
	}
	return playlist, nil
}

type ytPlaylistItem struct {
	videoID string
}

func (f *YouTubeFetcher) playlistItems(ctx context.Context, playlistID string) ([]ytPlaylistItem, error) {
	q := url.Values{
		"part":       {"contentDetails"},
		"playlistId": {playlistID},
		"maxResults": {"10"},
		"key":        {f.apiKey},
	}
	var out struct {
		Items []struct {
			ContentDetails struct {
				VideoID string `json:"videoId"`
			} `json:"contentDetails"`
		} `json:"items"`
	}
	if err := f.getJSON(ctx, "playlistItems", q, &out); err != nil {
		return nil, err
	}
	items := make([]ytPlaylistItem, 0, len(out.Items))
	for _, it := range out.Items {
		if it.ContentDetails.VideoID != "" {
			items = append(items, ytPlaylistItem{videoID: it.ContentDetails.VideoID})
		}
	}
	return items, nil
}

type ytVideoDetail struct {
	id           string
	title        string
	description  string
	channelID    string
	channelTitle string
	thumbnail    string
	durationSec  int
	published    *time.Time
	views        *int64
	likes        *int64
}

func (f *YouTubeFetcher) videoDetails(ctx context.Context, ids []string) ([]ytVideoDetail, error) {
	q := url.Values{
		"part": {"contentDetails,statistics,snippet"},
		"id":   {strings.Join(ids, ",")},
		"key":  {f.apiKey},
	}
	var out struct {
		Items []struct {
			ID      string `json:"id"`
			Snippet struct {
				Title        string `json:"title"`
				Description  string `json:"description"`
				ChannelID    string `json:"channelId"`
				ChannelTitle string `json:"channelTitle"`
				PublishedAt  string `json:"publishedAt"`
				Thumbnails   map[string]struct {
					URL string `json:"url"`
				} `json:"thumbnails"`
			} `json:"snippet"`
			ContentDetails struct {
				Duration string `json:"duration"`
			} `json:"contentDetails"`
			Statistics struct {
				ViewCount string `json:"viewCount"`
				LikeCount string `json:"likeCount"`
			} `json:"statistics"`
		} `json:"items"`
	}
	if err := f.getJSON(ctx, "videos", q, &out); err != nil {
		return nil, err
	}
	details := make([]ytVideoDetail, 0, len(out.Items))
	for _, it := range out.Items {
		d := ytVideoDetail{
			id:           it.ID,
			title:        it.Snippet.Title,
			description:  it.Snippet.Description,
			channelID:    it.Snippet.ChannelID,
			channelTitle: it.Snippet.ChannelTitle,
			durationSec:  parseISODuration(it.ContentDetails.Duration),
		}
		if t, err := time.Parse(time.RFC3339, it.Snippet.PublishedAt); err == nil {
			d.published = &t
		}
		d.thumbnail = bestThumbnail(it.Snippet.Thumbnails)
		if v, err := strconv.ParseInt(it.Statistics.ViewCount, 10, 64); err == nil {
			d.views = &v
		}
		if v, err := strconv.ParseInt(it.Statistics.LikeCount, 10, 64); err == nil {
			d.likes = &v
		}
		details = append(details, d)
	}
	return details, nil
}

func (f *YouTubeFetcher) getJSON(ctx context.Context, endpoint string, q url.Values, out any) error {
	u := "https://www.googleapis.com/youtube/v3/" + endpoint + "?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := f.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return &httpError{status: resp.StatusCode, url: "youtube/" + endpoint}
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func bestThumbnail(thumbs map[string]struct {
	URL string `json:"url"`
}) string {
	for _, key := range []string{"maxres", "standard", "high", "medium", "default"} {
		if t, ok := thumbs[key]; ok && t.URL != "" {
			return t.URL
		}
	}
	return ""
}

// parseISODuration parses an ISO 8601 duration like "PT1H2M10S" into seconds.
func parseISODuration(d string) int {
	if !strings.HasPrefix(d, "PT") {
		return 0
	}
	d = strings.TrimPrefix(d, "PT")
	var total, cur int
	for _, r := range d {
		switch {
		case r >= '0' && r <= '9':
			cur = cur*10 + int(r-'0')
		case r == 'H':
			total += cur * 3600
			cur = 0
		case r == 'M':
			total += cur * 60
			cur = 0
		case r == 'S':
			total += cur
			cur = 0
		}
	}
	return total
}

func truncate(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return strings.TrimSpace(string(r[:n])) + "…"
}
