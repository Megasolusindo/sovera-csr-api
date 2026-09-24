-- =============================================================================
-- 000006_cleanup_sources_and_indexes.up.sql
--
-- Tujuan:
--   1. Drop public.sources (semua source_id FK = 0, tabel 0 baris)
--   2. Drop FK source_id yang menggantung (setelah tabelnya hilang)
--   3. Tambah missing indexes untuk FK tanpa index
--   4. Tambah GIN index untuk kolom array yang dipakai matching
-- =============================================================================

BEGIN;

-- ----------------------------------------------------------------------------
-- 1. Lepas semua FK yang menunjuk ke public.sources
--    (semua source_id = NULL di data aktual, jadi aman di-drop constraint-nya)
-- ----------------------------------------------------------------------------
ALTER TABLE public.organization_focuses
  DROP CONSTRAINT IF EXISTS organization_focuses_source_id_fkey;

ALTER TABLE public.organization_programs
  DROP CONSTRAINT IF EXISTS organization_programs_source_id_fkey;

ALTER TABLE public.organization_partnerships
  DROP CONSTRAINT IF EXISTS organization_partnerships_source_id_fkey;

ALTER TABLE public.organization_signals
  DROP CONSTRAINT IF EXISTS organization_signals_source_id_fkey;

ALTER TABLE public.company_esg_profiles
  DROP CONSTRAINT IF EXISTS company_esg_profiles_source_id_fkey;

-- source_id di company_esg_profiles akan di-point ke source.sources nanti
-- jika schema source dipakai. Untuk sekarang biarkan kolom ada, constraint hilang.

-- ----------------------------------------------------------------------------
-- 2. Drop public.sources
-- ----------------------------------------------------------------------------
DROP TABLE IF EXISTS public.sources;

-- ----------------------------------------------------------------------------
-- 3. Missing indexes untuk FK (bikin cascade delete lambat & join mahal)
-- ----------------------------------------------------------------------------

-- users.org_id
CREATE INDEX IF NOT EXISTS idx_users_org_id
  ON public.users (org_id);

-- billing_invoices.plan_id
CREATE INDEX IF NOT EXISTS idx_billing_invoices_plan_id
  ON public.billing_invoices (plan_id);

-- payment_transactions.invoice_id
CREATE INDEX IF NOT EXISTS idx_payment_tx_invoice_id
  ON public.payment_transactions (invoice_id);

-- tenant_subscriptions.plan_id
CREATE INDEX IF NOT EXISTS idx_tenant_subscriptions_plan_id
  ON public.tenant_subscriptions (plan_id);

-- proposals.ngo_program_id
CREATE INDEX IF NOT EXISTS idx_proposals_ngo_program_id
  ON public.proposals (ngo_program_id);

-- proposals.submitted_by_user_id
CREATE INDEX IF NOT EXISTS idx_proposals_submitted_by
  ON public.proposals (submitted_by_user_id);

-- company_claims.requested_by_user_id
CREATE INDEX IF NOT EXISTS idx_company_claims_requested_by
  ON public.company_claims (requested_by_user_id);

-- crawling_logs.target_id
CREATE INDEX IF NOT EXISTS idx_crawling_logs_target_id
  ON public.crawling_logs (target_id);

-- ai_research_findings.reviewed_by
CREATE INDEX IF NOT EXISTS idx_ai_findings_reviewed_by
  ON public.ai_research_findings (reviewed_by)
  WHERE reviewed_by IS NOT NULL;

-- deal_pipelines.signal_id (FK gantung, tapi index tetap berguna untuk filter)
CREATE INDEX IF NOT EXISTS idx_deal_pipelines_signal_id
  ON public.deal_pipelines (signal_id)
  WHERE signal_id IS NOT NULL;

-- ----------------------------------------------------------------------------
-- 4. GIN indexes untuk pencarian & matching
-- ----------------------------------------------------------------------------

-- Install pg_trgm jika belum ada (butuh superuser)
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Fuzzy match nama perusahaan (hasil scraping sering typo/variasi)
CREATE INDEX IF NOT EXISTS idx_companies_name_trgm
  ON company.companies USING gin (name gin_trgm_ops);

-- alias_keywords array search
CREATE INDEX IF NOT EXISTS idx_companies_alias_keywords
  ON company.companies USING gin (alias_keywords);

-- CSR focus array di company_csr_profiles
CREATE INDEX IF NOT EXISTS idx_csr_profiles_focus
  ON public.company_csr_profiles USING gin (csr_focus);

-- SDG goals array di institution_programs
CREATE INDEX IF NOT EXISTS idx_institution_programs_sdgs
  ON public.institution_programs USING gin (target_sdgs);

-- monitoring_keywords di ai_company_watchlist
CREATE INDEX IF NOT EXISTS idx_watchlist_keywords
  ON public.ai_company_watchlist USING gin (monitoring_keywords);

-- ----------------------------------------------------------------------------
-- 5. Tambah updated_at trigger (generik, apply ke semua tabel yang butuh)
-- ----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION public.set_updated_at()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$;

-- Apply ke tabel-tabel utama yang punya updated_at tapi belum ada trigger
DO $$
DECLARE
  tbl TEXT;
  tbls TEXT[] := ARRAY[
    'company.companies',
    'public.organizations',
    'public.users',
    'public.company_csr_profiles',
    'public.company_csr_programs',
    'public.company_esg_profiles',
    'public.company_key_persons',
    'public.crawling_targets',
    'public.deal_pipelines',
    'public.billing_invoices',
    'public.payment_transactions',
    'public.tenant_subscriptions',
    'public.organization_ai_conversations',
    'public.proposals'
  ];
BEGIN
  FOREACH tbl IN ARRAY tbls LOOP
    IF to_regclass(tbl) IS NOT NULL THEN
      EXECUTE format(
        'DROP TRIGGER IF EXISTS trg_set_updated_at ON %s',
        tbl
      );
      EXECUTE format(
        'CREATE TRIGGER trg_set_updated_at
           BEFORE UPDATE ON %s
           FOR EACH ROW EXECUTE FUNCTION public.set_updated_at()',
        tbl
      );
      RAISE NOTICE 'Trigger updated_at ditambahkan ke %', tbl;
    ELSE
      RAISE NOTICE 'Trigger updated_at dilewati (tabel tidak ada): %', tbl;
    END IF;
  END LOOP;
END $$;

DO $$
BEGIN
  RAISE NOTICE 'Migration 006 selesai.';
END $$;

COMMIT;
