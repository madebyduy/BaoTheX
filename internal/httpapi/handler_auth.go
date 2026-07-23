package httpapi

import (
	"net/http"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"repwire/internal/auth"
	"repwire/internal/domain"
)

type registerReq struct {
	Email       string  `json:"email"`
	Password    string  `json:"password"`
	DisplayName *string `json:"display_name"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if !s.registerLimiter.allow(clientIP(r, s.trustedProxy)) {
		writeError(w, http.StatusTooManyRequests, "rate_limited", "Too many registrations, try again later")
		return
	}
	var req registerReq
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if !validEmail(req.Email) || len(req.Password) < 8 || len(req.Password) > 128 {
		writeError(w, http.StatusBadRequest, "validation", "Valid email and password (8-128 bytes) required")
		return
	}
	if req.DisplayName != nil {
		name := strings.TrimSpace(*req.DisplayName)
		if utf8.RuneCountInString(name) > 80 {
			writeError(w, http.StatusBadRequest, "validation", "Display name must be at most 80 characters")
			return
		}
		if name == "" {
			req.DisplayName = nil
		} else {
			req.DisplayName = &name
		}
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	u, err := s.db.User.Create(r.Context(), req.Email, hash, req.DisplayName)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	s.issueSession(w, r, u)
	writeJSON(w, http.StatusCreated, u, nil)
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if !validEmail(req.Email) || len(req.Password) == 0 || len(req.Password) > 128 {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password")
		return
	}

	// Rate limit by IP + email (spec section 17).
	ip := clientIP(r, s.trustedProxy)
	if !s.loginIPLimiter.allow(ip) || !s.loginLimiter.allow(ip+"|"+req.Email) {
		writeError(w, http.StatusTooManyRequests, "rate_limited", "Too many login attempts, try again later")
		return
	}

	u, err := s.db.User.ByEmail(r.Context(), req.Email)
	if err != nil {
		// Do not reveal whether the email exists.
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password")
		return
	}
	ok, err := auth.VerifyPassword(req.Password, u.PasswordHash)
	if err != nil || !ok {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password")
		return
	}
	s.issueSession(w, r, u)
	writeJSON(w, http.StatusOK, u, nil)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil && c.Value != "" {
		_ = s.db.User.DeleteSession(r.Context(), auth.HashToken(c.Value))
	}
	http.SetCookie(w, s.clearCookie())
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true}, nil)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, userFrom(r.Context()), nil)
}

type onboardingReq struct {
	Goals     []string `json:"goals"`
	TopicIDs  []int64  `json:"topic_ids"`
	EntityIDs []int64  `json:"entity_ids"`
}

func (s *Server) handleOnboarding(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	var req onboardingReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if len(req.TopicIDs) > 100 || len(req.EntityIDs) > 100 || len(req.Goals) > 20 {
		writeError(w, http.StatusBadRequest, "validation", "Quá nhiều tuỳ chọn onboarding")
		return
	}
	for _, goal := range req.Goals {
		if strings.TrimSpace(goal) == "" || len([]rune(goal)) > 80 {
			writeError(w, http.StatusBadRequest, "validation", "Mục tiêu không hợp lệ")
			return
		}
	}
	if !positiveIDs(req.TopicIDs) || !positiveIDs(req.EntityIDs) {
		writeError(w, http.StatusBadRequest, "validation", "Danh mục theo dõi không hợp lệ")
		return
	}
	if err := s.db.Sports.SyncPreferences(r.Context(), u.ID, nil, req.Goals, req.TopicIDs, req.EntityIDs); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true}, nil)
}

func positiveIDs(ids []int64) bool {
	for _, id := range ids {
		if id <= 0 {
			return false
		}
	}
	return true
}

// issueSession creates a session and sets the cookie.
func (s *Server) issueSession(w http.ResponseWriter, r *http.Request, u *domain.User) {
	token, hash, err := auth.NewSessionToken()
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	expires := time.Now().Add(auth.SessionTTLDays * 24 * time.Hour)
	if err := s.db.User.CreateSession(r.Context(), hash, u.ID, expires); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	http.SetCookie(w, s.sessionCookie(token, expires))
}

func (s *Server) sessionCookie(token string, expires time.Time) *http.Cookie {
	return &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   s.secureCookies(),
		SameSite: s.sameSiteMode(),
	}
}

func (s *Server) clearCookie() *http.Cookie {
	return &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.secureCookies(),
		SameSite: s.sameSiteMode(),
	}
}

// secureCookies enables the Secure flag when the public URL is https.
func (s *Server) secureCookies() bool {
	return strings.HasPrefix(s.cfg.PublicBaseURL, "https://")
}

// sameSiteMode picks the session cookie's SameSite policy. In production the
// browser frontend and this API live on different sites (e.g. the Cloudflare
// Worker origin vs. the API tunnel), so the credentialed /auth/* fetches are
// cross-site. A Lax cookie is silently dropped on those responses, which made
// login "succeed" without ever persisting a session once deployed. SameSite=None
// lets the browser store and send it cross-site — it requires Secure, which is
// exactly when secureCookies() is true. Over plain http (local dev, same-site
// localhost) None+Secure would be rejected, so fall back to Lax there.
func (s *Server) sameSiteMode() http.SameSite {
	if s.secureCookies() {
		return http.SameSiteNoneMode
	}
	return http.SameSiteLaxMode
}

func validEmail(value string) bool {
	if value == "" || len(value) > 254 {
		return false
	}
	address, err := mail.ParseAddress(value)
	return err == nil && strings.EqualFold(address.Address, value)
}
