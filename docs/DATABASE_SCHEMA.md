# Database Schema & RLS Specification (DATABASE_SCHEMA.md)

**Product:** Sovera (FundIQ) Core API  
**Database Engine:** PostgreSQL 16+ with `pgvector` and `uuid-ossp`  
**Security Model:** Multi-Tenant Row-Level Security (RLS) Enforced at Kernel Level  

---

## 1. Architectural Overview & Security Model

PostgreSQL dirancang dengan pemisahan tegas antara **Data Intelijen Publik & Korporasi (Shared Data)** dan **Data Lembaga Privat (Tenant-Isolated Data)**:

* **Tabel Publik & Intelijen Korporasi (Shared Market Data):** Berisi data induk perusahaan (`companies`), profil CSR (`company_csr_profiles`), program CSR (`company_csr_programs`), profil ESG (`company_esg_profiles`), taksonomi fokus (`csr_focuses`, `esg_material_topics`), serta sinyal CSR (`public_corporate_signals`) yang dapat dibaca oleh seluruh tenant terdaftar.
* **Tabel Privat (`institution_programs`, `deal_pipelines`, `users`):** Diberikan proteksi `ENABLE ROW LEVEL SECURITY` dan `FORCE ROW LEVEL SECURITY`. Akses data diisolasi secara otomatis berdasarkan parameter transaksi `app.current_org_id`.

---

## 2. Complete SQL DDL

