// Package config loads runtime configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all runtime settings for the api and worker binaries.
type Config struct {
	DatabaseURL   string
	SessionSecret string
	PublicBaseURL string

	// API
	APIAddr     string
	CORSOrigins []string

	// Worker
	WorkerConcurrency int

	// YouTube
	YouTubeAPIKey string

	// Telegram
	TelegramBotToken       string
	TelegramWebhookSecret  string
	TelegramBotUsername    string

	// LLM
	LLMAPIKey          string
	LLMBaseURL         string
	LLMModel           string
	LLMDailyBudgetUSD  float64
	LLMScoreThreshold  float64
	LLMMaxCallsPerHour int

	// Logging
	LogFormat string
	LogLevel  string
}

// Load reads configuration from the process environment, applying defaults.
// It returns an error only for values that are required and cannot be defaulted.
func Load() (*Config, error) {
	c := &Config{
		DatabaseURL:            env("DATABASE_URL", ""),
		SessionSecret:          env("SESSION_SECRET", ""),
		PublicBaseURL:          env("PUBLIC_BASE_URL", "http://localhost:3000"),
		APIAddr:                env("API_ADDR", ":8080"),
		CORSOrigins:            splitCSV(env("CORS_ORIGINS", "http://localhost:3000")),
		WorkerConcurrency:      envInt("WORKER_CONCURRENCY", 8),
		YouTubeAPIKey:          env("YOUTUBE_API_KEY", ""),
		TelegramBotToken:       env("TELEGRAM_BOT_TOKEN", ""),
		TelegramWebhookSecret:  env("TELEGRAM_WEBHOOK_SECRET", ""),
		TelegramBotUsername:    env("TELEGRAM_BOT_USERNAME", "RepWireBot"),
		LLMAPIKey:              env("LLM_API_KEY", ""),
		LLMBaseURL:             env("LLM_BASE_URL", "https://api.anthropic.com/v1/messages"),
		LLMModel:               env("LLM_MODEL", "claude-haiku-4-5-20251001"),
		LLMDailyBudgetUSD:      envFloat("LLM_DAILY_BUDGET_USD", 5),
		LLMScoreThreshold:      envFloat("LLM_SCORE_THRESHOLD", 25),
		LLMMaxCallsPerHour:     envInt("LLM_MAX_CALLS_PER_HOUR", 120),
		LogFormat:              env("LOG_FORMAT", "json"),
		LogLevel:               env("LOG_LEVEL", "info"),
	}

	if c.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if c.SessionSecret == "" {
		return nil, fmt.Errorf("SESSION_SECRET is required")
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
