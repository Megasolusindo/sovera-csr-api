-- =============================================================================
-- 000011_partnership_milestones.up.sql
--
-- Tujuan: Buat tabel untuk tracking eksekusi kemitraan pasca proposal diterima.
--
-- Struktur:
--   proposals
--     ├─ partnership_milestones   (tahapan eksekusi, bisa paralel)
--     │       ├─ milestone_disbursements  (pencairan dana: DP + pelunasan)
--     │       └─ milestone_reports        (bukti pelaksanaan oleh NGO)
--     └─ impact_reports           (laporan final per proposal)
--
-- Keputusan desain:
--   - Milestone bisa paralel → order_index untuk urutan tampilan saja
--   - Disbursement: DP di awal, pelunasan setelah verified
--   - Bukti: NGO upload → korporasi verifikasi
--   - Impact report: per milestone + laporan final per proposal
-- =============================================================================

BEGIN;

-- ----------------------------------------------------------------------------
-- 1. partnership_milestones
-- ----------------------------------------------------------------------------
CREATE TABLE public.partnership_milestones (
  id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
  proposal_id uuid NOT NULL
    REFERENCES public.proposals(id) ON DELETE RESTRICT,

  ngo_org_id uuid NOT NULL
    REFERENCES public.organizations(id) ON DELETE RESTRICT,
  corp_org_id uuid NOT NULL
    REFERENCES public.organizations(id) ON DELETE RESTRICT,

  title varchar(255) NOT NULL,
  description text,
  order_index integer NOT NULL DEFAULT 0,

  due_date date NOT NULL,
  started_at timestamptz,
  completed_at timestamptz,

  budget_allocated numeric(15,2) NOT NULL DEFAULT 0
    CHECK (budget_allocated >= 0),

  status varchar(50) NOT NULL DEFAULT 'PENDING'
    CHECK (status IN (
      'PENDING',
      'IN_PROGRESS',
      'SUBMITTED',
      'VERIFIED',
      'COMPLETED',
      'OVERDUE',
      'CANCELLED'
    )),

  created_by_org_id uuid REFERENCES public.organizations(id) ON DELETE SET NULL,
  created_by_user_id uuid REFERENCES public.users(id) ON DELETE SET NULL,
  created_at timestamptz DEFAULT now(),
  updated_at timestamptz DEFAULT now(),

  CONSTRAINT chk_milestone_dates
    CHECK (completed_at IS NULL OR completed_at >= started_at)
);

CREATE INDEX idx_milestones_proposal_id
  ON public.partnership_milestones (proposal_id);

CREATE INDEX idx_milestones_ngo_org_id
  ON public.partnership_milestones (ngo_org_id, status);

CREATE INDEX idx_milestones_corp_org_id
  ON public.partnership_milestones (corp_org_id, status);

CREATE INDEX idx_milestones_due_date
  ON public.partnership_milestones (due_date)
  WHERE status NOT IN ('COMPLETED', 'CANCELLED');

ALTER TABLE public.partnership_milestones ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.partnership_milestones FORCE ROW LEVEL SECURITY;

CREATE POLICY milestone_tenant_access ON public.partnership_milestones
  USING (
    ngo_org_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid
    OR corp_org_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid
  );

-- ----------------------------------------------------------------------------
-- 2. milestone_disbursements
--    budget_disbursed dihitung dari tabel ini, bukan disimpan di milestone
--    untuk menghindari dua sumber kebenaran
-- ----------------------------------------------------------------------------
CREATE TABLE public.milestone_disbursements (
  id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
  milestone_id uuid NOT NULL
    REFERENCES public.partnership_milestones(id) ON DELETE RESTRICT,

  disbursement_type varchar(20) NOT NULL
    CHECK (disbursement_type IN ('DOWN_PAYMENT', 'SETTLEMENT', 'ADDITIONAL')),

  amount numeric(15,2) NOT NULL
    CHECK (amount > 0),

  status varchar(50) NOT NULL DEFAULT 'PENDING'
    CHECK (status IN ('PENDING', 'PROCESSING', 'DISBURSED', 'FAILED')),

  disbursed_at timestamptz,
  notes text,

  payment_transaction_id uuid
    REFERENCES public.payment_transactions(id) ON DELETE SET NULL,

  created_by_user_id uuid REFERENCES public.users(id) ON DELETE SET NULL,
  created_at timestamptz DEFAULT now(),
  updated_at timestamptz DEFAULT now(),

  CONSTRAINT chk_disbursed_at
    CHECK (disbursed_at IS NULL OR status = 'DISBURSED')
);

