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
	edition := payload.Edition
	if edition != "evening" {
		edition = "morning"
	}
	if h.TTS == nil || !h.TTS.Enabled() {
		return fmt.Errorf("audio brief: TTS not configured")
	}
	candidates, _, err := h.DB.Content.List(ctx, postgres.ContentFilter{
		Type:      string(domain.ContentArticle),
		Sort:      "top",
		Limit:     60,
		OnlyReady: true,
	})
	if err != nil {
		return err
	}
	items := selectMorningStories(candidates, 14)
	if len(items) < 3 {
		return fmt.Errorf("audio brief: not enough ready stories")
	}
	var title, script string
	var ids []int64
	hourLabel := "6h"
	if edition == "evening" {
		title, script, ids = buildEveningScript(day, items)
		hourLabel = "20h"
	} else {
		title, script, ids = buildMorningScript(day, items)
	}
	relative := filepath.ToSlash(filepath.Join("audio", day.Format("2006-01-02")+"-the-thao-"+hourLabel+".wav"))
	output := filepath.Join(h.MediaDir, filepath.FromSlash(relative))
	duration, err := h.TTS.Render(ctx, script, output)
	if err != nil {
		return err
	}
	publicURL := strings.TrimRight(h.PublicBaseURL, "/") + "/media/" + relative
	return h.DB.Engagement.SaveAudioBrief(ctx, day, edition, title, script, publicURL, duration, ids)
}

func buildEveningScript(day time.Time, items []domain.ContentItem) (string, string, []int64) {
	title := "Thể thao 20h · " + day.Format("02/01/2006")
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Xin chào quý vị. Hôm nay là %s, ngày %s. Đây là Thể thao 20 giờ từ Báo Thể Ích, bản tổng kết những diễn biến đáng chú ý nhất trong ngày. Các thông tin đã được tuyển chọn, đối chiếu nhiều nguồn và biên tập bằng tiếng Việt.\n\n", weekdayVI(day.Weekday()), day.Format("02/01/2006")))
	ids := make([]int64, 0, len(items))
	for i, item := range items {
		if i == 5 {
			b.WriteString("Tiếp theo là các diễn biến quốc tế và những góc nhìn đáng chú ý sau một ngày thi đấu.\n\n")
		}
		if i == 10 {
			b.WriteString("Ở phần cuối bản tin là các kết quả, phát biểu và câu chuyện bên lề đáng chú ý.\n\n")
		}
		ids = append(ids, item.ID)
		b.WriteString(fmt.Sprintf("Tin thứ %d, từ %s. %s. ", i+1, sourceForSpeech(item.SourceName), item.Title))
		b.WriteString(clipWords(itemSynopsis(item), 100))
		for _, point := range item.KeyPoints[:min(2, len(item.KeyPoints))] {
			if strings.TrimSpace(point) != "" {
				b.WriteString(" " + clipWords(point, 34) + ".")
			}
		}
		b.WriteString("\n\n")
	}
	b.WriteString("Quý vị vừa nghe Thể thao 20 giờ của ngày " + day.Format("02/01/2006") + " từ Báo Thể Ích. Cảm ơn quý vị đã lắng nghe. Chúc quý vị một buổi tối thư giãn và hẹn gặp lại trong bản tin Thể thao 6 giờ sáng mai.")
	return title, b.String(), ids
}

