-- Migration 000044 Down
DELETE FROM public.crawling_targets WHERE source_type = 'OSS_REGISTRY';
DROP INDEX IF EXISTS idx_oss_investment_status;
DROP INDEX IF EXISTS idx_oss_company_id;
DROP INDEX IF EXISTS idx_oss_kbli_code;
DROP INDEX IF EXISTS idx_oss_nib;
DROP TABLE IF EXISTS public.oss_nib_registrations CASCADE;