```sql
-- ============================================================
-- 1. SETUP EXTENSIONS & ENUMS
-- ============================================================
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "vector";

DO $$ BEGIN
    CREATE TYPE deal_stage_enum AS ENUM (
        'DISCOVERED', 
        'RESEARCH', 
        'PITCHED', 
        'NEGOTIATION', 
        'CLOSED_WON', 
        'CLOSED_LOST'
    );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TYPE signal_source_enum AS ENUM (
        'BEI_REPORT', 
        'NEWS', 
        'CSR_PDF', 
        'SOCIAL'
    );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- ============================================================
-- 2. TENANT & AUTH TABLES (SYSTEM MANAGED)
-- ============================================================
CREATE TABLE IF NOT EXISTS organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    subscription_tier VARCHAR(50) DEFAULT 'PRO',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    role VARCHAR(50) DEFAULT 'FUNDRAISER',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- ============================================================
-- 3. CORPORATE INTELLIGENCE & SHARED MARKET DATA (PUBLIC READ)
-- ============================================================

-- Induk Perusahaan (Companies)
CREATE TABLE IF NOT EXISTS companies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    legal_name VARCHAR(255),
    slug VARCHAR(255) UNIQUE NOT NULL,
    industry_id VARCHAR(100),
    industry_sector VARCHAR(100) NOT NULL,
    company_type VARCHAR(50) DEFAULT 'SWASTA',
    website TEXT,
    phone VARCHAR(50),
    email VARCHAR(255),
    linkedin_url TEXT,
    headquarters VARCHAR(255),
    employee_range VARCHAR(50),
    revenue_range VARCHAR(50),
    is_public BOOLEAN DEFAULT FALSE,
    ticker VARCHAR(20),
    parent_company_id UUID REFERENCES companies(id) ON DELETE SET NULL,
    alias_keywords TEXT[] DEFAULT '{}',
    ahu_number VARCHAR(100),
    nib VARCHAR(30),
    kbli_code VARCHAR(10),
    legal_entity_type VARCHAR(50) DEFAULT 'PT',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Dataset Referensi KBLI 2020 BPS/OSS (Migrasi 000042)
CREATE TABLE IF NOT EXISTS kbli_reference (
    code VARCHAR(10) PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    category_code CHAR(1) NOT NULL,
    category_title VARCHAR(255) NOT NULL,
    csr_relevance_default VARCHAR(50) DEFAULT 'MEDIUM',
    risk_level VARCHAR(50) DEFAULT 'MENENGAH',
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Ditjen AHU Kemenkumham Corporate Registry & Entity Resolution (Migrasi 000043)
CREATE TABLE IF NOT EXISTS ahu_registrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID REFERENCES companies(id) ON DELETE SET NULL,
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

-- OSS RBA BKPM NIB Business Universe (Migrasi 000044)
CREATE TABLE IF NOT EXISTS oss_nib_registrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID REFERENCES companies(id) ON DELETE SET NULL,
    nib VARCHAR(30) UNIQUE NOT NULL,
    business_name VARCHAR(255) NOT NULL,
    legal_entity_type VARCHAR(50) DEFAULT 'PT',
    kbli_code VARCHAR(10) REFERENCES kbli_reference(code) ON DELETE SET NULL,
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

-- Profil Kebijakan CSR Perusahaan
CREATE TABLE IF NOT EXISTS company_csr_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID UNIQUE NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    has_csr BOOLEAN DEFAULT TRUE NOT NULL,
    csr_department_name VARCHAR(255),
    csr_email_public VARCHAR(255),
    csr_focus TEXT[] DEFAULT '{}',
    budget_range VARCHAR(100),
    proposal_acceptance VARCHAR(50) DEFAULT 'OPEN',
    website_source TEXT,
    last_verified_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Taksonomi Fokus CSR
CREATE TABLE IF NOT EXISTS csr_focuses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    category VARCHAR(100) NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Relasi Fokus CSR Perusahaan (Junction Table)
CREATE TABLE IF NOT EXISTS company_csr_focuses (
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    focus_id UUID NOT NULL REFERENCES csr_focuses(id) ON DELETE CASCADE,
    priority SMALLINT,
    confidence NUMERIC(5,2),
    source_id UUID,
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (company_id, focus_id)
);

-- Program CSR Perusahaan
CREATE TABLE IF NOT EXISTS company_csr_programs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    program_type VARCHAR(50),
    start_date DATE,
    end_date DATE,
    status VARCHAR(50) DEFAULT 'ACTIVE',
    budget_amount NUMERIC(15,2),
    impact_summary TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Relasi Program CSR dengan Taksonomi Fokus
CREATE TABLE IF NOT EXISTS company_csr_program_focuses (
    program_id UUID NOT NULL REFERENCES company_csr_programs(id) ON DELETE CASCADE,
    focus_id UUID NOT NULL REFERENCES csr_focuses(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (program_id, focus_id)
);

-- Profil ESG Perusahaan
CREATE TABLE IF NOT EXISTS company_esg_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    reporting_year SMALLINT NOT NULL,
    report_date DATE,
    overall_score NUMERIC(5,2),
    environmental_score NUMERIC(5,2),
    social_score NUMERIC(5,2),
    governance_score NUMERIC(5,2),
    esg_rating VARCHAR(50),
    sustainability_strategy TEXT,
    sdg_alignment JSONB DEFAULT '{}'::jsonb,
    source_id UUID,
    confidence NUMERIC(5,2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(company_id, reporting_year)
);

-- Taksonomi Topik Materialitas ESG (GRI/SASB)
CREATE TABLE IF NOT EXISTS esg_material_topics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    category VARCHAR(50) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Relasi Skor Materialitas ESG Perusahaan
CREATE TABLE IF NOT EXISTS company_esg_material_topics (
    esg_profile_id UUID NOT NULL REFERENCES company_esg_profiles(id) ON DELETE CASCADE,
    topic_id UUID NOT NULL REFERENCES esg_material_topics(id) ON DELETE CASCADE,
    materiality_score NUMERIC(5,2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (esg_profile_id, topic_id)
);

-- Sinyal Intelijen Pasar & Media (Public Signals)
CREATE TABLE IF NOT EXISTS public_corporate_signals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID REFERENCES companies(id) ON DELETE SET NULL,
    company_name VARCHAR(255) NOT NULL,
    industry_sector VARCHAR(100),
    source_type signal_source_enum NOT NULL,
    source_url TEXT,
    summary TEXT,
    extracted_pillar VARCHAR(100),
    target_regions TEXT[],
    estimated_budget_signal NUMERIC(15, 2),
    trigger_event VARCHAR(255),
    intent_score INT CHECK (intent_score BETWEEN 0 AND 100),
    content_hash VARCHAR(64) UNIQUE,
    signal_embedding vector(1536),
    published_date DATE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Target Crawling Intelijen Pasar
CREATE TABLE IF NOT EXISTS crawling_targets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID REFERENCES companies(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    target_url TEXT NOT NULL UNIQUE,
    source_type VARCHAR(50) NOT NULL,
    industry_sector VARCHAR(100),
    is_active BOOLEAN DEFAULT TRUE,
    crawl_frequency VARCHAR(50) DEFAULT 'WEEKLY',
    last_crawled_at TIMESTAMP WITH TIME ZONE,
    next_crawl_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- ============================================================
-- 4. TENANT ISOLATED TABLES (RLS ENFORCED)
-- ============================================================
CREATE TABLE IF NOT EXISTS institution_programs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    asnaf_category VARCHAR(100),
    esg_pillar VARCHAR(100),
    target_beneficiaries VARCHAR(255),
    program_embedding vector(1536),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS deal_pipelines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    signal_id UUID REFERENCES public_corporate_signals(id) ON DELETE SET NULL,
    company_name VARCHAR(255) NOT NULL,
    deal_stage deal_stage_enum DEFAULT 'DISCOVERED',
    estimated_value NUMERIC(15, 2),
    target_program_id UUID REFERENCES institution_programs(id) ON DELETE SET NULL,
    generated_icebreaker TEXT,
    generated_proposal TEXT,
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- ============================================================
-- 5. ROW-LEVEL SECURITY (RLS) POLICIES
-- ============================================================
ALTER TABLE institution_programs ENABLE ROW LEVEL SECURITY;
ALTER TABLE institution_programs FORCE ROW LEVEL SECURITY;

ALTER TABLE deal_pipelines ENABLE ROW LEVEL SECURITY;
ALTER TABLE deal_pipelines FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_programs ON institution_programs
    FOR ALL
    USING (org_id = NULLIF(current_setting('app.current_org_id', true), '')::UUID)
    WITH CHECK (org_id = NULLIF(current_setting('app.current_org_id', true), '')::UUID);

CREATE POLICY tenant_isolation_deals ON deal_pipelines
    FOR ALL
    USING (org_id = NULLIF(current_setting('app.current_org_id', true), '')::UUID)
    WITH CHECK (org_id = NULLIF(current_setting('app.current_org_id', true), '')::UUID);

-- ============================================================
-- 6. PERFORMANCE INDEXES
-- ============================================================
CREATE INDEX IF NOT EXISTS idx_signals_embedding ON public_corporate_signals USING hnsw (signal_embedding vector_cosine_ops);
CREATE INDEX IF NOT EXISTS idx_programs_embedding ON institution_programs USING hnsw (program_embedding vector_cosine_ops);

CREATE INDEX IF NOT EXISTS idx_companies_company_type ON companies(company_type);
CREATE INDEX IF NOT EXISTS idx_companies_is_public ON companies(is_public);
CREATE INDEX IF NOT EXISTS idx_companies_ticker ON companies(ticker);
CREATE INDEX IF NOT EXISTS idx_companies_parent ON companies(parent_company_id);

CREATE INDEX IF NOT EXISTS idx_csr_profiles_company ON company_csr_profiles(company_id);
CREATE INDEX IF NOT EXISTS idx_csr_focuses_code ON csr_focuses(code);
CREATE INDEX IF NOT EXISTS idx_esg_profiles_company_year ON company_esg_profiles(company_id, reporting_year);
CREATE INDEX IF NOT EXISTS idx_esg_topics_code ON esg_material_topics(code);

CREATE INDEX IF NOT EXISTS idx_programs_org_id ON institution_programs(org_id);
CREATE INDEX IF NOT EXISTS idx_deals_org_id ON deal_pipelines(org_id);
CREATE INDEX IF NOT EXISTS idx_deals_stage ON deal_pipelines(deal_stage);
```

