-- Migration 000023: Add account_status column to organizations and set Calon Tenant (PROSPECT) status

ALTER TABLE organizations
ADD COLUMN IF NOT EXISTS account_status VARCHAR(50) DEFAULT 'PROSPECT';

-- 1. Set active paying/system tenant accounts
UPDATE organizations SET account_status = 'ACTIVE' WHERE id IN (
  '00000000-0000-0000-0000-000000000000',
  '77123aaa-8819-4c12-99a1-00123456789a'
);

-- 2. Set all 25 newly discovered humanitarian institutions as Calon Tenant (PROSPECT)
UPDATE organizations SET account_status = 'PROSPECT' WHERE id::text LIKE 'b1000000%';
