-- =============================================================================
-- 000010_restructure_program_tables.up.sql
--
-- Tujuan: Restrukturisasi tabel program sesuai ADR-001.
--
-- Ringkasan perubahan:
--   1. Drop tabel kosong: ngo_programs, organization_programs
--   2. Rename institution_programs → ngo_managed_programs + tambah kolom
--   3. Rename company_csr_programs → company_enriched_programs
--   4. Buat tabel baru: company_managed_programs
--   5. Benahi csr_opportunities: drop company_id, rename tenant_id → org_id,
--      tambah company_program_id, enum status lengkap
--   6. Buat view kompatibilitas sementara (deprecated di migration 015)
--
-- Prasyarat: migration 009 selesai (pgvector sudah terinstall)
-- Data aktual: institution_programs=377, semua tabel lain yang di-drop=0
-- =============================================================================

BEGIN;

-- ----------------------------------------------------------------------------
-- 1. Drop tabel kosong
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS public.migration_archive_organization_programs_000010 AS
  SELECT * FROM public.organization_programs
  WHERE NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = 'migration_archive_organization_programs_000010');

INSERT INTO public.migration_archive_organization_programs_000010
SELECT *
FROM public.organization_programs p
WHERE NOT EXISTS (
  SELECT 1
  FROM public.migration_archive_organization_programs_000010 a
  WHERE a.id = p.id
);

-- proposals masih mereferensi ngo_programs; lepas FK dulu
ALTER TABLE public.proposals DROP CONSTRAINT IF EXISTS proposals_ngo_program_id_fkey;

DROP TABLE IF EXISTS public.ngo_programs;
DROP TABLE IF EXISTS public.organization_programs;
-- intelligence.company_csr_programs sudah di-drop di migration 003

-- ----------------------------------------------------------------------------
-- 2. Rename institution_programs → ngo_managed_programs + tambah kolom
-- ----------------------------------------------------------------------------
ALTER TABLE public.institution_programs
  RENAME TO ngo_managed_programs;

-- Tambah kolom yang ada di ngo_programs tapi tidak di institution_programs
ALTER TABLE public.ngo_managed_programs
  ADD COLUMN visibility varchar(20) NOT NULL DEFAULT 'private'
    CHECK (visibility IN ('private', 'published')),
  ADD COLUMN budget_needed numeric(15,2),
  ADD COLUMN fiqh_asnaf varchar(50),
  ADD COLUMN location varchar(100),
  ADD COLUMN target_beneficiaries_unit varchar(100),
  ADD COLUMN created_by_user_id uuid
    REFERENCES public.users(id) ON DELETE SET NULL;

-- target_beneficiaries di institution_programs berisi deskripsi teks bebas
-- (contoh: "Siswa SD, SMP, SMA kurang mampu", "Nelayan & masyarakat wilayah pesisir")
-- bukan angka — tidak bisa dikonversi langsung ke integer.
--
-- Strategi:
--   1. Tambah kolom target_beneficiaries_desc untuk tampung nilai lama
--   2. Pindahkan nilai lama ke kolom deskripsi
--   3. Konversi kolom lama ke integer (semua jadi NULL, diisi user nanti)

-- 1. Tambah kolom deskripsi
ALTER TABLE public.ngo_managed_programs
  ADD COLUMN target_beneficiaries_desc text;

-- 2. Selamatkan data lama ke kolom deskripsi
UPDATE public.ngo_managed_programs
  SET target_beneficiaries_desc = target_beneficiaries
  WHERE target_beneficiaries IS NOT NULL;

-- Verifikasi: jumlah baris yang diselamatkan harus sama dengan yang punya nilai
DO $$
DECLARE
  v_src BIGINT;
  v_dst BIGINT;
BEGIN
  SELECT count(*) INTO v_src
  FROM public.ngo_managed_programs WHERE target_beneficiaries IS NOT NULL;
  SELECT count(*) INTO v_dst
  FROM public.ngo_managed_programs WHERE target_beneficiaries_desc IS NOT NULL;
  IF v_src <> v_dst THEN
    RAISE EXCEPTION 'Data tidak tersalin sempurna: src=%, dst=%', v_src, v_dst;
  END IF;
  RAISE NOTICE 'Berhasil selamatkan % baris target_beneficiaries ke target_beneficiaries_desc', v_dst;
END $$;

-- 3. Konversi kolom lama ke integer (semua NULL, diisi user saat edit program)
ALTER TABLE public.ngo_managed_programs
  ALTER COLUMN target_beneficiaries DROP DEFAULT;

ALTER TABLE public.ngo_managed_programs
  ALTER COLUMN target_beneficiaries TYPE integer
  USING NULL;

