-- Add CVSS scoring columns to vulnerabilities.
-- Populated by the NVD adapter (source: nvd.nist.gov).
-- cvss_score uses the highest available version (v3.1 > v3.0 > v2).

ALTER TABLE vulnerabilities
    ADD COLUMN IF NOT EXISTS cvss_score    FLOAT   NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS cvss_severity TEXT    NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS cvss_vector   TEXT    NOT NULL DEFAULT '';

-- Register NVD as a sync source.
INSERT INTO source_status (source) VALUES ('nvd') ON CONFLICT DO NOTHING;
