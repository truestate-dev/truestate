# Changelog

All notable changes to TrueState are documented here.

## [Unreleased]

### 2026-04-16 (host collector)

- `ingestion/collect.sh` — SSH-based host package collector
  - Accepts `--host`, `--name`, `--type` (host|golden), `--api`, `--ssh-conf`, `--dry-run`
  - Reads `/etc/os-release` to detect platform (debian/ubuntu/proxmox) and release codename
  - Collects all installed packages via `dpkg-query -W` (name + version + arch)
  - POSTs JSON payload to `POST /api/v1/inventories`
  - Uses `~/.ssh/svc-automation.conf` by default; compatible with fleet-check SSH conf files

### 2026-04-16 (UI skeleton)

- `ui/` — React 19 + TypeScript + Tailwind CSS 4 + Vite 8 frontend
  - Typed API client (`src/api/client.ts`) matching all Go model types
  - **InventoryList** — table of all inventories with platform/release/type badges
  - **InventoryDetail** — packages table, evaluation history, "Run Evaluation" button
  - **EvaluationDetail** — summary cards, findings table (pkg/version/CVE/status/fix-version/source, drift flag), drift panel
  - **Sources** — sync status with health indicator per source
  - `/api` proxied to `:8080` in dev; production build outputs to `dist/`
- `ui/.gitignore` excludes `node_modules/` and `dist/`
- Note: run `npm install` in a local directory (CIFS mount doesn't support symlinks); source files committed to repo

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
