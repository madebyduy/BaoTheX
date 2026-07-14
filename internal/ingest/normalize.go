// Package ingest fetches raw items from sources and normalises them.
package ingest

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"slices"
	"strings"
	"time"

	"repwire/internal/domain"
)

// RawItem is the source-agnostic shape produced by every fetcher before it is
// normalised and inserted.
type RawItem struct {
	Type      domain.ContentType
	Title     string
	URL       string
	Excerpt   *string
	Body      *string
	ImageURL  *string
	Author    *string
	Published *time.Time
	Language  string

	// Subtype-specific payloads. Only the one matching Type is populated.
	Research *domain.ResearchPaper
	Video    *domain.Video
	Podcast  *domain.PodcastEpisode
	Article  *domain.Article
}

// trackingParams are stripped from URLs during normalisation.
var trackingParams = []string{"fbclid", "gclid", "ref", "source", "mc_cid", "mc_eid", "igshid", "spm"}

// Normalize returns a canonical form of a URL: https scheme, lowercased host
// without www, no fragment, tracking params removed, no trailing slash.
func Normalize(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", err
	}
	if u.Scheme == "http" || u.Scheme == "" {
		u.Scheme = "https"
	}
	u.Host = strings.ToLower(strings.TrimPrefix(u.Host, "www."))
	u.Fragment = ""

	q := u.Query()
	for k := range q {
		lk := strings.ToLower(k)
		if strings.HasPrefix(lk, "utm_") || slices.Contains(trackingParams, lk) {
			q.Del(k)
		}
	}
	u.RawQuery = q.Encode()
	u.Path = strings.TrimSuffix(u.Path, "/")
	return u.String(), nil
}

// URLHash returns the sha256 hex of a canonical URL.
func URLHash(canonical string) string {
	sum := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(sum[:])
}

// NormalizeTitle produces a lowercased, whitespace-collapsed title used as a
// soft-dedup signal.
func NormalizeTitle(title string) string {
	return strings.Join(strings.Fields(strings.ToLower(title)), " ")
}

func ptr[T any](v T) *T { return &v }
