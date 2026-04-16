-- Evaluations: persisted results of running the evaluation engine against an inventory
CREATE TABLE evaluations (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    inventory_id        UUID        NOT NULL REFERENCES inventories(id) ON DELETE CASCADE,
    golden_inventory_id UUID        REFERENCES inventories(id) ON DELETE SET NULL,
    evaluated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_evaluations_inventory_id ON evaluations(inventory_id);
CREATE INDEX idx_evaluations_evaluated_at ON evaluations(evaluated_at DESC);

-- Findings: individual vulnerability results within an evaluation
CREATE TABLE findings (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    evaluation_id   UUID        NOT NULL REFERENCES evaluations(id) ON DELETE CASCADE,
    inventory_id    UUID        NOT NULL,
    package_name    TEXT        NOT NULL,
    version         TEXT        NOT NULL,
    cve_id          TEXT        NOT NULL REFERENCES vulnerabilities(cve_id) ON DELETE CASCADE,
    status          TEXT        NOT NULL CHECK (status IN ('affected', 'fixed', 'not_affected', 'under_review')),
    fixed_version   TEXT        NOT NULL DEFAULT '',
    source          TEXT        NOT NULL,
    fetched_at      TIMESTAMPTZ NOT NULL,
    drift_related   BOOLEAN     NOT NULL DEFAULT FALSE
);

CREATE INDEX idx_findings_evaluation_id ON findings(evaluation_id);
CREATE INDEX idx_findings_cve_id        ON findings(cve_id);
CREATE INDEX idx_findings_package_name  ON findings(package_name);

-- Drift items: package deviations recorded alongside each evaluation
CREATE TABLE drift_items (
    id               UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    evaluation_id    UUID    NOT NULL REFERENCES evaluations(id) ON DELETE CASCADE,
    kind             TEXT    NOT NULL CHECK (kind IN ('missing_package', 'extra_package', 'version_mismatch')),
    package_name     TEXT    NOT NULL,
    host_version     TEXT    NOT NULL DEFAULT '',
    baseline_version TEXT    NOT NULL DEFAULT ''
);

CREATE INDEX idx_drift_items_evaluation_id ON drift_items(evaluation_id);
