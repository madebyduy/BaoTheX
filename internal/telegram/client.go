// Package telegram implements the bot client, digest building and webhook.
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Client is a minimal Telegram Bot API client with a global rate limiter.
type Client struct {
	token   string
	http    *http.Client
	limiter chan struct{} // token bucket, refilled by a background goroutine
}

// ErrBlocked indicates the bot was blocked by the user (HTTP 403).
var ErrBlocked = fmt.Errorf("telegram: bot blocked by user")

// NewClient constructs a Telegram client. It starts a refill goroutine that
// caps sends at ~25/second (below Telegram's ~30/s global limit).
func NewClient(token string) *Client {
	c := &Client{
		token:   token,
		http:    &http.Client{Timeout: 20 * time.Second},
		limiter: make(chan struct{}, 25),
	}
	for i := 0; i < 25; i++ {
		c.limiter <- struct{}{}
	}
	go c.refill()
	return c
}

// Enabled reports whether a bot token is configured.
func (c *Client) Enabled() bool { return c.token != "" }

func (c *Client) refill() {
	ticker := time.NewTicker(40 * time.Millisecond) // ~25/s
	defer ticker.Stop()
	for range ticker.C {
		select {
		case c.limiter <- struct{}{}:
		default:
		}
	}
}

func (c *Client) wait(ctx context.Context) error {
	select {
	case <-c.limiter:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// InlineButton is one inline keyboard button.
type InlineButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
	URL          string `json:"url,omitempty"`
}

// SendMessage sends a MarkdownV2 message and returns the message id.
func (c *Client) SendMessage(ctx context.Context, chatID int64, text string, buttons [][]InlineButton) (int64, error) {
	if !c.Enabled() {
		return 0, fmt.Errorf("telegram: TELEGRAM_BOT_TOKEN not configured")
	}
	if err := c.wait(ctx); err != nil {
		return 0, err
	}

	payload := map[string]any{
		"chat_id":                  chatID,
		"text":                     text,
		"parse_mode":               "MarkdownV2",
		"disable_web_page_preview": true,
	}
	if len(buttons) > 0 {
		payload["reply_markup"] = map[string]any{"inline_keyboard": buttons}
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL("sendMessage"), bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var out struct {
		OK     bool `json:"ok"`
		Result struct {
			MessageID int64 `json:"message_id"`
		} `json:"result"`
		ErrorCode   int    `json:"error_code"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0, err
	}
	if !out.OK {
		if out.ErrorCode == 403 {
			return 0, ErrBlocked
		}
		return 0, fmt.Errorf("telegram: %s", out.Description)
	}
	return out.Result.MessageID, nil
}

// SetWebhook registers the webhook URL with a secret token.
func (c *Client) SetWebhook(ctx context.Context, url, secret string) error {
	if !c.Enabled() {
		return fmt.Errorf("telegram: TELEGRAM_BOT_TOKEN not configured")
	}
	body, _ := json.Marshal(map[string]any{
		"url":                  url,
		"secret_token":         secret,
		"allowed_updates":      []string{"message", "callback_query"},
		"drop_pending_updates": true,
	})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL("setWebhook"), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("telegram: setWebhook http %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) apiURL(method string) string {
	return "https://api.telegram.org/bot" + c.token + "/" + method
}

// EscapeMarkdownV2 escapes the reserved characters for Telegram MarkdownV2.
func EscapeMarkdownV2(s string) string {
	const special = "_*[]()~`>#+-=|{}.!"
	var b strings.Builder
	for _, r := range s {
		if strings.ContainsRune(special, r) {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}
