-- Migration 000036: Seed Corporate Demo Tenant & Users (admin@corporate.com, csr@corporate.com, reviewer@corporate.com)

-- 1. Ensure user_org_role enum contains Corporate & Platform roles
ALTER TYPE user_org_role ADD VALUE IF NOT EXISTS 'CORP_ADMIN';
ALTER TYPE user_org_role ADD VALUE IF NOT EXISTS 'CSR_MANAGER';
ALTER TYPE user_org_role ADD VALUE IF NOT EXISTS 'REVIEWER';
ALTER TYPE user_org_role ADD VALUE IF NOT EXISTS 'SUPERADMIN';

-- 2. Seed Corporate Organization "Corporate (demo)"
INSERT INTO organizations (id, name, org_type, subscription_tier)
VALUES (
    '99999999-9999-4000-a000-000000000001',
    'Corporate (demo)',
    'CORPORATE',
    'ENTERPRISE'
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    org_type = EXCLUDED.org_type,
    subscription_tier = EXCLUDED.subscription_tier;

-- 3. Seed Corporate Demo Users (Default password: 'admin123', bcrypt cost 12 hash: $2a$12$4W50j7Bb2m9XT34xXWs4wugLlLd1n1WzmbZXLgN..YCRIMcfobVgC)
INSERT INTO users (id, org_id, email, password_hash, full_name, role, is_active)
VALUES (
    'bbbbbbbb-0001-4000-a000-000000000001',
    '99999999-9999-4000-a000-000000000001',
    'admin@corporate.com',
    '$2a$12$4W50j7Bb2m9XT34xXWs4wugLlLd1n1WzmbZXLgN..YCRIMcfobVgC',
    'Corporate Admin (Demo)',
    'CORP_ADMIN',
    true
)
ON CONFLICT (email) DO UPDATE SET
    password_hash = EXCLUDED.password_hash,
    role = EXCLUDED.role,
    is_active = true;

INSERT INTO users (id, org_id, email, password_hash, full_name, role, is_active)
VALUES (
    'bbbbbbbb-0002-4000-a000-000000000002',
    '99999999-9999-4000-a000-000000000001',
    'csr@corporate.com',
    '$2a$12$4W50j7Bb2m9XT34xXWs4wugLlLd1n1WzmbZXLgN..YCRIMcfobVgC',
    'CSR Manager (Demo)',
    'CSR_MANAGER',
    true
)
ON CONFLICT (email) DO UPDATE SET
    password_hash = EXCLUDED.password_hash,
    role = EXCLUDED.role,
    is_active = true;

INSERT INTO users (id, org_id, email, password_hash, full_name, role, is_active)
VALUES (
    'bbbbbbbb-0003-4000-a000-000000000003',
    '99999999-9999-4000-a000-000000000001',
    'reviewer@corporate.com',
    '$2a$12$4W50j7Bb2m9XT34xXWs4wugLlLd1n1WzmbZXLgN..YCRIMcfobVgC',
    'Proposal Assessor (Demo)',
    'REVIEWER',
    true
)
ON CONFLICT (email) DO UPDATE SET
    password_hash = EXCLUDED.password_hash,
    role = EXCLUDED.role,
    is_active = true;
