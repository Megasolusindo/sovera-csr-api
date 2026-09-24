-- Migration: 000008_create_agent_schema.up.sql
-- Description: Create dedicated agent schema and control plane tables (tasks, runs, tool_calls, policies, cost_logs)

CREATE SCHEMA IF NOT EXISTS agent;

-- 1. Agent Tasks Table
CREATE TABLE IF NOT EXISTS agent.tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_type VARCHAR(100) NOT NULL,
    scope_type VARCHAR(20) NOT NULL DEFAULT 'GLOBAL', -- 'GLOBAL' or 'TENANT'
    org_id UUID NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'PENDING',    -- 'PENDING', 'RUNNING', 'COMPLETED', 'FAILED', 'REJECTED'
    priority INT NOT NULL DEFAULT 100,
    payload JSONB DEFAULT '{}'::jsonb,
    result JSONB DEFAULT '{}'::jsonb,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_agent_tasks_status ON agent.tasks(status);
CREATE INDEX IF NOT EXISTS idx_agent_tasks_org_id ON agent.tasks(org_id) WHERE org_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_agent_tasks_scope ON agent.tasks(scope_type);

-- 2. Agent Runs Table
CREATE TABLE IF NOT EXISTS agent.runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES agent.tasks(id) ON DELETE CASCADE,
    agent_name VARCHAR(100) NOT NULL,
    step_count INT NOT NULL DEFAULT 0,
    max_steps INT NOT NULL DEFAULT 5,
    status VARCHAR(30) NOT NULL DEFAULT 'RUNNING',
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_agent_runs_task_id ON agent.runs(task_id);

-- 3. Agent Tool Calls Table
CREATE TABLE IF NOT EXISTS agent.tool_calls (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES agent.runs(id) ON DELETE CASCADE,
    tool_name VARCHAR(100) NOT NULL,
    arguments JSONB DEFAULT '{}'::jsonb,
    result JSONB DEFAULT '{}'::jsonb,
    status VARCHAR(30) NOT NULL DEFAULT 'SUCCESS',
    duration_ms INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_agent_tool_calls_run_id ON agent.tool_calls(run_id);

-- 4. Agent Policies Table (Permission Matrix & Risk Tiers)
CREATE TABLE IF NOT EXISTS agent.policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_name VARCHAR(100) NOT NULL,
    scope_type VARCHAR(20) NOT NULL DEFAULT 'GLOBAL',
    allowed_tools TEXT[] NOT NULL DEFAULT '{}',
    risk_tier VARCHAR(20) NOT NULL DEFAULT 'TIER_1', -- 'TIER_1', 'TIER_2', 'TIER_3'
    approval_required BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_policies_name_scope ON agent.policies(agent_name, scope_type);

-- 5. Agent Cost Logs Table
CREATE TABLE IF NOT EXISTS agent.cost_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID REFERENCES agent.tasks(id) ON DELETE SET NULL,
    org_id UUID NULL,
    agent_name VARCHAR(100) NOT NULL,
    model_name VARCHAR(100) NOT NULL,
    prompt_tokens INT NOT NULL DEFAULT 0,
    completion_tokens INT NOT NULL DEFAULT 0,
    total_tokens INT NOT NULL DEFAULT 0,
    cost_usd NUMERIC(12, 6) NOT NULL DEFAULT 0.000000,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_agent_cost_logs_org_id ON agent.cost_logs(org_id) WHERE org_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_agent_cost_logs_created_at ON agent.cost_logs(created_at DESC);
