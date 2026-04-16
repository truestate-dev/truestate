# Changelog

All notable changes to TrueState are documented here.

## [Unreleased]

### 2026-04-16 (inventory list stats + delete + pagination)

- `internal/db/inventory.go` — added `InventoryWithStats` struct; `ListInventoriesWithStats` lateral-join query (package count + most-recent evaluation summary per inventory); `DeleteInventory` (cascades to packages, evaluations, findings, drift_items)
- `backend/internal/api/handler.go` — `GET /api/v1/inventories` now returns `[]InventoryWithStats`; new `DELETE /api/v1/inventories/{id}` endpoint
- `ui/src/api/client.ts` — added `InventoryWithStats` type; `deleteInventory` API call
- `ui/src/pages/InventoryList.tsx` — split hosts/goldens into sections; new columns: Packages count, Findings (linked to last eval, colour-coded), Last Evaluated date
- `ui/src/pages/InventoryDetail.tsx` — packages search (name/version); package table pagination (100/page); inline delete with confirm/cancel flow (redirects to list on success)
- `ui/src/pages/EvaluationDetail.tsx` — findings table pagination (100/page); page resets on filter changes

### 2026-04-16 (Proxmox platform mapping + UI filters)

- `internal/db/assertion.go` — added `GetAssertionsForPackageOnPlatforms`: queries assertions across multiple platforms in one query (`platform = ANY($3)`); `GetAssertionsForPackage` now delegates to it
- `internal/engine/engine.go` — Proxmox platform fallback: when `platform=proxmox`, also queries `platform=debian` assertions for the same release codename (PVE 8.x = Debian bookworm, etc.), giving full CVE coverage for base system packages on PVE hosts
- `ui/src/pages/EvaluationDetail.tsx` — findings filter bar:
  - Text search on package name or CVE ID
  - Severity toggles (CRITICAL / HIGH / MEDIUM / LOW / Unscored) — multi-select chips
  - Status toggles (affected / fixed / under_review / not_affected) — multi-select chips
  - Drift-only toggle
  - Live counter: "Showing N of M findings"
  - "Clear filters" shortcut link
  - All filtering is client-side (no extra API calls)

### 2026-04-16 (CVSS severity enrichment)

- Migration 003: added `cvss_score FLOAT`, `cvss_severity TEXT`, `cvss_vector TEXT` to `vulnerabilities`; added `nvd` source_status row
- `ingestion/adapters/nvd/nvd.go` — NVD API 2.0 paginated adapter
  - Fetches all CVEs (2000/page), extracts best CVSS score (v3.1 > v3.0 > v2)
  - Rate limiting: 6.2s/req without key, 0.7s/req with `NVD_API_KEY` env var
  - Backoff on 429/403 (35s retry)
  - Progress logged every 10 pages
- `internal/db/vulnerability.go` — new file
  - `BulkUpdateVulnerabilityCVSS`: temp table + CopyFrom + single UPDATE for efficiency
  - `GetCVSSForCVEs`: batch lookup by CVE ID array for engine enrichment
- `internal/model/model.go` — added `CVSSScore`, `CVSSSeverity`, `CVSSVector` to `Vulnerability`; `CVSSScore`, `CVSSSeverity` to `Finding`; `SourceNVD` constant
- `internal/engine/engine.go` — after building findings, batch-enriches CVSS scores via single DB query
- `ingestion/cmd/sync/main.go` — added `-source nvd` and `syncNVD()` function
- `ui/src/api/client.ts` — added `CVSSSeverity` type, `cvss_score` and `cvss_severity` fields to `Finding`
- `ui/src/pages/EvaluationDetail.tsx` — severity badge (CRITICAL/HIGH/MEDIUM/LOW with score), findings sorted by severity descending, 4-card summary (total/critical/high/drift), drift-related count in section header

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
