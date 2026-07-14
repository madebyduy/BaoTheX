package httpapi

import (
	"net/http"
	"strings"
	"time"

	"repwire/internal/auth"
	"repwire/internal/domain"
)

type registerReq struct {
	Email       string  `json:"email"`
	Password    string  `json:"password"`
	DisplayName *string `json:"display_name"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if !strings.Contains(req.Email, "@") || len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "validation", "Valid email and password (>= 8 chars) required")
		return
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

	// Rate limit by IP + email (spec section 17).
	if !s.loginLimiter.allow(clientIP(r) + "|" + req.Email) {
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
	if err := s.db.User.SetOnboarding(r.Context(), u.ID, req.Goals); err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	for _, id := range req.TopicIDs {
		_ = s.db.Follow.FollowTopic(r.Context(), u.ID, id)
	}
	for _, id := range req.EntityIDs {
		_ = s.db.Follow.FollowEntity(r.Context(), u.ID, id)
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true}, nil)
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
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   s.secureCookies(),
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Server) clearCookie() *http.Cookie {
	return &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.secureCookies(),
		SameSite: http.SameSiteLaxMode,
	}
}

// secureCookies enables the Secure flag when the public URL is https.
func (s *Server) secureCookies() bool {
	return strings.HasPrefix(s.cfg.PublicBaseURL, "https://")
}
