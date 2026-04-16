package db

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"gitea.local.vjinx.de/truestate/truestate/internal/model"
)

// UpsertVulnerability inserts or updates a CVE record.
func UpsertVulnerability(ctx context.Context, pool *pgxpool.Pool, v model.Vulnerability) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO vulnerabilities (cve_id, summary, published_at, updated_at, fetched_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (cve_id) DO UPDATE SET
			summary     = EXCLUDED.summary,
			published_at = EXCLUDED.published_at,
			updated_at  = EXCLUDED.updated_at,
			fetched_at  = NOW()`,
		v.CVEID, v.Summary, nullTime(v.PublishedAt), nullTime(v.UpdatedAt),
	)
	return err
}

// UpsertAssertion inserts or updates a package status assertion.
// Returns the assertion ID.
func UpsertAssertion(ctx context.Context, pool *pgxpool.Pool, a model.Assertion) (string, error) {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO assertions
			(id, source, cve_id, package_name, platform, release, status, fixed_version, fetched_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT (source, cve_id, package_name, platform, release) DO UPDATE SET
			status        = EXCLUDED.status,
			fixed_version = EXCLUDED.fixed_version,
			fetched_at    = NOW()
		RETURNING id`,
		a.ID, a.Source, a.CVEID, a.PackageName,
		a.Platform, a.Release, a.Status, a.FixedVersion,
	)
	if err != nil {
		return "", fmt.Errorf("upsert assertion: %w", err)
	}
	return a.ID, nil
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
