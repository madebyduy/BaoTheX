package ingest

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
	"golang.org/x/net/html"

	"repwire/internal/domain"
)

// userAgent identifies our crawler per the legal requirements (spec section 22).
const userAgent = "RepWireBot/1.0 (+https://repwire.app/bot)"

// FetchResult carries fetched items plus updated conditional-GET headers.
type FetchResult struct {
	Items        []RawItem
	ETag         *string
	LastModified *string
	NotModified  bool
}

// RSSFetcher fetches and parses standard RSS/Atom feeds.
type RSSFetcher struct {
	client *http.Client
	parser *gofeed.Parser
}

// NewRSSFetcher constructs an RSSFetcher.
func NewRSSFetcher(client *http.Client) *RSSFetcher {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &RSSFetcher{client: client, parser: gofeed.NewParser()}
}

// Fetch retrieves the feed, honouring ETag / Last-Modified conditional GET.
func (f *RSSFetcher) Fetch(ctx context.Context, src *domain.Source) (*FetchResult, error) {
	feedURL := src.FeedURLOrEmpty()
	if feedURL == "" {
		return &FetchResult{}, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml, text/xml")
	if src.ETag != nil && *src.ETag != "" {
		req.Header.Set("If-None-Match", *src.ETag)
	}
	if src.LastModified != nil && *src.LastModified != "" {
		req.Header.Set("If-Modified-Since", *src.LastModified)
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

	res := &FetchResult{}
	if etag := resp.Header.Get("ETag"); etag != "" {
		res.ETag = &etag
	}
	if lm := resp.Header.Get("Last-Modified"); lm != "" {
		res.LastModified = &lm
	}

	enriched := 0
	for _, it := range feed.Items {
		if len(res.Items) >= 20 {
			break
		}
		if it.Link == "" || it.Title == "" {
			continue
		}
		if it.PublishedParsed != nil && time.Since(*it.PublishedParsed) > 72*time.Hour {
			continue
		}
		body := bodyFrom(it)
		imageURL := imageFrom(it)
		if (len([]rune(body)) < 600 || imageURL == "") && enriched < 12 {
			full, pageImage := f.FetchArticleData(ctx, it.Link)
			if len([]rune(full)) > len([]rune(body)) {
				body = full
			}
			if imageURL == "" {
				imageURL = pageImage
			}
			enriched++
		}
		raw := RawItem{
			Type:      contentTypeForSource(src),
			Title:     strings.TrimSpace(it.Title),
			URL:       it.Link,
			Published: it.PublishedParsed,
			Language:  src.DefaultLang,
		}
		if body != "" {
			raw.Body = ptr(body)
		}
		if excerpt := excerptFrom(it); excerpt != "" {
			raw.Excerpt = ptr(excerpt)
		}
		if imageURL != "" {
			raw.ImageURL = ptr(imageURL)
		}
		if it.Author != nil && it.Author.Name != "" {
			raw.Author = ptr(it.Author.Name)
			raw.Article = &domain.Article{Author: ptr(it.Author.Name)}
		}
		res.Items = append(res.Items, raw)
	}
	return res, nil
}

// fetchArticleBody is a bounded best-effort fallback for feeds that expose only
// an excerpt. It reads semantic article/main content, never scripts/styles,
// and is capped to a small number of items per feed refresh.
// FetchArticleData extracts readable text and the social preview image from an
// article page. It is also used by the periodic media repair pass.
func (f *RSSFetcher) FetchArticleData(ctx context.Context, articleURL string) (string, string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, articleURL, nil)
	if err != nil {
		return "", ""
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := f.client.Do(req)
	if err != nil {
		return "", ""
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", ""
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", ""
	}
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return "", ""
	}
	var candidates []*html.Node
	var imageURL string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "meta" && imageURL == "" {
			var key, content string
			for _, attr := range n.Attr {
				switch strings.ToLower(attr.Key) {
				case "property", "name":
					key = strings.ToLower(attr.Val)
				case "content":
					content = strings.TrimSpace(attr.Val)
				}
			}
			if key == "og:image" || key == "og:image:secure_url" || key == "twitter:image" || key == "twitter:image:src" {
				imageURL = absoluteURL(articleURL, content)
			}
		}
		if n.Type == html.ElementNode && n.Data == "link" && imageURL == "" {
			var rel, href string
			for _, attr := range n.Attr {
				if strings.ToLower(attr.Key) == "rel" {
					rel = strings.ToLower(attr.Val)
				}
				if strings.ToLower(attr.Key) == "href" {
					href = attr.Val
				}
			}
			if rel == "image_src" {
				imageURL = absoluteURL(articleURL, href)
			}
		}
		if n.Type == html.ElementNode && (n.Data == "article" || n.Data == "main") {
			candidates = append(candidates, n)
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)
	for _, candidate := range candidates {
		if text := cleanReadableText(nodeText(candidate)); len([]rune(text)) > 600 {
			return text, imageURL
		}
	}
	return "", imageURL
}

func absoluteURL(baseURL, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	if u.IsAbs() {
		return u.String()
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	return base.ResolveReference(u).String()
}

func nodeText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data + " "
	}
	if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style" || n.Data == "noscript" || n.Data == "nav" || n.Data == "footer") {
		return ""
	}
	var b strings.Builder
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		b.WriteString(nodeText(child))
	}
	return b.String()
}

// contentTypeForSource maps a source kind to the default content type.
func contentTypeForSource(src *domain.Source) domain.ContentType {
	switch src.Kind {
	case domain.SourcePodcastRSS:
		return domain.ContentPodcast
	default:
		return domain.ContentArticle
	}
}

// excerptFrom builds a short plain-text excerpt (<=200 chars) from a feed item.
func excerptFrom(it *gofeed.Item) string {
	text := bodyFrom(it)
	text = stripHTML(text)
	text = strings.Join(strings.Fields(text), " ")
	if len(text) > 200 {
		// Trim on a rune boundary.
		r := []rune(text)
		if len(r) > 200 {
			r = r[:200]
		}
		text = strings.TrimSpace(string(r)) + "…"
	}
	return text
}

func bodyFrom(it *gofeed.Item) string {
	text := it.Content
	if text == "" {
		text = it.Description
	}
	return cleanReadableText(text)
}

func imageFrom(it *gofeed.Item) string {
	if it.Image != nil && it.Image.URL != "" {
		return it.Image.URL
	}
	if len(it.Enclosures) > 0 {
		for _, e := range it.Enclosures {
			if strings.HasPrefix(e.Type, "image/") {
				return e.URL
			}
		}
	}
	if match := imageTagRE.FindStringSubmatch(it.Content + "\n" + it.Description); len(match) == 2 {
		return strings.TrimSpace(match[1])
	}
	return ""
}
