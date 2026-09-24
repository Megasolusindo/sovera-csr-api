-- Rollback Migration 000037
DROP TABLE IF EXISTS tenant_saved_items CASCADE;
DROP TYPE IF EXISTS saved_item_type_enum CASCADE;
