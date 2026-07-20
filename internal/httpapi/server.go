package httpapi

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"repwire/internal/config"
	"repwire/internal/feed"
	"repwire/internal/jobs"
	"repwire/internal/postgres"
	"repwire/internal/push"
	"repwire/internal/telegram"
)

// Server holds the API dependencies and builds the HTTP handler.
type Server struct {
	db         *postgres.DB
	cfg        *config.Config
	log        *slog.Logger
	homepage   *feed.Builder
	ranker     *feed.Ranker
	enqueue    *jobs.Enqueuer
	tgClient   *telegram.Client
	tgHook     *telegram.Handler
	pushClient *push.Client

	loginLimiter    *rateLimiter
	loginIPLimiter  *rateLimiter
	registerLimiter *rateLimiter
	searchLimiter   *rateLimiter
	writeLimiter    *rateLimiter
	trustedProxy    func(net.IP) bool
	metrics         *httpMetrics
}

// NewServer wires the API server.
func NewServer(db *postgres.DB, cfg *config.Config, log *slog.Logger, enqueue *jobs.Enqueuer, tgClient *telegram.Client, tgHook *telegram.Handler) *Server {
	return &Server{
		db:              db,
		cfg:             cfg,
		log:             log,
		homepage:        feed.NewBuilder(db),
		ranker:          feed.NewRanker(db),
		enqueue:         enqueue,
		tgClient:        tgClient,
		tgHook:          tgHook,
		pushClient:      push.NewClient(cfg.WebPushPublicKey, cfg.WebPushPrivateKey, cfg.WebPushSubject),
		loginLimiter:    newRateLimiter(5, 15*time.Minute),
		loginIPLimiter:  newRateLimiter(30, 15*time.Minute),
		registerLimiter: newRateLimiter(5, time.Hour),
		searchLimiter:   newRateLimiter(60, time.Minute),
		writeLimiter:    newRateLimiter(120, time.Minute),
		trustedProxy:    trustedProxyMatcher(cfg.TrustedProxyCIDRs),
		metrics:         &httpMetrics{},
	}
}

// Handler builds the routed http.Handler with global middleware applied.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	s.routes(mux)

	// Global middleware chain: recover → logging → cors → withUser → mux.
	var h http.Handler = mux
	h = s.withUser(h)
	h = browserWriteGuard(s.cfg.CORSOrigins, h)
	h = cors(s.cfg.CORSOrigins, h)
	h = logging(s.log, s.metrics, h)
	h = recoverer(s.log, h)
	return h
}

