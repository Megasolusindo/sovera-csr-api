-- Migration 000034 Down: Rollback Two-Sided Marketplace Schema

DROP TABLE IF EXISTS user_invitations;
DROP TABLE IF EXISTS proposals;
DROP TABLE IF EXISTS ngo_programs;
DROP TABLE IF EXISTS csr_opportunities;
DROP TABLE IF EXISTS company_claims;

ALTER TABLE companies 
    DROP COLUMN IF EXISTS claimed_by_tenant_id,
    DROP COLUMN IF EXISTS corporate_domain,
    DROP COLUMN IF EXISTS is_claimed;

ALTER TABLE organizations
    DROP COLUMN IF EXISTS is_verified,
    DROP COLUMN IF EXISTS logo_url,
    DROP COLUMN IF EXISTS slug,
    DROP COLUMN IF EXISTS company_id,
    DROP COLUMN IF EXISTS type;

DROP TYPE IF EXISTS proposal_status_enum;
DROP TYPE IF EXISTS claim_method_enum;
DROP TYPE IF EXISTS claim_status_enum;
DROP TYPE IF EXISTS tenant_type_enum;
