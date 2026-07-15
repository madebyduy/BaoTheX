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

// SendAudio sends a remotely hosted audio brief to a Telegram chat.
func (c *Client) SendAudio(ctx context.Context, chatID int64, audioURL, caption string) (int64, error) {
	if !c.Enabled() {
		return 0, fmt.Errorf("telegram: TELEGRAM_BOT_TOKEN not configured")
	}
	if err := c.wait(ctx); err != nil {
		return 0, err
	}
	payload := map[string]any{"chat_id": chatID, "audio": audioURL, "caption": caption}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL("sendAudio"), bytes.NewReader(body))
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

// DeleteWebhook switches the bot back to getUpdates mode. It is used by local
// development where Telegram cannot call a localhost webhook.
func (c *Client) DeleteWebhook(ctx context.Context) error {
	body, _ := json.Marshal(map[string]bool{"drop_pending_updates": false})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL("deleteWebhook"), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("telegram: deleteWebhook http %d", resp.StatusCode)
	}
	return nil
}

// Poll receives bot updates without requiring a public HTTPS URL. Production
// should normally use the webhook route instead.
func (c *Client) Poll(ctx context.Context, handler *Handler, onError func(error)) {
	if !c.Enabled() || handler == nil {
		return
	}
	if err := c.DeleteWebhook(ctx); err != nil && onError != nil {
		onError(err)
	}
	var offset int64
	for ctx.Err() == nil {
		updates, err := c.getUpdates(ctx, offset)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			if onError != nil {
				onError(err)
			}
			time.Sleep(time.Second)
			continue
		}
		for i := range updates {
			if updates[i].UpdateID >= offset {
				offset = updates[i].UpdateID + 1
			}
			if err := handler.Handle(ctx, &updates[i]); err != nil && onError != nil {
				onError(err)
			}
		}
	}
}

func (c *Client) getUpdates(ctx context.Context, offset int64) ([]Update, error) {
	body, _ := json.Marshal(map[string]any{
		"offset": offset, "timeout": 15,
		"allowed_updates": []string{"message", "callback_query"},
	})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL("getUpdates"), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out struct {
		OK          bool     `json:"ok"`
		Result      []Update `json:"result"`
		Description string   `json:"description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if !out.OK {
		return nil, fmt.Errorf("telegram: %s", out.Description)
	}
	return out.Result, nil
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
