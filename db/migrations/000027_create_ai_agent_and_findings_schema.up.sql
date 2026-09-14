-- Migration 000027: Create OpenClaw AI Agent Credentials, Audit Logs, and Research Findings schema

-- 1. Create ai_agent_credentials table for AI Agent Bearer Token Auth & RBAC
CREATE TABLE IF NOT EXISTS ai_agent_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_name VARCHAR(100) NOT NULL UNIQUE,
    api_key_hash VARCHAR(255) NOT NULL UNIQUE,
    scopes TEXT[] NOT NULL DEFAULT ARRAY['research:create', 'company:read', 'csr_program:create'],
    is_active BOOLEAN NOT NULL DEFAULT true,
    expires_at TIMESTAMP WITH TIME ZONE,
    last_used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 2. Create ai_audit_logs table for recording immutable AI Agent API actions
CREATE TABLE IF NOT EXISTS ai_audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_name VARCHAR(100) NOT NULL,
    action VARCHAR(100) NOT NULL,
    endpoint VARCHAR(255) NOT NULL,
    request_id VARCHAR(100),
    target_type VARCHAR(50),
    target_id VARCHAR(255),
    payload JSONB,
    status VARCHAR(20) NOT NULL DEFAULT 'SUCCESS',
    client_ip VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ai_audit_logs_agent ON ai_audit_logs(agent_name);
CREATE INDEX IF NOT EXISTS idx_ai_audit_logs_action ON ai_audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_ai_audit_logs_created_at ON ai_audit_logs(created_at DESC);

-- 3. Create ai_research_findings table for review queue
CREATE TABLE IF NOT EXISTS ai_research_findings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID REFERENCES company.companies(id) ON DELETE SET NULL,
    company_name VARCHAR(255),
    finding_type VARCHAR(50) NOT NULL DEFAULT 'csr_program',
    title VARCHAR(255) NOT NULL,
    summary TEXT,
    source_url TEXT NOT NULL,
    source_name VARCHAR(255) NOT NULL,
    source_type VARCHAR(50) NOT NULL DEFAULT 'news',
    published_at TIMESTAMP WITH TIME ZONE,
    discovered_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    confidence_score NUMERIC(3,2) NOT NULL DEFAULT 0.85,
    evidence_data JSONB,
    status VARCHAR(50) NOT NULL DEFAULT 'pending_review', -- pending_review, approved, rejected
    idempotency_key VARCHAR(255) UNIQUE,
    reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMP WITH TIME ZONE,
    review_notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ai_findings_status ON ai_research_findings(status);
CREATE INDEX IF NOT EXISTS idx_ai_findings_company ON ai_research_findings(company_id);
CREATE INDEX IF NOT EXISTS idx_ai_findings_confidence ON ai_research_findings(confidence_score DESC);
CREATE INDEX IF NOT EXISTS idx_ai_findings_created ON ai_research_findings(created_at DESC);
