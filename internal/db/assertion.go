package db

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"gitea.local.vjinx.de/truestate/truestate/internal/model"
)

const batchSize = 2000

// BulkUpsertVulnerabilities inserts or updates CVE records in batches.
func BulkUpsertVulnerabilities(ctx context.Context, pool *pgxpool.Pool, vulns []model.Vulnerability) error {
	now := time.Now().UTC()
	for i := 0; i < len(vulns); i += batchSize {
		end := i + batchSize
		if end > len(vulns) {
			end = len(vulns)
		}
		batch := &pgx.Batch{}
		for _, v := range vulns[i:end] {
			batch.Queue(`
				INSERT INTO vulnerabilities (cve_id, summary, published_at, updated_at, fetched_at)
				VALUES ($1, $2, $3, $4, $5)
				ON CONFLICT (cve_id) DO UPDATE SET
					summary      = EXCLUDED.summary,
					published_at = EXCLUDED.published_at,
					updated_at   = EXCLUDED.updated_at,
					fetched_at   = EXCLUDED.fetched_at`,
				v.CVEID, v.Summary, nullTime(v.PublishedAt), nullTime(v.UpdatedAt), now,
			)
		}
		if err := pool.SendBatch(ctx, batch).Close(); err != nil {
			return fmt.Errorf("bulk upsert vulnerabilities batch %d: %w", i/batchSize, err)
		}
		slog.Debug("vulnerabilities batch done", "offset", i, "count", end-i)
	}
	return nil
}

// BulkUpsertAssertions inserts or updates assertions in batches using a temp table
// for maximum throughput on large feeds (250k+ rows).
func BulkUpsertAssertions(ctx context.Context, pool *pgxpool.Pool, assertions []model.Assertion) error {
	now := time.Now().UTC()

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire conn: %w", err)
	}
	defer conn.Release()

	// Create a temp table matching assertions structure (no PK overhead during load).
	_, err = conn.Exec(ctx, `DROP TABLE IF EXISTS tmp_assertions`)
	if err != nil {
		return fmt.Errorf("drop old temp table: %w", err)
	}
	_, err = conn.Exec(ctx, `
		CREATE TEMP TABLE tmp_assertions (
			source        TEXT,
			cve_id        TEXT,
			package_name  TEXT,
			platform      TEXT,
			release       TEXT,
			status        TEXT,
			fixed_version TEXT,
			fetched_at    TIMESTAMPTZ
		)`)
	if err != nil {
		return fmt.Errorf("create temp table: %w", err)
	}

	// Bulk load into temp table via CopyFrom.
	rows := make([][]any, len(assertions))
	for i, a := range assertions {
		rows[i] = []any{
			string(a.Source), a.CVEID, a.PackageName,
			string(a.Platform), a.Release,
			string(a.Status), a.FixedVersion, now,
		}
	}

	_, err = conn.Conn().CopyFrom(ctx,
		pgx.Identifier{"tmp_assertions"},
		[]string{"source", "cve_id", "package_name", "platform", "release", "status", "fixed_version", "fetched_at"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return fmt.Errorf("copy into temp table: %w", err)
	}
	slog.Debug("copied to temp table", "rows", len(assertions))

	// Upsert from temp into assertions in one statement.
	// DISTINCT ON deduplicates rows with the same conflict key that may arrive
	// when the same (source, cve_id, package, platform, release) tuple appears
	// in multiple definitions (e.g. Ubuntu USN vs ESM variants).
	_, err = conn.Exec(ctx, `
		INSERT INTO assertions
			(id, source, cve_id, package_name, platform, release, status, fixed_version, fetched_at)
		SELECT
			gen_random_uuid(), source, cve_id, package_name, platform, release, status, fixed_version, fetched_at
		FROM (
			SELECT DISTINCT ON (source, cve_id, package_name, platform, release)
				source, cve_id, package_name, platform, release, status, fixed_version, fetched_at
			FROM tmp_assertions
			ORDER BY source, cve_id, package_name, platform, release
		) deduped
		ON CONFLICT (source, cve_id, package_name, platform, release) DO UPDATE SET
			status        = EXCLUDED.status,
			fixed_version = EXCLUDED.fixed_version,
			fetched_at    = EXCLUDED.fetched_at`)
	if err != nil {
		return fmt.Errorf("upsert from temp: %w", err)
	}

	return nil
}

// UpsertAssertion inserts or updates a single assertion (used by small callers).
func UpsertAssertion(ctx context.Context, pool *pgxpool.Pool, a model.Assertion) (string, error) {
	now := time.Now().UTC()
	var id string
	err := pool.QueryRow(ctx, `
		INSERT INTO assertions
			(id, source, cve_id, package_name, platform, release, status, fixed_version, fetched_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (source, cve_id, package_name, platform, release) DO UPDATE SET
			status        = EXCLUDED.status,
			fixed_version = EXCLUDED.fixed_version,
			fetched_at    = EXCLUDED.fetched_at
		RETURNING id`,
		a.Source, a.CVEID, a.PackageName,
		a.Platform, a.Release, a.Status, a.FixedVersion, now,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("upsert assertion: %w", err)
	}
	return id, nil
}

// GetAssertionsForPackage returns all assertions for a package on a given platform/release.
func GetAssertionsForPackage(ctx context.Context, pool *pgxpool.Pool,
	pkg, platform, release string) ([]model.Assertion, error) {

	rows, err := pool.Query(ctx, `
		SELECT id, source, cve_id, package_name, platform, release, status, fixed_version, fetched_at
		FROM assertions
		WHERE package_name = $1 AND platform = $2 AND release = $3
		ORDER BY source, cve_id`, pkg, platform, release)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.Assertion
	for rows.Next() {
		var a model.Assertion
		if err := rows.Scan(&a.ID, &a.Source, &a.CVEID, &a.PackageName,
			&a.Platform, &a.Release, &a.Status, &a.FixedVersion, &a.FetchedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// UpdateSourceStatus records the result of a sync run.
func UpdateSourceStatus(ctx context.Context, pool *pgxpool.Pool, s model.SourceStatus) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO source_status (source, last_sync_at, record_count, error)
		VALUES ($1, NOW(), $2, $3)
		ON CONFLICT (source) DO UPDATE SET
			last_sync_at = NOW(),
			record_count = EXCLUDED.record_count,
			error        = EXCLUDED.error`,
		s.Source, s.RecordCount, s.Error,
	)
	return err
}

// ListSourceStatus returns the sync state of all sources.
func ListSourceStatus(ctx context.Context, pool *pgxpool.Pool) ([]model.SourceStatus, error) {
	rows, err := pool.Query(ctx, `
		SELECT source, last_sync_at, record_count, error FROM source_status ORDER BY source`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.SourceStatus
	for rows.Next() {
		var s model.SourceStatus
		if err := rows.Scan(&s.Source, &s.LastSyncAt, &s.RecordCount, &s.Error); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
