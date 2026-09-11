-- Deduplicate crawling_targets and add unique constraint on target_url
-- to prevent future duplicates from being inserted.

-- Step 1: Remove duplicates, keeping the oldest record per target_url
DELETE FROM crawling_targets
WHERE id NOT IN (
  SELECT DISTINCT ON (target_url) id
  FROM crawling_targets
  ORDER BY target_url, created_at ASC
);

-- Step 2: Add unique index to prevent future duplicates
CREATE UNIQUE INDEX IF NOT EXISTS idx_crawling_targets_unique_url 
ON crawling_targets (target_url);
