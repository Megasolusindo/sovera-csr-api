-- Migration 000019 Up: Create Organization Intelligence and CRM tables with standardized org_id FK

-- 1. Master Data Sources Table
CREATE TABLE IF NOT EXISTS sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_type VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    url TEXT,
    domain VARCHAR(255),
    title VARCHAR(500),
    published_at TIMESTAMP WITH TIME ZONE,
    accessed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    confidence NUMERIC(5, 2) DEFAULT 1.00,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sources_domain ON sources(domain);
CREATE INDEX IF NOT EXISTS idx_sources_source_type ON sources(source_type);

-- 2. Organization Profiles Table
CREATE TABLE IF NOT EXISTS organization_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    description TEXT,
    coverage_scope VARCHAR(100),
    has_csr_partnerships BOOLEAN DEFAULT FALSE,
    has_corporate_partnership BOOLEAN DEFAULT FALSE,
    has_grant_program BOOLEAN DEFAULT FALSE,
    partnerships_page_url VARCHAR(500),
    confidence NUMERIC(5, 2) DEFAULT 1.00,
    last_verified_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT uq_organization_profiles_org_id UNIQUE (org_id)
);

CREATE INDEX IF NOT EXISTS idx_org_profiles_org_id ON organization_profiles(org_id);

-- 3. Organization Focuses Table (Links to csr_focuses)
CREATE TABLE IF NOT EXISTS organization_focuses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    focus_id UUID NOT NULL REFERENCES csr_focuses(id) ON DELETE CASCADE,
    priority VARCHAR(50),
    confidence NUMERIC(5, 2) DEFAULT 1.00,
    source_id UUID REFERENCES sources(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT uq_org_focuses_org_focus UNIQUE (org_id, focus_id)
);

CREATE INDEX IF NOT EXISTS idx_org_focuses_org_id ON organization_focuses(org_id);
CREATE INDEX IF NOT EXISTS idx_org_focuses_focus_id ON organization_focuses(focus_id);

-- 4. Organization Programs Table
CREATE TABLE IF NOT EXISTS organization_programs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    program_type VARCHAR(100),
    status VARCHAR(50) DEFAULT 'ACTIVE',
    location_scope TEXT,
    beneficiary_description TEXT,
    source_id UUID REFERENCES sources(id) ON DELETE SET NULL,
    confidence NUMERIC(5, 2) DEFAULT 1.00,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_org_programs_org_id ON organization_programs(org_id);
CREATE INDEX IF NOT EXISTS idx_org_programs_source_id ON organization_programs(source_id);

-- 5. Organization Corporate Partnerships Table (Links to companies)
CREATE TABLE IF NOT EXISTS organization_partnerships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    company_id UUID NOT NULL,
    partnership_type VARCHAR(100),

    program_name VARCHAR(255),
    description TEXT,
    start_date DATE,
    end_date DATE,
    source_id UUID REFERENCES sources(id) ON DELETE SET NULL,
    confidence NUMERIC(5, 2) DEFAULT 1.00,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_org_partnerships_org_id ON organization_partnerships(org_id);
CREATE INDEX IF NOT EXISTS idx_org_partnerships_company_id ON organization_partnerships(company_id);

-- 6. Organization Signals Table
CREATE TABLE IF NOT EXISTS organization_signals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    signal_press VARCHAR(255),
    signal_ext TEXT,
    source_id UUID REFERENCES sources(id) ON DELETE SET NULL,
    confidence NUMERIC(5, 2) DEFAULT 1.00,
    detected_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_org_signals_org_id ON organization_signals(org_id);

-- 7. Organization Prospects CRM Table
CREATE TABLE IF NOT EXISTS organization_prospects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    sales_status VARCHAR(50) DEFAULT 'LEAD',
    lead_score NUMERIC(5, 2) DEFAULT 0.00,
    source VARCHAR(100),
    assigned_to UUID REFERENCES users(id) ON DELETE SET NULL,
    first_contact_at TIMESTAMP WITH TIME ZONE,
    last_contact_at TIMESTAMP WITH TIME ZONE,
    next_followup_at TIMESTAMP WITH TIME ZONE,
    converted_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT uq_org_prospects_org_id UNIQUE (org_id)
);

CREATE INDEX IF NOT EXISTS idx_org_prospects_org_id ON organization_prospects(org_id);
CREATE INDEX IF NOT EXISTS idx_org_prospects_sales_status ON organization_prospects(sales_status);
CREATE INDEX IF NOT EXISTS idx_org_prospects_assigned_to ON organization_prospects(assigned_to);

-- 8. CRM Contacts Table
CREATE TABLE IF NOT EXISTS crm_contacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    position VARCHAR(150),
    email VARCHAR(255),
    phone VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_crm_contacts_org_id ON crm_contacts(org_id);
CREATE INDEX IF NOT EXISTS idx_crm_contacts_email ON crm_contacts(email);

-- 9. CRM Activities Table
CREATE TABLE IF NOT EXISTS crm_activities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    contact_id UUID REFERENCES crm_contacts(id) ON DELETE SET NULL,
    activity_type VARCHAR(50) NOT NULL,
    notes TEXT,
    activity_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_crm_activities_org_id ON crm_activities(org_id);
CREATE INDEX IF NOT EXISTS idx_crm_activities_contact_id ON crm_activities(contact_id);
