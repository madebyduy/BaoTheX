// Package config loads runtime configuration from environment variables.
package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"repwire/internal/process"
)

// Config holds all runtime settings for the api and worker binaries.
type Config struct {
	DatabaseURL   string
	SessionSecret string
	PublicBaseURL string

	// API
	APIAddr           string
	CORSOrigins       []string
	TrustedProxyCIDRs []string

	// Worker
	WorkerConcurrency int
	SportsDataMode    string
	FootballDataToken string
	PandaScoreToken   string
	OpenF1Enabled     bool
	TheSportsDBKey    string

	// YouTube
	YouTubeAPIKey string

	// Telegram
	TelegramBotToken      string
	TelegramWebhookSecret string
	TelegramBotUsername   string
	TelegramPolling       bool

	// Media / TTS
	TTSAPIKeys []string // dedicated audio-brief key pool (falls back to LLM keys)
	TTSModel   string
	TTSVoice   string
	// TTSMaxCallsPerMinute paces narration requests. It is separate from
	// LLMMaxCallsPerMinute because TTS_API_KEY is its own key pool with its own
	// quota: making the two share a pacer would have the audio brief queueing
	// behind article digests for an allowance it does not draw on. When
	// TTS_API_KEY is unset the pools are the same keys and this should be set to
	// match LLM_MAX_CALLS_PER_MINUTE. Zero disables pacing.
	TTSMaxCallsPerMinute int
	MediaStorageDir      string
	MediaPublicBaseURL   string

	// Web Push
	WebPushPublicKey  string
	WebPushPrivateKey string
	WebPushSubject    string

	// SePay / Premium
	SePayMerchant       string
	SePaySecretKey      string
	SePayBaseURL        string
	SePayIPNSecretKey   string
	PremiumMonthlyPrice int

	// LLM
	LLMAPIKey  string   // first key; kept for single-key callers (e.g. TTS default)
	LLMAPIKeys []string // full rotation pool: tried in order, next on quota exhaustion
	LLMBaseURL string
	LLMModel   string
	// LLMInputUSDPerMTok / LLMOutputUSDPerMTok price the daily budget meter and
	// must describe the provider behind LLM_BASE_URL. The defaults match the
	// default model (Anthropic Haiku-class); a free tier should set both to 0,
	// because a meter that charges for calls the provider gives away will stop
	// work over money that was never spent.
	LLMInputUSDPerMTok   float64
	LLMOutputUSDPerMTok  float64
	LLMDailyBudgetUSD    float64
	LLMScoreThreshold    float64
	LLMTranslateMinScore float64
	// LLMTranslateMaxAge is how old a foreign article may be and still be worth
	// translating. Past it the story is abandoned rather than queued: news
	// expires, and a queue with no expiry guarantees the worker falls
	// permanently behind, translating last week while today goes unread.
	LLMTranslateMaxAge time.Duration
	LLMMaxCallsPerHour int
	// LLMMaxCallsPerMinute paces requests so concurrent workers cannot burst
	// past the provider's rate limit. LLMMaxCallsPerHour bounds the total but
	// says nothing about the rate — 60 per hour permits all 60 in one minute,
	// which is how four worker slots came to fire twenty-plus requests at a
	// five-per-minute door and report the resulting rejections as dead keys.
	// Applies to the LLM key pool only; see TTSMaxCallsPerMinute for narration.
	// Zero disables pacing, which suits a paid tier with headroom.
	LLMMaxCallsPerMinute int

	// Editorial
	// EditorialStartHour is the hour (Vietnam time) from which the desk may
	// commit to a story. Before it, the day's headlines have not been
	// corroborated or followed up enough for the ranking to tell a real
	// controversy from one that is merely loud.
	//
	// It was DAILY_PICK_HOUR=21, and it meant something narrower: the single
	// moment the newsroom stopped watching and spent its whole LLM budget on one
	// story. That budget was a fiction produced by a pricing bug, and one article
	// an evening was all the fiction could afford. The desk now opens in the
	// morning and commits to EditorialPicksPerDay stories as the day earns them.
	EditorialStartHour int
	// EditorialPicksPerDay caps how many stories a day get an analysis. Each is
	// a couple of LLM calls plus translating its sources; the limiting factor is
	// the human who reviews the drafts, not the quota.
	EditorialPicksPerDay int

	// Logging
	LogFormat string
	LogLevel  string
}

