-- Migration 000039 Up: Add visibility column to company_csr_programs

ALTER TABLE company_csr_programs
    ADD COLUMN IF NOT EXISTS visibility VARCHAR(20) DEFAULT 'public';

-- Backfill existing rows to 'public' (default)
UPDATE company_csr_programs
    SET visibility = 'public'
    WHERE visibility IS NULL;

-- Enforce NOT NULL after backfill
ALTER TABLE company_csr_programs
    ALTER COLUMN visibility SET NOT NULL;

-- Add check constraint for valid visibility values
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid = 'company_csr_programs'::regclass
          AND conname = 'chk_program_visibility'
    ) THEN
        ALTER TABLE company_csr_programs
            ADD CONSTRAINT chk_program_visibility
            CHECK (visibility IN ('public', 'curated', 'private'));
    END IF;
END $$;

-- Index for visibility filtering on explore/catalog endpoints
CREATE INDEX IF NOT EXISTS idx_company_csr_programs_visibility ON company_csr_programs(visibility);
