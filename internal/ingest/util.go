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
var readerFooterRE = regexp.MustCompile(`(?i)^(?:tags?|copy link|link bài gốc|lấy link|đường dây nóng\s*:|hotline\s*:|gửi báo lỗi|report (?:an )?error)$`)

// boilerplateCutRE marks the start of end-of-article publisher blocks (tag
// lists, "read more" rails, related-article widgets and sponsored/product
// sections). Everything from the first match onward is dropped, since these
// blocks always trail the real body on Vietnamese news sites.
var boilerplateCutRE = regexp.MustCompile(`(?i)^(?:` +
	`tags?\s*[:：]|từ khóa\s*[:：]|` +
	`đọc nhiều\b|đọc thêm\b|xem thêm\s*[:：]|` +
	`tin liên quan\b|bài liên quan\b|các bài liên quan\b|tin tức liên quan\b|` +
	`có thể bạn quan tâm\b|cùng chuyên mục\b|tin cùng chuyên mục\b|` +
	`thông tin doanh nghiệp\b|tin tài trợ\b|nội dung tài trợ\b|quảng cáo\b|` +
	`theo dõi (?:chúng tôi|dân trí|báo)\b|bình luận\s*\(\d+\)` +
	`)`)
var readerPublisherJunkRE = regexp.MustCompile(`(?i)^(?:báo lỗi cho\b.*|\*?vui lòng nhập đủ thông tin.*|đóng|vff\s*\||soha|\d{1,2}/\d{1,2}/\d{4}\s+\d{1,2}:\d{2})$`)

// publisherWidgetMarkers are controls appended after the article body by
// some publishers. They are checked only after the first few lines so a
// header button such as "Trở lại chủ đề" is not mistaken for article text.
var publisherWidgetMarkers = []string{
	"trở lại chủ đề", "tặng sao cho bài viết", "tặng sao thành công",
	"chuyển sao tặng cho thành viên", "hoặc nhập số sao", "chủ đề:",
	"dòng sự kiện:", "tuổi trẻ online newsletters", "thông tin tài khoản",
	"đăng ký ngay để nhận gói tin", "xem thêm",
}

var publisherWidgetASCII = []string{
	"tags:", "\u0111\u1ecdc nhi\u1ec1u trong", "\u0111\u1ecdc th\u00eam",
	"\u0111\u1ecdc nhi\u1ec1u", "th\u00f4ng tin doanh nghi\u1ec7p", "tr\u1edf l\u1ea1i ch\u1ee7 \u0111\u1ec1",
	"t\u1eb7ng sao", "ch\u1ee7 \u0111\u1ec1:", "d\u00f2ng s\u1ef1 ki\u1ec7n:",
	"tu\u1ed5i tr\u1ebb online newsletters", "th\u00f4ng tin t\u00e0i kho\u1ea3n",
	"\u0111\u0103ng k\u00fd ngay \u0111\u1ec3 nh\u1eadn g\u00f3i tin", "xem th\u00eam",
}

var accessWallUnavailableMarkers = []string{
	"blog này hiện không khả dụng",
	"blog này không khả dụng",
	"this blog is currently unavailable",
	"this live blog is currently unavailable",
	"sorry, this blog is currently unavailable",
}

var accessWallConsentMarkers = []string{
	"bật cookie",
	"cho phép cookie một lần",
	"đồng ý với cookie",
	"xác nhận cài đặt cookie",
	"tùy chọn quyền riêng tư",
	"không thể xác minh xem bạn đã đồng ý",
	"enable cookies",
	"allow cookies once",
	"consented to cookies",
	"cookie settings",
	"privacy options",
	"unable to verify if you have consented",
	"we need your permission to use cookies",
}

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
	plain := stripHTML(s)
	if BlockedArticleText(plain) {
		return ""
	}
	lines := strings.Split(plain, "\n")
	clean := make([]string, 0, len(lines))
	skipNext := false
	for _, line := range lines {
		line = strings.Join(strings.Fields(line), " ")
		lowerLine := strings.ToLower(line)
		if readerFooterRE.MatchString(line) || boilerplateCutRE.MatchString(line) {
			break
		}
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
		if line != "" && !readerJunkRE.MatchString(line) && !readerPublisherJunkRE.MatchString(line) &&
			!strings.HasPrefix(lowerLine, "ảnh bìa:") &&
			!strings.HasPrefix(lowerLine, "phát sóng ngày:") {
			clean = append(clean, line)
		}
	}
	text := strings.TrimSpace(strings.Join(clean, "\n\n"))
	text = strings.TrimSpace(trailingReadMoreRE.ReplaceAllString(text, ""))
	text = trimPublisherWidgets(text)
	if len([]rune(text)) < 24 {
		return ""
	}
	return text
}

// TrimTrailingBoilerplate cuts a stored article body at the first end-of-article
// publisher block (tag list, "đọc nhiều", related/sponsored widgets). It is used
// to re-clean bodies ingested before the boilerplate filter was tightened, and
// is safe/idempotent: once the marker line is removed it no longer matches.
func TrimTrailingBoilerplate(text string) string {
	if strings.TrimSpace(text) == "" {
		return text
	}
	lines := strings.Split(text, "\n")
	cut := -1
	for i, line := range lines {
		l := strings.TrimSpace(line)
		if l == "" {
			continue
		}
		if boilerplateCutRE.MatchString(l) || readerFooterRE.MatchString(l) {
			cut = i
			break
		}
	}
	if cut < 0 {
		return trimPublisherWidgets(text)
	}
	return trimPublisherWidgets(strings.TrimSpace(strings.Join(lines[:cut], "\n")))
}

func trimPublisherWidgets(text string) string {
	trimmed := strings.TrimSpace(text)
	lower := strings.ToLower(trimmed)
	markers := append(append([]string{}, publisherWidgetMarkers...), publisherWidgetASCII...)
	cut := -1
	for _, marker := range markers {
		if len(lower) <= 400 {
			break
		}
		pos := strings.Index(lower[400:], marker)
		if pos >= 0 {
			pos += 400
			if cut < 0 || pos < cut {
				cut = pos
			}
		}
	}
	if cut >= 0 {
		return strings.TrimSpace(trimmed[:cut])
	}
	return trimmed
}

// BlockedArticleText detects consent/interstitial copy that can be mistaken
// for a full article. We do not attempt to bypass the publisher's access
// control: the item remains non-public until a real body is available.
func BlockedArticleText(s string) bool {
	plain := strings.ToLower(strings.Join(strings.Fields(stripHTML(s)), " "))
	if plain == "" {
		return false
	}
	unavailable := false
	for _, marker := range accessWallUnavailableMarkers {
		if strings.Contains(plain, marker) {
			unavailable = true
			break
		}
	}
	consentHits := 0
	for _, marker := range accessWallConsentMarkers {
		if strings.Contains(plain, marker) {
			consentHits++
		}
	}
	if unavailable && consentHits >= 2 {
		return true
	}
	return consentHits >= 4 && len(strings.Fields(plain)) < 500
}
