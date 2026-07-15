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
var trailingReadMoreRE = regexp.MustCompile(`(?i)(?:\n\s*)+(?:continue reading|read more|tiếp tục đọc|đọc tiếp)\s*(?:\.{3}|…)?\s*$`)
var consentBlockRE = regexp.MustCompile(`(?is)(?:để hiển thị nội dung này từ youtube|to display this content from youtube).*?(?:thử lại|try again)`)
var readerJunkRE = regexp.MustCompile(`(?i)^(?:chấp nhận|accept|quản lý lựa chọn của tôi|manage my choices|chia sẻ|share|đọc ít lại|read less|thử lại|try again|\d{1,2}:\d{2})$`)

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
	s = consentBlockRE.ReplaceAllString(s, "\n")
	lines := strings.Split(stripHTML(s), "\n")
	clean := make([]string, 0, len(lines))
	skipNext := false
	for _, line := range lines {
		line = strings.Join(strings.Fields(line), " ")
		lowerLine := strings.ToLower(line)
		if skipNext && line != "" {
			skipNext = false
			continue
		}
		if strings.HasPrefix(lowerLine, "đọc thêm") || strings.HasPrefix(lowerLine, "read more") {
			break
		}
		if strings.HasPrefix(lowerLine, "video thực hiện bởi") || strings.HasPrefix(lowerLine, "từ khóa cho bài viết") {
			skipNext = true
			continue
		}
		if line != "" && !readerJunkRE.MatchString(line) &&
			!strings.HasPrefix(lowerLine, "ảnh bìa:") &&
			!strings.HasPrefix(lowerLine, "phát sóng ngày:") {
			clean = append(clean, line)
		}
	}
	text := strings.TrimSpace(strings.Join(clean, "\n\n"))
	text = strings.TrimSpace(trailingReadMoreRE.ReplaceAllString(text, ""))
	if len([]rune(text)) < 24 {
		return ""
	}
	return text
}
