package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"repwire/internal/auth"
	"repwire/internal/domain"
)

// publicCache sets browser/CDN cache headers for endpoints whose response does
// not vary per user. A short max-age keeps sports news fresh while letting Caddy,
// the Next.js fetch cache and any CDN absorb read bursts instead of hitting
// Postgres for every visitor. Only wrap genuinely public GET handlers with it.
func publicCache(seconds int, h http.HandlerFunc) http.HandlerFunc {
	value := fmt.Sprintf(
		"public, max-age=%d, s-maxage=%d, stale-while-revalidate=%d",
		seconds, seconds, seconds*10,
	)
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", value)
		h(w, r)
	}
}

// sessionCookie is the name of the session cookie.
const sessionCookie = "rw_session"

type ctxKey int

const userKey ctxKey = iota

// userFrom returns the authenticated user from the request context, or nil.
func userFrom(ctx context.Context) *domain.User {
	u, _ := ctx.Value(userKey).(*domain.User)
	return u
}

// withUser resolves the session cookie into a user and stores it in context.
// It never rejects the request; use requireAuth to enforce authentication.
func (s *Server) withUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(sessionCookie)
		if err == nil && c.Value != "" {
			hash := auth.HashToken(c.Value)
			slideTo := time.Now().Add(auth.SessionTTLDays * 24 * time.Hour)
			if u, err := s.db.User.UserBySession(r.Context(), hash, slideTo); err == nil {
				r = r.WithContext(context.WithValue(r.Context(), userKey, u))
				// Keep the browser expiry aligned with the sliding database expiry.
				http.SetCookie(w, s.sessionCookie(c.Value, slideTo))
			}
		}
		next.ServeHTTP(w, r)
	})
}

// requireAuth wraps a handler, returning 401 when no user is present.
func requireAuth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if userFrom(r.Context()) == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
			return
		}
		h(w, r)
	}
}

// requireAdmin wraps a handler, returning 403 for non-admins.
func requireAdmin(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := userFrom(r.Context())
		if u == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
			return
		}
		if !u.IsAdmin() {
			writeError(w, http.StatusForbidden, "forbidden", "Admin only")
			return
		}
		h(w, r)
	}
}

// recoverer converts panics into 500s.
func recoverer(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Error("panic in handler", "err", rec, "path", r.URL.Path)
				writeError(w, http.StatusInternalServerError, "internal", "Internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// logging emits a structured access log per request.
type httpMetrics struct {
	requests       atomic.Uint64
	errors         atomic.Uint64
	inFlight       atomic.Int64
	durationMillis atomic.Uint64
}

func logging(log *slog.Logger, metrics *httpMetrics, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		metrics.requests.Add(1)
		metrics.inFlight.Add(1)
		defer metrics.inFlight.Add(-1)
		requestID := newRequestID()
		w.Header().Set("X-Request-ID", requestID)
		sw := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(sw, r)
		duration := time.Since(start).Milliseconds()
		metrics.durationMillis.Add(uint64(max(duration, 0)))
		if sw.status >= http.StatusInternalServerError {
			metrics.errors.Add(1)
		}
		log.Info("http",
			"request_id", requestID, "method", r.Method, "path", r.URL.Path,
			"status", sw.status, "bytes", sw.bytes,
			"duration_ms", duration)
	})
}

func newRequestID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err == nil {
		return hex.EncodeToString(b[:])
	}
	return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
}

type statusWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *statusWriter) WriteHeader(code int) {
	if w.status != 200 || code == 200 {
		return
	}
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(p []byte) (int, error) {
	n, err := w.ResponseWriter.Write(p)
	w.bytes += n
	return n, err
}

// cors applies a permissive-but-scoped CORS policy for the configured origins.
func cors(origins []string, next http.Handler) http.Handler {
	allowed := map[string]bool{}
	for _, o := range origins {
		allowed[o] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && allowed[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			// ngrok-skip-browser-warning lets the browser bypass the free-tier ngrok
			// interstitial (an HTML warning page served without CORS headers, which
			// otherwise blocks every credentialed fetch from the deployed frontend).
			// It is a custom header, so listing it here keeps the CORS preflight from
			// failing when the client sends it.
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, ngrok-skip-browser-warning")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// browserWriteGuard rejects cross-origin browser writes before they reach a
// cookie-authenticated handler. SameSite cookies are useful defence in depth,
// but are not a complete CSRF boundary (notably for compromised sibling
// subdomains). Requests without browser provenance remain available to trusted
// server-to-server integrations such as payment and Telegram webhooks, which
// authenticate with their own secrets.
func browserWriteGuard(origins []string, next http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(origins))
	for _, raw := range origins {
		if u, err := url.Parse(raw); err == nil && u.Scheme != "" && u.Host != "" {
			allowed[u.Scheme+"://"+u.Host] = struct{}{}
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		origin := r.Header.Get("Origin")
		if origin != "" {
			if _, ok := allowed[origin]; !ok {
				writeError(w, http.StatusForbidden, "csrf_rejected", "Cross-origin request rejected")
				return
			}
		} else if strings.EqualFold(r.Header.Get("Sec-Fetch-Site"), "cross-site") {
			writeError(w, http.StatusForbidden, "csrf_rejected", "Cross-site request rejected")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// rateLimiter is a tiny fixed-window limiter keyed by an arbitrary string.
type rateLimiter struct {
	mu      sync.Mutex
	hits    map[string][]time.Time
	limit   int
	window  time.Duration
	maxKeys int
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{hits: map[string][]time.Time{}, limit: limit, window: window, maxKeys: 10_000}
}

// allow reports whether key is under its limit, recording the hit if so.
func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-rl.window)
	if _, exists := rl.hits[key]; !exists && len(rl.hits) >= rl.maxKeys {
		// Fail closed under a key-flood attack. A distributed limiter should be
		// used at larger scale; this hard ceiling protects a single process now.
		return false
	}
	kept := rl.hits[key][:0]
	for _, t := range rl.hits[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= rl.limit {
		rl.hits[key] = kept
		return false
	}
	rl.hits[key] = append(kept, now)
	// Opportunistic cleanup bounds memory when an attacker continuously invents
	// new keys. Cleanup is deliberately infrequent so the hot path stays cheap.
	if len(rl.hits)%256 == 0 {
		for candidate, timestamps := range rl.hits {
			if len(timestamps) == 0 || !timestamps[len(timestamps)-1].After(cutoff) {
				delete(rl.hits, candidate)
			}
		}
	}
	return true
}

// clientIP extracts a rate-limit key. Forwarded headers are accepted only from
// an explicitly trusted proxy; otherwise a client could rotate a forged XFF
// value and bypass every IP limit.
func clientIP(r *http.Request, trustedProxy func(net.IP) bool) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	remoteIP := net.ParseIP(strings.Trim(host, "[]"))
	if remoteIP != nil && trustedProxy != nil && trustedProxy(remoteIP) {
		xff := r.Header.Get("X-Forwarded-For")
		if i := strings.IndexByte(xff, ','); i >= 0 {
			return strings.TrimSpace(xff[:i])
		}
		if strings.TrimSpace(xff) != "" {
			return strings.TrimSpace(xff)
		}
	}
	return host
}

func trustedProxyMatcher(cidrs []string) func(net.IP) bool {
	networks := make([]*net.IPNet, 0, len(cidrs))
	for _, raw := range cidrs {
		_, network, err := net.ParseCIDR(strings.TrimSpace(raw))
		if err == nil {
			networks = append(networks, network)
		}
	}
	return func(ip net.IP) bool {
		for _, network := range networks {
			if network.Contains(ip) {
				return true
			}
		}
		return false
	}
}
