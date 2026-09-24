-- Migration 000043: Create AHU Kemenkumham Corporate Registry and Entity Resolution Tables
CREATE TABLE IF NOT EXISTS public.ahu_registrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID REFERENCES company.companies(id) ON DELETE SET NULL,
    company_name VARCHAR(255) NOT NULL,
    legal_name VARCHAR(255) NOT NULL,
    ahu_number VARCHAR(100) UNIQUE NOT NULL,
    legal_entity_type VARCHAR(50) NOT NULL DEFAULT 'PT',
    deed_number VARCHAR(100),
    deed_date DATE,
    notary_name VARCHAR(255),
    status VARCHAR(50) DEFAULT 'AKTIF',
    headquarters VARCHAR(255),
    capital_amount NUMERIC(18,2),
    verified_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ahu_number ON public.ahu_registrations(ahu_number);
CREATE INDEX IF NOT EXISTS idx_ahu_company_name ON public.ahu_registrations(company_name);
CREATE INDEX IF NOT EXISTS idx_ahu_company_id ON public.ahu_registrations(company_id);

-- Insert Real Official AHU SK Kemenkumham Targets into crawling_targets
INSERT INTO public.crawling_targets (
    id, source_name, source_type, target_url, check_interval_hours, is_active, health_status, created_at, updated_at
) VALUES (
    gen_random_uuid(),
    'Ditjen AHU Kemenkumham Official Portal - Profil Perseroan & Badan Usaha',
    'AHU_REGISTRY',
    'https://ahu.go.id/profil-pt',
    24,
    true,
    'HEALTHY',
    NOW(),
    NOW()
) ON CONFLICT (target_url) DO UPDATE SET is_active = true, source_type = 'AHU_REGISTRY', updated_at = NOW();