CREATE INDEX idx_disbursements_milestone_id
  ON public.milestone_disbursements (milestone_id);

CREATE INDEX idx_disbursements_pending
  ON public.milestone_disbursements (status)
  WHERE status IN ('PENDING', 'PROCESSING');

ALTER TABLE public.milestone_disbursements ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.milestone_disbursements FORCE ROW LEVEL SECURITY;

CREATE POLICY disbursement_tenant_access ON public.milestone_disbursements
  USING (
    EXISTS (
      SELECT 1 FROM public.partnership_milestones m
      WHERE m.id = milestone_id
        AND (
          m.ngo_org_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid
          OR m.corp_org_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid
        )
    )
  );

-- View helper: total disbursed per milestone
CREATE VIEW public.milestone_budget_summary AS
SELECT
  m.id AS milestone_id,
  m.budget_allocated,
  COALESCE(SUM(d.amount) FILTER (WHERE d.status = 'DISBURSED'), 0)
    AS budget_disbursed,
  m.budget_allocated -
    COALESCE(SUM(d.amount) FILTER (WHERE d.status = 'DISBURSED'), 0)
    AS budget_remaining,
  COALESCE(SUM(d.amount) FILTER (
    WHERE d.disbursement_type = 'DOWN_PAYMENT' AND d.status = 'DISBURSED'
  ), 0) AS down_payment_disbursed,
  COALESCE(SUM(d.amount) FILTER (
    WHERE d.disbursement_type = 'SETTLEMENT' AND d.status = 'DISBURSED'
  ), 0) AS settlement_disbursed
FROM public.partnership_milestones m
LEFT JOIN public.milestone_disbursements d ON d.milestone_id = m.id
GROUP BY m.id, m.budget_allocated;

-- ----------------------------------------------------------------------------
-- 3. milestone_reports
--    NGO upload bukti → korporasi verifikasi
-- ----------------------------------------------------------------------------
CREATE TABLE public.milestone_reports (
  id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
  milestone_id uuid NOT NULL
    REFERENCES public.partnership_milestones(id) ON DELETE RESTRICT,

  submitted_by_org_id uuid NOT NULL
    REFERENCES public.organizations(id) ON DELETE RESTRICT,
  submitted_by_user_id uuid
    REFERENCES public.users(id) ON DELETE SET NULL,

  title varchar(255) NOT NULL,
  narrative text NOT NULL,

  beneficiaries_reached integer DEFAULT 0,
  beneficiaries_reached_unit varchar(100),

  sdg_outcomes jsonb DEFAULT '{}',
  -- {"SDG-4": {"metric": "siswa terbantu", "value": 120}}

  evidence_urls text[] DEFAULT '{}',

  verification_status varchar(50) NOT NULL DEFAULT 'PENDING'
    CHECK (verification_status IN (
      'PENDING',
      'APPROVED',
      'REJECTED',
      'REVISION'
    )),

  verified_by_user_id uuid REFERENCES public.users(id) ON DELETE SET NULL,
  verified_at timestamptz,
  verification_notes text,

  CONSTRAINT chk_verified_at
    CHECK (verified_at IS NULL OR verification_status IN ('APPROVED', 'REJECTED', 'REVISION')),

  submitted_at timestamptz DEFAULT now(),
  created_at timestamptz DEFAULT now(),
  updated_at timestamptz DEFAULT now()
);

CREATE INDEX idx_milestone_reports_milestone_id
  ON public.milestone_reports (milestone_id);

CREATE INDEX idx_milestone_reports_pending
  ON public.milestone_reports (verification_status)
  WHERE verification_status = 'PENDING';

ALTER TABLE public.milestone_reports ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.milestone_reports FORCE ROW LEVEL SECURITY;

CREATE POLICY report_tenant_access ON public.milestone_reports
  USING (
    submitted_by_org_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid
    OR EXISTS (
      SELECT 1 FROM public.partnership_milestones m
      WHERE m.id = milestone_id
        AND m.corp_org_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid
    )
  );

