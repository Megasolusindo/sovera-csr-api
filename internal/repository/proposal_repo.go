package repository

import (
	"context"
	"fmt"

	"sovera-core-api/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ProposalRepository struct {
	pool *pgxpool.Pool
}

func NewProposalRepository(pool *pgxpool.Pool) *ProposalRepository {
	return &ProposalRepository{pool: pool}
}

func (r *ProposalRepository) Create(ctx context.Context, p model.Proposal) (*model.Proposal, error) {
	query := `
		INSERT INTO proposals (opportunity_id, ngo_program_id, org_tenant_id, corp_tenant_id, company_id, title, summary, proposal_file_url, budget_requested, status, submitted_by_user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, opportunity_id, ngo_program_id, org_tenant_id, corp_tenant_id, company_id, title, summary, proposal_file_url, budget_requested, status, reviewer_notes, submitted_by_user_id, created_at, updated_at;
	`
	var created model.Proposal
	err := r.pool.QueryRow(ctx, query,
		p.OpportunityID, p.NGOProgramID, p.OrgTenantID, p.CorpTenantID, p.CompanyID,
		p.Title, p.Summary, p.ProposalFileURL, p.BudgetRequested, p.Status, p.SubmittedByUserID,
	).Scan(
		&created.ID, &created.OpportunityID, &created.NGOProgramID, &created.OrgTenantID,
		&created.CorpTenantID, &created.CompanyID, &created.Title, &created.Summary,
		&created.ProposalFileURL, &created.BudgetRequested, &created.Status,
		&created.ReviewerNotes, &created.SubmittedByUserID, &created.CreatedAt, &created.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to submit proposal: %w", err)
	}
	return &created, nil
}

func (r *ProposalRepository) ListByOrgTenant(ctx context.Context, orgTenantID string, page, pageSize int) ([]model.Proposal, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM proposals WHERE org_tenant_id = $1", orgTenantID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count NGO proposals: %w", err)
	}

	query := `
		SELECT id, opportunity_id, ngo_program_id, org_tenant_id, corp_tenant_id, company_id, title, COALESCE(summary, ''), COALESCE(proposal_file_url, ''), budget_requested, status, COALESCE(reviewer_notes, ''), submitted_by_user_id, created_at, updated_at
		FROM proposals
		WHERE org_tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3;
	`
	rows, err := r.pool.Query(ctx, query, orgTenantID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list proposals: %w", err)
	}
	defer rows.Close()

	items := []model.Proposal{}
	for rows.Next() {
		var item model.Proposal
		if err := rows.Scan(
			&item.ID, &item.OpportunityID, &item.NGOProgramID, &item.OrgTenantID,
			&item.CorpTenantID, &item.CompanyID, &item.Title, &item.Summary,
			&item.ProposalFileURL, &item.BudgetRequested, &item.Status,
			&item.ReviewerNotes, &item.SubmittedByUserID, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan proposal: %w", err)
		}
		items = append(items, item)
	}
	return items, total, nil
}

func (r *ProposalRepository) ListByCorpTenant(ctx context.Context, companyID string, page, pageSize int) ([]model.Proposal, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM proposals WHERE company_id = $1", companyID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count corporate proposals: %w", err)
	}

	query := `
		SELECT id, opportunity_id, ngo_program_id, org_tenant_id, corp_tenant_id, company_id, title, COALESCE(summary, ''), COALESCE(proposal_file_url, ''), budget_requested, status, COALESCE(reviewer_notes, ''), submitted_by_user_id, created_at, updated_at
		FROM proposals
		WHERE company_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3;
	`
	rows, err := r.pool.Query(ctx, query, companyID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list proposals for corporate: %w", err)
	}
	defer rows.Close()

	items := []model.Proposal{}
	for rows.Next() {
		var item model.Proposal
		if err := rows.Scan(
			&item.ID, &item.OpportunityID, &item.NGOProgramID, &item.OrgTenantID,
			&item.CorpTenantID, &item.CompanyID, &item.Title, &item.Summary,
			&item.ProposalFileURL, &item.BudgetRequested, &item.Status,
			&item.ReviewerNotes, &item.SubmittedByUserID, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan proposal: %w", err)
		}
		items = append(items, item)
	}
	return items, total, nil
}

func (r *ProposalRepository) UpdateStatus(ctx context.Context, id string, status model.ProposalStatus, notes string) (*model.Proposal, error) {
	query := `
		UPDATE proposals
		SET status = $2, reviewer_notes = COALESCE(NULLIF($3, ''), reviewer_notes), updated_at = NOW()
		WHERE id = $1
		RETURNING id, opportunity_id, ngo_program_id, org_tenant_id, corp_tenant_id, company_id, title, COALESCE(summary, ''), COALESCE(proposal_file_url, ''), budget_requested, status, COALESCE(reviewer_notes, ''), submitted_by_user_id, created_at, updated_at;
	`
	var item model.Proposal
	err := r.pool.QueryRow(ctx, query, id, status, notes).Scan(
		&item.ID, &item.OpportunityID, &item.NGOProgramID, &item.OrgTenantID,
		&item.CorpTenantID, &item.CompanyID, &item.Title, &item.Summary,
		&item.ProposalFileURL, &item.BudgetRequested, &item.Status,
		&item.ReviewerNotes, &item.SubmittedByUserID, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update proposal status: %w", err)
	}
	return &item, nil
}
