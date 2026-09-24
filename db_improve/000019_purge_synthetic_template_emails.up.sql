-- =============================================================================
-- 000019_purge_synthetic_template_emails.up.sql
--
-- Tujuan: Mengosongkan (SET NULL) email template sintetis berformat 'csr@{slug}.co.id'
--         pada tabel public.company_csr_profiles agar sesuai dengan Zero Data Mocking Directive.
-- =============================================================================

BEGIN;

UPDATE public.company_csr_profiles
SET csr_email_public = NULL,
    updated_at = NOW()
WHERE csr_email_public ILIKE 'csr@%';

COMMIT;
