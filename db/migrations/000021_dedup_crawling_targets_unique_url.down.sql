-- Rollback: remove the unique constraint (duplicates cannot be restored)
DROP INDEX IF EXISTS idx_crawling_targets_unique_url;
