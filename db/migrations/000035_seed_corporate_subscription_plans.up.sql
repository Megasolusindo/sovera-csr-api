-- Migration 000035 Up: Add target_persona column and seed Corporate Free & Corporate Enterprise plans

-- 1. Add target_persona column to subscription_plans
ALTER TABLE subscription_plans
    ADD COLUMN IF NOT EXISTS target_persona VARCHAR(50) NOT NULL DEFAULT 'ORGANIZATION';

-- 2. Update existing NGO plans to target_persona = 'ORGANIZATION'
UPDATE subscription_plans
    SET target_persona = 'ORGANIZATION'
    WHERE target_persona IS NULL OR target_persona = 'ORGANIZATION';

-- 3. Seed Corporate Subscription Plans (Free, Starter, Enterprise)
INSERT INTO subscription_plans (code, name, target_persona, description, price_monthly, price_yearly, crawl_quota, ai_query_quota, max_user_seats, is_active)
VALUES 
    ('CORPORATE_FREE', 'Corporate Free (Verified Claim)', 'CORPORATE', 'Klaim & verifikasi profil korporasi, unggah laporan ESG/CSR publik, terima proposal terverifikasi dari NGO', 0, 0, 100, 200, 3, TRUE),
    ('CORPORATE_STARTER', 'Corporate Starter RFP', 'CORPORATE', 'Terbitkan hingga 3 CSR Opportunities (RFP) per bulan, Kurasi & AI Vetting proposal masuk, Dashboard ESG & SDGs', 1500000, 15000000, 2000, 5000, 10, TRUE),
    ('CORPORATE_ENTERPRISE', 'Corporate Enterprise Solution', 'CORPORATE', 'Unlimited RFP opportunities, Custom Audit Integration, Dedicated ESG Analyst & Onboarding Pendampingan', 14900000, 149000000, 99999, 99999, 999, TRUE)
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name,
    target_persona = EXCLUDED.target_persona,
    description = EXCLUDED.description,
    price_monthly = EXCLUDED.price_monthly,
    price_yearly = EXCLUDED.price_yearly,
    crawl_quota = EXCLUDED.crawl_quota,
    ai_query_quota = EXCLUDED.ai_query_quota,
    max_user_seats = EXCLUDED.max_user_seats,
    is_active = TRUE;
