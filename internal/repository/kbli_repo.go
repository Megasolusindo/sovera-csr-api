package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"sovera-core-api/internal/model"
)

type KBLIRepository struct {
	pool *pgxpool.Pool
}

func NewKBLIRepository(pool *pgxpool.Pool) *KBLIRepository {
	return &KBLIRepository{pool: pool}
}

// GetAll retrieves all KBLI reference codes filtered by optional category code or relevance tier.
func (r *KBLIRepository) GetAll(ctx context.Context, categoryCode, minRelevance string) ([]model.KBLIReference, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if categoryCode != "" {
		whereClause += fmt.Sprintf(" AND k.category_code = $%d", argIdx)
		args = append(args, categoryCode)
		argIdx++
	}

	if minRelevance != "" {
		whereClause += fmt.Sprintf(" AND k.csr_relevance_default = $%d", argIdx)
		args = append(args, minRelevance)
		argIdx++
	}

	query := fmt.Sprintf(`
		SELECT 
			k.code, k.title, k.category_code, k.category_title,
			k.csr_relevance_default, k.risk_level, k.description,
			k.created_at, k.updated_at,
			COUNT(c.id) AS company_count
		FROM public.kbli_reference k
		LEFT JOIN company.companies c ON c.kbli_code = k.code
		%s
		GROUP BY k.code, k.title, k.category_code, k.category_title, k.csr_relevance_default, k.risk_level, k.description, k.created_at, k.updated_at
		ORDER BY k.category_code ASC, k.code ASC;
	`, whereClause)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query KBLI references: %w", err)
	}
	defer rows.Close()

	var result []model.KBLIReference
	for rows.Next() {
		var item model.KBLIReference
		err := rows.Scan(
			&item.Code, &item.Title, &item.CategoryCode, &item.CategoryTitle,
			&item.CSRRelevanceDefault, &item.RiskLevel, &item.Description,
			&item.CreatedAt, &item.UpdatedAt,
			&item.CompanyCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan KBLI row: %w", err)
		}
		result = append(result, item)
	}

	return result, nil
}

// GetByCode retrieves a single KBLI code reference detail.
func (r *KBLIRepository) GetByCode(ctx context.Context, code string) (*model.KBLIReference, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	query := `
		SELECT 
			k.code, k.title, k.category_code, k.category_title,
			k.csr_relevance_default, k.risk_level, k.description,
			k.created_at, k.updated_at,
			COUNT(c.id) AS company_count
		FROM public.kbli_reference k
		LEFT JOIN company.companies c ON c.kbli_code = k.code
		WHERE k.code = $1
		GROUP BY k.code, k.title, k.category_code, k.category_title, k.csr_relevance_default, k.risk_level, k.description, k.created_at, k.updated_at
		LIMIT 1;
	`

	var item model.KBLIReference
	err := r.pool.QueryRow(ctx, query, code).Scan(
		&item.Code, &item.Title, &item.CategoryCode, &item.CategoryTitle,
		&item.CSRRelevanceDefault, &item.RiskLevel, &item.Description,
		&item.CreatedAt, &item.UpdatedAt,
		&item.CompanyCount,
	)
	if err != nil {
		return nil, fmt.Errorf("KBLI code '%s' not found: %w", code, err)
	}

	return &item, nil
}

// Search searches KBLI codes by keyword in title, description, or code.
func (r *KBLIRepository) Search(ctx context.Context, searchQuery string, limit int) ([]model.KBLIReference, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	if limit <= 0 {
		limit = 20
	}

	pattern := "%" + searchQuery + "%"
	query := `
		SELECT 
			k.code, k.title, k.category_code, k.category_title,
			k.csr_relevance_default, k.risk_level, k.description,
			k.created_at, k.updated_at,
			COUNT(c.id) AS company_count
		FROM public.kbli_reference k
		LEFT JOIN company.companies c ON c.kbli_code = k.code
		WHERE k.code ILIKE $1 OR k.title ILIKE $1 OR k.category_title ILIKE $1 OR k.description ILIKE $1
		GROUP BY k.code, k.title, k.category_code, k.category_title, k.csr_relevance_default, k.risk_level, k.description, k.created_at, k.updated_at
		ORDER BY k.code ASC
		LIMIT $2;
	`

	rows, err := r.pool.Query(ctx, query, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search KBLI references: %w", err)
	}
	defer rows.Close()

	var result []model.KBLIReference
	for rows.Next() {
		var item model.KBLIReference
		err := rows.Scan(
			&item.Code, &item.Title, &item.CategoryCode, &item.CategoryTitle,
			&item.CSRRelevanceDefault, &item.RiskLevel, &item.Description,
			&item.CreatedAt, &item.UpdatedAt,
			&item.CompanyCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan searched KBLI row: %w", err)
		}
		result = append(result, item)
	}

	return result, nil
}

// LinkCompanyKBLI links a company to a KBLI code and automatically updates the company's industry_sector to match the KBLI category.
func (r *KBLIRepository) LinkCompanyKBLI(ctx context.Context, companyID, kbliCode string) error {
	if r.pool == nil {
		return fmt.Errorf("database pool is nil")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var categoryTitle string
	err = tx.QueryRow(ctx, "SELECT category_title FROM public.kbli_reference WHERE code = $1", kbliCode).Scan(&categoryTitle)
	if err != nil {
		return fmt.Errorf("invalid KBLI code '%s': %w", kbliCode, err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE company.companies
		SET kbli_code = $1,
		    industry_sector = COALESCE(NULLIF(industry_sector, ''), $2),
		    updated_at = NOW()
		WHERE id = $3::uuid;
	`, kbliCode, categoryTitle, companyID)

	if err != nil {
		return fmt.Errorf("failed to link company KBLI code: %w", err)
	}

	return tx.Commit(ctx)
}