---

## 3. Data Dictionary

| Table Name | Description | Isolation Level | Primary Keys / Foreign Keys |
| --- | --- | --- | --- |
| `organizations` | Data tenant/lembaga zakat/NGO terdaftar | Global System | PK: `id` |
| `users` | Akun pengguna/fundraiser di bawah tenant | Global System | PK: `id`, FK: `org_id` -> `organizations.id` |
| `companies` | Induk direktori entitas korporasi | Shared (Public) | PK: `id`, FK: `parent_company_id` -> `companies.id` |
| `company_csr_profiles` | Kebijakan, kontak, & estimasi budget CSR perusahaan | Shared (Public) | PK: `id`, FK: `company_id` -> `companies.id` |
| `csr_focuses` | Taksonomi bidang fokus CSR | Shared (Public) | PK: `id`, UNIQUE: `code` |
| `company_csr_focuses` | Relasi prioritas fokus CSR per perusahaan | Shared (Public) | PK: (`company_id`, `focus_id`) |
| `company_csr_programs` | Portofolio program CSR milik korporasi | Shared (Public) | PK: `id`, FK: `company_id` -> `companies.id` |
| `company_csr_program_focuses` | Relasi program CSR korporasi dengan bidang fokus | Shared (Public) | PK: (`program_id`, `focus_id`) |
| `company_esg_profiles` | Skor, rating, dan strategi keberlanjutan ESG | Shared (Public) | PK: `id`, FK: `company_id` -> `companies.id`, UNIQUE: (`company_id`, `reporting_year`) |
| `esg_material_topics` | Taksonomi topik materialitas ESG (GRI/SASB) | Shared (Public) | PK: `id`, UNIQUE: `code` |
| `company_esg_material_topics` | Skor materialitas topik ESG per profil perusahaan | Shared (Public) | PK: (`esg_profile_id`, `topic_id`) |
| `public_corporate_signals` | Data intelijen CSR/ESG hasil ekstraksi crawler | Shared (Public) | PK: `id`, FK: `company_id` -> `companies.id` |
| `crawling_targets` | Target situs web & portal berita crawling | Shared (Public) | PK: `id`, FK: `company_id` -> `companies.id` |
| `sources` | Referensi rujukan data crawling & scraping | Shared (Public) | PK: `id` |
| `organization_profiles` | Profil kapabilitas & kualifikasi hibah/kemitraan | Shared (Public) | PK: `id`, FK: `org_id` -> `organizations.id` |
| `organization_focuses` | Relasi prioritas fokus CSR per organisasi/lembaga | Shared (Public) | PK: `id`, FK: `org_id`, FK: `focus_id` -> `csr_focuses.id` |
| `organization_programs` | Portofolio program sosial/CSR milik organisasi | Shared (Public) | PK: `id`, FK: `org_id`, FK: `source_id` -> `sources.id` |
| `organization_partnerships` | Riwayat kemitraan korporasi-lembaga CSR | Shared (Public) | PK: `id`, FK: `org_id`, FK: `company_id` -> `companies.id` |
| `organization_signals` | Sinyal berita publisitas media per organisasi | Shared (Public) | PK: `id`, FK: `org_id`, FK: `source_id` -> `sources.id` |
| `organization_prospects` | Funnel sales CRM prospek calon tenant | Internal CRM | PK: `id`, FK: `org_id`, FK: `assigned_to` -> `users.id` |
| `crm_contacts` | Kontak personil pengurus organisasi | Internal CRM | PK: `id`, FK: `org_id` -> `organizations.id` |
| `crm_activities` | Log aktivitas penjualan CRM calon tenant | Internal CRM | PK: `id`, FK: `org_id`, FK: `contact_id` -> `crm_contacts.id` |
| `institution_programs` | Portofolio program unggulan & embedding program | **RLS Enforced** | PK: `id`, FK: `org_id` -> `organizations.id` |
| `deal_pipelines` | Prospek, naskah proposal, dan tracking negosiasi | **RLS Enforced** | PK: `id`, FK: `org_id`, FK: `signal_id`, FK: `target_program_id` |


---

## 4. Query Execution Pattern (Go pgx/v5 Backend Wrapper)

Semua query pada tabel privat wajib dieksekusi melalui transaksi berpagar RLS:

```go
package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WithTenantContext executes database operations within an RLS-enforced PostgreSQL transaction.
func WithTenantContext(ctx context.Context, pool *pgxpool.Pool, orgID string, fn func(tx pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()

	// Set tenant context for PostgreSQL RLS policies
	_, err = tx.Exec(ctx, "SET LOCAL app.current_org_id = $1", orgID)
	if err != nil {
		_ = tx.Rollback(ctx)
		return fmt.Errorf("failed to set tenant RLS context: %w", err)
	}

	if err := fn(tx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
```
