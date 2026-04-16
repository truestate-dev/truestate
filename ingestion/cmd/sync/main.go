package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"gitea.local.vjinx.de/truestate/truestate/internal/db"
	"gitea.local.vjinx.de/truestate/truestate/ingestion/adapters/debian"
	"gitea.local.vjinx.de/truestate/truestate/ingestion/adapters/ubuntu"
	"gitea.local.vjinx.de/truestate/truestate/internal/model"
)

func main() {
	source := flag.String("source", "all", "Source to sync: all|debian|ubuntu")
	dsn := flag.String("db",
		getenv("DATABASE_URL", "postgres://truestate:truestate@localhost:5432/truestate?sslmode=disable"),
		"PostgreSQL DSN")
	flag.Parse()

	ctx := context.Background()

	pool, err := db.Connect(ctx, *dsn)
	if err != nil {
		slog.Error("db connect", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if *source == "all" || *source == "debian" {
		if err := syncDebian(ctx, pool); err != nil {
			slog.Error("debian sync failed", "err", err)
		}
	}
	if *source == "all" || *source == "ubuntu" {
		if err := syncUbuntu(ctx, pool); err != nil {
			slog.Error("ubuntu sync failed", "err", err)
		}
	}
}

func syncDebian(ctx context.Context, pool *pgxpool.Pool) error {
	result, err := debian.Fetch(ctx)
	if err != nil {
		return err
	}

	for _, v := range result.Vulnerabilities {
		if err := db.UpsertVulnerability(ctx, pool, v); err != nil {
			slog.Warn("upsert vulnerability", "cve", v.CVEID, "err", err)
		}
	}
	for _, a := range result.Assertions {
		if _, err := db.UpsertAssertion(ctx, pool, a); err != nil {
			slog.Warn("upsert assertion", "cve", a.CVEID, "pkg", a.PackageName, "err", err)
		}
	}

	return db.UpdateSourceStatus(ctx, pool, model.SourceStatus{
		Source:      model.SourceDebian,
		RecordCount: len(result.Assertions),
	})
}

func syncUbuntu(ctx context.Context, pool *pgxpool.Pool) error {
	result, err := ubuntu.Fetch(ctx)
	if err != nil {
		return err
	}

	for _, v := range result.Vulnerabilities {
		if err := db.UpsertVulnerability(ctx, pool, v); err != nil {
			slog.Warn("upsert vulnerability", "cve", v.CVEID, "err", err)
		}
	}
	for _, a := range result.Assertions {
		if _, err := db.UpsertAssertion(ctx, pool, a); err != nil {
			slog.Warn("upsert assertion", "cve", a.CVEID, "pkg", a.PackageName, "err", err)
		}
	}

	return db.UpdateSourceStatus(ctx, pool, model.SourceStatus{
		Source:      model.SourceUbuntu,
		RecordCount: len(result.Assertions),
	})
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
