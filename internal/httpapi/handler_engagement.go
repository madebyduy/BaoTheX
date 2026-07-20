package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"repwire/internal/domain"
)

func (s *Server) handlePublicCapabilities(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"telegram_enabled":      s.tgClient != nil && s.tgClient.Enabled(),
		"telegram_bot":          s.cfg.TelegramBotUsername,
		"web_push_enabled":      s.cfg.WebPushPublicKey != "" && s.cfg.WebPushPrivateKey != "",
		"web_push_public_key":   s.cfg.WebPushPublicKey,
		"premium_monthly_price": s.cfg.PremiumMonthlyPrice,
		"sepay_enabled":         s.cfg.SePayMerchant != "" && s.cfg.SePaySecretKey != "",
	}, nil)
}

func (s *Server) handlePremiumStatus(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	sub, err := s.db.Engagement.Subscription(r.Context(), u.ID)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"subscription":  sub,
		"active":        sub.Active(time.Now()),
		"monthly_price": s.cfg.PremiumMonthlyPrice,
	}, nil)
}

func (s *Server) handlePremiumCheckout(w http.ResponseWriter, r *http.Request) {
	if s.cfg.SePayMerchant == "" || s.cfg.SePaySecretKey == "" {
		writeError(w, http.StatusServiceUnavailable, "sepay_not_configured", "SePay is not configured")
		return
	}
	u := userFrom(r.Context())
	invoice := fmt.Sprintf("BTX%d%d", u.ID, time.Now().Unix())
	amount := int64(s.cfg.PremiumMonthlyPrice)
	if _, err := s.db.Engagement.CreatePaymentOrder(r.Context(), u.ID, invoice, amount); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	base := strings.TrimRight(s.cfg.PublicBaseURL, "/")
	fields := [][2]string{
		{"order_amount", strconv.FormatInt(amount, 10)},
		{"merchant", s.cfg.SePayMerchant},
		{"currency", "VND"},
		{"operation", "PURCHASE"},
		{"order_description", "BaoTheX Premium 30 ngay"},
		{"order_invoice_number", invoice},
		{"customer_id", strconv.FormatInt(u.ID, 10)},
		{"success_url", base + "/premium?payment=success"},
		{"error_url", base + "/premium?payment=error"},
		{"cancel_url", base + "/premium?payment=cancel"},
	}
	pairs := make([]string, 0, len(fields))
	responseFields := make(map[string]string, len(fields)+1)
	for _, field := range fields {
		pairs = append(pairs, field[0]+"="+field[1])
		responseFields[field[0]] = field[1]
	}
	mac := hmac.New(sha256.New, []byte(s.cfg.SePaySecretKey))
	_, _ = mac.Write([]byte(strings.Join(pairs, ",")))
	responseFields["signature"] = base64.StdEncoding.EncodeToString(mac.Sum(nil))
	writeJSON(w, http.StatusOK, map[string]any{
		"action":  strings.TrimRight(s.cfg.SePayBaseURL, "/") + "/v1/checkout/init",
		"method":  "POST",
		"fields":  responseFields,
		"invoice": invoice,
	}, nil)
}

type sepayIPN struct {
	NotificationType string `json:"notification_type"`
	Order            struct {
		ID            string `json:"id"`
		Status        string `json:"order_status"`
		Amount        string `json:"order_amount"`
		InvoiceNumber string `json:"order_invoice_number"`
	} `json:"order"`
	Transaction struct {
		ID     string `json:"transaction_id"`
		Status string `json:"transaction_status"`
		Amount string `json:"transaction_amount"`
	} `json:"transaction"`
}

func (s *Server) handleSePayIPN(w http.ResponseWriter, r *http.Request) {
	provided := r.Header.Get("X-Secret-Key")
	expected := s.cfg.SePayIPNSecretKey
	if expected == "" || len(provided) != len(expected) || subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
		writeError(w, http.StatusUnauthorized, "unauthorized", "invalid SePay secret")
		return
	}
	var payload sepayIPN
	if !decodeJSON(w, r, &payload) {
		return
	}
	if payload.NotificationType != "ORDER_PAID" || payload.Order.Status != "CAPTURED" || payload.Transaction.Status != "APPROVED" {
		writeJSON(w, http.StatusOK, map[string]bool{"acknowledged": true}, nil)
		return
	}
	amountText := payload.Transaction.Amount
	if amountText == "" {
		amountText = payload.Order.Amount
	}
	amount, err := parseVND(amountText)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_amount", "invalid payment amount")
		return
	}
	_, newlyPaid, err := s.db.Engagement.MarkPaymentPaid(r.Context(), payload.Order.InvoiceNumber, payload.Transaction.ID, payload.Order.ID, amount)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"acknowledged": true, "activated": newlyPaid}, nil)
}