func (s *Server) routes(mux *http.ServeMux) {
	mux.Handle("GET /media/", http.StripPrefix("/media/", http.FileServer(http.Dir(s.cfg.MediaStorageDir))))
	// Health
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /readyz", s.handleReadyz)
	// Intentionally not proxied by the production Caddyfile. Monitoring agents
	// on the private container network may scrape this low-cardinality endpoint.
	mux.HandleFunc("GET /metrics", s.handleMetrics)

	const v1 = "/api/v1"

	// ---- Public content ----
	// Public GET endpoints carry Cache-Control so Caddy, the Next.js fetch cache
	// and any CDN can absorb read bursts. Durations track how fast each surface
	// changes: live scores short, taxonomy long. Personalized or mutating routes
	// (home, feed, likes, translate) are deliberately left uncached.
	mux.HandleFunc("GET "+v1+"/content", publicCache(30, s.handleListContent))
	mux.HandleFunc("GET "+v1+"/content/{id}", publicCache(30, s.handleGetContent))
	mux.HandleFunc("GET "+v1+"/content/{id}/related", publicCache(120, s.handleRelated))
	mux.HandleFunc("GET "+v1+"/content/{id}/reactions", s.handleReactions)
	mux.HandleFunc("POST "+v1+"/content/{id}/like", s.handleLike)
	mux.HandleFunc("DELETE "+v1+"/content/{id}/like", s.handleUnlike)
	mux.HandleFunc("GET "+v1+"/clusters/{id}", publicCache(60, s.handleGetStoryCluster))
	mux.HandleFunc("GET "+v1+"/analyses", publicCache(120, s.handlePublishedAnalyses))
	mux.HandleFunc("POST "+v1+"/content/{id}/translate", s.handleTranslate)

	mux.HandleFunc("GET "+v1+"/research", publicCache(300, s.handleListResearch))
	mux.HandleFunc("GET "+v1+"/research/{id}", publicCache(600, s.handleGetResearch))

	mux.HandleFunc("GET "+v1+"/videos", publicCache(60, s.handleListVideos))
	mux.HandleFunc("GET "+v1+"/videos/{id}", publicCache(300, s.handleGetVideo))

	mux.HandleFunc("GET "+v1+"/topics", publicCache(300, s.handleListTopics))
	mux.HandleFunc("GET "+v1+"/topics/{slug}", publicCache(300, s.handleGetTopic))
	mux.HandleFunc("GET "+v1+"/sources", publicCache(600, s.handleListSources))
	mux.HandleFunc("GET "+v1+"/entities", publicCache(600, s.handleListEntities))
	mux.HandleFunc("GET "+v1+"/entities/{slug}", publicCache(600, s.handleGetEntity))

	mux.HandleFunc("GET "+v1+"/search", s.handleSearch)
	mux.HandleFunc("GET "+v1+"/search/suggest", s.handleSuggest)
	mux.HandleFunc("GET "+v1+"/home", s.handleHome)
	mux.HandleFunc("GET "+v1+"/capabilities", publicCache(600, s.handlePublicCapabilities))
	mux.HandleFunc("GET "+v1+"/audio-briefs/latest", publicCache(300, s.handleLatestAudioBrief))
	mux.HandleFunc("GET "+v1+"/sports", publicCache(600, s.handleSports))
	mux.HandleFunc("GET "+v1+"/competitions", publicCache(600, s.handleCompetitions))
	mux.HandleFunc("GET "+v1+"/competitions/{slug}", publicCache(60, s.handleCompetition))
	mux.HandleFunc("GET "+v1+"/events", publicCache(15, s.handleEvents))
	mux.HandleFunc("GET "+v1+"/events/{id}", publicCache(15, s.handleEvent))
	mux.HandleFunc("GET "+v1+"/events/{id}/content", publicCache(60, s.handleEventContent))
	mux.HandleFunc("GET "+v1+"/events/{id}/calendar.ics", s.handleEventCalendar)
	mux.HandleFunc("GET "+v1+"/catch-up", s.handleCatchUp)
	mux.HandleFunc("POST "+v1+"/product-events", s.handleProductEvent)
	mux.HandleFunc("POST "+v1+"/payments/sepay/ipn", s.handleSePayIPN)

	// ---- Auth ----
	mux.HandleFunc("POST "+v1+"/auth/register", s.handleRegister)
	mux.HandleFunc("POST "+v1+"/auth/login", s.handleLogin)
	mux.HandleFunc("POST "+v1+"/auth/logout", s.handleLogout)
	mux.HandleFunc("GET "+v1+"/auth/me", requireAuth(s.handleMe))

	mux.HandleFunc("POST "+v1+"/onboarding", requireAuth(s.handleOnboarding))

	// ---- Feed ----
	mux.HandleFunc("GET "+v1+"/feed", requireAuth(s.handleFeed))
	mux.HandleFunc("GET "+v1+"/me/dashboard", requireAuth(s.handleDashboard))
	mux.HandleFunc("PATCH "+v1+"/me/dashboard", requireAuth(s.handleDashboard))
	mux.HandleFunc("POST "+v1+"/me/preferences/sync", requireAuth(s.handlePreferenceSync))
	mux.HandleFunc("GET "+v1+"/me/fan-passport", requireAuth(s.handleFanPassport))
	mux.HandleFunc("POST "+v1+"/clusters/{id}/follow", requireAuth(s.handleFollowCluster))
	mux.HandleFunc("DELETE "+v1+"/clusters/{id}/follow", requireAuth(s.handleFollowCluster))
	mux.HandleFunc("POST "+v1+"/clusters/{id}/read", requireAuth(s.handleReadCluster))
	mux.HandleFunc("POST "+v1+"/events/{id}/follow", requireAuth(s.handleFollowEvent))
	mux.HandleFunc("DELETE "+v1+"/events/{id}/follow", requireAuth(s.handleFollowEvent))
	mux.HandleFunc("GET "+v1+"/predictions", requireAuth(s.handlePredictions))
	mux.HandleFunc("POST "+v1+"/predictions/{id}/answer", requireAuth(s.handlePredictionAnswer))
	mux.HandleFunc("GET "+v1+"/premium/status", requireAuth(s.handlePremiumStatus))
	mux.HandleFunc("POST "+v1+"/premium/checkout", requireAuth(s.handlePremiumCheckout))
	mux.HandleFunc("POST "+v1+"/push/subscribe", requireAuth(s.handlePushSubscribe))
	mux.HandleFunc("DELETE "+v1+"/push/subscribe", requireAuth(s.handlePushUnsubscribe))
	mux.HandleFunc("POST "+v1+"/push/test", requireAuth(s.handlePushTest))

	// ---- Follows ----
	mux.HandleFunc("GET "+v1+"/follows", requireAuth(s.handleListFollows))
	mux.HandleFunc("GET "+v1+"/follows/status", requireAuth(s.handleFollowStatus))
	mux.HandleFunc("POST "+v1+"/follows/topics/{id}", requireAuth(s.handleFollowTopic))
	mux.HandleFunc("DELETE "+v1+"/follows/topics/{id}", requireAuth(s.handleUnfollowTopic))
	mux.HandleFunc("PATCH "+v1+"/follows/topics/{id}", requireAuth(s.handlePatchTopicFollow))
	mux.HandleFunc("POST "+v1+"/follows/entities/{id}", requireAuth(s.handleFollowEntity))
	mux.HandleFunc("DELETE "+v1+"/follows/entities/{id}", requireAuth(s.handleUnfollowEntity))
	mux.HandleFunc("PATCH "+v1+"/follows/entities/{id}", requireAuth(s.handlePatchEntityFollow))
	mux.HandleFunc("POST "+v1+"/follows/sources/{id}", requireAuth(s.handleFollowSource))
	mux.HandleFunc("DELETE "+v1+"/follows/sources/{id}", requireAuth(s.handleUnfollowSource))

	// ---- Mutes ----
	mux.HandleFunc("POST "+v1+"/mutes/topics/{id}", requireAuth(s.handleMuteTopic))
	mux.HandleFunc("DELETE "+v1+"/mutes/topics/{id}", requireAuth(s.handleUnmuteTopic))
	mux.HandleFunc("POST "+v1+"/mutes/sources/{id}", requireAuth(s.handleMuteSource))
	mux.HandleFunc("DELETE "+v1+"/mutes/sources/{id}", requireAuth(s.handleUnmuteSource))

	// ---- Save / collections / history / hidden ----
	mux.HandleFunc("POST "+v1+"/saved/{id}", requireAuth(s.handleSave))
	mux.HandleFunc("DELETE "+v1+"/saved/{id}", requireAuth(s.handleUnsave))
	mux.HandleFunc("GET "+v1+"/saved", requireAuth(s.handleListSaved))
	mux.HandleFunc("GET "+v1+"/saved/{id}/status", requireAuth(s.handleSavedStatus))
	mux.HandleFunc("GET "+v1+"/collections", requireAuth(s.handleListCollections))
	mux.HandleFunc("POST "+v1+"/collections", requireAuth(s.handleCreateCollection))
	mux.HandleFunc("DELETE "+v1+"/collections/{id}", requireAuth(s.handleDeleteCollection))
	mux.HandleFunc("POST "+v1+"/hidden/{id}", requireAuth(s.handleHide))
	mux.HandleFunc("POST "+v1+"/history/{id}", requireAuth(s.handleMarkRead))

	// ---- Telegram / notifications ----
	mux.HandleFunc("GET "+v1+"/telegram/status", requireAuth(s.handleTelegramStatus))
	mux.HandleFunc("GET "+v1+"/telegram/link", requireAuth(s.handleTelegramLink))
	mux.HandleFunc("DELETE "+v1+"/telegram", requireAuth(s.handleTelegramUnlink))
	mux.HandleFunc("GET "+v1+"/notifications/prefs", requireAuth(s.handleGetPrefs))
	mux.HandleFunc("PATCH "+v1+"/notifications/prefs", requireAuth(s.handleUpdatePrefs))
	mux.HandleFunc("POST "+v1+"/notifications/test", requireAuth(s.handleTestNotification))

	// ---- Telegram webhook (secret-token gated) ----
	mux.HandleFunc("POST "+v1+"/telegram/webhook", s.handleTelegramWebhook)

	// ---- Admin ----
	mux.HandleFunc("GET "+v1+"/admin/sources", requireAdmin(s.handleAdminListSources))
	mux.HandleFunc("POST "+v1+"/admin/sources", requireAdmin(s.handleAdminCreateSource))
	mux.HandleFunc("PATCH "+v1+"/admin/sources/{id}", requireAdmin(s.handleAdminUpdateSource))
	mux.HandleFunc("POST "+v1+"/admin/sources/{id}/fetch", requireAdmin(s.handleAdminFetchSource))

	mux.HandleFunc("GET "+v1+"/admin/content", requireAdmin(s.handleAdminListContent))
	mux.HandleFunc("GET "+v1+"/admin/content/{id}", requireAdmin(s.handleAdminGetContent))
	mux.HandleFunc("PATCH "+v1+"/admin/content/{id}", requireAdmin(s.handleAdminUpdateContent))
	mux.HandleFunc("POST "+v1+"/admin/content/{id}/topics", requireAdmin(s.handleAdminSetTopics))
	mux.HandleFunc("POST "+v1+"/admin/content/{id}/analysis-queue", requireAdmin(s.handleAdminQueueContentAnalysis))
	mux.HandleFunc("POST "+v1+"/admin/content/{id}/perspective", requireAdmin(s.handleAdminGeneratePerspective))
	mux.HandleFunc("POST "+v1+"/admin/content/{id}/highlight", requireAdmin(s.handleAdminHighlight))
	mux.HandleFunc("POST "+v1+"/admin/content/{id}/hide", requireAdmin(s.handleAdminHideContent))
	mux.HandleFunc("PATCH "+v1+"/admin/research/{id}", requireAdmin(s.handleAdminUpdateResearch))
	mux.HandleFunc("GET "+v1+"/admin/analysis-candidates", requireAdmin(s.handleAdminAnalysisCandidates))
	mux.HandleFunc("POST "+v1+"/admin/analysis-candidates/{id}/generate", requireAdmin(s.handleAdminGenerateAnalysis))
	mux.HandleFunc("POST "+v1+"/admin/analysis-candidates/{id}/publish", requireAdmin(s.handleAdminPublishAnalysis))
	mux.HandleFunc("POST "+v1+"/admin/analysis-candidates/{id}/dismiss", requireAdmin(s.handleAdminDismissAnalysis))

	mux.HandleFunc("GET "+v1+"/admin/jobs", requireAdmin(s.handleAdminListJobs))
	mux.HandleFunc("POST "+v1+"/admin/jobs/{id}/retry", requireAdmin(s.handleAdminRetryJob))
	mux.HandleFunc("GET "+v1+"/admin/jobs/stats", requireAdmin(s.handleAdminJobStats))
	mux.HandleFunc("GET "+v1+"/admin/llm-usage", requireAdmin(s.handleAdminLLMUsage))

	mux.HandleFunc("GET "+v1+"/admin/entities", requireAdmin(s.handleAdminListEntities))
	mux.HandleFunc("POST "+v1+"/admin/entities", requireAdmin(s.handleAdminCreateEntity))
	mux.HandleFunc("PATCH "+v1+"/admin/entities/{id}", requireAdmin(s.handleAdminUpdateEntity))

	mux.HandleFunc("GET "+v1+"/admin/topics", requireAdmin(s.handleAdminListTopics))
	mux.HandleFunc("POST "+v1+"/admin/topics", requireAdmin(s.handleAdminCreateTopic))
	mux.HandleFunc("PATCH "+v1+"/admin/topics/{id}", requireAdmin(s.handleAdminUpdateTopic))
	mux.HandleFunc("POST "+v1+"/admin/events", requireAdmin(s.handleAdminEvents))
	mux.HandleFunc("PATCH "+v1+"/admin/events/{id}", requireAdmin(s.handleAdminEvent))
	mux.HandleFunc("POST "+v1+"/admin/events/{id}/result", requireAdmin(s.handleAdminEventResult))
	mux.HandleFunc("POST "+v1+"/admin/predictions", requireAdmin(s.handleAdminPredictions))
	mux.HandleFunc("PATCH "+v1+"/admin/predictions/{id}", requireAdmin(s.handleAdminPrediction))
	mux.HandleFunc("POST "+v1+"/admin/predictions/{id}/settle", requireAdmin(s.handleAdminSettlePrediction))
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"}, nil)
}

func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	if err := s.db.Ping(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "not_ready", "database unreachable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"}, nil)
}

func (s *Server) handleMetrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = fmt.Fprintf(w,
		"# TYPE baothex_http_requests_total counter\n"+
			"baothex_http_requests_total %d\n"+
			"# TYPE baothex_http_errors_total counter\n"+
			"baothex_http_errors_total %d\n"+
			"# TYPE baothex_http_requests_in_flight gauge\n"+
			"baothex_http_requests_in_flight %d\n"+
			"# TYPE baothex_http_duration_milliseconds_total counter\n"+
			"baothex_http_duration_milliseconds_total %d\n",
		s.metrics.requests.Load(), s.metrics.errors.Load(), s.metrics.inFlight.Load(),
		s.metrics.durationMillis.Load())
}
