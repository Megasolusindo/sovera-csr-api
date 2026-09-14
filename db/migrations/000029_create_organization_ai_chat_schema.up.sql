-- Migration 000029 Up: Create Organization AI Chat & Logs schema with org_id FK

CREATE TABLE IF NOT EXISTS organization_ai_conversations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title       TEXT,
    status      TEXT DEFAULT 'active',
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_org_ai_conv_org_user ON organization_ai_conversations(org_id, user_id);
CREATE INDEX IF NOT EXISTS idx_org_ai_conv_org_status ON organization_ai_conversations(org_id, status);

CREATE TABLE IF NOT EXISTS organization_ai_chat_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    conversation_id UUID NOT NULL REFERENCES organization_ai_conversations(id) ON DELETE CASCADE,
    message         TEXT NOT NULL,
    reply           TEXT NOT NULL,
    tools_called    JSONB DEFAULT '[]',
    latency_ms      INTEGER,
    model_name      TEXT,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_org_ai_chat_logs_org_id ON organization_ai_chat_logs(org_id);
CREATE INDEX IF NOT EXISTS idx_org_ai_chat_logs_conversation_id ON organization_ai_chat_logs(conversation_id);
CREATE INDEX IF NOT EXISTS idx_org_ai_chat_logs_org_user ON organization_ai_chat_logs(org_id, user_id);
