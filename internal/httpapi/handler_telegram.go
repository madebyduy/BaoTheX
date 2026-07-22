package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"repwire/internal/auth"
	"repwire/internal/domain"
	"repwire/internal/telegram"
)

func (s *Server) handleTelegramStatus(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	configured := s.tgClient != nil && s.tgClient.Enabled() && s.cfg.TelegramBotUsername != ""
	conn, err := s.db.Telegram.ConnectionByUser(r.Context(), u.ID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		writeDomainError(w, s.log, err)
		return
	}
	data := map[string]any{
		"configured":   configured,
		"linked":       conn != nil,
		"bot_username": s.cfg.TelegramBotUsername,
	}
	if conn != nil {
		data["username"] = conn.Username
		data["linked_at"] = conn.LinkedAt
	}
	writeJSON(w, http.StatusOK, data, nil)
}

func (s *Server) handleTelegramLink(w http.ResponseWriter, r *http.Request) {
	if s.tgClient == nil || !s.tgClient.Enabled() || s.cfg.TelegramBotUsername == "" {
		writeError(w, http.StatusServiceUnavailable, "telegram_not_configured", "Telegram bot is not configured")
		return
	}
	u := userFrom(r.Context())
	code, err := auth.NewLinkCode()
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	expires := time.Now().Add(10 * time.Minute)
	if err := s.db.Telegram.CreateLinkCode(r.Context(), code, u.ID, expires); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	deepLink := fmt.Sprintf("https://t.me/%s?start=%s", s.cfg.TelegramBotUsername, code)
	writeJSON(w, http.StatusOK, map[string]any{
		"deep_link":  deepLink,
		"code":       code,
		"expires_at": expires,
	}, nil)
}

func (s *Server) handleTelegramUnlink(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	if err := s.db.Telegram.Unlink(r.Context(), u.ID); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"unlinked": true}, nil)
}

func (s *Server) handleGetPrefs(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	prefs, err := s.db.Telegram.GetPrefs(r.Context(), u.ID)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, prefs, nil)
}

func (s *Server) handleUpdatePrefs(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	// Start from current prefs so a partial PATCH keeps unspecified fields.
	prefs, err := s.db.Telegram.GetPrefs(r.Context(), u.ID)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	wasFollowingOnly := prefs.FeedFollowingOnly
	if !decodeJSON(w, r, prefs) {
		return
	}
	prefs.UserID = u.ID
	if prefs.DailyMaxItems < 3 || prefs.DailyMaxItems > 7 {
		writeError(w, http.StatusBadRequest, "validation", "daily_max_items must be between 3 and 7")
		return
	}
	if prefs.FeedFollowingOnly && !wasFollowingOnly {
		premium, premiumErr := s.hasActivePremium(r.Context(), u.ID)
		if premiumErr != nil {
			writeDomainError(w, s.log, premiumErr)
			return
		}
		if !premium {
			writeError(w, http.StatusForbidden, "premium_required", "Premium is required for following-only feed mode")
			return
		}
		hasTopics, checkErr := s.db.Follow.HasFeedTopics(r.Context(), u.ID)
		if checkErr != nil {
			writeDomainError(w, s.log, checkErr)
			return
		}
		if !hasTopics {
			writeError(w, http.StatusBadRequest, "validation", "follow at least one feed topic before enabling strict mode")
			return
		}
	}
	if err := s.db.Telegram.UpdatePrefs(r.Context(), prefs); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, prefs, nil)
}

func (s *Server) handleTestNotification(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	if _, err := s.db.Telegram.ConnectionByUser(r.Context(), u.ID); err != nil {
		writeError(w, http.StatusBadRequest, "not_linked", "Connect Telegram first")
		return
	}
	if err := s.enqueue.EnqueueSendDaily(r.Context(), u.ID, true); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"queued": true}, nil)
}

// handleTelegramWebhook receives bot updates. It verifies the secret token
// header before processing (spec section 18).
func (s *Server) handleTelegramWebhook(w http.ResponseWriter, r *http.Request) {
	if s.cfg.TelegramWebhookSecret != "" {
		if r.Header.Get("X-Telegram-Bot-Api-Secret-Token") != s.cfg.TelegramWebhookSecret {
			writeError(w, http.StatusUnauthorized, "unauthorized", "bad secret token")
			return
		}
	}
	var update telegram.Update
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&update); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid update")
		return
	}
	if s.tgHook != nil {
		if err := s.tgHook.Handle(r.Context(), &update); err != nil {
			s.log.Error("telegram webhook handler failed", "err", err)
		}
	}
	// Always 200 so Telegram does not retry indefinitely.
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true}, nil)
}
