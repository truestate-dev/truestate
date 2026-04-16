package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"gitea.local.vjinx.de/truestate/truestate/backend/internal/api"
	"gitea.local.vjinx.de/truestate/truestate/internal/db"
)

func main() {
	ctx := context.Background()

	dsn := getenv("DATABASE_URL", "postgres://truestate:truestate@localhost:5432/truestate?sslmode=disable")
	addr := getenv("LISTEN_ADDR", ":8080")
	migrationsPath := getenv("MIGRATIONS_PATH", "migrations")

	if err := db.Migrate(dsn, migrationsPath); err != nil {
		slog.Error("migration failed", "err", err)
		os.Exit(1)
	}

	pool, err := db.Connect(ctx, dsn)
	if err != nil {
		slog.Error("db connect failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	router := api.Router(pool)

	slog.Info("truestate api starting", "addr", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
