package jobs

import (
	"strings"
	"testing"
	"time"

	"repwire/internal/domain"
)

func TestBuildMorningScriptHasDatedGreetingAndClosing(t *testing.T) {
	summary := strings.Repeat("Thông tin đáng chú ý trong ngày. ", 8)
	items := []domain.ContentItem{
		{ID: 1, Title: "Tin thứ nhất", SourceName: "Nguồn A", Summary: &summary},
		{ID: 2, Title: "Tin thứ hai", SourceName: "Nguồn B", Summary: &summary},
		{ID: 3, Title: "Tin thứ ba", SourceName: "Nguồn C", Summary: &summary},
	}
	day := time.Date(2026, 7, 15, 6, 0, 0, 0, time.Local)
	_, script, ids := buildMorningScript(day, items)
	for _, expected := range []string{"Xin chào quý vị", "ngày 15/07/2026", "Tin thứ 3", "Cảm ơn quý vị đã lắng nghe", "hẹn gặp lại"} {
		if !strings.Contains(script, expected) {
			t.Fatalf("script missing %q: %s", expected, script)
		}
	}
	if len(ids) != 3 {
		t.Fatalf("expected 3 content ids, got %d", len(ids))
	}
}

func TestSelectMorningStoriesOnlyAllowsVietnameseReadyArticles(t *testing.T) {
	viSummary := "Đội tuyển Việt Nam đã hoàn tất buổi tập và sẵn sàng cho trận đấu quan trọng sắp tới."
	translatedSummary := "Câu lạc bộ đã xác nhận thương vụ sau khi hai bên hoàn tất kiểm tra y tế trong ngày hôm nay."
	englishSummary := "The club confirmed the transfer after the player completed his medical examination."
	candidates := []domain.ContentItem{
		{ID: 1, Type: domain.ContentVideo, Language: "en", Title: "Full game highlights", Summary: &englishSummary},
		{ID: 2, Type: domain.ContentArticle, Language: "en", Title: "Premier League transfer confirmed", Summary: &englishSummary},
		{ID: 3, Type: domain.ContentArticle, Language: "en", Title: "Thương vụ tại Ngoại hạng Anh đã được xác nhận", Summary: &translatedSummary},
		{ID: 4, Type: domain.ContentArticle, Language: "vi", Title: "Đội tuyển Việt Nam sẵn sàng ra sân", Summary: &viSummary},
	}

	selected := selectMorningStories(candidates, 10)
	if len(selected) != 2 {
		t.Fatalf("expected 2 Vietnamese-ready articles, got %d: %#v", len(selected), selected)
	}
	if selected[0].ID != 3 || selected[1].ID != 4 {
		t.Fatalf("unexpected selected ids: %d, %d", selected[0].ID, selected[1].ID)
	}
}
