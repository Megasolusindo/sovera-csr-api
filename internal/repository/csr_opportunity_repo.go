package repository

import (
	"context"
	"fmt"

	"sovera-core-api/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CSROpportunityRepository struct {
	pool *pgxpool.Pool
}

func NewCSROpportunityRepository(pool *pgxpool.Pool) *CSROpportunityRepository {
	return &CSROpportunityRepository{pool: pool}
}

func (r *CSROpportunityRepository) Create(ctx context.Context, opp model.CSROpportunity) (*model.CSROpportunity, error) {
	query := `
		INSERT INTO csr_opportunities (company_id, tenant_id, title, description, category, target_location, budget_amount, open_until, status, created_by_user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, company_id, tenant_id, title, description, category, target_location, budget_amount, open_until, status, created_by_user_id, created_at, updated_at;
	`
	var created model.CSROpportunity
	err := r.pool.QueryRow(ctx, query,
		opp.CompanyID, opp.TenantID, opp.Title, opp.Description, opp.Category,
		opp.TargetLocation, opp.BudgetAmount, opp.OpenUntil, opp.Status, opp.CreatedByUserID,
	).Scan(
		&created.ID, &created.CompanyID, &created.TenantID, &created.Title, &created.Description,
		&created.Category, &created.TargetLocation, &created.BudgetAmount, &created.OpenUntil,
		&created.Status, &created.CreatedByUserID, &created.CreatedAt, &created.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create opportunity: %w", err)
	}
	return &created, nil
}

func (r *CSROpportunityRepository) ListPublic(ctx context.Context, page, pageSize int, category, search string) ([]model.CSROpportunity, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	baseWhere := "WHERE status = 'OPEN'"
	args := []interface{}{}
	argIdx := 1

	if category != "" {
		baseWhere += fmt.Sprintf(" AND category = $%d", argIdx)
		args = append(args, category)
		argIdx++
	}

	if search != "" {
		baseWhere += fmt.Sprintf(" AND (title ILIKE $%d OR COALESCE(description, '') ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM csr_opportunities %s", baseWhere)
	var total int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count opportunities: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, company_id, tenant_id, title, COALESCE(description, ''), category, COALESCE(target_location, ''), budget_amount, open_until, status, created_by_user_id, created_at, updated_at
		FROM csr_opportunities
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d;
	`, baseWhere, argIdx, argIdx+1)

	args = append(args, pageSize, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list opportunities: %w", err)
	}
	defer rows.Close()

	items := []model.CSROpportunity{}
	for rows.Next() {
		var item model.CSROpportunity
		if err := rows.Scan(
			&item.ID, &item.CompanyID, &item.TenantID, &item.Title, &item.Description,
			&item.Category, &item.TargetLocation, &item.BudgetAmount, &item.OpenUntil,
			&item.Status, &item.CreatedByUserID, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan opportunity: %w", err)
		}
		items = append(items, item)
	}

	return items, total, nil
}
