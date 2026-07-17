package telegram

import (
	"context"
	"errors"
	"net/url"
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
	// baseURL is PUBLIC_BASE_URL, used to point a lost user back at the web app.
	baseURL string
}

// NewHandler constructs a webhook Handler. baseURL is the public address of the
// web app (PUBLIC_BASE_URL); it may be empty or a dev address, in which case
// replies degrade to plain text rather than failing.
func NewHandler(db *postgres.DB, client *Client, enqueue Enqueuer, baseURL string) *Handler {
	return &Handler{db: db, client: client, enqueue: enqueue, baseURL: strings.TrimRight(baseURL, "/")}
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
		// A bare /start is someone who found the bot before the web app. The bot
		// cannot link them — it has no idea which account this chat belongs to
		// until the site issues a code — so the only useful thing it can do is
		// hand them the way back, as one tap rather than a set of directions.
		return h.replyWithButtons(ctx, chatID,
			"Chào bạn! Bot chưa biết đây là tài khoản nào. Hãy mở BaoTheX, đăng nhập rồi bấm \"Kết nối Telegram\" — Telegram sẽ tự mở lại với mã liên kết, bạn không phải gõ gì cả.",
			h.connectButton())
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

// replyWithButtons sends text with an optional inline keyboard. Passing nil
// buttons is the same as reply, which is what makes connectButton safe to call
// unconditionally.
func (h *Handler) replyWithButtons(ctx context.Context, chatID int64, text string, buttons [][]InlineButton) error {
	_, err := h.client.SendMessage(ctx, chatID, EscapeMarkdownV2(text), buttons)
	return err
}

// connectButton returns a one-tap link into the web app's settings page, or nil
// when PUBLIC_BASE_URL cannot legally go in a Telegram button.
//
// Telegram validates inline URL buttons server-side and rejects a host it will
// not resolve — a dev machine's http://localhost:3000 comes back as
// BUTTON_URL_INVALID, and that error fails the whole sendMessage. A user who
// typed /start would then get no reply at all, which is worse than the plain
// directions this button is replacing. So on a dev address the button is
// dropped and the text stands on its own: an imperfect answer that arrives
// beats a better one that errors.
func (h *Handler) connectButton() [][]InlineButton {
	if !buttonURLAllowed(h.baseURL) {
		return nil
	}
	return [][]InlineButton{{{Text: "Mở BaoTheX để kết nối", URL: h.baseURL + "/cai-dat"}}}
}

// buttonURLAllowed reports whether a URL is public enough for Telegram to
// accept it in an inline button.
func buttonURLAllowed(raw string) bool {
	if raw == "" {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if host == "localhost" || host == "127.0.0.1" || host == "::1" || strings.HasSuffix(host, ".local") {
		return false
	}
	// A bare hostname with no dot is not resolvable from Telegram's servers.
	return strings.Contains(host, ".")
}
