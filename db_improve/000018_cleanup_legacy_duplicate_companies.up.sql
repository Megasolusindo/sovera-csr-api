-- =============================================================================
-- 000018_cleanup_legacy_duplicate_companies.up.sql
--
-- Tujuan: Mereroute relasi dan menghapus baris duplikat legacy ber-suffix [LEGACY <uuid>]
--         yang dihasilkan dari migrasi awal, sehingga data perusahaan di company.companies
--         bersih dari nama duplikat dan tag [LEGACY].
-- =============================================================================

BEGIN;

-- 1. Reroute company_enriched_programs references
UPDATE company_enriched_programs p
SET company_id = c_canonical.id
FROM company.companies c_legacy
JOIN company.companies c_canonical 
  ON LOWER(TRIM(REGEXP_REPLACE(c_legacy.name, '\s*\[LEGACY\s+[a-f0-9-]+\]', '', 'gi'))) = LOWER(TRIM(c_canonical.name))
 AND c_canonical.id != c_legacy.id
 AND c_canonical.name NOT LIKE '%[LEGACY%'
WHERE p.company_id = c_legacy.id;

-- 2. Reroute intelligence.company_signals references
UPDATE intelligence.company_signals s
SET company_id = c_canonical.id
FROM company.companies c_legacy
JOIN company.companies c_canonical 
  ON LOWER(TRIM(REGEXP_REPLACE(c_legacy.name, '\s*\[LEGACY\s+[a-f0-9-]+\]', '', 'gi'))) = LOWER(TRIM(c_canonical.name))
 AND c_canonical.id != c_legacy.id
 AND c_canonical.name NOT LIKE '%[LEGACY%'
WHERE s.company_id = c_legacy.id;

-- 3. Delete orphaned LEGACY duplicate companies
DELETE FROM company.companies c
WHERE c.name LIKE '%[LEGACY%'
  AND NOT EXISTS (SELECT 1 FROM crawling_targets t WHERE t.company_id = c.id)
  AND NOT EXISTS (SELECT 1 FROM company_enriched_programs p WHERE p.company_id = c.id)
  AND NOT EXISTS (SELECT 1 FROM intelligence.company_signals s WHERE s.company_id = c.id);

-- 4. Strip [LEGACY ...] suffix for any remaining unique legacy entries
UPDATE company.companies
SET name = TRIM(REGEXP_REPLACE(name, '\s*\[LEGACY\s+[a-f0-9-]+\]', '', 'gi')),
    updated_at = NOW()
WHERE name LIKE '%[LEGACY%';

COMMIT;
