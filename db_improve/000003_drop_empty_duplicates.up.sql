-- =============================================================================
-- 000003_drop_empty_duplicates.up.sql
--
-- Tujuan: Hapus tabel-tabel kosong yang merupakan duplikat dari versi
--         public yang otoritatif.
--
-- Semua tabel di bawah sudah dikonfirmasi 0 baris.
-- Diurutkan: junction/child dulu, parent belakangan.
-- =============================================================================

BEGIN;

-- Schema intelligence (versi kosong, public yang otoritatif)
DROP TABLE IF EXISTS intelligence.company_csr_focuses;
DROP TABLE IF EXISTS intelligence.company_csr_programs;
DROP TABLE IF EXISTS intelligence.company_esg_profiles;

-- Schema crm (semua kosong, public yang otoritatif)
DROP TABLE IF EXISTS crm.proposals;
DROP TABLE IF EXISTS crm.opportunities;
DROP TABLE IF EXISTS crm.contacts;
-- crm.ai_token_logs dan crm.tenant_templates tidak di-drop di sini
-- karena belum dikonfirmasi jumlah barisnya

-- Schema source (kosong)
DROP TABLE IF EXISTS source.documents;
DROP TABLE IF EXISTS source.sources;

-- Schema scraper (kosong, crawling_targets di public yang otoritatif)
DROP TABLE IF EXISTS scraper.scraping_jobs;

-- public: tabel kosong yang tidak punya padanan aktif
ALTER TABLE public.proposals DROP CONSTRAINT IF EXISTS proposals_opportunity_id_fkey;
DROP TABLE IF EXISTS public.csr_opportunities;    -- 0 baris, digantikan deal_pipelines
DROP TABLE IF EXISTS public.organization_signals; -- 0 baris, digantikan intelligence.company_signals
DROP TABLE IF EXISTS public.company_csr_focuses;  -- 0 baris junction table
-- public.company_csr_program_focuses: 76 baris, JANGAN drop, dipakai

-- public.sources: tergantung hasil query C (source_id FK berisi data atau tidak)
-- PLACEHOLDER — uncomment setelah konfirmasi query C = semua 0:
-- DROP TABLE IF EXISTS public.sources;

DO $$
BEGIN
  RAISE NOTICE 'Tabel duplikat kosong berhasil di-drop.';
END $$;

COMMIT;
