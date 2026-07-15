// Command seed-admin creates or updates one administrator from environment variables.
// It is safe to run repeatedly and never prints the password or password hash.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"repwire/internal/auth"
	"repwire/internal/domain"
	"repwire/internal/postgres"
)

func main() {
	email := strings.ToLower(strings.TrimSpace(os.Getenv("ADMIN_EMAIL")))
	password := os.Getenv("ADMIN_PASSWORD")
	name := strings.TrimSpace(os.Getenv("ADMIN_NAME"))
	if !strings.Contains(email, "@") || len(password) < 12 {
		panic("ADMIN_EMAIL hợp lệ và ADMIN_PASSWORD từ 12 ký tự là bắt buộc")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := postgres.Open(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		panic(err)
	}
	defer db.Close()

	hash, err := auth.HashPassword(password)
	if err != nil {
		panic(err)
	}
	user, err := db.User.ByEmail(ctx, email)
	if errors.Is(err, domain.ErrNotFound) {
		var displayName *string
		if name != "" {
			displayName = &name
		}
		user, err = db.User.Create(ctx, email, hash, displayName)
	}
	if err != nil {
		panic(err)
	}
	_, err = db.Pool.Exec(ctx, `UPDATE users SET role='admin', password_hash=$2, display_name=COALESCE(NULLIF($3,''),display_name) WHERE id=$1`, user.ID, hash, name)
	if err != nil {
		panic(err)
	}
	fmt.Printf("admin ready: %s (id=%d)\n", email, user.ID)
}
