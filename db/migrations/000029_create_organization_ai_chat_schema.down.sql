-- Migration 000029 Down: Drop Organization AI Chat & Logs schema

DROP TABLE IF EXISTS organization_ai_chat_logs;
DROP TABLE IF EXISTS organization_ai_conversations;
