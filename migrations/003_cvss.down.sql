ALTER TABLE vulnerabilities
    DROP COLUMN IF EXISTS cvss_score,
    DROP COLUMN IF EXISTS cvss_severity,
    DROP COLUMN IF EXISTS cvss_vector;

DELETE FROM source_status WHERE source = 'nvd';
