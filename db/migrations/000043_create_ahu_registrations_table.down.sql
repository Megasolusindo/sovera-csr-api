-- Migration 000043 Down
DELETE FROM public.crawling_targets WHERE source_type = 'AHU_REGISTRY';
DROP INDEX IF EXISTS idx_ahu_company_id;
DROP INDEX IF EXISTS idx_ahu_company_name;
DROP INDEX IF EXISTS idx_ahu_number;
DROP TABLE IF EXISTS public.ahu_registrations CASCADE;
