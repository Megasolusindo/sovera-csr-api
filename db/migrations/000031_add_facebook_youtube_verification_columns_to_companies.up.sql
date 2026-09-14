-- Add Facebook & YouTube verification columns to company.companies table
ALTER TABLE company.companies 
ADD COLUMN IF NOT EXISTS facebook_status VARCHAR(50) DEFAULT 'UNVERIFIED',
ADD COLUMN IF NOT EXISTS facebook_verified_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS facebook_last_error TEXT,
ADD COLUMN IF NOT EXISTS youtube_status VARCHAR(50) DEFAULT 'UNVERIFIED',
ADD COLUMN IF NOT EXISTS youtube_verified_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS youtube_last_error TEXT;

-- Index for fast batch worker queries
CREATE INDEX IF NOT EXISTS idx_companies_facebook_status ON company.companies (facebook_status);
CREATE INDEX IF NOT EXISTS idx_companies_youtube_status ON company.companies (youtube_status);
