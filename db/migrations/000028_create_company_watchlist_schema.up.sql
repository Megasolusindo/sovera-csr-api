-- Migration 000028: Create OpenClaw Company Watchlist schema

CREATE TABLE IF NOT EXISTS ai_company_watchlist (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL UNIQUE REFERENCES company.companies(id) ON DELETE CASCADE,
    company_name VARCHAR(255) NOT NULL,
    monitoring_keywords TEXT[] NOT NULL DEFAULT ARRAY['CSR', 'TJSL', 'Keberlanjutan', 'ESG'],
    check_interval_hours INT NOT NULL DEFAULT 12,
    last_monitored_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ai_watchlist_active ON ai_company_watchlist(is_active) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_ai_watchlist_last_monitored ON ai_company_watchlist(last_monitored_at);
