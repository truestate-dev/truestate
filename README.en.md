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
├─ Debian Security Tracker├─ Host inventory uploads
└─ Ubuntu OVAL / USN      └─ Golden image inventories
         ↓
Adapters / Collectors (ingestion/)
         ↓
Normalized Data Model (internal/model/)
         ↓
Evaluation Engine (internal/engine/)
  ├─ dpkg-correct version comparison
  ├─ Vulnerability matching per platform/release
  └─ Drift comparison against golden baseline
         ↓
REST API (backend/)
         ↓
React UI (ui/)
```

## Prerequisites

- Go 1.22+
- PostgreSQL 15+
- Node.js 20+ (for the UI)

## Quickstart

```bash
git clone http://gitea.local.vjinx.de:3000/truestate-dev/truestate.git
cd truestate
```

### API server

```bash
DATABASE_URL=postgres://truestate:truestate@localhost:5432/truestate?sslmode=disable \
MIGRATIONS_PATH=./migrations \
LISTEN_ADDR=:8080 \
go run ./backend/cmd/api
```

The server applies migrations on startup. API available at `http://localhost:8080/api/v1`.

### Ingestion sync

```bash
# Sync all sources (Debian + Ubuntu)
go run ./ingestion/cmd/sync -source all \
  -db "postgres://truestate:truestate@localhost:5432/truestate?sslmode=disable"

# Sync a single source
go run ./ingestion/cmd/sync -source debian
go run ./ingestion/cmd/sync -source ubuntu
```

### Host collector

Register a host's full package inventory into TrueState via SSH:

```bash
# Load automation cert first
eval $(~/bin/load-automation-cert)

# Register a host
./ingestion/collect.sh --host automation --api http://localhost:8080

# Register a golden image baseline
./ingestion/collect.sh --host ubuntu-golden --type golden --name ubuntu-24.04-golden

# Dry-run (print payload, no POST)
./ingestion/collect.sh --host automation --dry-run

# Use a custom SSH config (e.g. fleet-check conf)
./ingestion/collect.sh --host pve01 \
  --ssh-conf /mnt/skripte/projects/fleet-check/hosts/fleet-ssh.conf
```

`collect.sh` reads `/etc/os-release` to auto-detect platform/release, then collects all
installed packages via `dpkg-query -W` (name + version + arch).

### UI (development)

> **Note — CIFS/NFS mount:** `npm install` creates symlinks in `node_modules/` that fail on CIFS or
> NFS mounts. Install dependencies and run the dev server from a **local** directory:

```bash
# One-time setup (run once per machine)
mkdir -p ~/.local/truestate-ui
rsync -a --exclude node_modules /path/to/truestate/ui/ ~/.local/truestate-ui/
cd ~/.local/truestate-ui
npm install

# Daily dev workflow
rsync -a --exclude node_modules /path/to/truestate/ui/ ~/.local/truestate-ui/
cd ~/.local/truestate-ui
npm run dev          # http://localhost:5173  (proxies /api → :8080)
```

For production builds:

```bash
npm run build        # outputs to dist/
```

## Configuration

| Variable | Description | Default |
|---|---|---|
| `DATABASE_URL` | PostgreSQL connection string | required |
| `LISTEN_ADDR` | HTTP listen address | `:8080` |
| `LOG_LEVEL` | Log verbosity (debug/info/warn/error) | `info` |
| `MIGRATIONS_PATH` | Path to SQL migration files | `./migrations` |

## API Reference

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/inventories` | Create inventory + packages |
| `GET` | `/api/v1/inventories` | List all inventories |
| `GET` | `/api/v1/inventories/{id}` | Get inventory + packages |
| `POST` | `/api/v1/inventories/{id}/relation` | Link host to golden baseline |
| `GET` | `/api/v1/inventories/{id}/evaluations` | List evaluation history |
| `POST` | `/api/v1/evaluate/{id}` | Run + persist evaluation |
| `GET` | `/api/v1/evaluations/{id}` | Retrieve stored evaluation |
| `GET` | `/api/v1/sources` | Source sync status |

## File Overview

```
backend/
  cmd/api/              — API server entrypoint
  internal/api/         — HTTP handlers and routing
ingestion/
  cmd/sync/             — sync worker entrypoint
  collect.sh            — SSH host collector: registers a host's dpkg inventory via API
  adapters/
    debian/             — Debian Security Tracker JSON feed
    ubuntu/             — Ubuntu OVAL XML bzip2 files (focal/jammy/noble)
internal/
  model/                — shared domain types
  engine/               — evaluation engine + dpkg version comparison
  db/                   — database layer (shared by backend + ingestion)
ui/
  src/
    api/client.ts       — typed API client
    pages/              — InventoryList, InventoryDetail, EvaluationDetail, Sources
migrations/             — SQL migrations (001 schema, 002 evaluations)
docs/                   — design notes and outline
```

## Version Roadmap

| Version | Focus | Status |
|---|---|---|
| 0.1 | Core engine: inventory model, Debian/Ubuntu matchers, evaluation engine, API, UI skeleton | **released** |
| 0.2 | Baselines and drift: persist drift history, UI polish | planned |
| 0.3 | Proxmox advisory adapter | planned |
| 0.4 | BSI enrichment, CVSS scores, evidence panels | planned |
| 0.5 | Multi-tenancy, policy enforcement | planned |

## Data Sources

| Source | Feed | Coverage |
|---|---|---|
| Debian Security Tracker | JSON (`security-tracker.debian.org`) | ~247k assertions |
| Ubuntu OVAL | bzip2 XML per release | ~6.25M assertions (focal/jammy/noble) |

## License

Proprietary — vjinx lab internal tool.
