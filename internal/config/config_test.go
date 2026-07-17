package config

import (
	"strings"
	"testing"
	"time"
)

// setRequired sets the two variables Load insists on, so each test only has to
// express the setting it actually cares about.
func setRequired(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("SESSION_SECRET", "test-secret")
}

func TestLoadRejectsUnreachableScoreThreshold(t *testing.T) {
	// An 80 threshold reads like "be strict" but silently disables summarizing
	// altogether, because an article can never score above 40. Loading must fail
	// rather than run a pipeline whose main gate is inert.
	setRequired(t)
	t.Setenv("LLM_SCORE_THRESHOLD", "80")

	_, err := Load()
	if err == nil {
		t.Fatal("expected an unreachable threshold to be rejected")
	}
	if !strings.Contains(err.Error(), "unreachable") {
		t.Fatalf("error should explain the ceiling, got: %v", err)
	}
}

func TestLoadRejectsUnreachableTranslateMinScore(t *testing.T) {
	setRequired(t)
	t.Setenv("LLM_TRANSLATE_MIN_SCORE", "60")

	if _, err := Load(); err == nil {
		t.Fatal("expected an unreachable translate floor to be rejected")
	}
}

func TestLoadAcceptsReachableThresholds(t *testing.T) {
	setRequired(t)
	t.Setenv("LLM_SCORE_THRESHOLD", "25")
	t.Setenv("LLM_TRANSLATE_MIN_SCORE", "30")

	c, err := Load()
	if err != nil {
		t.Fatalf("reachable thresholds rejected: %v", err)
	}
	if c.LLMScoreThreshold != 25 || c.LLMTranslateMinScore != 30 {
		t.Fatalf("thresholds not loaded: %+v", c)
	}
}

func TestLoadDefaultsAreReachable(t *testing.T) {
	// The defaults must themselves satisfy the guard, or a fresh install fails
	// to boot.
	setRequired(t)

	c, err := Load()
	if err != nil {
		t.Fatalf("defaults rejected: %v", err)
	}
	if c.EditorialStartHour != 9 {
		t.Fatalf("EditorialStartHour = %d, want 9", c.EditorialStartHour)
	}
	if c.EditorialPicksPerDay != 3 {
		t.Fatalf("EditorialPicksPerDay = %d, want 3", c.EditorialPicksPerDay)
	}
}

func TestTranslateMaxAgeDefaultsAndParses(t *testing.T) {
	setRequired(t)
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.LLMTranslateMaxAge != 36*time.Hour {
		t.Fatalf("default max age = %v, want 36h", c.LLMTranslateMaxAge)
	}

	t.Setenv("LLM_TRANSLATE_MAX_AGE_HOURS", "12")
	c, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.LLMTranslateMaxAge != 12*time.Hour {
		t.Fatalf("max age = %v, want 12h", c.LLMTranslateMaxAge)
	}
}

func TestLoadRejectsNonPositiveTranslateMaxAge(t *testing.T) {
	// Zero would mean "translate nothing" while reading like "no limit" — the
	// same silent-disable trap as an unreachable score threshold.
	setRequired(t)
	t.Setenv("LLM_TRANSLATE_MAX_AGE_HOURS", "0")

	if _, err := Load(); err == nil {
		t.Fatal("expected a zero max age to be rejected")
	}
}

func TestLoadRejectsGeminiModelUrlMismatch(t *testing.T) {
	// The trap: Gemini reads the model from the URL path, so this config bills
	// gemini-3.5-flash (5x the input price) while every usage row says
	// gemini-2.5-flash. Silent, and it corrupts the cost reporting you would use
	// to notice.
	setRequired(t)
	t.Setenv("LLM_BASE_URL", "https://generativelanguage.googleapis.com/v1beta/models/gemini-3.5-flash:generateContent")
	t.Setenv("LLM_MODEL", "gemini-2.5-flash")

	_, err := Load()
	if err == nil {
		t.Fatal("expected a model/URL mismatch to be rejected")
	}
	if !strings.Contains(err.Error(), "LLM_BASE_URL") {
		t.Fatalf("error should point at the URL: %v", err)
	}
}

func TestLoadAcceptsMatchingGeminiModelAndUrl(t *testing.T) {
	setRequired(t)
	t.Setenv("LLM_BASE_URL", "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent")
	t.Setenv("LLM_MODEL", "gemini-2.5-flash")

	if _, err := Load(); err != nil {
		t.Fatalf("matching model and URL rejected: %v", err)
	}
}

func TestLoadLeavesNonGeminiUrlsAlone(t *testing.T) {
	// Anthropic takes the model in the body, so there is nothing to cross-check
	// and the guard must not fire.
	setRequired(t)
	t.Setenv("LLM_BASE_URL", "https://api.anthropic.com/v1/messages")
	t.Setenv("LLM_MODEL", "claude-haiku-4-5-20251001")

	if _, err := Load(); err != nil {
		t.Fatalf("Anthropic config rejected: %v", err)
	}
}

func TestLoadRejectsBadEditorialStartHour(t *testing.T) {
	setRequired(t)
	t.Setenv("EDITORIAL_START_HOUR", "25")

	if _, err := Load(); err == nil {
		t.Fatal("expected hour 25 to be rejected")
	}
}

// Zero would read as "no editorial limit" and mean the opposite: a desk that
// never commits to anything. It is a typo, not a setting.
func TestLoadRejectsZeroEditorialPicks(t *testing.T) {
	setRequired(t)
	t.Setenv("EDITORIAL_PICKS_PER_DAY", "0")

	if _, err := Load(); err == nil {
		t.Fatal("expected zero picks per day to be rejected")
	}
}

func TestTTSFallsBackToLLMKeysOnlyWhenUnset(t *testing.T) {
	setRequired(t)
	t.Setenv("LLM_API_KEY", "llm-one,llm-two")
	t.Setenv("TTS_API_KEY", "tts-only")

	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(c.TTSAPIKeys) != 1 || c.TTSAPIKeys[0] != "tts-only" {
		t.Fatalf("dedicated TTS keys were not honoured: %v", c.TTSAPIKeys)
	}
	if len(c.LLMAPIKeys) != 2 {
		t.Fatalf("LLM key pool = %v, want two keys", c.LLMAPIKeys)
	}
}
