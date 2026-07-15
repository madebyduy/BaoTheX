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
