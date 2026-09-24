-- Migration: Add tenant_id column and RLS policy to crawling_targets
-- NOTE: Column renamed to org_id for naming consistency in migration 000005

-- 1. Add tenant_id column safely if neither tenant_id nor org_id exists
DO $$ 
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'crawling_targets' AND column_name IN ('tenant_id', 'org_id')
    ) THEN
        ALTER TABLE crawling_targets ADD COLUMN tenant_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
    END IF;
END $$;

-- 2. Index for tenant query optimization (only if tenant_id column exists)
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'crawling_targets' AND column_name = 'tenant_id'
    ) THEN
        CREATE INDEX IF NOT EXISTS idx_crawling_targets_tenant ON crawling_targets (tenant_id);
    END IF;
END $$;

-- 3. Enable Row Level Security (RLS)
ALTER TABLE crawling_targets ENABLE ROW LEVEL SECURITY;

-- 4. Drop policy if exists to ensure idempotency
DROP POLICY IF EXISTS crawling_targets_tenant_isolation ON crawling_targets;

-- 5. Create Hybrid RLS Policy (Public Global Targets + Tenant Private Targets)
-- Use org_id if tenant_id was already renamed, otherwise use tenant_id
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'crawling_targets' AND column_name = 'org_id'
    ) THEN
        CREATE POLICY crawling_targets_tenant_isolation ON crawling_targets
            FOR ALL
            USING (
                org_id IS NULL
                OR org_id = NULLIF(current_setting('app.current_org_id', true), '')::uuid
            );
    ELSIF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'crawling_targets' AND column_name = 'tenant_id'
    ) THEN
        CREATE POLICY crawling_targets_tenant_isolation ON crawling_targets
            FOR ALL
            USING (
                tenant_id IS NULL
                OR tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid
            );
    END IF;
END $$;
