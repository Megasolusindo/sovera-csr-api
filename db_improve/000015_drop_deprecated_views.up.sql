-- =============================================================================
-- 000015_drop_deprecated_views.up.sql
--
-- Tujuan: Drop view kompatibilitas yang dibuat di migration 010.
--
-- View ini dibuat sementara agar Go backend bisa diupdate bertahap
-- tanpa harus deploy database dan aplikasi bersamaan.
--
-- JANGAN jalankan migration ini sebelum memastikan tidak ada query
-- di sovera-csr-api yang masih memakai nama lama.
--
-- Checklist sebelum jalankan:
--   [ ] grep -r "institution_programs" di repo sovera-csr-api → 0 hasil
--   [ ] grep -r "ngo_programs" di repo sovera-csr-api → 0 hasil
--   [ ] grep -r "company_csr_programs" di repo sovera-csr-api → 0 hasil
--   [ ] Semua endpoint sudah ditest di staging dengan nama tabel baru
--   [ ] Migration 010–012 sudah berjalan di production minimal 2 minggu
-- =============================================================================

BEGIN;

-- Verifikasi: pastikan view masih ada sebelum di-drop
-- (idempotent — tidak error kalau sudah di-drop sebelumnya)
DO $$
DECLARE
  v_views TEXT[] := ARRAY[
    'institution_programs',
    'ngo_programs',
    'company_csr_programs'
  ];
  v_view TEXT;
  v_exists BOOLEAN;
BEGIN
  FOREACH v_view IN ARRAY v_views LOOP
    SELECT EXISTS (
      SELECT 1 FROM information_schema.views
      WHERE table_schema = 'public' AND table_name = v_view
    ) INTO v_exists;

    IF v_exists THEN
      RAISE NOTICE 'View public.% akan di-drop.', v_view;
    ELSE
      RAISE NOTICE 'View public.% sudah tidak ada, dilewati.', v_view;
    END IF;
  END LOOP;
END $$;

DROP VIEW IF EXISTS public.institution_programs;
DROP VIEW IF EXISTS public.ngo_programs;
DROP VIEW IF EXISTS public.company_csr_programs;

-- Verifikasi akhir: pastikan tabel baru masih ada dan berisi data
DO $$
DECLARE
  v_ngo BIGINT;
  v_enriched BIGINT;
BEGIN
  SELECT count(*) INTO v_ngo FROM public.ngo_managed_programs;
  SELECT count(*) INTO v_enriched FROM public.company_enriched_programs;

  RAISE NOTICE 'ngo_managed_programs: % baris', v_ngo;
  RAISE NOTICE 'company_enriched_programs: % baris', v_enriched;

  IF v_ngo = 0 THEN
    RAISE WARNING 'ngo_managed_programs kosong — periksa apakah data masih ada.';
  END IF;
END $$;

DO $$
BEGIN
  RAISE NOTICE 'Migration 015 selesai. View deprecated berhasil di-drop.';
END $$;

COMMIT;
