-- Migration 000033 Up: Create Subscription, Midtrans Billing, Invoicing, and Webhook Log tables

-- 1. Subscription Plans Table
CREATE TABLE IF NOT EXISTS subscription_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(50) UNIQUE NOT NULL, -- 'FREE_TRIAL', 'PRO', 'ENTERPRISE'
    name VARCHAR(100) NOT NULL,
    description TEXT,
    price_monthly NUMERIC(15, 2) NOT NULL DEFAULT 0,
    price_yearly NUMERIC(15, 2) NOT NULL DEFAULT 0,
    crawl_quota INTEGER NOT NULL DEFAULT 50,
    ai_query_quota INTEGER NOT NULL DEFAULT 100,
    max_user_seats INTEGER NOT NULL DEFAULT 3,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 2. Tenant Subscriptions Table
CREATE TABLE IF NOT EXISTS tenant_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES subscription_plans(id),
    status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE', -- 'ACTIVE', 'EXPIRED', 'CANCELLED', 'PAST_DUE'
    billing_cycle VARCHAR(20) NOT NULL DEFAULT 'MONTHLY', -- 'MONTHLY', 'YEARLY'
    current_period_start TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    current_period_end TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT (NOW() + INTERVAL '30 days'),
    auto_renew BOOLEAN DEFAULT TRUE,
    midtrans_subscription_id VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT uq_tenant_subscriptions_org UNIQUE (org_id)
);

-- 3. Billing Invoices Table
CREATE TABLE IF NOT EXISTS billing_invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_number VARCHAR(100) UNIQUE NOT NULL,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES subscription_plans(id),
    amount NUMERIC(15, 2) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING', -- 'PENDING', 'PAID', 'FAILED', 'EXPIRED'
    billing_cycle VARCHAR(20) NOT NULL DEFAULT 'MONTHLY',
    due_date TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT (NOW() + INTERVAL '7 days'),
    paid_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 4. Payment Transactions Table (Midtrans Snap)
CREATE TABLE IF NOT EXISTS payment_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID NOT NULL REFERENCES billing_invoices(id) ON DELETE CASCADE,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL DEFAULT 'midtrans',
    order_id VARCHAR(100) UNIQUE NOT NULL,
    gross_amount NUMERIC(15, 2) NOT NULL,
    payment_type VARCHAR(50),
    snap_token VARCHAR(255),
    snap_redirect_url TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING', -- 'PENDING', 'SETTLEMENT', 'EXPIRE', 'CANCEL', 'DENY'
    raw_response JSONB,
    paid_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 5. Webhook Inbound Log Table (Idempotent Webhook Processing)
CREATE TABLE IF NOT EXISTS webhook_inbound_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source VARCHAR(50) NOT NULL DEFAULT 'midtrans',
    event_type VARCHAR(100),
    external_id VARCHAR(100), -- order_id from Midtrans
    payload JSONB NOT NULL,
    signature_hash VARCHAR(255),
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING', -- 'PENDING', 'PROCESSED', 'FAILED'
    error_message TEXT,
    processed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tenant_subscriptions_org ON tenant_subscriptions(org_id);
CREATE INDEX IF NOT EXISTS idx_billing_invoices_org ON billing_invoices(org_id);
CREATE INDEX IF NOT EXISTS idx_payment_tx_order_id ON payment_transactions(order_id);
CREATE INDEX IF NOT EXISTS idx_webhook_logs_ext_id ON webhook_inbound_logs(external_id);

-- 6. Seed Default Subscription Plans
INSERT INTO subscription_plans (code, name, description, price_monthly, price_yearly, crawl_quota, ai_query_quota, max_user_seats, is_active)
VALUES 
    ('FREE_TRIAL', 'Free Trial', '14-Day evaluation tier with basic directory access', 0, 0, 10, 20, 1, TRUE),
    ('PRO', 'Professional CSR Tier', 'Full Corporate Directory, ESG Index, AI Proposal Icebreakers', 4900000, 49000000, 500, 1000, 5, TRUE),
    ('ENTERPRISE', 'Enterprise Unlimited', 'Dedicated Crawler Pipeline, Custom ESG Taxonomies, Unlimited Seats', 14900000, 149000000, 99999, 99999, 999, TRUE)
ON CONFLICT (code) DO NOTHING;

-- 7. Seed Default Subscriptions for existing active organizations
INSERT INTO tenant_subscriptions (org_id, plan_id, status, billing_cycle, current_period_start, current_period_end)
SELECT 
    o.id,
    sp.id,
    'ACTIVE',
    'MONTHLY',
    NOW(),
    NOW() + INTERVAL '30 days'
FROM organizations o
JOIN subscription_plans sp ON sp.code = COALESCE(o.subscription_tier, 'PRO')
ON CONFLICT (org_id) DO NOTHING;
