-- Migration 000037: Create tenant_saved_items table for bookmarking organizations, programs, opportunities, and signals

DO $$ BEGIN
    CREATE TYPE saved_item_type_enum AS ENUM ('ORGANIZATION', 'PROGRAM', 'OPPORTUNITY', 'SIGNAL');
EXCEPTION WHEN duplicate_object THEN null; END $$;

CREATE TABLE IF NOT EXISTS tenant_saved_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    item_type saved_item_type_enum NOT NULL,
    item_id VARCHAR(255) NOT NULL,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_tenant_saved_item UNIQUE (org_id, item_type, item_id)
);

CREATE INDEX IF NOT EXISTS idx_tenant_saved_items_org_id ON tenant_saved_items(org_id);
CREATE INDEX IF NOT EXISTS idx_tenant_saved_items_type ON tenant_saved_items(item_type);
