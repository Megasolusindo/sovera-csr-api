-- 1. SETUP EXTENSIONS
DO $$ BEGIN
    CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
EXCEPTION WHEN OTHERS THEN NULL; END $$;

DO $$ BEGIN
    CREATE EXTENSION IF NOT EXISTS "vector";
EXCEPTION WHEN OTHERS THEN NULL; END $$;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'vector') THEN
        CREATE DOMAIN vector AS float8[];
    END IF;
END $$;

-- 2. ENUM TYPES
DO $$ BEGIN
    CREATE TYPE deal_stage_enum AS ENUM (
        'DISCOVERED', 
        'RESEARCH', 
        'PITCHED', 
        'NEGOTIATION', 
        'CLOSED_WON', 
        'CLOSED_LOST'
    );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TYPE signal_source_enum AS ENUM (
        'BEI_REPORT', 
        'NEWS', 
        'CSR_PDF', 
        'SOCIAL'
    );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- 3. TENANT & AUTH TABLES (PUBLIC READ / SYSTEM MANAGED)
CREATE TABLE IF NOT EXISTS organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    subscription_tier VARCHAR(50) DEFAULT 'PRO',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    role VARCHAR(50) DEFAULT 'FUNDRAISER',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 4. SHARED MARKET DATA (NO RLS - SHARED ACROSS TENANTS)
CREATE TABLE IF NOT EXISTS public_corporate_signals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_name VARCHAR(255) NOT NULL,
    industry_sector VARCHAR(100),
    source_type signal_source_enum NOT NULL,
    source_url TEXT,
    summary TEXT,
    extracted_pillar VARCHAR(100),
    target_regions TEXT[],
    estimated_budget_signal NUMERIC(15, 2),
    trigger_event VARCHAR(255),
    intent_score INT CHECK (intent_score BETWEEN 0 AND 100),
    content_hash VARCHAR(64) UNIQUE,
    signal_embedding vector,
    published_date DATE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 5. TENANT ISOLATED TABLES (RLS ENFORCED)
CREATE TABLE IF NOT EXISTS institution_programs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    asnaf_category VARCHAR(100),
    esg_pillar VARCHAR(100),
    target_beneficiaries VARCHAR(255),
    program_embedding vector,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS deal_pipelines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    signal_id UUID REFERENCES public_corporate_signals(id) ON DELETE SET NULL,
    company_name VARCHAR(255) NOT NULL,
    deal_stage deal_stage_enum DEFAULT 'DISCOVERED',
    estimated_value NUMERIC(15, 2),
    target_program_id UUID REFERENCES institution_programs(id) ON DELETE SET NULL,
    generated_icebreaker TEXT,
    generated_proposal TEXT,
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 6. ROW-LEVEL SECURITY (RLS) POLICIES
ALTER TABLE institution_programs ENABLE ROW LEVEL SECURITY;
ALTER TABLE institution_programs FORCE ROW LEVEL SECURITY;

ALTER TABLE deal_pipelines ENABLE ROW LEVEL SECURITY;
ALTER TABLE deal_pipelines FORCE ROW LEVEL SECURITY;

DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_policies WHERE policyname = 'tenant_isolation_programs' AND tablename = 'institution_programs'
    ) THEN
        CREATE POLICY tenant_isolation_programs ON institution_programs
            FOR ALL
            USING (org_id = NULLIF(current_setting('app.current_org_id', true), '')::UUID)
            WITH CHECK (org_id = NULLIF(current_setting('app.current_org_id', true), '')::UUID);
    END IF;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_policies WHERE policyname = 'tenant_isolation_deals' AND tablename = 'deal_pipelines'
    ) THEN
        CREATE POLICY tenant_isolation_deals ON deal_pipelines
            FOR ALL
            USING (org_id = NULLIF(current_setting('app.current_org_id', true), '')::UUID)
            WITH CHECK (org_id = NULLIF(current_setting('app.current_org_id', true), '')::UUID);
    END IF;
END $$;

-- 7. PERFORMANCE INDEXES (HNSW & BTREE)
DO $$ BEGIN
    CREATE INDEX IF NOT EXISTS idx_signals_embedding 
        ON public_corporate_signals 
        USING hnsw (signal_embedding vector_cosine_ops);
EXCEPTION WHEN OTHERS THEN NULL; END $$;

DO $$ BEGIN
    CREATE INDEX IF NOT EXISTS idx_programs_embedding 
        ON institution_programs 
        USING hnsw (program_embedding vector_cosine_ops);
EXCEPTION WHEN OTHERS THEN NULL; END $$;

CREATE INDEX IF NOT EXISTS idx_programs_org_id ON institution_programs(org_id);
CREATE INDEX IF NOT EXISTS idx_deals_org_id ON deal_pipelines(org_id);
CREATE INDEX IF NOT EXISTS idx_deals_stage ON deal_pipelines(deal_stage);
CREATE INDEX IF NOT EXISTS idx_signals_intent ON public_corporate_signals(intent_score DESC);
CREATE INDEX IF NOT EXISTS idx_signals_hash ON public_corporate_signals(content_hash);
