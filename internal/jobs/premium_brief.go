package jobs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"repwire/internal/domain"
	"repwire/internal/telegram"
)

// handleSendPremiumBrief delivers the exact ready-made 6h/20h issue. The
// audio is already generated solely from Vietnamese, editorially-ready items.
func (h *Handlers) handleSendPremiumBrief(ctx context.Context, j *domain.Job) error {
	var payload domain.PremiumBriefPayload
	if err := j.Unmarshal(&payload); err != nil {
		return err
	}
	if payload.UserID == 0 || (payload.Edition != "morning" && payload.Edition != "evening") {
		return fmt.Errorf("premium brief: invalid payload")
	}
	day, err := time.Parse("2006-01-02", payload.Date)
	if err != nil {
		return fmt.Errorf("premium brief: invalid date: %w", err)
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
	sub, err := h.DB.Engagement.Subscription(ctx, payload.UserID)
	if err != nil || !sub.Active(time.Now()) {
		return err
	}

	// Claim before sending: a worker retry cannot duplicate a finished issue.
	claimed, err := h.DB.Engagement.RecordAudioBriefDelivery(ctx, brief.ID, payload.UserID, nil, nil)
	if err != nil || !claimed {
		return err
	}
	caption := brief.Title + " · Báo Thể Ích Premium"
	msgID, err := h.Telegram.SendAudio(ctx, conn.ChatID, *brief.AudioURL, caption)
	if errors.Is(err, telegram.ErrBlocked) {
		_ = h.DB.Telegram.SetDailyEnabled(ctx, payload.UserID, false)
		return nil
	}
	if err != nil {
		_ = h.DB.Engagement.ReleaseAudioBriefDelivery(ctx, brief.ID, payload.UserID)
		return err
	}
	return h.DB.Engagement.CompleteAudioBriefDelivery(ctx, brief.ID, payload.UserID, msgID)
}
