package repository

import (
	"context"
	"fmt"
	"time"

	"sovera-core-api/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CompanyClaimRepository struct {
	pool *pgxpool.Pool
}

func NewCompanyClaimRepository(pool *pgxpool.Pool) *CompanyClaimRepository {
	return &CompanyClaimRepository{pool: pool}
}

// Create inserts a new company claim request.
func (r *CompanyClaimRepository) Create(ctx context.Context, claim model.CompanyClaim) (*model.CompanyClaim, error) {
	query := `
		INSERT INTO company_claims (company_id, tenant_id, requested_by_user_id, work_email, document_proof_url, method, status, verification_notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, company_id, tenant_id, requested_by_user_id, work_email, document_proof_url, method, status, verification_notes, verified_at, verified_by_user_id, created_at, updated_at;
	`
	var created model.CompanyClaim
	err := r.pool.QueryRow(ctx, query,
		claim.CompanyID, claim.TenantID, claim.RequestedByUserID,
		claim.WorkEmail, claim.DocumentProofURL, claim.Method, claim.Status, claim.VerificationNotes,
	).Scan(
		&created.ID, &created.CompanyID, &created.TenantID, &created.RequestedByUserID,
		&created.WorkEmail, &created.DocumentProofURL, &created.Method, &created.Status,
		&created.VerificationNotes, &created.VerifiedAt, &created.VerifiedByUserID,
		&created.CreatedAt, &created.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create company claim: %w", err)
	}
	return &created, nil
}

// FindByID retrieves a claim by ID.
func (r *CompanyClaimRepository) FindByID(ctx context.Context, id string) (*model.CompanyClaim, error) {
	query := `
		SELECT id, company_id, tenant_id, requested_by_user_id, work_email, document_proof_url, method, status, verification_notes, verified_at, verified_by_user_id, created_at, updated_at
		FROM company_claims
		WHERE id = $1;
	`
	var c model.CompanyClaim
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.CompanyID, &c.TenantID, &c.RequestedByUserID,
		&c.WorkEmail, &c.DocumentProofURL, &c.Method, &c.Status,
		&c.VerificationNotes, &c.VerifiedAt, &c.VerifiedByUserID,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("claim not found: %w", err)
	}
	return &c, nil
}

// UpdateStatus approves or rejects a company claim request.
func (r *CompanyClaimRepository) UpdateStatus(ctx context.Context, id string, status model.ClaimStatus, verifierUserID, notes string) (*model.CompanyClaim, error) {
	now := time.Now()
	query := `
		UPDATE company_claims
		SET status = $2, verified_by_user_id = $3, verification_notes = $4, verified_at = $5, updated_at = NOW()
		WHERE id = $1
		RETURNING id, company_id, tenant_id, requested_by_user_id, work_email, document_proof_url, method, status, verification_notes, verified_at, verified_by_user_id, created_at, updated_at;
	`
	var c model.CompanyClaim
	err := r.pool.QueryRow(ctx, query, id, status, verifierUserID, notes, now).Scan(
		&c.ID, &c.CompanyID, &c.TenantID, &c.RequestedByUserID,
		&c.WorkEmail, &c.DocumentProofURL, &c.Method, &c.Status,
		&c.VerificationNotes, &c.VerifiedAt, &c.VerifiedByUserID,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update claim status: %w", err)
	}

	// If approved, update company and tenant records
	if status == model.ClaimStatusApproved {
		_, _ = r.pool.Exec(ctx, `
			UPDATE company.companies SET is_claimed = true, claimed_by_tenant_id = $1 WHERE id = $2;
		`, c.TenantID, c.CompanyID)

		_, _ = r.pool.Exec(ctx, `
			UPDATE organizations SET type = 'CORPORATE', company_id = $1, is_verified = true WHERE id = $2;
		`, c.CompanyID, c.TenantID)
	}

	return &c, nil
}
