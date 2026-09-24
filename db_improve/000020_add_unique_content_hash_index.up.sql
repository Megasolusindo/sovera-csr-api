-- Migration: Add unique index on content_hash for intelligence.company_signals
CREATE UNIQUE INDEX IF NOT EXISTS idx_company_signals_content_hash ON intelligence.company_signals (content_hash);
