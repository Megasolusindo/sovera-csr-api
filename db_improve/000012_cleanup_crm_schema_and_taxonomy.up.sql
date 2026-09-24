-- =============================================================================
-- 000012_cleanup_crm_schema_and_taxonomy.up.sql
--
-- Tujuan:
--   1. Pindah crm.ai_token_logs → public.ai_token_logs
--   2. Pindah crm.tenant_templates → public.tenant_templates
--   3. Drop schema crm
--   4. Buat tenant_csr_focuses (subset master per tenant)
--   5. Buat tenant_esg_material_topics (subset master per tenant)
--   6. RLS organization_prospects (internal Sovera/admin saja)
--
-- Data aktual:
--   crm.ai_token_logs    = 1.496 baris (aktif, SIGNAL_LLM_EXTRACTION + PITCH_STRATEGY)
--   crm.tenant_templates = 1 baris (template dokumen per tenant)
-- =============================================================================

BEGIN;

-- ----------------------------------------------------------------------------
-- 1. Pindah crm.ai_token_logs → public.ai_token_logs
-- ----------------------------------------------------------------------------
CREATE TABLE public.ai_token_logs (
  id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
  org_id uuid REFERENCES public.organizations(id) ON DELETE SET NULL,
  deal_id uuid REFERENCES public.deal_pipelines(id) ON DELETE SET NULL,
  feature_name varchar(100) NOT NULL,
  model_name varchar(100) NOT NULL,
  prompt_tokens integer NOT NULL DEFAULT 0,
  completion_tokens integer NOT NULL DEFAULT 0,
  total_tokens integer NOT NULL DEFAULT 0,
  estimated_cost_usd numeric(10,6) DEFAULT 0,
  created_at timestamptz DEFAULT now(),

  CONSTRAINT chk_token_counts
    CHECK (total_tokens = prompt_tokens + completion_tokens)
);

-- Migrate data
INSERT INTO public.ai_token_logs
  SELECT * FROM crm.ai_token_logs;

-- Verifikasi
DO $$
DECLARE
  v_src BIGINT;
  v_dst BIGINT;
BEGIN
  SELECT count(*) INTO v_src FROM crm.ai_token_logs;
  SELECT count(*) INTO v_dst FROM public.ai_token_logs;
  IF v_src <> v_dst THEN
    RAISE EXCEPTION 'ai_token_logs: data tidak lengkap! src=%, dst=%', v_src, v_dst;
  END IF;
  RAISE NOTICE 'ai_token_logs: % baris berhasil dipindah', v_dst;
END $$;

-- Index untuk monitoring biaya per tenant per fitur
CREATE INDEX idx_ai_token_logs_org_feature
  ON public.ai_token_logs (org_id, feature_name, created_at DESC);

CREATE INDEX idx_ai_token_logs_model
  ON public.ai_token_logs (model_name, created_at DESC);

-- Partisi bulanan akan berguna di masa depan (1.491 baris/bulan dari SIGNAL_LLM_EXTRACTION)
-- Tidak dipartisi sekarang, tapi tambahkan index created_at untuk query range
CREATE INDEX idx_ai_token_logs_created_at
  ON public.ai_token_logs (created_at DESC);

-- RLS: tenant hanya lihat log token mereka sendiri
-- Admin Sovera lihat semua (via BYPASSRLS atau role khusus)
ALTER TABLE public.ai_token_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.ai_token_logs FORCE ROW LEVEL SECURITY;

CREATE POLICY ai_token_logs_tenant_access ON public.ai_token_logs
  USING (org_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid);

-- ----------------------------------------------------------------------------
-- 2. Pindah crm.tenant_templates → public.tenant_templates
-- ----------------------------------------------------------------------------

