package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"gitea.local.vjinx.de/truestate/truestate/internal/db"
	"gitea.local.vjinx.de/truestate/truestate/ingestion/adapters/debian"
	"gitea.local.vjinx.de/truestate/truestate/ingestion/adapters/nvd"
	"gitea.local.vjinx.de/truestate/truestate/ingestion/adapters/ubuntu"
	"gitea.local.vjinx.de/truestate/truestate/internal/model"
)

func main() {
	source := flag.String("source", "all", "Source to sync: all|debian|ubuntu|nvd")
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
	if *source == "all" || *source == "nvd" {
		if err := syncNVD(ctx, pool); err != nil {
			slog.Error("nvd sync failed", "err", err)
		}
	}
}

func syncDebian(ctx context.Context, pool *pgxpool.Pool) error {
	result, err := debian.Fetch(ctx)
	if err != nil {
		return err
	}

	slog.Info("debian: writing vulnerabilities", "count", len(result.Vulnerabilities))
	if err := db.BulkUpsertVulnerabilities(ctx, pool, result.Vulnerabilities); err != nil {
		return fmt.Errorf("debian: bulk upsert vulns: %w", err)
	}

	slog.Info("debian: writing assertions", "count", len(result.Assertions))
	if err := db.BulkUpsertAssertions(ctx, pool, result.Assertions); err != nil {
		return fmt.Errorf("debian: bulk upsert assertions: %w", err)
	}

	slog.Info("debian: sync complete", "assertions", len(result.Assertions))
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

	slog.Info("ubuntu: writing vulnerabilities", "count", len(result.Vulnerabilities))
	if err := db.BulkUpsertVulnerabilities(ctx, pool, result.Vulnerabilities); err != nil {
		return fmt.Errorf("ubuntu: bulk upsert vulns: %w", err)
	}

	slog.Info("ubuntu: writing assertions", "count", len(result.Assertions))
	if err := db.BulkUpsertAssertions(ctx, pool, result.Assertions); err != nil {
		return fmt.Errorf("ubuntu: bulk upsert assertions: %w", err)
	}

	slog.Info("ubuntu: sync complete", "assertions", len(result.Assertions))
	return db.UpdateSourceStatus(ctx, pool, model.SourceStatus{
		Source:      model.SourceUbuntu,
		RecordCount: len(result.Assertions),
	})
}

func syncNVD(ctx context.Context, pool *pgxpool.Pool) error {
	records, err := nvd.Fetch(ctx)
	if err != nil {
		return err
	}

	slog.Info("nvd: updating vulnerabilities with CVSS scores", "count", len(records))
	updated, err := db.BulkUpdateVulnerabilityCVSS(ctx, pool, records)
	if err != nil {
		return fmt.Errorf("nvd: bulk update cvss: %w", err)
	}

	slog.Info("nvd: sync complete", "cvss_records_fetched", len(records), "vulnerabilities_updated", updated)
	return db.UpdateSourceStatus(ctx, pool, model.SourceStatus{
		Source:      model.SourceNVD,
		RecordCount: int(updated),
	})
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
