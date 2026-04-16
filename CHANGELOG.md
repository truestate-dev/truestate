# Changelog

All notable changes to TrueState are documented here.

## [Unreleased]

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
