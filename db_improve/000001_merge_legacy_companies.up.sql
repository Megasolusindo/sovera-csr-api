-- =============================================================================
-- 000001_merge_legacy_companies.up.sql
--
-- Tujuan: Gabungkan semua baris dari public.companies_legacy_backup
--         ke company.companies, sehingga company.companies menjadi
--         satu-satunya sumber kebenaran perusahaan.
--
-- Kondisi awal (terukur):
--   company.companies        = 3.452 baris
--   companies_legacy_backup  = 11.780 baris
--   yatim di master baru     = 1.493 (ada di master, tidak ada di legacy)
--   yatim di legacy          = 9.821 (belum di-migrate ke master)
--
-- Strategi: INSERT ... ON CONFLICT DO NOTHING
--   - Baris yang sudah ada di master (berdasarkan id) dilewati.
--   - Baris yang belum ada (9.821) di-insert dengan kolom yang tersedia.
--   - Kolom yang ada di company.companies tapi tidak di legacy
--     (social media fields, is_claimed, dll) dibiarkan NULL/default.
-- =============================================================================

BEGIN;

-- Simpan jumlah awal untuk verifikasi akhir
DO $$
DECLARE
  v_before BIGINT;
  v_legacy BIGINT;
BEGIN
  SELECT count(*) INTO v_before FROM company.companies;
  SELECT count(*) INTO v_legacy FROM public.companies_legacy_backup;
  RAISE NOTICE 'Sebelum merge: company.companies=%, legacy=%', v_before, v_legacy;
END $$;

-- Insert perusahaan yang ada di legacy tapi belum ada di master baru.
-- Kolom yang tidak ada di legacy di-set ke default/NULL.
INSERT INTO company.companies (
  id,
  name,
  legal_name,
  slug,
  industry_id,
  industry_sector,
  company_type,
  priority_tier,
  csr_category,
  website,
  linkedin_url,
  headquarters,
  employee_range,
  revenue_range,
  is_public,
  ticker,
  parent_company_id,
  alias_keywords,
  created_at,
  updated_at,
  -- Kolom baru yang tidak ada di legacy, set ke default
  linkedin_status,
  instagram_status,
  facebook_status,
  youtube_status,
  is_claimed
)
SELECT
  b.id,
  CASE
    WHEN EXISTS (
      SELECT 1
      FROM company.companies c
      WHERE lower(trim(c.name)) = lower(trim(b.name))
    ) OR (
      SELECT count(*)
      FROM public.companies_legacy_backup b2
      WHERE lower(trim(b2.name)) = lower(trim(b.name))
        AND NOT EXISTS (
          SELECT 1
          FROM company.companies c2
          WHERE c2.id = b2.id
        )
    ) > 1
      THEN left(b.name, 200) || ' [LEGACY ' || b.id::text || ']'
    ELSE b.name
  END,
  b.legal_name,
  -- Slug bisa konflik karena ada UNIQUE constraint di company.companies.
  -- Append '_legacy' sementara, nanti di-clean up manual jika perlu.
  CASE
    WHEN EXISTS (
      SELECT 1 FROM company.companies c WHERE c.slug = b.slug
    ) THEN b.slug || '_legacy_' || LEFT(b.id::text, 8)
    ELSE b.slug
  END,
  b.industry_id,
  b.industry_sector,
  b.company_type,
  b.priority_tier,
  b.csr_category,
  b.website,
  b.linkedin_url,
  b.headquarters,
  b.employee_range,
  b.revenue_range,
  b.is_public,
  b.ticker,
  NULL, -- parent_company_id: legacy FK ke dirinya sendiri, resolve terpisah
  b.alias_keywords,
  b.created_at,
  b.updated_at,
  'UNVERIFIED',  -- linkedin_status default
  'UNVERIFIED',  -- instagram_status default
  'UNVERIFIED',  -- facebook_status default
  'UNVERIFIED',  -- youtube_status default
  false          -- is_claimed default
FROM public.companies_legacy_backup b
WHERE NOT EXISTS (
  SELECT 1 FROM company.companies c WHERE c.id = b.id
);

-- Verifikasi: jumlah akhir harus = 3.452 + 9.821 = 13.273
DO $$
DECLARE
  v_after BIGINT;
  v_expected BIGINT := 13273;
  v_name_conflicts BIGINT;
BEGIN
  SELECT count(*) INTO v_after FROM company.companies;
  RAISE NOTICE 'Setelah merge: company.companies=%', v_after;
  IF v_after < v_expected THEN
    RAISE WARNING 'Jumlah kurang dari expected (%). Periksa slug conflict.', v_expected;
  END IF;

  SELECT count(*) INTO v_name_conflicts
  FROM (
    SELECT lower(trim(name)) AS normalized_name
    FROM company.companies
    GROUP BY lower(trim(name))
    HAVING count(*) > 1
  ) conflicts;

  IF v_name_conflicts > 0 THEN
    RAISE EXCEPTION 'Terdapat % normalized name conflict setelah merge.', v_name_conflicts;
  END IF;
END $$;

-- parent_company_id yang tadinya FK ke legacy, sekarang aman karena
-- semua id sudah ada di company.companies.
-- Update parent_company_id dari legacy ke company.companies
-- (tadinya FK ini menunjuk ke companies_legacy_backup)
-- Catatan: FK ini akan di-drop dan di-recreate di migration berikutnya.

COMMIT;
