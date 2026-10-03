-- Add a website column to organizations so a discovered/verified official site
-- can be stored (previously the /discover payload carried it but it was dropped).
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS website VARCHAR(512);
