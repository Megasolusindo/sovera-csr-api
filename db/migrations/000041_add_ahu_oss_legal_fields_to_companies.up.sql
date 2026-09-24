-- Migration 000041: Add AHU and OSS legal registration fields to company.companies
ALTER TABLE company.companies
ADD COLUMN IF NOT EXISTS ahu_number VARCHAR(100),
ADD COLUMN IF NOT EXISTS nib VARCHAR(30),
ADD COLUMN IF NOT EXISTS kbli_code VARCHAR(20),
ADD COLUMN IF NOT EXISTS legal_entity_type VARCHAR(50) DEFAULT 'PT';

CREATE INDEX IF NOT EXISTS idx_companies_nib ON company.companies (nib);
CREATE INDEX IF NOT EXISTS idx_companies_ahu ON company.companies (ahu_number);
