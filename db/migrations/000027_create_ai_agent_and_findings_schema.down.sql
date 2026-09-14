-- Migration 000027 Down: Drop OpenClaw AI Agent schema
DROP TABLE IF EXISTS ai_research_findings CASCADE;
DROP TABLE IF EXISTS ai_audit_logs CASCADE;
DROP TABLE IF EXISTS ai_agent_credentials CASCADE;
