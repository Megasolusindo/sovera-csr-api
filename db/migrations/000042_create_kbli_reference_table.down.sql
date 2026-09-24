-- Migration 000042 Down
ALTER TABLE company.companies DROP CONSTRAINT IF EXISTS companies_kbli_code_fkey;
DROP INDEX IF EXISTS idx_kbli_csr_relevance;
DROP INDEX IF EXISTS idx_kbli_category;
DROP TABLE IF EXISTS public.kbli_reference CASCADE;
