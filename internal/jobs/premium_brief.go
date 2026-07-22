package jobs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"repwire/internal/domain"
	"repwire/internal/telegram"
)

// handleSendAudioBrief delivers the exact ready-made 6h/20h issue. The
// audio is already generated solely from Vietnamese, editorially-ready items.
func (h *Handlers) handleSendAudioBrief(ctx context.Context, j *domain.Job) error {
	var payload domain.AudioBriefDeliveryPayload
	if err := j.Unmarshal(&payload); err != nil {
		return err
	}
	if payload.UserID == 0 || (payload.Edition != "morning" && payload.Edition != "evening") {
		return fmt.Errorf("audio brief delivery: invalid payload")
	}
	day, err := time.Parse("2006-01-02", payload.Date)
	if err != nil {
		return fmt.Errorf("audio brief delivery: invalid date: %w", err)
	}
	brief, err := h.DB.Engagement.AudioBriefForDate(ctx, day, payload.Edition)
	if errors.Is(err, domain.ErrNotFound) {
		return nil // generation can be delayed; next scheduler pass will retry.
	}
	if err != nil {
		return err
	}
	if brief.AudioURL == nil || *brief.AudioURL == "" {
		return nil
	}
	conn, err := h.DB.Telegram.ConnectionByUser(ctx, payload.UserID)
	if errors.Is(err, domain.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	// Claim before sending: a worker retry cannot duplicate a finished issue.
	claimed, err := h.DB.Engagement.RecordAudioBriefDelivery(ctx, brief.ID, payload.UserID, nil, nil)
	if err != nil || !claimed {
		return err
	}
	caption := brief.Title + " · BaoTheX"
	msgID, err := h.Telegram.SendAudio(ctx, conn.ChatID, *brief.AudioURL, caption)
	if errors.Is(err, telegram.ErrBlocked) {
		_ = h.DB.Telegram.SetBriefDeliveryEnabled(ctx, payload.UserID, false)
		return nil
	}
	if err != nil {
		_ = h.DB.Engagement.ReleaseAudioBriefDelivery(ctx, brief.ID, payload.UserID)
		return err
	}
	return h.DB.Engagement.CompleteAudioBriefDelivery(ctx, brief.ID, payload.UserID, msgID)
}
