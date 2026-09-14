-- Migration 000034: Create Two-Sided Marketplace Schema (Claims, Opportunities, Proposals, Invitations)

-- 1. Create Enum Types
DO $$ BEGIN
    CREATE TYPE tenant_type_enum AS ENUM ('ORGANIZATION', 'CORPORATE', 'ADMIN');
EXCEPTION WHEN duplicate_object THEN null; END $$;

DO $$ BEGIN
    CREATE TYPE claim_status_enum AS ENUM ('PENDING', 'APPROVED', 'REJECTED', 'REVOKED');
EXCEPTION WHEN duplicate_object THEN null; END $$;

DO $$ BEGIN
    CREATE TYPE claim_method_enum AS ENUM ('CORPORATE_EMAIL', 'DOCUMENT_UPLOAD');
EXCEPTION WHEN duplicate_object THEN null; END $$;

DO $$ BEGIN
    CREATE TYPE proposal_status_enum AS ENUM ('SUBMITTED', 'UNDER_REVIEW', 'MEETING', 'ACCEPTED', 'REJECTED');
EXCEPTION WHEN duplicate_object THEN null; END $$;

-- 2. Enhance organizations table (Safe for 105 existing records)
ALTER TABLE organizations
    ADD COLUMN IF NOT EXISTS type tenant_type_enum NOT NULL DEFAULT 'ORGANIZATION',
    ADD COLUMN IF NOT EXISTS company_id UUID REFERENCES company.companies(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS slug VARCHAR(255),
    ADD COLUMN IF NOT EXISTS logo_url TEXT,
    ADD COLUMN IF NOT EXISTS is_verified BOOLEAN DEFAULT FALSE;

-- Ensure all existing records are marked as ORGANIZATION
UPDATE organizations SET type = 'ORGANIZATION' WHERE type IS NULL OR type = 'ORGANIZATION';

-- System Admin Account
UPDATE organizations SET type = 'ADMIN', is_verified = TRUE WHERE id = '00000000-0000-0000-0000-000000000000';

CREATE INDEX IF NOT EXISTS idx_organizations_type ON organizations(type);
CREATE INDEX IF NOT EXISTS idx_organizations_company_id ON organizations(company_id);

-- 3. Enhance company.companies table for corporate claims
ALTER TABLE company.companies
    ADD COLUMN IF NOT EXISTS is_claimed BOOLEAN DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS corporate_domain VARCHAR(100),
    ADD COLUMN IF NOT EXISTS claimed_by_tenant_id UUID REFERENCES organizations(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_companies_is_claimed ON company.companies(is_claimed);
CREATE INDEX IF NOT EXISTS idx_companies_corporate_domain ON company.companies(corporate_domain);

-- 4. Update Public View `companies`
DROP VIEW IF EXISTS companies CASCADE;

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
    c.instagram_last_error,
    c.is_claimed,
    c.corporate_domain,
    c.claimed_by_tenant_id
   FROM company.companies c;

-- 5. Create Company Claims Table
CREATE TABLE IF NOT EXISTS company_claims (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES company.companies(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    requested_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    work_email VARCHAR(255),
    document_proof_url TEXT,
    method claim_method_enum NOT NULL,
    status claim_status_enum DEFAULT 'PENDING',
    
    verification_notes TEXT,
    verified_at TIMESTAMPTZ,
    verified_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_company_claims_company_id ON company_claims(company_id);
CREATE INDEX IF NOT EXISTS idx_company_claims_tenant_id ON company_claims(tenant_id);
CREATE INDEX IF NOT EXISTS idx_company_claims_status ON company_claims(status);

-- 6. Create CSR Opportunities Table (Corporate Grant / RFP Openings)
CREATE TABLE IF NOT EXISTS csr_opportunities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES company.companies(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    
    title VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100) NOT NULL, -- Education, Health, Environment, Economic, Social
    target_location VARCHAR(100),
    budget_amount NUMERIC(15, 2),
    open_until TIMESTAMPTZ,
    status VARCHAR(50) DEFAULT 'OPEN', -- OPEN, CLOSED, IN_REVIEW
    
    created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_opportunities_company_id ON csr_opportunities(company_id);
CREATE INDEX IF NOT EXISTS idx_opportunities_tenant_id ON csr_opportunities(tenant_id);
CREATE INDEX IF NOT EXISTS idx_opportunities_status ON csr_opportunities(status);

-- 7. Create NGO Programs Table (if not exists)
CREATE TABLE IF NOT EXISTS ngo_programs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    
    title VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100) NOT NULL,
    location VARCHAR(100),
    target_beneficiaries INT DEFAULT 0,
    budget_needed NUMERIC(15, 2),
    sdg_goals INT[],
    fiqh_asnaf VARCHAR(50),
    
    created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ngo_programs_tenant_id ON ngo_programs(tenant_id);

-- 8. Create Proposals Table (NGO Proposals to Corporate Opportunities)
CREATE TABLE IF NOT EXISTS proposals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    opportunity_id UUID REFERENCES csr_opportunities(id) ON DELETE CASCADE,
    ngo_program_id UUID REFERENCES ngo_programs(id) ON DELETE SET NULL,
    
    org_tenant_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    corp_tenant_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    company_id UUID NOT NULL REFERENCES company.companies(id) ON DELETE CASCADE,
    
    title VARCHAR(255) NOT NULL,
    summary TEXT,
    proposal_file_url TEXT,
    budget_requested NUMERIC(15, 2),
    status proposal_status_enum DEFAULT 'SUBMITTED',
    reviewer_notes TEXT,
    
    submitted_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_proposals_opportunity_id ON proposals(opportunity_id);
CREATE INDEX IF NOT EXISTS idx_proposals_org_tenant ON proposals(org_tenant_id);
CREATE INDEX IF NOT EXISTS idx_proposals_corp_tenant ON proposals(corp_tenant_id);
CREATE INDEX IF NOT EXISTS idx_proposals_company ON proposals(company_id);
CREATE INDEX IF NOT EXISTS idx_proposals_status ON proposals(status);

-- 9. Create User Invitations Table (Team Invites)
CREATE TABLE IF NOT EXISTS user_invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    invited_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    email VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL,
    invite_token VARCHAR(128) UNIQUE NOT NULL,
    status VARCHAR(50) DEFAULT 'PENDING', -- PENDING, ACCEPTED, EXPIRED, REVOKED
    
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_invitations_token ON user_invitations(invite_token);
CREATE INDEX IF NOT EXISTS idx_user_invitations_org_id ON user_invitations(org_id);
