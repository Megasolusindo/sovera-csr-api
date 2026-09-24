-- Migration 000038 Up: Add Instagram URL & verification columns to company.companies table and companies view

ALTER TABLE company.companies
ADD COLUMN IF NOT EXISTS instagram_url TEXT,
ADD COLUMN IF NOT EXISTS instagram_status VARCHAR(50) DEFAULT 'UNVERIFIED',
ADD COLUMN IF NOT EXISTS instagram_verified_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS instagram_last_error TEXT;

CREATE INDEX IF NOT EXISTS idx_companies_instagram_status ON company.companies(instagram_status);

-- Drop and recreate view to avoid "cannot drop columns from view" error
DROP VIEW IF EXISTS companies;

CREATE VIEW companies AS
 SELECT c.id,
    c.name,
    c.legal_name,
    c.slug,
    c.industry_id,
    c.industry_sector,
    c.company_type,
    c.priority_tier,
    c.csr_category,
    c.website,
    c.linkedin_url,
    c.headquarters,
    c.employee_range,
    c.revenue_range,
    c.is_public,
    c.ticker,
    c.parent_company_id,
    c.alias_keywords,
    c.created_at,
    c.updated_at,
    c.linkedin_status,
    c.linkedin_verified_at,
    c.linkedin_last_error,
    c.instagram_url,
    c.instagram_status,
    c.instagram_verified_at,
    c.instagram_last_error
   FROM company.companies c;