// parseVND parses the provider's integer currency amount without float
// rounding/truncation. Providers sometimes serialise VND as "39000.00"; only a
// zero fractional part is valid because the currency has no minor unit here.
func parseVND(raw string) (int64, error) {
	whole, fraction, hasFraction := strings.Cut(strings.TrimSpace(raw), ".")
	if whole == "" {
		return 0, fmt.Errorf("empty amount")
	}
	for _, r := range whole {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("non-decimal amount")
		}
	}
	if hasFraction {
		if fraction == "" {
			return 0, fmt.Errorf("empty fraction")
		}
		for _, r := range fraction {
			if r != '0' {
				return 0, fmt.Errorf("fractional VND")
			}
		}
	}
	amount, err := strconv.ParseInt(whole, 10, 64)
	if err != nil || amount <= 0 {
		return 0, fmt.Errorf("invalid amount")
	}
	return amount, nil
}

func (s *Server) handleLatestAudioBrief(w http.ResponseWriter, r *http.Request) {
	edition := r.URL.Query().Get("edition")
	if edition != "" && edition != "morning" && edition != "evening" {
		writeError(w, http.StatusBadRequest, "validation", "edition must be morning or evening")
		return
	}
	brief, err := s.db.Engagement.LatestAudioBrief(r.Context(), edition)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	if s.isMissingLocalAudio(brief.AudioURL) {
		// A database row can outlive a local media volume after a cleanup or a
		// machine restart. Requeue it here so readers never get a convincing
		// looking player attached to a 404 audio file.
		if err := s.enqueue.EnqueueGenerateAudio(r.Context(), brief.BriefDate, brief.Edition); err != nil {
			s.log.Warn("requeue missing audio brief failed", "brief", brief.ID, "err", err)
		}
		writeError(w, http.StatusServiceUnavailable, "audio_processing", "Audio brief is being regenerated")
		return
	}
	writeJSON(w, http.StatusOK, brief, nil)
}

// isMissingLocalAudio only verifies files owned by this deployment. External
// URLs are intentionally left alone; the audio renderer itself writes /media/.
func (s *Server) isMissingLocalAudio(audioURL *string) bool {
	if audioURL == nil || strings.TrimSpace(*audioURL) == "" {
		return true
	}
	u, err := url.Parse(*audioURL)
	if err != nil || !strings.HasPrefix(u.Path, "/media/") {
		return false
	}
	rel := strings.TrimPrefix(u.Path, "/media/")
	clean := filepath.Clean(filepath.FromSlash(rel))
	if clean == "." || strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
		return true
	}
	info, err := os.Stat(filepath.Join(s.cfg.MediaStorageDir, clean))
	return err != nil || info.IsDir() || info.Size() == 0
}

func (s *Server) handlePushSubscribe(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	var input struct {
		Endpoint string `json:"endpoint"`
		Keys     struct {
			P256DH string `json:"p256dh"`
			Auth   string `json:"auth"`
		} `json:"keys"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if input.Endpoint == "" || input.Keys.P256DH == "" || input.Keys.Auth == "" {
		writeError(w, http.StatusBadRequest, "validation", "endpoint and push keys are required")
		return
	}
	if _, err := url.ParseRequestURI(input.Endpoint); err != nil {
		writeError(w, http.StatusBadRequest, "bad_endpoint", "invalid push endpoint")
		return
	}
	err := s.db.Engagement.UpsertPushSubscription(r.Context(), u.ID, domain.PushSubscription{
		Endpoint: input.Endpoint, P256DH: input.Keys.P256DH, Auth: input.Keys.Auth,
		UserAgent: r.UserAgent(),
	})
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"subscribed": true}, nil)
}

func (s *Server) handlePushUnsubscribe(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	var input struct {
		Endpoint string `json:"endpoint"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if err := s.db.Engagement.DeletePushSubscription(r.Context(), u.ID, input.Endpoint); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"unsubscribed": true}, nil)
}

func (s *Server) handlePushTest(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	if s.pushClient == nil || !s.pushClient.Enabled() {
		writeError(w, http.StatusServiceUnavailable, "push_not_configured", "Web Push is not configured")
		return
	}
	subs, err := s.db.Engagement.PushSubscriptions(r.Context(), u.ID)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	if len(subs) == 0 {
		writeError(w, http.StatusBadRequest, "not_subscribed", "No push subscription")
		return
	}
	sent := 0
	for _, sub := range subs {
		if err := s.pushClient.Send(r.Context(), sub, "BaoTheX đã sẵn sàng", "Bạn sẽ nhận tin quan trọng từ các đội và chủ đề đang theo dõi.", "/"); err == nil {
			sent++
		}
	}
	writeJSON(w, http.StatusOK, map[string]int{"sent": sent}, nil)
}
