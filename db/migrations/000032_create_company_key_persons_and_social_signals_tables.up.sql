-- Migration 000032 Up: Create company_key_persons and key_person_social_signals tables

CREATE TABLE IF NOT EXISTS company_key_persons (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES company.companies(id) ON DELETE CASCADE,
    full_name VARCHAR(255) NOT NULL,
    normalized_name VARCHAR(255) NOT NULL,
    current_title VARCHAR(255),
    role_category VARCHAR(50), -- 'csr_lead', 'corp_sec', 'c_level', 'foundation_head'
    linkedin_url VARCHAR(255),
    twitter_handle VARCHAR(100),
    instagram_handle VARCHAR(100),
    is_decision_maker BOOLEAN DEFAULT FALSE,
    is_monitored BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS key_person_social_signals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id UUID REFERENCES company_key_persons(id) ON DELETE CASCADE,
    company_id UUID NOT NULL REFERENCES company.companies(id) ON DELETE CASCADE,
    platform VARCHAR(30) NOT NULL, -- 'linkedin', 'twitter', 'instagram', 'news_web'
    post_url VARCHAR(500) UNIQUE,
    post_text TEXT NOT NULL,
    posted_at TIMESTAMP WITH TIME ZONE,
    matched_keywords TEXT[] DEFAULT '{}',
    sentiment VARCHAR(20), -- 'positive', 'call_for_proposal', 'event_recap', 'neutral'
    ai_summary TEXT,
    is_actionable BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_key_persons_company_id ON company_key_persons(company_id);
CREATE INDEX IF NOT EXISTS idx_key_persons_role ON company_key_persons(role_category, is_monitored);
CREATE INDEX IF NOT EXISTS idx_social_signals_company ON key_person_social_signals(company_id);
CREATE INDEX IF NOT EXISTS idx_social_signals_person ON key_person_social_signals(person_id);
CREATE INDEX IF NOT EXISTS idx_social_signals_actionable ON key_person_social_signals(is_actionable) WHERE is_actionable = TRUE;
