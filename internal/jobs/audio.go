package jobs

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"repwire/internal/domain"
	"repwire/internal/postgres"
)

func (h *Handlers) handleGenerateAudio(ctx context.Context, j *domain.Job) error {
	var payload domain.BriefPayload
	if err := j.Unmarshal(&payload); err != nil {
		return err
	}
	day := time.Now()
	if payload.Date != "" {
		if parsed, err := time.Parse("2006-01-02", payload.Date); err == nil {
			day = parsed
		}
	}
	if h.TTS == nil || !h.TTS.Enabled() {
		return fmt.Errorf("audio brief: TTS not configured")
	}
	items, _, err := h.DB.Content.List(ctx, postgres.ContentFilter{Sort: "top", Limit: 8, OnlyReady: true})
	if err != nil {
		return err
	}
	if len(items) < 3 {
		return fmt.Errorf("audio brief: not enough ready stories")
	}
	title, script, ids := buildMorningScript(day, items)
	relative := filepath.ToSlash(filepath.Join("audio", day.Format("2006-01-02")+"-the-thao-6h.wav"))
	output := filepath.Join(h.MediaDir, filepath.FromSlash(relative))
	duration, err := h.TTS.Render(ctx, script, output)
	if err != nil {
		return err
	}
	publicURL := strings.TrimRight(h.PublicBaseURL, "/") + "/media/" + relative
	return h.DB.Engagement.SaveAudioBrief(ctx, day, title, script, publicURL, duration, ids)
}

func buildMorningScript(day time.Time, items []domain.ContentItem) (string, string, []int64) {
	title := "Thể thao 6h · " + day.Format("02/01/2006")
	var b strings.Builder
	b.WriteString("Xin chào, đây là Thể thao 6 giờ từ Báo Thể X. Trong khoảng năm phút tới, mời bạn điểm qua những diễn biến đáng chú ý nhất, được tổng hợp từ nhiều nguồn và biên tập bằng tiếng Việt.\n\n")
	ids := make([]int64, 0, len(items))
	for i, item := range items {
		ids = append(ids, item.ID)
		b.WriteString(fmt.Sprintf("Tin thứ %d. %s. ", i+1, item.Title))
		if item.Summary != nil && strings.TrimSpace(*item.Summary) != "" {
			b.WriteString(clipWords(*item.Summary, 85))
		} else if item.Excerpt != nil {
			b.WriteString(clipWords(*item.Excerpt, 85))
		}
		for _, point := range item.KeyPoints {
			if strings.TrimSpace(point) != "" {
				b.WriteString(" " + clipWords(point, 35) + ".")
			}
		}
		if item.SourceName != "" {
			b.WriteString(" Nguồn: " + item.SourceName + ".")
		}
		b.WriteString("\n\n")
	}
	b.WriteString("Bạn vừa nghe Thể thao 6 giờ từ Báo Thể X. Hãy theo dõi đội bóng, giải đấu và vận động viên bạn quan tâm để nhận bản tin phù hợp hơn. Chúc bạn một ngày nhiều năng lượng.")
	return title, b.String(), ids
}

func clipWords(text string, max int) string {
	words := strings.Fields(text)
	if len(words) <= max {
		return strings.Join(words, " ")
	}
	return strings.Join(words[:max], " ") + "…"
}
