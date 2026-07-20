package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientIPIgnoresForwardedHeaderFromUntrustedPeer(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "203.0.113.9:1234"
	r.Header.Set("X-Forwarded-For", "198.51.100.7")

	if got := clientIP(r, trustedProxyMatcher([]string{"127.0.0.1/32"})); got != "203.0.113.9" {
		t.Fatalf("clientIP = %q, want direct peer", got)
	}
}

func TestClientIPAcceptsForwardedHeaderFromTrustedPeer(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "127.0.0.1:1234"
	r.Header.Set("X-Forwarded-For", "198.51.100.7, 127.0.0.1")

	if got := clientIP(r, trustedProxyMatcher([]string{"127.0.0.1/32"})); got != "198.51.100.7" {
		t.Fatalf("clientIP = %q, want first forwarded address", got)
	}
}

func TestBrowserWriteGuardRejectsForeignOrigin(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	h := browserWriteGuard([]string{"https://baothex.vn"}, next)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/saved/1", nil)
	r.Header.Set("Origin", "https://evil.example")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestBrowserWriteGuardAllowsConfiguredOriginAndServerWebhook(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	h := browserWriteGuard([]string{"https://baothex.vn"}, next)
	for _, origin := range []string{"https://baothex.vn", ""} {
		r := httptest.NewRequest(http.MethodPost, "/api/v1/payments/sepay/ipn", nil)
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusNoContent {
			t.Fatalf("origin %q: status = %d, want 204", origin, w.Code)
		}
	}
}

func TestRateLimiterCleansExpiredKeys(t *testing.T) {
	rl := newRateLimiter(2, time.Nanosecond)
	for i := 0; i < 512; i++ {
		rl.allow(string(rune(i + 1)))
		time.Sleep(time.Nanosecond)
	}
	if len(rl.hits) >= 512 {
		t.Fatalf("expired limiter keys were not reclaimed: %d", len(rl.hits))
	}
}

func TestRateLimiterHasHardKeyCeiling(t *testing.T) {
	rl := newRateLimiter(1, time.Hour)
	rl.maxKeys = 2
	if !rl.allow("one") || !rl.allow("two") {
		t.Fatal("limiter rejected keys below its ceiling")
	}
	if rl.allow("three") {
		t.Fatal("limiter accepted a new key above its memory ceiling")
	}
	if len(rl.hits) != 2 {
		t.Fatalf("key count = %d, want 2", len(rl.hits))
	}
}
