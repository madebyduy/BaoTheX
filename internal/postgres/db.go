// Package postgres holds the pgx connection pool and all repositories.
package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps a pgx connection pool and exposes the repositories.
type DB struct {
	Pool *pgxpool.Pool

	Content    *ContentRepo
	Source     *SourceRepo
	User       *UserRepo
	Follow     *FollowRepo
	Topic      *TopicRepo
	Entity     *EntityRepo
	Telegram   *TelegramRepo
	Engagement *EngagementRepo
	Job        *JobRepo
	Search     *SearchRepo
	Analysis   *AnalysisRepo
}

// Open creates a connection pool and pings the database.
func Open(ctx context.Context, dsn string) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	// Supabase's session pooler has a small per-project connection ceiling.
	// The API and worker run as separate processes, so keeping five connections
	// per process leaves room for migrations and operational commands.
	cfg.MaxConns = 5
	cfg.MinConns = 1
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute
	cfg.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("new pool: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	db := &DB{Pool: pool}
	db.Content = &ContentRepo{db: db}
	db.Source = &SourceRepo{db: db}
	db.User = &UserRepo{db: db}
	db.Follow = &FollowRepo{db: db}
	db.Topic = &TopicRepo{db: db}
	db.Entity = &EntityRepo{db: db}
	db.Telegram = &TelegramRepo{db: db}
	db.Engagement = &EngagementRepo{db: db}
	db.Job = &JobRepo{db: db}
	db.Search = &SearchRepo{db: db}
	db.Analysis = &AnalysisRepo{db: db}
	return db, nil
}

// Close releases the pool.
func (db *DB) Close() { db.Pool.Close() }

// Ping verifies connectivity (used by /readyz).
func (db *DB) Ping(ctx context.Context) error { return db.Pool.Ping(ctx) }

// WithTx runs fn inside a transaction, committing on success and rolling back
// on error or panic.
func (db *DB) WithTx(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		// Rollback is a no-op if the tx already committed.
		_ = tx.Rollback(ctx)
	}()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
