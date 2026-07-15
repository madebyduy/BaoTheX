// Command apply-migration applies one SQL migration inside a transaction.
// It is intentionally small so local development does not require Docker.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
)

func main() {
	if len(os.Args) != 2 {
		panic("usage: go run ./tools/apply-migration <migration.sql>")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		panic("DATABASE_URL is required")
	}
	sql, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		panic(fmt.Errorf("connect database: %w", err))
	}
	defer conn.Close(context.Background())
	tx, err := conn.Begin(ctx)
	if err != nil {
		panic(err)
	}
	defer tx.Rollback(context.Background())
	if _, err := tx.Exec(ctx, string(sql)); err != nil {
		panic(fmt.Errorf("execute migration: %w", err))
	}
	if err := tx.Commit(ctx); err != nil {
		panic(err)
	}
	fmt.Println("migration applied successfully")
}