func buildMorningScript(day time.Time, items []domain.ContentItem) (string, string, []int64) {
	title := "Thể thao 6h · " + day.Format("02/01/2006")
	var b strings.Builder
	b.WriteString(fmt.Sprintf(
		"Xin chào quý vị. Hôm nay là %s, ngày %s. Đây là Thể thao 6 giờ từ Báo Thể Ích. Trong ít phút tới, mời quý vị cùng điểm qua những diễn biến đáng chú ý nhất trong nước và quốc tế, từ bóng đá, bóng rổ, quần vợt đến các môn thể thao khác. Các thông tin được tuyển chọn từ nhiều nguồn uy tín, đối chiếu và biên tập bằng tiếng Việt.\n\n",
		weekdayVI(day.Weekday()), day.Format("02/01/2006"),
	))
	ids := make([]int64, 0, len(items))
	for i, item := range items {
		if i == 5 {
			b.WriteString("Tiếp theo là những diễn biến quốc tế và các câu chuyện đang thu hút sự chú ý của người hâm mộ.\n\n")
		}
		if i == 10 {
			b.WriteString("Ở phần cuối bản tin, mời quý vị đến với các kết quả, video và thông tin bên lề đáng chú ý.\n\n")
		}
		ids = append(ids, item.ID)
		b.WriteString(fmt.Sprintf("Tin thứ %d, từ %s. %s. ", i+1, sourceForSpeech(item.SourceName), item.Title))
		if item.Summary != nil && strings.TrimSpace(*item.Summary) != "" {
			b.WriteString(clipWords(*item.Summary, 95))
		} else if item.Excerpt != nil {
			b.WriteString(clipWords(*item.Excerpt, 95))
		}
		for _, point := range item.KeyPoints[:min(2, len(item.KeyPoints))] {
			if strings.TrimSpace(point) != "" {
				b.WriteString(" " + clipWords(point, 32) + ".")
			}
		}
		b.WriteString("\n\n")
	}
	b.WriteString("Quý vị vừa nghe Thể thao 6 giờ của ngày " + day.Format("02/01/2006") + " từ Báo Thể Ích. Cảm ơn quý vị đã lắng nghe. Hãy theo dõi đội bóng, giải đấu và vận động viên mình quan tâm để nhận bản tin phù hợp hơn. Kính chúc quý vị một ngày nhiều năng lượng và hẹn gặp lại trong bản tin tiếp theo.")
	return title, b.String(), ids
}

// selectMorningStories keeps one event from dominating the edition and gives
// the listener a broader mix of publishers and formats.
func selectMorningStories(candidates []domain.ContentItem, limit int) []domain.ContentItem {
	selected := make([]domain.ContentItem, 0, limit)
	sourceCount := map[string]int{}
	clusters := map[int64]bool{}
	for _, item := range candidates {
		if !readyForVietnameseBrief(item) {
			continue
		}
		if item.StoryClusterID != nil && clusters[*item.StoryClusterID] {
			continue
		}
		source := strings.ToLower(strings.TrimSpace(item.SourceName))
		if sourceCount[source] >= 2 {
			continue
		}
		selected = append(selected, item)
		sourceCount[source]++
		if item.StoryClusterID != nil {
			clusters[*item.StoryClusterID] = true
		}
		if len(selected) == limit {
			break
		}
	}
	return selected
}

// readyForVietnameseBrief is the final safety gate before text reaches TTS.
// Vietnamese publishers can be used directly. Foreign stories must already
// expose both a Vietnamese title and Vietnamese synopsis; untranslated videos,
// podcasts and English excerpts are never allowed into the morning edition.
func readyForVietnameseBrief(item domain.ContentItem) bool {
	if item.Type != domain.ContentArticle {
		return false
	}
	title := strings.TrimSpace(item.Title)
	synopsis := strings.TrimSpace(itemSynopsis(item))
	if title == "" || synopsis == "" {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(item.Language), "vi") {
		return true
	}
	return looksVietnamese(title) && looksVietnamese(synopsis)
}

func itemSynopsis(item domain.ContentItem) string {
	if item.Summary != nil && strings.TrimSpace(*item.Summary) != "" {
		return *item.Summary
	}
	if item.Excerpt != nil {
		return *item.Excerpt
	}
	return ""
}

func sourceForSpeech(source string) string {
	if strings.TrimSpace(source) == "" {
		return "Báo Thể Ích"
	}
	return source
}

func weekdayVI(day time.Weekday) string {
	return [...]string{"Chủ nhật", "thứ Hai", "thứ Ba", "thứ Tư", "thứ Năm", "thứ Sáu", "thứ Bảy"}[day]
}

func clipWords(text string, max int) string {
	words := strings.Fields(text)
	if len(words) <= max {
		return strings.Join(words, " ")
	}
	return strings.Join(words[:max], " ") + "…"
}