// Load reads configuration from the process environment, applying defaults.
// It returns an error only for values that are required and cannot be defaulted.
func Load() (*Config, error) {
	// LLM_API_KEY may hold several comma-separated keys; the summarizer rotates
	// through them when one is rate-limited/quota-exhausted.
	llmKeys := splitCSV(env("LLM_API_KEY", ""))
	firstLLMKey := ""
	if len(llmKeys) > 0 {
		firstLLMKey = llmKeys[0]
	}

	// TTS uses its own key pool so the audio brief has reserved quota; when
	// TTS_API_KEY is unset it falls back to sharing the LLM keys.
	ttsKeys := splitCSV(env("TTS_API_KEY", ""))
	if len(ttsKeys) == 0 {
		ttsKeys = llmKeys
	}

	c := &Config{
		DatabaseURL:           env("DATABASE_URL", ""),
		SessionSecret:         env("SESSION_SECRET", ""),
		PublicBaseURL:         env("PUBLIC_BASE_URL", "http://localhost:3000"),
		APIAddr:               env("API_ADDR", ":8080"),
		CORSOrigins:           splitCSV(env("CORS_ORIGINS", "http://localhost:3000")),
		TrustedProxyCIDRs:     splitCSV(env("TRUSTED_PROXY_CIDRS", "127.0.0.1/32,::1/128")),
		WorkerConcurrency:     envInt("WORKER_CONCURRENCY", 8),
		SportsDataMode:        env("SPORTS_DATA_MODE", "free"),
		FootballDataToken:     env("FOOTBALL_DATA_TOKEN", ""),
		PandaScoreToken:       env("PANDASCORE_TOKEN", ""),
		OpenF1Enabled:         envBool("OPENF1_ENABLED", true),
		TheSportsDBKey:        env("THESPORTSDB_KEY", "123"),
		YouTubeAPIKey:         env("YOUTUBE_API_KEY", ""),
		TelegramBotToken:      env("TELEGRAM_BOT_TOKEN", ""),
		TelegramWebhookSecret: env("TELEGRAM_WEBHOOK_SECRET", ""),
		TelegramBotUsername:   env("TELEGRAM_BOT_USERNAME", "RepWireBot"),
		TelegramPolling:       envBool("TELEGRAM_POLLING", strings.Contains(env("PUBLIC_BASE_URL", ""), "localhost")),
		TTSAPIKeys:            ttsKeys,
		TTSModel:              env("TTS_MODEL", "gemini-2.5-flash-preview-tts"),
		TTSVoice:              env("TTS_VOICE", "Erinome"),
		TTSMaxCallsPerMinute:  envInt("TTS_MAX_CALLS_PER_MINUTE", 3),
		MediaStorageDir:       env("MEDIA_STORAGE_DIR", "./var/media"),
		MediaPublicBaseURL:    env("MEDIA_PUBLIC_BASE_URL", env("PUBLIC_BASE_URL", "http://localhost:3000")),
		WebPushPublicKey:      env("WEB_PUSH_PUBLIC_KEY", ""),
		WebPushPrivateKey:     env("WEB_PUSH_PRIVATE_KEY", ""),
		WebPushSubject:        env("WEB_PUSH_SUBJECT", "mailto:admin@example.com"),
		SePayMerchant:         env("SEPAY_MERCHANT", ""),
		SePaySecretKey:        env("SEPAY_SECRET_KEY", ""),
		SePayBaseURL:          env("SEPAY_BASE_URL", "https://pay.sepay.vn"),
		SePayIPNSecretKey:     env("SEPAY_IPN_SECRET_KEY", ""),
		PremiumMonthlyPrice:   envInt("PREMIUM_MONTHLY_PRICE", 39000),
		LLMAPIKey:             firstLLMKey,
		LLMAPIKeys:            llmKeys,
		LLMBaseURL:            env("LLM_BASE_URL", "https://api.anthropic.com/v1/messages"),
		LLMModel:              env("LLM_MODEL", "claude-haiku-4-5-20251001"),
		LLMInputUSDPerMTok:    envFloat("LLM_INPUT_USD_PER_MTOK", 1),
		LLMOutputUSDPerMTok:   envFloat("LLM_OUTPUT_USD_PER_MTOK", 5),
		LLMDailyBudgetUSD:     envFloat("LLM_DAILY_BUDGET_USD", 5),
		LLMScoreThreshold:     envFloat("LLM_SCORE_THRESHOLD", 25),
		LLMTranslateMinScore:  envFloat("LLM_TRANSLATE_MIN_SCORE", 30),
		LLMTranslateMaxAge:    time.Duration(envInt("LLM_TRANSLATE_MAX_AGE_HOURS", 36)) * time.Hour,
		LLMMaxCallsPerHour:    envInt("LLM_MAX_CALLS_PER_HOUR", 120),
		LLMMaxCallsPerMinute:  envInt("LLM_MAX_CALLS_PER_MINUTE", 4),
		EditorialStartHour:    envInt("EDITORIAL_START_HOUR", 9),
		EditorialPicksPerDay:  envInt("EDITORIAL_PICKS_PER_DAY", 3),
		LogFormat:             env("LOG_FORMAT", "json"),
		LogLevel:              env("LOG_LEVEL", "info"),
	}

	if c.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if c.SessionSecret == "" {
		return nil, fmt.Errorf("SESSION_SECRET is required")
	}
	for _, cidr := range c.TrustedProxyCIDRs {
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			return nil, fmt.Errorf("TRUSTED_PROXY_CIDRS contains invalid CIDR %q", cidr)
		}
	}
	if c.EditorialStartHour < 0 || c.EditorialStartHour > 23 {
		return nil, fmt.Errorf("EDITORIAL_START_HOUR must be between 0 and 23, got %d", c.EditorialStartHour)
	}
	if c.EditorialPicksPerDay < 1 {
		return nil, fmt.Errorf("EDITORIAL_PICKS_PER_DAY must be at least 1, got %d", c.EditorialPicksPerDay)
	}
	if c.LLMTranslateMaxAge <= 0 {
		return nil, fmt.Errorf("LLM_TRANSLATE_MAX_AGE_HOURS must be positive, got %v", c.LLMTranslateMaxAge)
	}
	// Gemini selects the model in the URL path, not the request body, so
	// LLM_BASE_URL is what you are billed for and LLM_MODEL is only the label on
	// the usage records. Letting them disagree means spending on one model while
	// every cost report names another — the reports look fine and are wrong.
	if strings.Contains(c.LLMBaseURL, "generativelanguage.googleapis.com") &&
		!strings.Contains(c.LLMBaseURL, c.LLMModel) {
		return nil, fmt.Errorf(
			"LLM_MODEL=%q does not appear in LLM_BASE_URL=%q: Gemini takes the model from the URL, "+
				"so this would bill the URL's model while recording usage as %q",
			c.LLMModel, c.LLMBaseURL, c.LLMModel)
	}
	// Score thresholds are compared against process.BaseScore, which tops out at
	// process.MaxArticleScore for an article. A threshold above that ceiling
	// silently disables the gate it guards instead of tightening it — the exact
	// trap that once left LLM_SCORE_THRESHOLD=80 blocking every summary while
	// translation ran unchecked. Fail loudly rather than degrade quietly.
	if c.LLMScoreThreshold > process.MaxArticleScore {
		return nil, fmt.Errorf(
			"LLM_SCORE_THRESHOLD=%g is unreachable: article scores never exceed %g, so no article would ever be summarized",
			c.LLMScoreThreshold, process.MaxArticleScore)
	}
	if c.LLMTranslateMinScore > process.MaxArticleScore {
		return nil, fmt.Errorf(
			"LLM_TRANSLATE_MIN_SCORE=%g is unreachable: article scores never exceed %g, so nothing would ever be translated",
			c.LLMTranslateMinScore, process.MaxArticleScore)
	}
	return c, nil
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return n
		}
	}
	return def
}

func envFloat(key string, def float64) float64 {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
			return f
		}
	}
	return def
}

func envBool(key string, def bool) bool {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if b, err := strconv.ParseBool(strings.TrimSpace(v)); err == nil {
			return b
		}
	}
	return def
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
