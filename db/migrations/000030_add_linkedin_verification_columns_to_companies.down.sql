-- Migration 000030 Down: Revert LinkedIn verification columns

DROP INDEX IF EXISTS idx_companies_linkedin_status;

ALTER TABLE companies
DROP COLUMN IF EXISTS linkedin_status,
DROP COLUMN IF EXISTS linkedin_verified_at,
DROP COLUMN IF EXISTS linkedin_last_error;
