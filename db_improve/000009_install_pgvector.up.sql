-- =============================================================================
-- 000009_install_pgvector.up.sql
--
-- Tujuan: Ganti domain public.vector (double precision[]) dengan pgvector,
--         sehingga similarity search bisa pakai index HNSW.
--
-- Prasyarat: pgvector extension sudah terinstall di server PostgreSQL.
--   Cek: SELECT * FROM pg_available_extensions WHERE name = 'vector';
--   Install di OS (Debian): apt install postgresql-18-pgvector
--
-- Kondisi awal (terukur):
--   institution_programs: 377 baris, 0 ada embedding → konversi trivial
--   company_signals.signal_embedding: perlu dicek dimensinya
-- =============================================================================

BEGIN;

-- ----------------------------------------------------------------------------
-- 1. Buang kolom yang masih memakai domain public.vector
-- ----------------------------------------------------------------------------
ALTER TABLE public.institution_programs
  DROP COLUMN IF EXISTS program_embedding;

ALTER TABLE intelligence.company_signals
  DROP COLUMN IF EXISTS signal_embedding;

-- ----------------------------------------------------------------------------
-- 2. Drop domain public.vector yang bentrok namanya dengan tipe ekstensi
-- ----------------------------------------------------------------------------
DROP DOMAIN IF EXISTS public.vector CASCADE;

-- ----------------------------------------------------------------------------
-- 3. Pasang pgvector extension
-- ----------------------------------------------------------------------------
CREATE EXTENSION IF NOT EXISTS vector;

-- ----------------------------------------------------------------------------
-- 4. Recreate kolom embedding dengan tipe pgvector
--    Sesuaikan 768 dengan dimensi model yang dipakai (text-embedding-004 = 768)
-- ----------------------------------------------------------------------------
ALTER TABLE public.institution_programs
  ADD COLUMN IF NOT EXISTS program_embedding vector(768);

ALTER TABLE intelligence.company_signals
  ADD COLUMN IF NOT EXISTS signal_embedding vector(768);

-- ----------------------------------------------------------------------------
-- 5. HNSW index untuk similarity search (kosong dulu, tapi struktur jadi ada)
-- ----------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_institution_programs_embedding
  ON public.institution_programs
  USING hnsw (program_embedding vector_cosine_ops)
  WITH (m = 16, ef_construction = 64);

CREATE INDEX IF NOT EXISTS idx_company_signals_embedding
  ON intelligence.company_signals
  USING hnsw (signal_embedding vector_cosine_ops)
  WITH (m = 16, ef_construction = 64);

-- ----------------------------------------------------------------------------
-- 6. Helper function untuk partner matching (NGO vs perusahaan)
-- ----------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION public.match_institution_programs(
  query_embedding vector(768),
  match_threshold float DEFAULT 0.7,
  match_count int DEFAULT 10,
  p_org_id uuid DEFAULT NULL
)
RETURNS TABLE (
  id uuid,
  org_id uuid,
  name varchar,
  description text,
  primary_cluster varchar,
  target_sdgs text[],
  similarity float
)
LANGUAGE plpgsql AS $$
BEGIN
  RETURN QUERY
  SELECT
    ip.id,
    ip.org_id,
    ip.title,
    ip.description,
    ip.primary_cluster,
    ip.target_sdgs,
    1 - (ip.program_embedding <=> query_embedding) AS similarity
  FROM public.institution_programs ip
  WHERE
    ip.program_embedding IS NOT NULL
    AND 1 - (ip.program_embedding <=> query_embedding) > match_threshold
    AND (p_org_id IS NULL OR ip.org_id = p_org_id)
  ORDER BY ip.program_embedding <=> query_embedding
  LIMIT match_count;
END;
$$;

DO $$
BEGIN
  RAISE NOTICE 'pgvector berhasil diinstall. Index HNSW dan fungsi matching siap.';
END $$;

COMMIT;
