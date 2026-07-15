// Command worker runs the RepWire background worker: scheduler + job pool.
package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"repwire/internal/config"
	"repwire/internal/ingest"
	"repwire/internal/jobs"
	"repwire/internal/logging"
	"repwire/internal/postgres"
	"repwire/internal/process"
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

	httpClient := &http.Client{Timeout: 30 * time.Second}
	enqueue := jobs.NewEnqueuer(db.Job)
	tgClient := telegram.NewClient(cfg.TelegramBotToken)

	handlers := &jobs.Handlers{
		DB:             db,
		Enqueue:        enqueue,
		Log:            log,
		RSS:            ingest.NewRSSFetcher(httpClient),
		YouTube:        ingest.NewYouTubeFetcher(httpClient, cfg.YouTubeAPIKey, db),
		PMC:            ingest.NewEuropePMCFetcher(httpClient),
		Podcast:        ingest.NewPodcastFetcher(httpClient),
		Summarizer:     process.NewSummarizer(cfg.LLMAPIKey, cfg.LLMBaseURL, cfg.LLMModel, cfg.LLMDailyBudgetUSD, cfg.LLMMaxCallsPerHour, db.LLM()),
		Telegram:       tgClient,
		Digest:         telegram.NewDigest(db, cfg.PublicBaseURL),
		ScoreThreshold: cfg.LLMScoreThreshold,
	}

	worker := jobs.NewWorker(hostID(), db.Job, handlers.Register(), log)
	scheduler := jobs.NewScheduler(db, enqueue, log)

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
