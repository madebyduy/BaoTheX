// Command api runs the RepWire HTTP API server.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"repwire/internal/config"
	"repwire/internal/httpapi"
	"repwire/internal/jobs"
	"repwire/internal/logging"
	"repwire/internal/postgres"
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

	enqueue := jobs.NewEnqueuer(db.Job)
	tgClient := telegram.NewClient(cfg.TelegramBotToken)
	tgHook := telegram.NewHandler(db, tgClient, enqueue)

	srv := httpapi.NewServer(db, cfg, log, enqueue, tgClient, tgHook)

	httpServer := &http.Server{
		Addr:              cfg.APIAddr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Info("api listening", "addr", cfg.APIAddr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server failed", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	log.Info("api shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", "err", err)
	}
	log.Info("api stopped")
}