ALTER TABLE public.ngo_managed_programs
  ALTER COLUMN target_beneficiaries SET DEFAULT 0;

-- Skema benefisiari akhir:
--   target_beneficiaries      integer     → jumlah (diisi user, misal: 500)
--   target_beneficiaries_unit varchar     → satuan (diisi user, misal: "siswa")
--   target_beneficiaries_desc text        → deskripsi bebas (data lama + opsional baru)

-- Index untuk marketplace query (published programs)
CREATE INDEX idx_ngo_managed_programs_visibility
  ON public.ngo_managed_programs (org_id, visibility)
  WHERE visibility = 'published';

-- Index untuk AI matching (program dengan embedding)
CREATE INDEX idx_ngo_managed_programs_cluster
  ON public.ngo_managed_programs (primary_cluster, org_id);

-- ----------------------------------------------------------------------------
-- 3. Rename company_csr_programs → company_enriched_programs
-- ----------------------------------------------------------------------------
ALTER TABLE public.company_csr_programs
  RENAME TO company_enriched_programs;

-- Update nama index yang ikut nama tabel lama
-- (Postgres tidak otomatis rename index saat tabel di-rename)
DO $$
DECLARE
  idx RECORD;
BEGIN
  FOR idx IN
    SELECT indexname
    FROM pg_indexes
    WHERE tablename = 'company_enriched_programs'
      AND indexname LIKE '%company_csr_programs%'
  LOOP
    EXECUTE format(
      'ALTER INDEX %I RENAME TO %I',
      idx.indexname,
      replace(idx.indexname, 'company_csr_programs', 'company_enriched_programs')
    );
    RAISE NOTICE 'Renamed index: % → %',
      idx.indexname,
      replace(idx.indexname, 'company_csr_programs', 'company_enriched_programs');
  END LOOP;
END $$;

-- ----------------------------------------------------------------------------
-- 4. Buat tabel baru: company_managed_programs
-- ----------------------------------------------------------------------------
CREATE TABLE public.company_managed_programs (
  id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
  org_id uuid NOT NULL REFERENCES public.organizations(id) ON DELETE RESTRICT,
  enriched_from_id uuid
    REFERENCES public.company_enriched_programs(id) ON DELETE SET NULL,

  -- Identitas program
  title varchar(255) NOT NULL,
  description text,
  csr_pillar varchar(100),
  target_regions text[] DEFAULT '{}',
  sdg_alignment text[] DEFAULT '{}',

  -- Periode
  start_date date NOT NULL,
  end_date date,

  -- Budget (input manual)
  budget_allocated numeric(15,2) NOT NULL DEFAULT 0,
  budget_disbursed numeric(15,2) NOT NULL DEFAULT 0,

  -- Status
  status varchar(50) NOT NULL DEFAULT 'DRAFT'
    CHECK (status IN ('DRAFT', 'ACTIVE', 'PAUSED', 'COMPLETED', 'CANCELLED')),

  -- Audit
  created_by_user_id uuid REFERENCES public.users(id) ON DELETE SET NULL,
  created_at timestamptz DEFAULT now(),
  updated_at timestamptz DEFAULT now(),

  CONSTRAINT chk_company_program_budget
    CHECK (budget_disbursed <= budget_allocated),
  CONSTRAINT chk_company_program_dates
    CHECK (end_date IS NULL OR end_date > start_date)
);

CREATE INDEX idx_company_managed_programs_org_id
  ON public.company_managed_programs (org_id);

CREATE INDEX idx_company_managed_programs_status
  ON public.company_managed_programs (org_id, status)
  WHERE status = 'ACTIVE';

CREATE INDEX idx_company_managed_programs_enriched_from
  ON public.company_managed_programs (enriched_from_id)
  WHERE enriched_from_id IS NOT NULL;

-- RLS
ALTER TABLE public.company_managed_programs ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.company_managed_programs FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_company_programs ON public.company_managed_programs
  USING (org_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid)
  WITH CHECK (org_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid);

-- RLS untuk ngo_managed_programs (belum ada sebelumnya)
ALTER TABLE public.ngo_managed_programs ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.ngo_managed_programs FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_ngo_programs ON public.ngo_managed_programs
  USING (org_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid)
  WITH CHECK (org_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid);

