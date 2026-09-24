-- Migration 000041 Down
DROP INDEX IF EXISTS idx_companies_ahu;
DROP INDEX IF EXISTS idx_companies_nib;

ALTER TABLE company.companies
DROP COLUMN IF EXISTS legal_entity_type,
DROP COLUMN IF EXISTS kbli_code,
DROP COLUMN IF EXISTS nib,
DROP COLUMN IF EXISTS ahu_number;