-- ----------------------------------------------------------------------------
-- 4. impact_reports
--    Laporan final per proposal — satu per kemitraan
-- ----------------------------------------------------------------------------
CREATE TABLE public.impact_reports (
  id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
  proposal_id uuid NOT NULL
    REFERENCES public.proposals(id) ON DELETE RESTRICT,

  CONSTRAINT uq_impact_report_proposal UNIQUE (proposal_id),

  submitted_by_org_id uuid NOT NULL
    REFERENCES public.organizations(id) ON DELETE RESTRICT,
  submitted_by_user_id uuid
    REFERENCES public.users(id) ON DELETE SET NULL,

  reporting_period_start date NOT NULL,
  reporting_period_end date NOT NULL,

  CONSTRAINT chk_impact_report_period
    CHECK (reporting_period_end > reporting_period_start),

  total_beneficiaries_reached integer DEFAULT 0,
  total_beneficiaries_reached_unit varchar(100),
  total_budget_utilized numeric(15,2) DEFAULT 0,

  sdg_outcomes jsonb DEFAULT '{}',

  executive_summary text NOT NULL,
  challenges text,
  lessons_learned text,
  recommendations text,

  evidence_urls text[] DEFAULT '{}',
  full_report_url text,

  verification_status varchar(50) NOT NULL DEFAULT 'PENDING'
    CHECK (verification_status IN (
      'PENDING',
      'APPROVED',
      'REJECTED',
      'REVISION'
    )),

  verified_by_user_id uuid REFERENCES public.users(id) ON DELETE SET NULL,
  verified_at timestamptz,
  verification_notes text,

  submitted_at timestamptz DEFAULT now(),
  created_at timestamptz DEFAULT now(),
  updated_at timestamptz DEFAULT now()
);

CREATE INDEX idx_impact_reports_proposal_id
  ON public.impact_reports (proposal_id);

CREATE INDEX idx_impact_reports_pending
  ON public.impact_reports (verification_status)
  WHERE verification_status = 'PENDING';

ALTER TABLE public.impact_reports ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.impact_reports FORCE ROW LEVEL SECURITY;

CREATE POLICY impact_report_tenant_access ON public.impact_reports
  USING (
    submitted_by_org_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid
    OR EXISTS (
      SELECT 1 FROM public.proposals p
      WHERE p.id = proposal_id
        AND p.corp_tenant_id = (NULLIF(current_setting('app.current_org_id', true), ''))::uuid
    )
  );

CREATE OR REPLACE FUNCTION public.set_updated_at()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$;

DO $$
DECLARE
  tbl TEXT;
BEGIN
  FOREACH tbl IN ARRAY ARRAY[
    'public.partnership_milestones',
    'public.milestone_disbursements',
    'public.milestone_reports',
    'public.impact_reports'
  ]
  LOOP
    EXECUTE format('DROP TRIGGER IF EXISTS trg_set_updated_at_%s ON %s', replace(tbl, 'public.', ''), tbl);
    EXECUTE format('CREATE TRIGGER trg_set_updated_at_%s BEFORE UPDATE ON %s FOR EACH ROW EXECUTE FUNCTION public.set_updated_at()', replace(tbl, 'public.', ''), tbl);
  END LOOP;
END $$;

-- ----------------------------------------------------------------------------
-- 6. Trigger: auto-update status milestone saat report diverifikasi
-- ----------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION public.sync_milestone_status_on_report()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.verification_status = 'APPROVED'
    AND OLD.verification_status <> 'APPROVED' THEN
    UPDATE public.partnership_milestones
    SET status = 'VERIFIED', updated_at = now()
    WHERE id = NEW.milestone_id AND status = 'SUBMITTED';
  END IF;

  IF NEW.verification_status IN ('REJECTED', 'REVISION')
    AND OLD.verification_status = 'PENDING' THEN
    UPDATE public.partnership_milestones
    SET status = 'IN_PROGRESS', updated_at = now()
    WHERE id = NEW.milestone_id AND status = 'SUBMITTED';
  END IF;

  RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_sync_milestone_status ON public.milestone_reports;
CREATE TRIGGER trg_sync_milestone_status
  AFTER UPDATE OF verification_status ON public.milestone_reports
  FOR EACH ROW EXECUTE FUNCTION public.sync_milestone_status_on_report();

DO $$
BEGIN
  RAISE NOTICE 'Migration 011 selesai.';
END $$;

COMMIT;
