// Command worker runs the RepWire background worker: scheduler + job pool.
package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"repwire/internal/briefmedia"
	"repwire/internal/config"
	"repwire/internal/ingest"
	"repwire/internal/jobs"
	"repwire/internal/logging"
	"repwire/internal/postgres"
	"repwire/internal/process"
	"repwire/internal/ratelimit"
	"repwire/internal/sportsdata"
	"repwire/internal/telegram"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	log := logging.New(cfg.LogFormat, cfg.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("db connect failed", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	httpClient := ingest.NewSafeHTTPClient(30 * time.Second)
	enqueue := jobs.NewEnqueuer(db.Job)
	tgClient := telegram.NewClient(cfg.TelegramBotToken)
	tgHandler := telegram.NewHandler(db, tgClient, enqueue, cfg.PublicBaseURL)
	if cfg.TelegramPolling && tgClient.Enabled() {
		go tgClient.Poll(ctx, tgHandler, func(err error) {
			log.Error("telegram polling failed", "err", err)
		})
		log.Info("telegram polling started")
	}

	// A pacer per key pool, not per process. TTS_API_KEY is its own pool with its
	// own quota, so the audio brief must not queue behind article digests for an
	// allowance it never draws on. When TTS_API_KEY is unset the pools are the
	// same keys and the two pacers should be configured to match.
	llmPacer := ratelimit.NewPacer(cfg.LLMMaxCallsPerMinute)
	ttsPacer := ratelimit.NewPacer(cfg.TTSMaxCallsPerMinute)

	handlers := &jobs.Handlers{
		DB:      db,
		Enqueue: enqueue,
		Log:     log,
		RSS:     ingest.NewRSSFetcher(httpClient),
		YouTube: ingest.NewYouTubeFetcher(httpClient, cfg.YouTubeAPIKey, db),
		PMC:     ingest.NewEuropePMCFetcher(httpClient),
		Podcast: ingest.NewPodcastFetcher(httpClient),
		Summarizer: process.NewSummarizer(cfg.LLMAPIKeys, cfg.LLMBaseURL, cfg.LLMModel,
			process.Pricing{InputUSDPerMTok: cfg.LLMInputUSDPerMTok, OutputUSDPerMTok: cfg.LLMOutputUSDPerMTok},
			cfg.LLMDailyBudgetUSD, cfg.LLMMaxCallsPerHour, db.LLM(), llmPacer),
		Telegram:          tgClient,
		Digest:            telegram.NewDigest(db, cfg.PublicBaseURL),
		TTS:               briefmedia.NewTTS(cfg.TTSAPIKeys, cfg.TTSModel, cfg.TTSVoice, ttsPacer),
		MediaDir:          cfg.MediaStorageDir,
		PublicBaseURL:     cfg.MediaPublicBaseURL,
		ScoreThreshold:    cfg.LLMScoreThreshold,
		TranslateMinScore: cfg.LLMTranslateMinScore,
	}

	worker := jobs.NewWorker(hostID(), db.Job, handlers.Register(), log)
	sportsMode := !strings.EqualFold(cfg.SportsDataMode, "off")
	footballToken, pandaToken, sportsDBKey := cfg.FootballDataToken, cfg.PandaScoreToken, cfg.TheSportsDBKey
	if !sportsMode {
		footballToken, pandaToken, sportsDBKey = "", "", ""
	}
	sportsSync := sportsdata.NewSyncer(db.Sports, log,
		sportsdata.NewFootballData(httpClient, footballToken),
		sportsdata.NewPandaScore(httpClient, pandaToken),
		sportsdata.NewOpenF1(httpClient, sportsMode && cfg.OpenF1Enabled),
		sportsdata.NewTheSportsDB(httpClient, sportsDBKey),
	)
	scheduler := jobs.NewScheduler(db, enqueue, log, cfg.LLMTranslateMinScore, cfg.LLMTranslateMaxAge, cfg.EditorialStartHour, cfg.EditorialPicksPerDay, sportsSync)
	scheduler.SetGenerationCapabilities(len(cfg.LLMAPIKeys) > 0, len(cfg.TTSAPIKeys) > 0)

	go scheduler.Run(ctx)
	worker.Run(ctx, cfg.WorkerConcurrency) // blocks until ctx is done

	log.Info("worker exited")
}

// hostID returns a stable-ish worker identifier for job locking.
func hostID() string {
	h, err := os.Hostname()
	if err != nil || h == "" {
		h = "worker"
	}
	return h + "-" + time.Now().Format("150405")
}
