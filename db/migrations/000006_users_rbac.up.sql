-- Migration 000006: Implement RBAC - user_org_role enum, role column on users, pic_user_id on deal_pipelines, seed default user

-- 1. Create user_org_role enum type
DO $$ BEGIN
    CREATE TYPE user_org_role AS ENUM ('ORG_ADMIN', 'DIRECTOR', 'FUNDRAISER');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- 2. Add role column (typed enum) to users table
--    Must drop default first before casting type, then re-apply
ALTER TABLE users ALTER COLUMN role DROP DEFAULT;
ALTER TABLE users ALTER COLUMN role TYPE user_org_role USING role::user_org_role;
ALTER TABLE users ALTER COLUMN role SET DEFAULT 'FUNDRAISER';

-- 3. Add is_active column to users if not exists
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT true;

-- 4. Add pic_user_id to deal_pipelines for FUNDRAISER ownership isolation
ALTER TABLE deal_pipelines
    ADD COLUMN IF NOT EXISTS pic_user_id UUID REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_deals_pic_user ON deal_pipelines(pic_user_id);

-- 5. Seed default organization & users for LAZ Peduli Ummat (org: 77123aaa-...)
INSERT INTO organizations (id, name, subscription_tier)
VALUES ('77123aaa-8819-4c12-99a1-00123456789a', 'LAZ Peduli Ummat', 'PRO')
ON CONFLICT (id) DO NOTHING;

--    password: 'admin123' hashed with bcrypt cost 12
--    Hash generated offline: $2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewKyDAJAVpSm8O2
INSERT INTO users (id, org_id, email, password_hash, full_name, role, is_active)
VALUES (
    'aaaaaaaa-0001-4000-a000-000000000001',
    '77123aaa-8819-4c12-99a1-00123456789a',
    'admin@laz.id',
    '$2a$12$4W50j7Bb2m9XT34xXWs4wugLlLd1n1WzmbZXLgN..YCRIMcfobVgC',
    'Administrator LAZ',
    'ORG_ADMIN',
    true
)
ON CONFLICT (email) DO NOTHING;

INSERT INTO users (id, org_id, email, password_hash, full_name, role, is_active)
VALUES (
    'aaaaaaaa-0002-4000-a000-000000000002',
    '77123aaa-8819-4c12-99a1-00123456789a',
    'director@laz.id',
    '$2a$12$4W50j7Bb2m9XT34xXWs4wugLlLd1n1WzmbZXLgN..YCRIMcfobVgC',
    'Head of Partnership LAZ',
    'DIRECTOR',
    true
)
ON CONFLICT (email) DO NOTHING;

INSERT INTO users (id, org_id, email, password_hash, full_name, role, is_active)
VALUES (
    'aaaaaaaa-0003-4000-a000-000000000003',
    '77123aaa-8819-4c12-99a1-00123456789a',
    'fundraiser@laz.id',
    '$2a$12$4W50j7Bb2m9XT34xXWs4wugLlLd1n1WzmbZXLgN..YCRIMcfobVgC',
    'Account Executive LAZ',
    'FUNDRAISER',
    true
)
ON CONFLICT (email) DO NOTHING;
