package telegram

import (
	"context"
	"errors"
	"strings"

	"repwire/internal/domain"
	"repwire/internal/postgres"
)

// Update is the subset of a Telegram update we handle.
type Update struct {
	UpdateID int64 `json:"update_id"`
	Message  *struct {
		MessageID int64 `json:"message_id"`
		From      struct {
			ID       int64  `json:"id"`
			Username string `json:"username"`
		} `json:"from"`
		Chat struct {
			ID int64 `json:"id"`
		} `json:"chat"`
		Text string `json:"text"`
	} `json:"message"`
	CallbackQuery *struct {
		ID   string `json:"id"`
		Data string `json:"data"`
		From struct {
			ID int64 `json:"id"`
		} `json:"from"`
	} `json:"callback_query"`
}

// Enqueuer lets the webhook trigger an immediate digest (used by /today).
type Enqueuer interface {
	EnqueueSendDaily(ctx context.Context, userID int64, immediate bool) error
}

// Handler processes incoming Telegram updates.
type Handler struct {
	db      *postgres.DB
	client  *Client
	enqueue Enqueuer
}

// NewHandler constructs a webhook Handler.
func NewHandler(db *postgres.DB, client *Client, enqueue Enqueuer) *Handler {
	return &Handler{db: db, client: client, enqueue: enqueue}
}

// Handle dispatches an update. Errors are logged by the caller; user-facing
// failures are surfaced as bot replies where possible.
func (h *Handler) Handle(ctx context.Context, u *Update) error {
	if u.Message == nil {
		return nil // ignore callback queries for MVP
	}
	chatID := u.Message.Chat.ID
	text := strings.TrimSpace(u.Message.Text)

	switch {
	case strings.HasPrefix(text, "/start"):
		return h.handleStart(ctx, u, text)
	case text == "/pause":
		return h.handlePause(ctx, chatID, true)
	case text == "/resume":
		return h.handlePause(ctx, chatID, false)
	case text == "/today":
		return h.handleToday(ctx, chatID)
	case text == "/settings":
		return h.reply(ctx, chatID, "⚙️ Mở BaoTheX → Cài đặt → Telegram để chỉnh giờ gửi, số lượng và loại nội dung.")
	default:
		return h.reply(ctx, chatID, "Lệnh không nhận diện được. Dùng /today, /pause, /resume, /settings.")
	}
}

func (h *Handler) handleStart(ctx context.Context, u *Update, text string) error {
	chatID := u.Message.Chat.ID
	parts := strings.Fields(text)
	if len(parts) < 2 {
		return h.reply(ctx, chatID, "Chào bạn! Mở BaoTheX → Cài đặt → Telegram và bấm \"Kết nối Telegram\" để nhận mã liên kết.")
	}
	code := parts[1]
	var username *string
	if u.Message.From.Username != "" {
		username = &u.Message.From.Username
	}
	_, err := h.db.Telegram.ConsumeLinkCode(ctx, code, chatID, username)
	if errors.Is(err, domain.ErrNotFound) {
		return h.reply(ctx, chatID, "Mã liên kết không hợp lệ hoặc đã hết hạn. Hãy tạo mã mới trên web.")
	}
	if err != nil {
		return err
	}
	return h.reply(ctx, chatID, "✅ Đã kết nối. Bạn sẽ nhận Daily Brief lúc 7:00 mỗi sáng.")
}

func (h *Handler) handlePause(ctx context.Context, chatID int64, paused bool) error {
	conn, err := h.db.Telegram.ConnectionByChat(ctx, chatID)
	if errors.Is(err, domain.ErrNotFound) {
		return h.reply(ctx, chatID, "Chưa liên kết tài khoản. Dùng /start <mã> để kết nối.")
	}
	if err != nil {
		return err
	}
	if err := h.db.Telegram.SetDailyEnabled(ctx, conn.UserID, !paused); err != nil {
		return err
	}
	if paused {
		return h.reply(ctx, chatID, "⏸ Đã tạm dừng tất cả digest. Dùng /resume để bật lại.")
	}
	return h.reply(ctx, chatID, "▶️ Đã bật lại digest.")
}

func (h *Handler) handleToday(ctx context.Context, chatID int64) error {
	conn, err := h.db.Telegram.ConnectionByChat(ctx, chatID)
	if errors.Is(err, domain.ErrNotFound) {
		return h.reply(ctx, chatID, "Chưa liên kết tài khoản. Dùng /start <mã> để kết nối.")
	}
	if err != nil {
		return err
	}
	if h.enqueue != nil {
		if err := h.enqueue.EnqueueSendDaily(ctx, conn.UserID, true); err != nil {
			return err
		}
	}
	return h.reply(ctx, chatID, "📬 Đang chuẩn bị brief cho bạn…")
}

func (h *Handler) reply(ctx context.Context, chatID int64, text string) error {
	_, err := h.client.SendMessage(ctx, chatID, EscapeMarkdownV2(text), nil)
	return err
}
