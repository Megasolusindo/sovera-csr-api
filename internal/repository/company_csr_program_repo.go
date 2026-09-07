package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"sovera-core-api/internal/model"
)

type CompanyCSRProgramRepository struct {
	pool *pgxpool.Pool
}

func NewCompanyCSRProgramRepository(pool *pgxpool.Pool) *CompanyCSRProgramRepository {
	return &CompanyCSRProgramRepository{pool: pool}
}

// ListByCompanyID retrieves all CSR programs for a given company ID, including linked focus areas.
func (r *CompanyCSRProgramRepository) ListByCompanyID(ctx context.Context, companyID string) ([]model.CompanyCSRProgram, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	query := `
		SELECT 
			id::text, company_id::text, name, description, program_type,
			start_date, end_date, status, budget_amount, partner_ngo, impact_summary,
			created_at, updated_at
		FROM company_csr_programs
		WHERE company_id::text = $1
		ORDER BY created_at DESC;
	`

	rows, err := r.pool.Query(ctx, query, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to query company_csr_programs: %w", err)
	}
	defer rows.Close()

	var programs []model.CompanyCSRProgram
	for rows.Next() {
		var p model.CompanyCSRProgram
		err := rows.Scan(
			&p.ID, &p.CompanyID, &p.Name, &p.Description, &p.ProgramType,
			&p.StartDate, &p.EndDate, &p.Status, &p.BudgetAmount, &p.PartnerNGO, &p.ImpactSummary,
			&p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan company_csr_program row: %w", err)
		}

		// Load linked focus areas
		focuses, err := r.getProgramFocuses(ctx, p.ID)
		if err == nil {
			p.Focuses = focuses
		}

		programs = append(programs, p)
	}

	return programs, nil
}

// GetByID retrieves a single CSR program by ID with linked focus areas.
func (r *CompanyCSRProgramRepository) GetByID(ctx context.Context, id string) (*model.CompanyCSRProgram, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	query := `
		SELECT 
			id::text, company_id::text, name, description, program_type,
			start_date, end_date, status, budget_amount, partner_ngo, impact_summary,
			created_at, updated_at
		FROM company_csr_programs
		WHERE id::text = $1
		LIMIT 1;
	`

	var p model.CompanyCSRProgram
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.CompanyID, &p.Name, &p.Description, &p.ProgramType,
		&p.StartDate, &p.EndDate, &p.Status, &p.BudgetAmount, &p.PartnerNGO, &p.ImpactSummary,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("company_csr_program not found: %w", err)
	}

	focuses, err := r.getProgramFocuses(ctx, p.ID)
	if err == nil {
		p.Focuses = focuses
	}

	return &p, nil
}

// Create inserts a new CSR program and associates its focus IDs.
func (r *CompanyCSRProgramRepository) Create(ctx context.Context, p *model.CompanyCSRProgram, focusIDs []string) (*model.CompanyCSRProgram, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO company_csr_programs (
			company_id, name, description, program_type,
			start_date, end_date, status, budget_amount, partner_ngo, impact_summary,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, COALESCE(NULLIF($7, ''), 'ACTIVE'), $8, $9, $10, NOW(), NOW()
		)
		RETURNING 
			id::text, company_id::text, name, description, program_type,
			start_date, end_date, status, budget_amount, partner_ngo, impact_summary,
			created_at, updated_at;
	`

	var created model.CompanyCSRProgram
	err = tx.QueryRow(ctx, query,
		p.CompanyID, p.Name, p.Description, p.ProgramType,
		p.StartDate, p.EndDate, p.Status, p.BudgetAmount, p.PartnerNGO, p.ImpactSummary,
	).Scan(
		&created.ID, &created.CompanyID, &created.Name, &created.Description, &created.ProgramType,
		&created.StartDate, &created.EndDate, &created.Status, &created.BudgetAmount, &created.PartnerNGO, &created.ImpactSummary,
		&created.CreatedAt, &created.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert company_csr_program: %w", err)
	}

	for _, fid := range focusIDs {
		_, err := tx.Exec(ctx, `INSERT INTO company_csr_program_focuses (program_id, focus_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;`, created.ID, fid)
		if err != nil {
			return nil, fmt.Errorf("failed to associate focus ID %s: %w", fid, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	focuses, _ := r.getProgramFocuses(ctx, created.ID)
	created.Focuses = focuses

	return &created, nil
}

// Delete removes a CSR program by ID.
func (r *CompanyCSRProgramRepository) Delete(ctx context.Context, id string) error {
	if r.pool == nil {
		return fmt.Errorf("database pool is nil")
	}

	query := `DELETE FROM company_csr_programs WHERE id::text = $1;`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete company_csr_program: %w", err)
	}

	return nil
}

// Helper to fetch linked CSRFocus objects for a program.
func (r *CompanyCSRProgramRepository) getProgramFocuses(ctx context.Context, programID string) ([]model.CSRFocus, error) {
	query := `
		SELECT f.id::text, f.code, f.name, f.category, f.description, f.created_at, f.updated_at
		FROM company_csr_program_focuses pf
		JOIN csr_focuses f ON f.id = pf.focus_id
		WHERE pf.program_id::text = $1
		ORDER BY f.name ASC;
	`

	rows, err := r.pool.Query(ctx, query, programID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var focuses []model.CSRFocus
	for rows.Next() {
		var f model.CSRFocus
		if err := rows.Scan(&f.ID, &f.Code, &f.Name, &f.Category, &f.Description, &f.CreatedAt, &f.UpdatedAt); err == nil {
			focuses = append(focuses, f)
		}
	}

	return focuses, nil
}
