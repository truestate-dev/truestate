# TrueState — Project Outline

This document captures the original product outline for reference.

## Purpose

TrueState determines the actual vulnerability state of infrastructure by combining
distro-aware security data, inventory state, and baseline intent.

## Core Questions

- Is this system actually vulnerable?
- Is a fix available for this distro and release?
- Does the current state drift from the approved baseline?
- What source says so?

## Version Roadmap

| Version | Focus |
|---|---|
| 0.1 | Core engine: inventory model, Debian/Ubuntu matchers, evaluation engine, API-only |
| 0.2 | Baselines and drift: golden inventories, drift engine, drift-aware findings |
| 0.3 | Proxmox overlay + minimal UI (dashboard, inventory detail, source health) |
| 0.4 | BSI enrichment, evidence panels, source disagreement visibility |
| 0.5 | Multi-tenancy, tenant model, environment tags, simple policies |

## Core Data Domains (stable)

`inventories`, `packages`, `vulnerabilities`, `assertions`, `advisories`,
`relations`, `evaluations`, `source_status`

## Architecture

```
Security Sources
├─ CVE.org
├─ Debian Security Tracker
├─ Ubuntu Security Tracker / USN
├─ Proxmox advisories
└─ BSI enrichment (optional)

Inventory Sources
├─ Host inventory uploads
└─ Golden image inventories

Adapters / Collectors → Normalized Data Model
                       → Evaluation Engine
                         ├─ Vulnerability matching
                         ├─ Drift comparison
                         └─ Source resolution / evidence
                       → REST API → UI / External consumers
```

## Key Design Constraints

- Distro-correct logic: use proper Debian/Ubuntu version comparison
- Source transparency: every finding carries source, timestamp, advisory refs
- Evidence-first: evaluation engine resolves source precedence explicitly
- No agent-based scanning in v0.x scope

## Non-functional Requirements

- **Accuracy**: distro-correct logic, proper version comparison, no silent conflict flattening
- **Explainability**: every finding must be traceable to a source
- **Freshness**: sync status tracked, stale-source risk visible
- **Extensibility**: new source = new adapter + same core model
- **Performance**: evaluation cacheable, no full recompute on every request
