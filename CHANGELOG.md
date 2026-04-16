# Changelog

All notable changes to TrueState are documented here.

## [Unreleased]

### 2026-04-16 (evaluation persistence)

- Migration 002: added `evaluations`, `findings`, and `drift_items` tables
- `internal/db/evaluation.go`: SaveEvaluation (CopyFrom bulk insert), GetEvaluation, ListEvaluationsForInventory
- API changes:
  - `POST /api/v1/evaluate/{id}` — triggers evaluation, persists result, returns 201 + full evaluation
  - `GET /api/v1/evaluations/{id}` — retrieves a stored evaluation by ID
  - `GET /api/v1/inventories/{id}/evaluations` — lists all evaluations for an inventory (summary, newest first)

## [v0.1.0] — 2026-04-16

### 2026-04-16

- Initial project scaffold: directory structure, README, CHANGELOG, go.mod, core model
- Defined core data model: Inventory, Package, Vulnerability, Assertion, Evaluation
- Added project outline to docs/

### 2026-04-16 (continued)

- Added Go dependencies: chi, pgx/v5, golang-migrate, uuid
- SQL migrations: inventories, packages, inventory_relations, vulnerabilities, assertions, source_status
- internal/db: connection, migrations, inventory CRUD, assertion upsert, source status
- internal/engine: evaluation engine with vulnerability matching and drift detection
- ingestion/adapters/debian: Debian Security Tracker JSON feed ingestion
- ingestion/adapters/ubuntu: Ubuntu CVE JSON feed ingestion (paginated)
- backend/internal/api: chi router with inventory, evaluate, and source endpoints
- Moved db package to internal/db (shared between backend and ingestion)

### 2026-04-16 (fixes)

- Fixed NULL scan panic: SourceStatus.LastSyncAt changed to *time.Time
- Fixed NOT NULL violation: nil Metadata map defaulted to empty map on inventory insert
- End-to-end test passed: inventory creation, relations, evaluate endpoint, drift detection

### 2026-04-16 (version comparison)

- Implemented proper Debian version comparison (dpkg Policy §5.6.12)
  - epoch:upstream-revision format
  - alternating non-digit/digit run algorithm
  - tilde pre-release ordering (1.0~rc1 < 1.0)
  - correct numeric comparison (1.10 > 1.9)
- 25 test cases covering epochs, tildes, revisions, real Ubuntu/Debian versions

### 2026-04-16 (performance + Ubuntu fix)

- Replaced single-row assertion upserts with CopyFrom+temp table bulk upsert (247k rows in 14s vs 2+ hours)
- Added BulkUpsertVulnerabilities using pgx SendBatch (2000-row batches)
- Fixed Ubuntu adapter: total_results field name, reduced page limit to 100 (API rejects >100)

### 2026-04-16 (Ubuntu OVAL rewrite)

- Replaced Ubuntu JSON API adapter with OVAL XML bulk file ingestion
  - Source: security-metadata.canonical.com/oval/com.ubuntu.<release>.usn.oval.xml.bz2
  - Covers focal, jammy, noble (6.25M assertions across 3 releases)
- Fixed CVE extraction: now reads `<reference source="CVE" ref_id="...">` attributes
- Fixed package extraction: descriptions list packages as `pkgname - version` on one
  space-separated line after "following package versions:"; regex now parses correctly
- Added DISTINCT ON deduplication in bulk upsert SQL to handle same package/CVE appearing
  in multiple USN definitions (ESM vs main channel variants)
