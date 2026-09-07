-- Migration 000019 Down: Drop Organization Intelligence and CRM tables

DROP TABLE IF EXISTS crm_activities CASCADE;
DROP TABLE IF EXISTS crm_contacts CASCADE;
DROP TABLE IF EXISTS organization_prospects CASCADE;
DROP TABLE IF EXISTS organization_signals CASCADE;
DROP TABLE IF EXISTS organization_partnerships CASCADE;
DROP TABLE IF EXISTS organization_programs CASCADE;
DROP TABLE IF EXISTS organization_focuses CASCADE;
DROP TABLE IF EXISTS organization_profiles CASCADE;
DROP TABLE IF EXISTS sources CASCADE;
