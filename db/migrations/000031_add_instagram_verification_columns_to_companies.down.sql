-- Migration 000031 Down: Revert Instagram verification columns

DROP INDEX IF EXISTS idx_companies_instagram_status;

ALTER TABLE companies
DROP COLUMN IF EXISTS instagram_status,
DROP COLUMN IF EXISTS instagram_verified_at,
DROP COLUMN IF EXISTS instagram_last_error;
