-- Migration 000040 Up: Add phone and email contact columns to company.companies table
ALTER TABLE company.companies
ADD COLUMN IF NOT EXISTS phone VARCHAR(50),
ADD COLUMN IF NOT EXISTS email VARCHAR(255);

CREATE INDEX IF NOT EXISTS idx_companies_phone ON company.companies (phone) WHERE phone IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_companies_email ON company.companies (email) WHERE email IS NOT NULL;
