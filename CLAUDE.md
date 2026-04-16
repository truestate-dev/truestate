# TrueState — Claude Instructions

## Project

TrueState is a Go + React/TypeScript vulnerability and drift intelligence tool for Debian-based infrastructure.

## Tech stack

- `backend/` — Go, PostgreSQL, REST API (net/http or chi router)
- `ingestion/` — Go, source adapter workers (CVE, Debian, Ubuntu, Proxmox)
- `ui/` — React, TypeScript
- Deployment: containerized
- Single Go module at repo root; shared domain types under `internal/model/` and `internal/engine/`

## Key design constraints

- Distro-correct logic: always use proper Debian/Ubuntu version comparison, never flatten away source conflicts
- Source transparency: every finding must carry source, timestamp, and advisory references
- Evidence-first: the evaluation engine resolves source precedence, never silently picks one
- No agent-based scanning in scope for v0.x

## Domain vocabulary

- **Inventory**: package + platform snapshot (host or golden)
- **Assertion**: what a source says about a package in a platform/release (affected/fixed/not-affected)
- **Evaluation**: result of matching an inventory against assertions + optional golden baseline
- **Drift**: deviation between host inventory and its linked golden inventory

## Gitea

- Org: `truestate`
- Repo: `truestate/truestate`
- URL: `http://gitea.local.vjinx.de:3000/truestate/truestate`

## Core domains (stable)

`inventories`, `packages`, `vulnerabilities`, `assertions`, `advisories`, `relations`, `evaluations`, `source_status`

## IaC rules

Read `/mnt/skripte/projects/obsidian/00 Context/60 IaC Rules.md` and `61 IaC Checklist.md` when writing deployment scripts or automation. Not required for Go application code.
