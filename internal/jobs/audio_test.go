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

func TestBuildEveningScriptHasFixedAppointmentAndClosing(t *testing.T) {
	summary := strings.Repeat("Thông tin thể thao đã được đối chiếu trong ngày. ", 8)
	items := []domain.ContentItem{
		{ID: 1, Title: "Tin thứ nhất", SourceName: "Nguồn A", Summary: &summary},
		{ID: 2, Title: "Tin thứ hai", SourceName: "Nguồn B", Summary: &summary},
		{ID: 3, Title: "Tin thứ ba", SourceName: "Nguồn C", Summary: &summary},
	}
	day := time.Date(2026, 7, 15, 20, 0, 0, 0, time.Local)
	title, script, ids := buildEveningScript(day, items)
	for _, expected := range []string{"Thể thao 20h", "Xin chào quý vị", "bản tổng kết", "Báo Thể Ích", "Cảm ơn quý vị đã lắng nghe", "6 giờ sáng mai"} {
		if !strings.Contains(title+script, expected) {
			t.Fatalf("evening edition missing %q", expected)
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

func TestAudioScriptDoesNotCutOrRepeatStory(t *testing.T) {
	summary := "Câu thứ nhất cung cấp bối cảnh đầy đủ cho người nghe. " +
		"Câu thứ hai giải thích diễn biến chính của trận đấu. " +
		strings.Repeat("Phần cuối chưa hoàn tất ", 90)
	repeatedPoint := "Câu thứ hai giải thích diễn biến chính của trận đấu."
	items := []domain.ContentItem{{
		ID: 1, Title: "Một tin thể thao", SourceName: "Nguồn A", Summary: &summary,
		KeyPoints: []string{repeatedPoint},
	}}

	_, script, _ := buildMorningScript(time.Date(2026, 7, 22, 6, 0, 0, 0, time.Local), items)
	if strings.Contains(script, "…") {
		t.Fatalf("audio script still contains an abrupt ellipsis: %s", script)
	}
	if got := strings.Count(script, repeatedPoint); got != 1 {
		t.Fatalf("duplicate key point was narrated %d times", got)
	}
	if strings.Contains(script, "Phần cuối chưa hoàn tất") {
		t.Fatalf("incomplete trailing sentence should have been omitted: %s", script)
	}
}

func TestClipSpeechPrefersCompleteSentence(t *testing.T) {
	text := strings.Repeat("Một câu hoàn chỉnh có nội dung rõ ràng. ", 8) +
		strings.Repeat("Đoạn dở dang ", 30)
	got := clipSpeech(text, 55)
	if !strings.HasSuffix(got, ".") || strings.Contains(got, "Đoạn dở dang") {
		t.Fatalf("clipSpeech cut a spoken sentence incorrectly: %q", got)
	}
}
