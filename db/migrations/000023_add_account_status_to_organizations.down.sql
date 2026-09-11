-- Migration 000023 Down: Remove account_status column

ALTER TABLE organizations DROP COLUMN IF EXISTS account_status;
