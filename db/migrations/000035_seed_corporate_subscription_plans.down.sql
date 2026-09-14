-- Migration 000035 Down: Rollback Corporate subscription plans

DELETE FROM subscription_plans WHERE code IN ('CORPORATE_FREE', 'CORPORATE_STARTER', 'CORPORATE_ENTERPRISE');

ALTER TABLE subscription_plans DROP COLUMN IF EXISTS target_persona;
