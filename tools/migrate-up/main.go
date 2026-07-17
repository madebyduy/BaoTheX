// Command migrate-up applies every pending .up.sql migration in version order
// and keeps golang-migrate's schema_migrations row in sync, so local work and
// deploys need neither Docker nor the migrate CLI installed.
//
// Each migration runs in its own transaction together with its version bump: a
// failure rolls back both, so the recorded version can never claim a migration
// that did not actually apply. Separate transactions also matter for real
// migrations — 0022 adds enum values that 0023 inserts with, and Postgres
// refuses to use an enum value in the transaction that added it.
//
// Usage: DATABASE_URL=... go run ./tools/migrate-up [migrations-dir]
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

var upFileRE = regexp.MustCompile(`^(\d+)_.*\.up\.sql$`)

type migration struct {
	version int64
	name    string
	path    string
}

func main() {
	dir := "migrations"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		fail(errors.New("DATABASE_URL is required"))
	}

	list, err := load(dir)
	if err != nil {
		fail(err)
	}
	if len(list) == 0 {
		fail(fmt.Errorf("no .up.sql migrations found in %s", dir))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		fail(fmt.Errorf("connect database: %w", err))
	}
	defer conn.Close(context.Background())

	current, dirty, err := currentVersion(ctx, conn)
	if err != nil {
		fail(err)
	}
	if dirty {
		fail(fmt.Errorf("schema_migrations is dirty at version %d: a previous run failed part-way; "+
			"reconcile the schema by hand before migrating again", current))
	}
	fmt.Printf("current version: %d\n", current)

	applied := 0
	for _, m := range list {
		if m.version <= current {
			continue
		}
		if err := apply(ctx, conn, m); err != nil {
			fail(err)
		}
		fmt.Printf("applied %s\n", m.name)
		applied++
		current = m.version
	}
	if applied == 0 {
		fmt.Println("no change")
		return
	}
	fmt.Printf("done: %d migration(s) applied, now at version %d\n", applied, current)
}

// load lists the up migrations sorted by version.
func load(dir string) ([]migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}
	var out []migration
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		match := upFileRE.FindStringSubmatch(e.Name())
		if match == nil {
			continue
		}
		version, err := strconv.ParseInt(match[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("bad version in %s: %w", e.Name(), err)
		}
		out = append(out, migration{version: version, name: e.Name(), path: filepath.Join(dir, e.Name())})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].version < out[j].version })
	return out, nil
}

// currentVersion reads golang-migrate's single-row version table, creating it
// when this is the first run.
func currentVersion(ctx context.Context, conn *pgx.Conn) (int64, bool, error) {
	if _, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version bigint NOT NULL PRIMARY KEY,
		dirty   boolean NOT NULL)`); err != nil {
		return 0, false, fmt.Errorf("ensure schema_migrations: %w", err)
	}
	var version int64
	var dirty bool
	err := conn.QueryRow(ctx, `SELECT version, dirty FROM schema_migrations LIMIT 1`).Scan(&version, &dirty)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("read schema_migrations: %w", err)
	}
	return version, dirty, nil
}

// apply runs one migration and records its version in the same transaction.
func apply(ctx context.Context, conn *pgx.Conn, m migration) error {
	sql, err := os.ReadFile(m.path)
	if err != nil {
		return fmt.Errorf("read %s: %w", m.name, err)
	}
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())

	if _, err := tx.Exec(ctx, string(sql)); err != nil {
		return fmt.Errorf("apply %s: %w", m.name, err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM schema_migrations`); err != nil {
		return fmt.Errorf("clear version: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO schema_migrations (version, dirty) VALUES ($1, false)`, m.version); err != nil {
		return fmt.Errorf("record version %d: %w", m.version, err)
	}
	return tx.Commit(ctx)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "migrate-up:", err)
	os.Exit(1)
}
