-- Core schema for TrueState v0.1

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Inventories: host or golden image package snapshots
CREATE TABLE inventories (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT        NOT NULL,
    type        TEXT        NOT NULL CHECK (type IN ('host', 'golden')),
    platform    TEXT        NOT NULL CHECK (platform IN ('debian', 'ubuntu', 'proxmox')),
    release     TEXT        NOT NULL,  -- e.g. 'bookworm', 'jammy'
    metadata    JSONB       NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Packages: installed packages within an inventory
CREATE TABLE packages (
    id              UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    inventory_id    UUID    NOT NULL REFERENCES inventories(id) ON DELETE CASCADE,
    name            TEXT    NOT NULL,
    version         TEXT    NOT NULL,
    arch            TEXT    NOT NULL DEFAULT '',
    UNIQUE (inventory_id, name, arch)
);

CREATE INDEX idx_packages_inventory_id ON packages(inventory_id);
CREATE INDEX idx_packages_name ON packages(name);

-- Relations: links a host inventory to its golden baseline
CREATE TABLE inventory_relations (
    host_inventory_id   UUID NOT NULL REFERENCES inventories(id) ON DELETE CASCADE,
    golden_inventory_id UUID NOT NULL REFERENCES inventories(id) ON DELETE CASCADE,
    PRIMARY KEY (host_inventory_id, golden_inventory_id)
);

-- Vulnerabilities: CVE identity and metadata
CREATE TABLE vulnerabilities (
    cve_id          TEXT        PRIMARY KEY,
    summary         TEXT        NOT NULL DEFAULT '',
    published_at    TIMESTAMPTZ,
    updated_at      TIMESTAMPTZ,
    fetched_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE vulnerability_references (
    cve_id  TEXT NOT NULL REFERENCES vulnerabilities(cve_id) ON DELETE CASCADE,
    url     TEXT NOT NULL,
    PRIMARY KEY (cve_id, url)
);

-- Assertions: what a source says about a package+CVE for a platform/release
CREATE TABLE assertions (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    source          TEXT        NOT NULL,
    cve_id          TEXT        NOT NULL REFERENCES vulnerabilities(cve_id) ON DELETE CASCADE,
    package_name    TEXT        NOT NULL,
    platform        TEXT        NOT NULL,
    release         TEXT        NOT NULL,
    status          TEXT        NOT NULL CHECK (status IN ('affected', 'fixed', 'not_affected', 'under_review')),
    fixed_version   TEXT        NOT NULL DEFAULT '',
    fetched_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (source, cve_id, package_name, platform, release)
);

CREATE INDEX idx_assertions_cve_id ON assertions(cve_id);
CREATE INDEX idx_assertions_package ON assertions(package_name, platform, release);
CREATE INDEX idx_assertions_source ON assertions(source);

CREATE TABLE assertion_refs (
    assertion_id    UUID NOT NULL REFERENCES assertions(id) ON DELETE CASCADE,
    url             TEXT NOT NULL,
    PRIMARY KEY (assertion_id, url)
);

-- Source status: health and freshness of each data source
CREATE TABLE source_status (
    source          TEXT        PRIMARY KEY,
    last_sync_at    TIMESTAMPTZ,
    record_count    INT         NOT NULL DEFAULT 0,
    error           TEXT        NOT NULL DEFAULT ''
);

INSERT INTO source_status (source) VALUES
    ('cve.org'),
    ('debian_tracker'),
    ('ubuntu_tracker'),
    ('proxmox'),
    ('bsi');
