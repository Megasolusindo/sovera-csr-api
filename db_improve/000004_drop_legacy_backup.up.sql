-- =============================================================================
-- 000004_drop_legacy_backup.up.sql
--
-- Tujuan: Hapus companies_legacy_backup setelah semua FK sudah di-repoint
--         ke company.companies (selesai di migration 000002).
--
-- Prasyarat:
--   - migration 000001: semua baris legacy sudah di-merge ke master
--   - migration 000002: semua FK sudah menunjuk ke company.companies
--   - migration 000003: duplikat kosong sudah di-drop
-- =============================================================================

BEGIN;

-- Pastikan tidak ada FK aktif yang masih menunjuk ke tabel ini
DO $$
DECLARE
  v_count BIGINT;
BEGIN
  SELECT count(*) INTO v_count
  FROM pg_constraint
  WHERE confrelid = 'public.companies_legacy_backup'::regclass
    AND contype = 'f';

  IF v_count > 0 THEN
    RAISE EXCEPTION
      'Masih ada % FK yang menunjuk ke companies_legacy_backup. Abort.', v_count;
  END IF;
END $$;

-- Drop view public.companies yang saat ini adalah wrapper dari company.companies
-- (view ini ketinggalan 7 kolom dari tabel aslinya, tidak berguna lagi)
DROP VIEW IF EXISTS public.companies;

-- Drop view public.public_corporate_signals (wrapper dari intelligence.company_signals)
-- Akan di-recreate dengan nama yang lebih bersih jika masih dibutuhkan
DROP VIEW IF EXISTS public.public_corporate_signals;

-- Sekarang aman untuk drop
DROP TABLE public.companies_legacy_backup;

-- Recreate view companies yang proper, lengkap semua kolom dari master
CREATE VIEW public.companies AS
SELECT
  id, name, legal_name, slug, industry_id, industry_sector,
  company_type, priority_tier, csr_category,
  website, linkedin_url, headquarters, employee_range, revenue_range,
  is_public, ticker, parent_company_id, alias_keywords,
  created_at, updated_at,
  -- Kolom yang sebelumnya tidak ada di view lama:
  partner_ngo, instagram_url, facebook_url, youtube_url,
  linkedin_status, linkedin_verified_at, linkedin_last_error,
  instagram_status, instagram_verified_at, instagram_last_error,
  facebook_status, facebook_verified_at, facebook_last_error,
  youtube_status, youtube_verified_at, youtube_last_error,
  is_claimed, corporate_domain, claimed_by_tenant_id
FROM company.companies;

-- Recreate signal view dengan nama yang lebih eksplisit
CREATE VIEW public.corporate_signals AS
SELECT
  id, company_id, company_name, industry_sector,
  source_type, source_url, summary,
  extracted_pillar, target_regions, estimated_budget_signal,
  trigger_event, intent_score, content_hash,
  -- Kolom baru yang tidak ada di view lama (sensitif, hanya expose yang perlu):
  csr_relevance, activity_focus, action_type, opportunity_alert,
  published_date, created_at
  -- signal_embedding sengaja tidak di-expose di view
FROM intelligence.company_signals;

DO $$
BEGIN
  RAISE NOTICE 'companies_legacy_backup berhasil di-drop. View public.companies di-recreate lengkap.';
END $$;

COMMIT;
