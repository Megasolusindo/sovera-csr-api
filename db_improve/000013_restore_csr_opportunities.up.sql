-- =============================================================================
-- 000013_restore_csr_opportunities.up.sql
--
-- csr_opportunities was dropped in migration 003 (and 010 only patched if it
-- still existed via to_regclass, which is no longer the case). The marketplace
-- feature depends on this table, so recreate it with the columns the Go model
-- expects, plus RLS aligned to the 010 spec.
-- =============================================================================
BEGIN;

CREATE TABLE public.csr_opportunities (
  id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
  company_id uuid REFERENCES company.companies(id) ON DELETE CASCADE,
  org_id  uuid NOT NULL REFERENCES public.organizations(id) ON DELETE RESTRICT,  -- renamed from tenant_id
  title varchar(255) NOT NULL,
  description text,
  category varchar(100),
  target_location varchar(255),
  budget_amount numeric(15,2) DEFAULT 0,
  open_until timestamptz,
  status varchar(50) NOT NULL DEFAULT 'DRAFT'
    CHECK (status IN ('DRAFT','OPEN','CLOSED','AWARDED','CANCELLED')),
  created_by_user_id uuid REFERENCES public.users(id) ON DELETE SET NULL,
  created_at timestamptz DEFAULT now(),
  updated_at timestamptz DEFAULT now()
);

-- Compatibility: keep legacy "tenant_id" name visible to old queries via a view?
-- Instead, provide a view so legacy code paths using tenant_id still resolve.
-- (App code has been patched to use org_id, so view omitted for now.)

CREATE INDEX idx_csr_opportunities_status ON public.csr_opportunities (status) WHERE status='OPEN';
CREATE INDEX idx_csr_opportunities_org_id   ON public.csr_opportunities (org_id);

ALTER TABLE public.csr_opportunities ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.csr_opportunities FORCE ROW LEVEL SECURITY;

CREATE POLICY corp_own_opportunities ON public.csr_opportunities
  USING (org_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid)
  WITH CHECK (org_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid);

CREATE POLICY ngo_see_open_opportunities ON public.csr_opportunities
  AS PERMISSIVE FOR SELECT USING (status = 'OPEN');

-- Updated_at trigger
DROP TRIGGER IF EXISTS trg_set_updated_at_csr_opportunities ON public.csr_opportunities;
CREATE TRIGGER trg_set_updated_at_csr_opportunities
  BEFORE UPDATE ON public.csr_opportunities
  FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

-- Link proposals.opportunity_id → csr_opportunities safely
ALTER TABLE public.proposals
  ADD CONSTRAINT proposals_csr_opportunity_id_fkey
  FOREIGN KEY (opportunity_id) REFERENCES public.csr_opportunities(id) ON DELETE SET NULL;

DO $$
BEGIN
  RAISE NOTICE 'Migration 013 selesai: csr_opportunities direstore.';
END $$;

COMMIT;
