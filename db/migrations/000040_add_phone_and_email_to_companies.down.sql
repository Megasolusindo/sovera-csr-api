-- Migration 000040 Down: Remove phone and email contact columns from company.companies table
DROP INDEX IF EXISTS idx_companies_phone;
DROP INDEX IF EXISTS idx_companies_email;

ALTER TABLE company.companies
DROP COLUMN IF EXISTS phone,
DROP COLUMN IF EXISTS email;
