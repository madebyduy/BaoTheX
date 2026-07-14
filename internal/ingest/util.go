package ingest

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

type httpError struct {
	status int
	url    string
}

func (e *httpError) Error() string { return fmt.Sprintf("http %d fetching %s", e.status, e.url) }

var tagRE = regexp.MustCompile(`<[^>]*>`)
var blockTagRE = regexp.MustCompile(`(?is)<(?:script|style|noscript|svg)[^>]*>.*?</(?:script|style|noscript|svg)>`)
var commentRE = regexp.MustCompile(`(?s)<!--.*?-->`)
var imageTagRE = regexp.MustCompile(`(?i)<img[^>]+(?:src|data-src)=["']([^"']+)["']`)
var junkMarkers = []string{".custom-article-container", "--color-foreground", "var(--color-", "sourcemappingurl", "font-family:", "color-text:", "webpack", "__next_f.push", "application/ld+json"}

func stripHTML(s string) string {
	s = blockTagRE.ReplaceAllString(s, "\n")
	s = commentRE.ReplaceAllString(s, "\n")
	s = html.UnescapeString(s)
	s = tagRE.ReplaceAllString(s, "\n")
	return strings.TrimSpace(s)
}

// cleanReadableText filters CSS/JSON/code accidentally exposed by RSS feeds.
func cleanReadableText(s string) string {
	if strings.TrimSpace(s) == "" {
		return ""
	}
	lower := strings.ToLower(s)
	for _, marker := range junkMarkers {
		if strings.Contains(lower, marker) {
			return ""
		}
	}
	lines := strings.Split(stripHTML(s), "\n")
	clean := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.Join(strings.Fields(line), " ")
		if line != "" {
			clean = append(clean, line)
		}
	}
	text := strings.TrimSpace(strings.Join(clean, "\n\n"))
	if len([]rune(text)) < 24 {
		return ""
	}
	return text
}