-- ----------------------------------------------------------------------------
-- 5. Benahi csr_opportunities
--    (tabel mungkin sudah tidak ada jika migration 000003 sudah jalan;
--     semua perubahan ini dijalankan hanya bila tabel masih ada.)
-- ----------------------------------------------------------------------------
DO $$
BEGIN
  IF to_regclass('public.csr_opportunities') IS NOT NULL THEN
    EXECUTE $b$ALTER TABLE public.csr_opportunities DROP CONSTRAINT IF EXISTS csr_opportunities_company_id_fkey$b$;
    EXECUTE $b$ALTER TABLE public.csr_opportunities DROP COLUMN IF EXISTS company_id$b$;
    EXECUTE $b$ALTER TABLE public.csr_opportunities RENAME COLUMN tenant_id TO org_id$b$;
    EXECUTE $b$ALTER TABLE public.csr_opportunities DROP CONSTRAINT IF EXISTS csr_opportunities_tenant_id_fkey$b$;
    EXECUTE $b$ALTER TABLE public.csr_opportunities ADD CONSTRAINT csr_opportunities_org_id_fkey FOREIGN KEY (org_id) REFERENCES public.organizations(id) ON DELETE RESTRICT$b$;
    EXECUTE $b$ALTER TABLE public.csr_opportunities ADD COLUMN company_program_id uuid REFERENCES public.company_managed_programs(id) ON DELETE SET NULL$b$;
    EXECUTE $b$CREATE INDEX IF NOT EXISTS idx_csr_opportunities_company_program_id ON public.csr_opportunities (company_program_id) WHERE company_program_id IS NOT NULL$b$;
    EXECUTE $b$CREATE INDEX IF NOT EXISTS idx_csr_opportunities_org_id ON public.csr_opportunities (org_id)$b$;
    EXECUTE $b$ALTER TABLE public.csr_opportunities DROP CONSTRAINT IF EXISTS csr_opportunities_status_check$b$;
    EXECUTE $b$ALTER TABLE public.csr_opportunities ALTER COLUMN status SET DEFAULT 'DRAFT'$b$;
    EXECUTE $b$ALTER TABLE public.csr_opportunities ADD CONSTRAINT csr_opportunities_status_check CHECK (status IN ('DRAFT', 'OPEN', 'CLOSED', 'AWARDED', 'CANCELLED'))$b$;
    EXECUTE $b$ALTER TABLE public.csr_opportunities ENABLE ROW LEVEL SECURITY$b$;
    EXECUTE $b$ALTER TABLE public.csr_opportunities FORCE ROW LEVEL SECURITY$b$;
    EXECUTE $b$CREATE POLICY corp_own_opportunities ON public.csr_opportunities USING (org_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid) WITH CHECK (org_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid)$b$;
    EXECUTE $b$CREATE POLICY ngo_see_open_opportunities ON public.csr_opportunities AS PERMISSIVE FOR SELECT USING (status = 'OPEN')$b$;
  END IF;
END $$;

-- ----------------------------------------------------------------------------
-- 6. View kompatibilitas sementara
--    DEPRECATED — akan di-drop di migration 015
--    Beri batas waktu: 3 bulan setelah migration 010 deploy
-- ----------------------------------------------------------------------------
CREATE VIEW public.institution_programs AS
  SELECT * FROM public.ngo_managed_programs;

COMMENT ON VIEW public.institution_programs IS
  'DEPRECATED sejak migration 010. Gunakan ngo_managed_programs. Akan di-drop di migration 015.';

CREATE VIEW public.ngo_programs AS
  SELECT * FROM public.ngo_managed_programs
  WHERE visibility = 'published';

COMMENT ON VIEW public.ngo_programs IS
  'DEPRECATED sejak migration 010. Gunakan ngo_managed_programs WHERE visibility = ''published''. Akan di-drop di migration 015.';

CREATE VIEW public.company_csr_programs AS
  SELECT * FROM public.company_enriched_programs;

COMMENT ON VIEW public.company_csr_programs IS
  'DEPRECATED sejak migration 010. Gunakan company_enriched_programs. Akan di-drop di migration 015.';

-- ----------------------------------------------------------------------------
-- Verifikasi akhir
-- ----------------------------------------------------------------------------
DO $$
DECLARE
  v_ngo BIGINT;
  v_enriched BIGINT;
BEGIN
  SELECT count(*) INTO v_ngo FROM public.ngo_managed_programs;
  SELECT count(*) INTO v_enriched FROM public.company_enriched_programs;

  RAISE NOTICE 'ngo_managed_programs: % baris (expected 377)', v_ngo;
  RAISE NOTICE 'company_enriched_programs: % baris (expected 76)', v_enriched;

  IF v_ngo <> 377 THEN
    RAISE EXCEPTION 'ngo_managed_programs jumlah baris tidak sesuai! expected=377, got=%', v_ngo;
  END IF;

  IF v_enriched <> 76 THEN
    RAISE EXCEPTION 'company_enriched_programs jumlah baris tidak sesuai! expected=76, got=%', v_enriched;
  END IF;
END $$;

DO $$
BEGIN
  RAISE NOTICE 'Migration 010 selesai.';
END $$;

COMMIT;