-- Cek struktur crm.tenant_templates dulu
-- Dari screenshot: id, org_id, docx_s3_key, pptx_s3_key
CREATE TABLE public.tenant_templates (
  id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
  org_id uuid NOT NULL REFERENCES public.organizations(id) ON DELETE CASCADE,
  docx_s3_key text,   -- S3 key template Word (.docx)
  pptx_s3_key text,   -- S3 key template PowerPoint (.pptx)
  brand_primary_color varchar(32) DEFAULT '#047857',
  logo_s3_key text,   -- S3 key logo tenant
  -- Satu tenant satu template set
  CONSTRAINT uq_tenant_templates_org UNIQUE (org_id),
  created_at timestamptz DEFAULT now(),
  updated_at timestamptz DEFAULT now()
);

-- Migrate data (explicit column mapping: source has brand_primary_color + logo_s3_key, no created_at)
INSERT INTO public.tenant_templates (
  id, org_id, docx_s3_key, pptx_s3_key, brand_primary_color, logo_s3_key, updated_at
)
SELECT
  id, org_id, docx_s3_key, pptx_s3_key, brand_primary_color, logo_s3_key, updated_at
FROM crm.tenant_templates;

-- Verifikasi
DO $$
DECLARE
  v_src BIGINT;
  v_dst BIGINT;
BEGIN
  SELECT count(*) INTO v_src FROM crm.tenant_templates;
  SELECT count(*) INTO v_dst FROM public.tenant_templates;
  IF v_src <> v_dst THEN
    RAISE EXCEPTION 'tenant_templates: data tidak lengkap! src=%, dst=%', v_src, v_dst;
  END IF;
  RAISE NOTICE 'tenant_templates: % baris berhasil dipindah', v_dst;
END $$;

DROP TRIGGER IF EXISTS trg_set_updated_at_tenant_templates ON public.tenant_templates;
CREATE TRIGGER trg_set_updated_at_tenant_templates
  BEFORE UPDATE ON public.tenant_templates
  FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

-- RLS
ALTER TABLE public.tenant_templates ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.tenant_templates FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_templates_access ON public.tenant_templates
  USING (org_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid)
  WITH CHECK (org_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid);

-- ----------------------------------------------------------------------------
-- 3. Drop schema crm (semua tabel sudah dipindah atau di-drop di migration 003)
-- ----------------------------------------------------------------------------
DROP TABLE IF EXISTS crm.ai_token_logs;
DROP TABLE IF EXISTS crm.tenant_templates;
DROP SCHEMA IF EXISTS crm;

-- ----------------------------------------------------------------------------
-- 4. tenant_csr_focuses
--    Tenant pilih subset dari master csr_focuses
-- ----------------------------------------------------------------------------
CREATE TABLE public.tenant_csr_focuses (
  org_id uuid NOT NULL
    REFERENCES public.organizations(id) ON DELETE CASCADE,
  focus_id uuid NOT NULL
    REFERENCES public.csr_focuses(id) ON DELETE CASCADE,
  PRIMARY KEY (org_id, focus_id),
  created_at timestamptz DEFAULT now()
);

CREATE INDEX idx_tenant_csr_focuses_org_id
  ON public.tenant_csr_focuses (org_id);

CREATE INDEX idx_tenant_csr_focuses_focus_id
  ON public.tenant_csr_focuses (focus_id);

-- RLS
ALTER TABLE public.tenant_csr_focuses ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.tenant_csr_focuses FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_csr_focuses_access ON public.tenant_csr_focuses
  USING (org_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid)
  WITH CHECK (org_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid);

-- View helper: csr_focuses yang dipilih tenant, fallback ke semua jika belum pilih
-- Berguna untuk dropdown di UI — jika tenant belum setup, tampilkan semua master
CREATE VIEW public.effective_csr_focuses AS
SELECT
  f.id,
  f.name,
  f.description,
  tf.org_id,
  -- NULL org_id berarti ini dari master (belum ada pilihan tenant)
  CASE WHEN tf.org_id IS NOT NULL THEN true ELSE false END AS is_selected
FROM public.csr_focuses f
LEFT JOIN public.tenant_csr_focuses tf
  ON tf.focus_id = f.id
  AND tf.org_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid
ORDER BY f.name;

