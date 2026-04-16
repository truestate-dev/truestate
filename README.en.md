# TrueState

Distro-correct, source-backed vulnerability and drift intelligence for Debian-based infrastructure.

## Overview

TrueState determines the actual vulnerability state of infrastructure by combining distro-aware security data, inventory state, and baseline intent.

It answers:

- Is this system actually vulnerable?
- Is a fix available for this distro and release?
- Does the current state drift from the approved baseline?
- What source says so?

## Architecture

```
Security Sources          Inventory Sources
├─ CVE.org                ├─ Host inventory uploads
├─ Debian Security Tracker└─ Golden image inventories
├─ Ubuntu Security Tracker / USN
└─ Proxmox advisories
         ↓
Adapters / Collectors
         ↓
Normalized Data Model
         ↓
Evaluation Engine
  ├─ Vulnerability matching
  ├─ Drift comparison
  └─ Source resolution / evidence
         ↓
REST API
         ↓
UI / External consumers
```

## Prerequisites

- Go 1.22+
- PostgreSQL 15+
- Node.js 20+ (for frontend)

## Installation

```bash
# Clone the repository
git clone http://gitea.local.vjinx.de:3000/<org>/truestate.git
cd truestate

# Build backend
go build ./cmd/truestate/...

# Apply database migrations
psql -d truestate < migrations/001_initial.sql
```

## Configuration

| Variable | Description | Default |
|---|---|---|
| `DATABASE_URL` | PostgreSQL connection string | required |
| `LISTEN_ADDR` | HTTP listen address | `:8080` |
| `LOG_LEVEL` | Log verbosity (debug/info/warn/error) | `info` |

## Usage

```bash
# Start API server
./truestate serve

# Evaluate an inventory
curl http://localhost:8080/api/v1/evaluate/{inventory_id}
```

## File Overview

```
backend/
  cmd/api/         — API server entrypoint
  internal/
    api/           — HTTP handlers and routing
    db/            — database layer
ingestion/
  cmd/sync/        — sync worker entrypoint
  adapters/
    cve/           — CVE.org adapter
    debian/        — Debian Security Tracker adapter
    ubuntu/        — Ubuntu Security Tracker / USN adapter
    proxmox/       — Proxmox advisory overlay adapter
internal/
  model/           — shared domain types (used by backend + ingestion)
  engine/          — vulnerability matching and drift evaluation
ui/                — React/TypeScript frontend
migrations/        — SQL migrations
docs/              — design notes, ADRs, runbooks
scripts/           — operational helpers
```

## Version Roadmap

| Version | Focus |
|---|---|
| 0.1 | Core engine: inventory model, Debian/Ubuntu matchers, evaluation engine, API |
| 0.2 | Baselines and drift: golden inventories, drift engine |
| 0.3 | Proxmox overlay + minimal UI |
| 0.4 | BSI enrichment, evidence panels, source trust |
| 0.5 | Multi-tenancy, policy enforcement |

## License

Proprietary — vjinx lab internal tool.
