-- Migration 000020: Seed Platform Admin Organization & Internal Operators

-- 1. Seed System Organization "Sovera Platform Admin"
INSERT INTO organizations (id, name, org_type, subscription_tier)
VALUES ('00000000-0000-0000-0000-000000000000', 'Sovera Platform Admin', 'SYSTEM_ADMIN', 'ENTERPRISE')
ON CONFLICT (id) DO NOTHING;

-- 2. Seed Platform Admin Users (Default password: 'admin123')
INSERT INTO users (id, org_id, email, password_hash, full_name, role, is_active)
VALUES (
    'aaaaaaaa-0000-4000-a000-000000000001',
    '00000000-0000-0000-0000-000000000000',
    'admin@sovera.id',
    '$2a$12$4W50j7Bb2m9XT34xXWs4wugLlLd1n1WzmbZXLgN..YCRIMcfobVgC',
    'Super Admin Operator',
    'ORG_ADMIN',
    true
)
ON CONFLICT (email) DO NOTHING;

INSERT INTO users (id, org_id, email, password_hash, full_name, role, is_active)
VALUES (
    'aaaaaaaa-0000-4000-a000-000000000002',
    '00000000-0000-0000-0000-000000000000',
    'ops@sovera.id',
    '$2a$12$4W50j7Bb2m9XT34xXWs4wugLlLd1n1WzmbZXLgN..YCRIMcfobVgC',
    'Data Engineer Ops',
    'ORG_ADMIN',
    true
)
ON CONFLICT (email) DO NOTHING;

INSERT INTO users (id, org_id, email, password_hash, full_name, role, is_active)
VALUES (
    'aaaaaaaa-0000-4000-a000-000000000003',
    '00000000-0000-0000-0000-000000000000',
    'support@sovera.id',
    '$2a$12$4W50j7Bb2m9XT34xXWs4wugLlLd1n1WzmbZXLgN..YCRIMcfobVgC',
    'Customer Success Spec',
    'ORG_ADMIN',
    true
)
ON CONFLICT (email) DO NOTHING;