-- ----------------------------------------------------------------------------
-- 5. tenant_esg_material_topics
--    Pola sama dengan tenant_csr_focuses
-- ----------------------------------------------------------------------------
CREATE TABLE public.tenant_esg_material_topics (
  org_id uuid NOT NULL
    REFERENCES public.organizations(id) ON DELETE CASCADE,
  topic_id uuid NOT NULL
    REFERENCES public.esg_material_topics(id) ON DELETE CASCADE,
  PRIMARY KEY (org_id, topic_id),
  created_at timestamptz DEFAULT now()
);

CREATE INDEX idx_tenant_esg_topics_org_id
  ON public.tenant_esg_material_topics (org_id);

CREATE INDEX idx_tenant_esg_topics_topic_id
  ON public.tenant_esg_material_topics (topic_id);

-- RLS
ALTER TABLE public.tenant_esg_material_topics ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.tenant_esg_material_topics FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_esg_topics_access ON public.tenant_esg_material_topics
  USING (org_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid)
  WITH CHECK (org_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid);

-- View helper: esg_material_topics yang dipilih tenant
CREATE VIEW public.effective_esg_material_topics AS
SELECT
  t.id,
  t.name,
  t.description,
  t.category,  -- asumsi ada kolom category di esg_material_topics
  tt.org_id,
  CASE WHEN tt.org_id IS NOT NULL THEN true ELSE false END AS is_selected
FROM public.esg_material_topics t
LEFT JOIN public.tenant_esg_material_topics tt
  ON tt.topic_id = t.id
  AND tt.org_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid
ORDER BY t.category, t.name;

-- ----------------------------------------------------------------------------
-- 6. organization_prospects — RLS untuk internal Sovera
--
-- organization_prospects hanya untuk tim internal Sovera.
-- Tidak ada app.current_org_id yang cocok karena ini data lintas tenant.
-- Solusi: gunakan role Postgres khusus untuk akses internal.
--
-- Pola:
--   - Role 'sovera_internal' di-BYPASSRLS atau punya policy khusus
--   - Role 'sovera_app' (yang dipakai API biasa) tidak bisa akses sama sekali
-- ----------------------------------------------------------------------------
ALTER TABLE public.organization_prospects ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.organization_prospects FORCE ROW LEVEL SECURITY;

-- Policy: hanya role internal yang bisa akses
-- current_user adalah Postgres role, bukan app user
CREATE POLICY prospects_internal_only ON public.organization_prospects
  USING (current_user = 'sovera_internal');

-- Jika belum ada role sovera_internal, buat dulu (tanpa password; ops set via ALTER ROLE)
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'sovera_internal') THEN
    CREATE ROLE sovera_internal NOSUPERUSER NOCREATEDB NOCREATEROLE;
    RAISE NOTICE 'Role sovera_internal dibuat (LOGIN/password di-set manual oleh ops).';
  ELSE
    RAISE NOTICE 'Role sovera_internal sudah ada.';
  END IF;
END $$;
GRANT SELECT, INSERT, UPDATE, DELETE ON public.organization_prospects TO sovera_internal;
--
-- Role sovera_app (dipakai API publik) tidak di-GRANT ke tabel ini,
-- sehingga query dari API biasa akan return 0 baris (bukan error).
--
-- Catatan untuk Go backend:
-- Koneksi database untuk internal admin dashboard pakai kredensial
-- dengan role sovera_internal, bukan role yang sama dengan API publik.

-- ----------------------------------------------------------------------------
-- Verifikasi akhir
-- ----------------------------------------------------------------------------
DO $$
BEGIN
  -- Pastikan schema crm sudah kosong dan bisa di-drop
  IF EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema = 'crm'
  ) THEN
    RAISE WARNING 'Schema crm masih punya tabel. Periksa manual.';
  ELSE
    RAISE NOTICE 'Schema crm berhasil di-drop.';
  END IF;

  RAISE NOTICE 'Migration 012 selesai.';
END $$;

COMMIT;
