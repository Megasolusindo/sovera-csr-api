-- =============================================================================
-- 000002_repoint_fk_to_master.up.sql
--
-- Tujuan: Pindahkan semua FK yang menunjuk ke companies_legacy_backup
--         agar menunjuk ke company.companies.
--
-- Prasyarat: migration 000001 sudah selesai (semua id sudah ada di master).
--
-- Tabel yang terdampak:
--   public.company_csr_focuses      (company_id → legacy)
--   public.company_csr_profiles     (company_id → legacy)
--   public.company_csr_programs     (company_id → legacy)
--   public.company_esg_profiles     (company_id → legacy)
--   public.companies_legacy_backup  (parent_company_id → dirinya sendiri)
-- =============================================================================

BEGIN;

CREATE TABLE IF NOT EXISTS public.migration_orphan_company_csr_profiles_000002 AS
SELECT p.*, now()::timestamptz AS quarantined_at
FROM public.company_csr_profiles p
WHERE false;

CREATE TABLE IF NOT EXISTS public.migration_orphan_company_csr_programs_000002 AS
SELECT p.*, now()::timestamptz AS quarantined_at
FROM public.company_csr_programs p
WHERE false;

CREATE TABLE IF NOT EXISTS public.migration_orphan_company_esg_profiles_000002 AS
SELECT p.*, now()::timestamptz AS quarantined_at
FROM public.company_esg_profiles p
WHERE false;

ALTER TABLE public.migration_orphan_company_csr_profiles_000002
  ADD COLUMN IF NOT EXISTS quarantined_at timestamptz;

ALTER TABLE public.migration_orphan_company_csr_programs_000002
  ADD COLUMN IF NOT EXISTS quarantined_at timestamptz;

ALTER TABLE public.migration_orphan_company_esg_profiles_000002
  ADD COLUMN IF NOT EXISTS quarantined_at timestamptz;

INSERT INTO public.migration_orphan_company_csr_profiles_000002
SELECT p.*, now()
FROM public.company_csr_profiles p
WHERE NOT EXISTS (SELECT 1 FROM company.companies c WHERE c.id = p.company_id)
  AND NOT EXISTS (
    SELECT 1
    FROM public.migration_orphan_company_csr_profiles_000002 a
    WHERE a.id = p.id
  );

INSERT INTO public.migration_orphan_company_csr_programs_000002
SELECT p.*, now()
FROM public.company_csr_programs p
WHERE NOT EXISTS (SELECT 1 FROM company.companies c WHERE c.id = p.company_id)
  AND NOT EXISTS (
    SELECT 1
    FROM public.migration_orphan_company_csr_programs_000002 a
    WHERE a.id = p.id
  );

INSERT INTO public.migration_orphan_company_esg_profiles_000002
SELECT p.*, now()
FROM public.company_esg_profiles p
WHERE NOT EXISTS (SELECT 1 FROM company.companies c WHERE c.id = p.company_id)
  AND NOT EXISTS (
    SELECT 1
    FROM public.migration_orphan_company_esg_profiles_000002 a
    WHERE a.id = p.id
  );

DELETE FROM public.company_csr_profiles p
WHERE NOT EXISTS (SELECT 1 FROM company.companies c WHERE c.id = p.company_id);

DELETE FROM public.company_csr_programs p
WHERE NOT EXISTS (SELECT 1 FROM company.companies c WHERE c.id = p.company_id);

DELETE FROM public.company_esg_profiles p
WHERE NOT EXISTS (SELECT 1 FROM company.companies c WHERE c.id = p.company_id);

-- ----------------------------------------------------------------------------
-- 1. company_csr_focuses
-- ----------------------------------------------------------------------------
ALTER TABLE public.company_csr_focuses
  DROP CONSTRAINT IF EXISTS company_csr_focuses_company_id_fkey;

ALTER TABLE public.company_csr_focuses
  ADD CONSTRAINT company_csr_focuses_company_id_fkey
  FOREIGN KEY (company_id) REFERENCES company.companies(id)
  ON DELETE CASCADE;

-- ----------------------------------------------------------------------------
-- 2. company_csr_profiles
-- ----------------------------------------------------------------------------
ALTER TABLE public.company_csr_profiles
  DROP CONSTRAINT IF EXISTS company_csr_profiles_company_id_fkey;

ALTER TABLE public.company_csr_profiles
  ADD CONSTRAINT company_csr_profiles_company_id_fkey
  FOREIGN KEY (company_id) REFERENCES company.companies(id)
  ON DELETE CASCADE;

-- ----------------------------------------------------------------------------
-- 3. company_csr_programs
-- ----------------------------------------------------------------------------
ALTER TABLE public.company_csr_programs
  DROP CONSTRAINT IF EXISTS company_csr_programs_company_id_fkey;

ALTER TABLE public.company_csr_programs
  ADD CONSTRAINT company_csr_programs_company_id_fkey
  FOREIGN KEY (company_id) REFERENCES company.companies(id)
  ON DELETE CASCADE;

-- ----------------------------------------------------------------------------
-- 4. company_esg_profiles
-- ----------------------------------------------------------------------------
ALTER TABLE public.company_esg_profiles
  DROP CONSTRAINT IF EXISTS company_esg_profiles_company_id_fkey;

ALTER TABLE public.company_esg_profiles
  ADD CONSTRAINT company_esg_profiles_company_id_fkey
  FOREIGN KEY (company_id) REFERENCES company.companies(id)
  ON DELETE CASCADE;

-- ----------------------------------------------------------------------------
-- 5. Self-FK parent_company_id di legacy → sekarang ke master
--    (FK ini bernama companies_parent_company_id_fkey di public schema)
-- ----------------------------------------------------------------------------
ALTER TABLE public.companies_legacy_backup
  DROP CONSTRAINT IF EXISTS companies_parent_company_id_fkey;

-- Tidak di-recreate ke company.companies karena tabel ini akan di-drop
-- di migration 000004. Biarkan kolom parent_company_id jadi orphan sementara.

-- ----------------------------------------------------------------------------
-- Verifikasi: tidak boleh ada FK violation setelah ini
-- ----------------------------------------------------------------------------
DO $$
DECLARE
  v_broken BIGINT;
BEGIN
  SELECT count(*) INTO v_broken
  FROM public.company_csr_profiles p
  WHERE NOT EXISTS (SELECT 1 FROM company.companies c WHERE c.id = p.company_id);
  IF v_broken > 0 THEN
    RAISE EXCEPTION 'company_csr_profiles masih ada % baris orphan!', v_broken;
  END IF;

  SELECT count(*) INTO v_broken
  FROM public.company_csr_programs p
  WHERE NOT EXISTS (SELECT 1 FROM company.companies c WHERE c.id = p.company_id);
  IF v_broken > 0 THEN
    RAISE EXCEPTION 'company_csr_programs masih ada % baris orphan!', v_broken;
  END IF;

  SELECT count(*) INTO v_broken
  FROM public.company_esg_profiles p
  WHERE NOT EXISTS (SELECT 1 FROM company.companies c WHERE c.id = p.company_id);
  IF v_broken > 0 THEN
    RAISE EXCEPTION 'company_esg_profiles masih ada % baris orphan!', v_broken;
  END IF;

  RAISE NOTICE 'Semua FK berhasil di-repoint ke company.companies.';
END $$;

COMMIT;
