-- Migration 000025 Down: Remove contact person columns

ALTER TABLE organizations
DROP COLUMN IF EXISTS contact_name,
DROP COLUMN IF EXISTS contact_email,
DROP COLUMN IF EXISTS contact_phone;
