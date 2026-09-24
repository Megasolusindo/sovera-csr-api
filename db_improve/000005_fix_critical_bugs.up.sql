-- =============================================================================
-- 000005_fix_critical_bugs.up.sql
--
-- Tujuan: Perbaiki bug kritis yang sudah aktif di production:
--
--   1. billing_invoices + payment_transactions: CASCADE → RESTRICT
--      (hapus tenant = hapus seluruh riwayat keuangan)
--
--   2. users.email UNIQUE global → UNIQUE (org_id, email)
--      (satu orang tidak bisa join dua tenant)
--
--   3. crawling_targets.target_url UNIQUE global → UNIQUE (org_id, target_url)
--      (dua tenant tidak bisa monitor URL yang sama)
--
--   4. company_signals.content_hash UNIQUE global → UNIQUE (company_id, content_hash)
--      (satu artikel tidak bisa disimpan untuk dua perusahaan sekaligus)
--
--   5. Hapus index duplikat crawling_targets
-- =============================================================================

BEGIN;

-- ----------------------------------------------------------------------------
-- 1. Billing: ganti CASCADE ke RESTRICT untuk data finansial
-- ----------------------------------------------------------------------------
ALTER TABLE public.billing_invoices
  DROP CONSTRAINT IF EXISTS billing_invoices_org_id_fkey;

ALTER TABLE public.billing_invoices
  ADD CONSTRAINT billing_invoices_org_id_fkey
  FOREIGN KEY (org_id) REFERENCES public.organizations(id)
  ON DELETE RESTRICT;

ALTER TABLE public.payment_transactions
  DROP CONSTRAINT IF EXISTS payment_transactions_org_id_fkey;

ALTER TABLE public.payment_transactions
  ADD CONSTRAINT payment_transactions_org_id_fkey
  FOREIGN KEY (org_id) REFERENCES public.organizations(id)
  ON DELETE RESTRICT;

-- Soft delete untuk organizations (jangan hard delete tenant aktif)
ALTER TABLE public.organizations
  ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP WITH TIME ZONE DEFAULT NULL;

CREATE INDEX IF NOT EXISTS idx_organizations_deleted_at
  ON public.organizations (deleted_at)
  WHERE deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- 2. users.email: UNIQUE global → UNIQUE per org
--
-- PERHATIAN: Jika ada email yang sama di dua org berbeda, step ini akan gagal.
-- Cek dulu:
--   SELECT email, count(*) FROM public.users GROUP BY email HAVING count(*) > 1;
-- Jika hasilnya kosong, lanjutkan.
-- ----------------------------------------------------------------------------

-- Cek konflik dulu
DO $$
DECLARE
  v_conflicts BIGINT;
BEGIN
  SELECT count(*) INTO v_conflicts
  FROM (
    SELECT email FROM public.users GROUP BY email HAVING count(*) > 1
  ) t;

  IF v_conflicts > 0 THEN
    RAISE EXCEPTION
      'Ada % email duplikat lintas org. Resolve dulu sebelum migration ini.', v_conflicts;
  END IF;
END $$;

ALTER TABLE public.users
  DROP CONSTRAINT IF EXISTS users_email_key;

ALTER TABLE public.users
  ADD CONSTRAINT users_email_org_key UNIQUE (org_id, email);

-- Index untuk lookup by email saja (login flow) tetap perlu
CREATE INDEX IF NOT EXISTS idx_users_email ON public.users (email);

-- ----------------------------------------------------------------------------
-- 3. crawling_targets.target_url: UNIQUE global → UNIQUE per org
--
-- NULL org_id = target global (tidak terikat tenant), boleh duplikat antar org.
-- UNIQUE hanya berlaku jika org_id IS NOT NULL.
-- ----------------------------------------------------------------------------

-- Hapus dua unique index lama (keduanya identik = bug tersendiri)
DROP INDEX IF EXISTS public.idx_crawling_targets_target_url;
DROP INDEX IF EXISTS public.idx_crawling_targets_unique_url;

-- Unique per (org_id, target_url) untuk yang punya org_id
CREATE UNIQUE INDEX idx_crawling_targets_org_url
  ON public.crawling_targets (org_id, target_url)
  WHERE org_id IS NOT NULL;

-- Global unique hanya untuk target tanpa org (target sistem)
CREATE UNIQUE INDEX idx_crawling_targets_global_url
  ON public.crawling_targets (target_url)
  WHERE org_id IS NULL;

-- ----------------------------------------------------------------------------
-- 4. company_signals.content_hash: UNIQUE global → UNIQUE per company
-- ----------------------------------------------------------------------------
ALTER TABLE intelligence.company_signals
  DROP CONSTRAINT IF EXISTS company_signals_content_hash_key;

-- Partial: hanya enforce uniqueness jika company_id tidak NULL
-- (sinyal tanpa company bisa saja sama content_hash-nya dari monitoring berbeda)
CREATE UNIQUE INDEX IF NOT EXISTS idx_company_signals_company_content_hash
  ON intelligence.company_signals (company_id, content_hash)
  WHERE company_id IS NOT NULL AND content_hash IS NOT NULL;

-- ----------------------------------------------------------------------------
-- 5. company_claims: hapus redundansi is_claimed di companies
--    Ganti dengan partial unique index di company_claims
-- ----------------------------------------------------------------------------
CREATE UNIQUE INDEX IF NOT EXISTS uq_company_claim_approved
  ON public.company_claims (company_id)
  WHERE status = 'APPROVED';

-- is_claimed dan claimed_by_tenant_id di company.companies sekarang
-- bisa di-derive dari company_claims. Biarkan kolom ini untuk sekarang
-- (bisa jadi computed column atau trigger nanti), tapi tambahkan trigger
-- untuk keep-in-sync:

CREATE OR REPLACE FUNCTION public.sync_company_claim_status()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.status = 'APPROVED' THEN
    UPDATE company.companies
    SET is_claimed = true,
        claimed_by_tenant_id = NEW.tenant_id
    WHERE id = NEW.company_id;
  ELSIF NEW.status IN ('REJECTED', 'REVOKED') THEN
    UPDATE company.companies
    SET is_claimed = false,
        claimed_by_tenant_id = NULL
    WHERE id = NEW.company_id;
  END IF;
  RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_sync_company_claim_status ON public.company_claims;

CREATE TRIGGER trg_sync_company_claim_status
  AFTER INSERT OR UPDATE OF status ON public.company_claims
  FOR EACH ROW EXECUTE FUNCTION public.sync_company_claim_status();

DO $$
BEGIN
  RAISE NOTICE 'Bug kritis berhasil diperbaiki.';
END $$;

COMMIT;
