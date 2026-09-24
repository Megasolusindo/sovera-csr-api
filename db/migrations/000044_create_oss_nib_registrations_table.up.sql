-- Migration 000044: Create OSS NIB Registrations Table & Crawling Target
CREATE TABLE IF NOT EXISTS public.oss_nib_registrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID REFERENCES company.companies(id) ON DELETE SET NULL,
    nib VARCHAR(30) UNIQUE NOT NULL,
    business_name VARCHAR(255) NOT NULL,
    legal_entity_type VARCHAR(50) DEFAULT 'PT',
    kbli_code VARCHAR(10) REFERENCES public.kbli_reference(code) ON DELETE SET NULL,
    risk_level VARCHAR(50) DEFAULT 'MENENGAH',
    investment_status VARCHAR(50) DEFAULT 'PMDN',
    province VARCHAR(100),
    regency_city VARCHAR(100),
    district VARCHAR(100),
    address TEXT,
    license_status VARCHAR(50) DEFAULT 'TERBIT',
    issued_date DATE,
    verified_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_oss_nib ON public.oss_nib_registrations(nib);
CREATE INDEX IF NOT EXISTS idx_oss_kbli_code ON public.oss_nib_registrations(kbli_code);
CREATE INDEX IF NOT EXISTS idx_oss_company_id ON public.oss_nib_registrations(company_id);
CREATE INDEX IF NOT EXISTS idx_oss_investment_status ON public.oss_nib_registrations(investment_status);

-- Register Official OSS Target into crawling_targets
INSERT INTO public.crawling_targets (
    id, source_name, source_type, target_url, check_interval_hours, is_active, health_status, created_at, updated_at
) VALUES 
(
    gen_random_uuid(),
    'OSS RBA BKPM Official Portal - Perizinan Berusaha Berbasis Risiko',
    'OSS_REGISTRY',
    'https://oss.go.id/id',
    24,
    true,
    'HEALTHY',
    NOW(),
    NOW()
),
(
    gen_random_uuid(),
    'Google News RSS - Terbit NIB Perusahaan & Perizinan Usaha OSS',
    'OSS_REGISTRY',
    'https://news.google.com/rss/search?q=site%3Aoss.go.id+NIB+OR+KBLI+OR+Izin+Usaha&hl=id&gl=ID&ceid=ID:id',
    6,
    true,
    'HEALTHY',
    NOW(),
    NOW()
)
ON CONFLICT (target_url) DO UPDATE SET is_active = true, source_type = 'OSS_REGISTRY', updated_at = NOW();
