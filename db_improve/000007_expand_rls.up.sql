-- =============================================================================
-- 000007_expand_rls.up.sql
--
-- Tujuan: Perluas RLS ke tabel multi-tenant yang saat ini tidak terlindungi.
--
-- Saat ini RLS hanya aktif di:
--   - deal_pipelines        (app.current_org_id)
--   - institution_programs  (app.current_org_id)
--   - crawling_targets      (app.current_tenant_id) ← nama variabel berbeda!
--
-- Setelah migration ini: semua tabel sensitif per-tenant dilindungi RLS,
-- dan nama variabel sesi distandarkan ke app.current_org_id.
--
-- Catatan untuk Go backend (sovera-csr-api):
--   Setiap request harus SET LOCAL app.current_org_id = '<uuid>' dalam transaksi.
--   Jangan pakai SET (tanpa LOCAL) — akan bocor ke koneksi berikutnya di pool.
-- =============================================================================

BEGIN;

-- Standarisasi nama variabel di crawling_targets (sebelumnya app.current_tenant_id)
DROP POLICY IF EXISTS crawling_targets_org_isolation ON public.crawling_targets;

CREATE POLICY crawling_targets_org_isolation ON public.crawling_targets
  USING (
    org_id IS NULL  -- target sistem/global, semua bisa baca
    OR org_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid
  );

-- ----------------------------------------------------------------------------
-- Tabel yang perlu RLS baru
-- Pola: ENABLE → CREATE POLICY USING + WITH CHECK
-- ----------------------------------------------------------------------------

-- proposals
ALTER TABLE public.proposals ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.proposals FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_proposals ON public.proposals
  USING (org_tenant_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid
    OR corp_tenant_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid)
  WITH CHECK (org_tenant_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid);

-- ngo_programs
ALTER TABLE public.ngo_programs ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.ngo_programs FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_ngo_programs ON public.ngo_programs
  USING (tenant_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid)
  WITH CHECK (tenant_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid);

-- crm_contacts
ALTER TABLE public.crm_contacts ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.crm_contacts FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_crm_contacts ON public.crm_contacts
  USING (org_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid)
  WITH CHECK (org_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid);

-- crm_activities
ALTER TABLE public.crm_activities ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.crm_activities FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_crm_activities ON public.crm_activities
  USING (org_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid)
  WITH CHECK (org_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid);

-- tenant_saved_items
ALTER TABLE public.tenant_saved_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.tenant_saved_items FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_saved_items ON public.tenant_saved_items
  USING (org_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid)
  WITH CHECK (org_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid);

-- billing_invoices (setelah CASCADE → RESTRICT di migration 005)
ALTER TABLE public.billing_invoices ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.billing_invoices FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_billing ON public.billing_invoices
  USING (org_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid)
  WITH CHECK (org_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid);

-- payment_transactions
ALTER TABLE public.payment_transactions ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.payment_transactions FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_payments ON public.payment_transactions
  USING (org_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid)
  WITH CHECK (org_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid);

-- ----------------------------------------------------------------------------
-- AI Chat: isolasi per user dalam tenant
-- Ini untuk fitur dashboard chat yang sedang dibangun
-- ----------------------------------------------------------------------------
ALTER TABLE public.organization_ai_conversations ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.organization_ai_conversations FORCE ROW LEVEL SECURITY;

-- User hanya lihat conversation miliknya sendiri
CREATE POLICY conv_user_isolation ON public.organization_ai_conversations
  USING (
    org_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid
    AND user_id = (NULLIF(current_setting('app.current_user_id', true), ''))::uuid
  )
  WITH CHECK (
    org_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid
    AND user_id = (NULLIF(current_setting('app.current_user_id', true), ''))::uuid
  );

ALTER TABLE public.organization_ai_chat_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.organization_ai_chat_logs FORCE ROW LEVEL SECURITY;

CREATE POLICY chat_log_user_isolation ON public.organization_ai_chat_logs
  USING (
    org_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid
    AND user_id = (NULLIF(current_setting('app.current_user_id', true), ''))::uuid
  )
  WITH CHECK (
    org_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid
    AND user_id = (NULLIF(current_setting('app.current_user_id', true), ''))::uuid
  );

-- Tambah variabel sesi yang dibutuhkan ke Go backend:
-- Di setiap request handler, sebelum query:
--   db.ExecContext(ctx, "SET LOCAL app.current_org_id = $1", orgID)
--   db.ExecContext(ctx, "SET LOCAL app.current_user_id = $1", userID)
-- Keduanya dalam satu transaksi yang sama dengan query utama.

-- ----------------------------------------------------------------------------
-- Composite FK untuk memastikan user_id berasal dari org yang sama
-- (mencegah user dari org A membuat conversation di org B)
-- ----------------------------------------------------------------------------

-- Tambahkan unique constraint dulu di users
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='public.users'::regclass AND conname='users_id_org_key') THEN
    ALTER TABLE public.users ADD CONSTRAINT users_id_org_key UNIQUE (id, org_id);
  END IF;
END $$;

-- FK composite di conversations
ALTER TABLE public.organization_ai_conversations
  ADD CONSTRAINT conv_user_in_org
  FOREIGN KEY (user_id, org_id) REFERENCES public.users(id, org_id)
  ON DELETE CASCADE;

-- FK composite di chat logs
ALTER TABLE public.organization_ai_chat_logs
  ADD CONSTRAINT chat_log_user_in_org
  FOREIGN KEY (user_id, org_id) REFERENCES public.users(id, org_id)
  ON DELETE CASCADE;

DO $$
BEGIN
  RAISE NOTICE 'RLS berhasil diperluas ke semua tabel multi-tenant.';
END $$;

COMMIT;
