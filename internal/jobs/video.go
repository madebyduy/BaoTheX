package jobs

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"repwire/internal/briefmedia"
	"repwire/internal/domain"
	"repwire/internal/postgres"
)

func (h *Handlers) handleGenerateVideo(ctx context.Context, j *domain.Job) error {
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
	if h.TTS == nil || !h.TTS.Enabled() || h.VideoRenderer == nil || !h.VideoRenderer.Enabled() {
		return fmt.Errorf("video brief: TTS or FFmpeg not configured")
	}
	candidates, _, err := h.DB.Content.List(ctx, postgres.ContentFilter{Sort: "top", Limit: 24, OnlyReady: true})
	if err != nil {
		return err
	}
	items := make([]domain.ContentItem, 0, 5)
	for _, item := range candidates {
		if item.Language == "vi" || looksVietnamese(item.Title) {
			items = append(items, item)
		}
		if len(items) == 5 {
			break
		}
	}
	if len(items) < 3 {
		return fmt.Errorf("video brief: not enough ready stories")
	}
	title, script, stories, ids := buildVideoScript(day, items)
	baseName := day.Format("2006-01-02") + "-tin-nhanh"
	relativeAudio := filepath.ToSlash(filepath.Join("video", baseName+".wav"))
	relativeVideo := filepath.ToSlash(filepath.Join("video", baseName+".mp4"))
	relativeThumb := filepath.ToSlash(filepath.Join("video", baseName+".jpg"))
	audioPath := filepath.Join(h.MediaDir, filepath.FromSlash(relativeAudio))
	videoPath := filepath.Join(h.MediaDir, filepath.FromSlash(relativeVideo))
	thumbPath := filepath.Join(h.MediaDir, filepath.FromSlash(relativeThumb))
	duration, err := h.TTS.Render(ctx, script, audioPath)
	if err != nil {
		return err
	}
	if err := h.VideoRenderer.Render(ctx, title, stories, audioPath, videoPath, thumbPath, duration); err != nil {
		return err
	}
	base := strings.TrimRight(h.PublicBaseURL, "/") + "/media/"
	return h.DB.Engagement.SaveVideoBrief(ctx, day, title, script, base+relativeVideo, base+relativeThumb, duration, ids)
}

func buildVideoScript(day time.Time, items []domain.ContentItem) (string, string, []briefmedia.VideoStory, []int64) {
	title := "Tin nhanh thể thao · " + day.Format("02/01/2006")
	stories := make([]briefmedia.VideoStory, 0, len(items))
	ids := make([]int64, 0, len(items))
	var script strings.Builder
	script.WriteString("Đây là tin nhanh thể thao hôm nay từ Báo Thể Ích. ")
	for i, item := range items {
		ids = append(ids, item.ID)
		stories = append(stories, briefmedia.VideoStory{Title: cleanVideoText(item.Title), Source: cleanVideoText(item.SourceName)})
		script.WriteString(fmt.Sprintf("Tin %d. %s. ", i+1, item.Title))
		if item.Summary != nil && strings.TrimSpace(*item.Summary) != "" {
			script.WriteString(clipWords(*item.Summary, 38))
		} else if item.Excerpt != nil {
			script.WriteString(clipWords(*item.Excerpt, 38))
		}
		script.WriteString(". ")
	}
	script.WriteString("Theo dõi Báo Thể Ích để đọc nhiều góc nhìn và nhận bản tin theo đội bóng bạn quan tâm.")
	return title, script.String(), stories, ids
}

func looksVietnamese(text string) bool {
	return strings.ContainsAny(strings.ToLower(text), "ăâđêôơưáàảãạấầẩẫậắằẳẵặéèẻẽẹếềểễệíìỉĩịóòỏõọốồổỗộớờởỡợúùủũụứừửữựýỳỷỹỵ")
}

func cleanVideoText(text string) string {
	text = strings.ReplaceAll(text, "|", "·")
	return strings.Map(func(r rune) rune {
		if unicode.Is(unicode.So, r) || unicode.IsControl(r) {
			return -1
		}
		return r
	}, text)
}
