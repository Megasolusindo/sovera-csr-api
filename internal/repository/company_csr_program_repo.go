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

type CSRProgramWithCompany struct {
	ID            string   `json:"id"`
	CompanyID     string   `json:"company_id"`
	Name          string   `json:"name"`
	Description   *string  `json:"description,omitempty"`
	ProgramType   *string  `json:"program_type,omitempty"`
	StartDate     *string  `json:"start_date,omitempty"`
	EndDate       *string  `json:"end_date,omitempty"`
	Status        string   `json:"status"`
	BudgetAmount  *float64 `json:"budget_amount,omitempty"`
	PartnerNGO    *string  `json:"partner_ngo,omitempty"`
	ImpactSummary *string  `json:"impact_summary,omitempty"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
	CompanyName   string   `json:"company_name"`
	CompanyTicker *string  `json:"company_ticker,omitempty"`
}

// ListAllPrograms returns a paginated list of all CSR programs across companies.
func (r *CompanyCSRProgramRepository) ListAllPrograms(ctx context.Context, limit, offset int, search, programType string) ([]CSRProgramWithCompany, int, error) {
	if r.pool == nil {
		return nil, 0, fmt.Errorf("database pool is nil")
	}

	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if search != "" {
		whereClause += fmt.Sprintf(" AND (p.name ILIKE $%d OR c.name ILIKE $%d OR p.partner_ngo ILIKE $%d OR p.program_type ILIKE $%d)", argIdx, argIdx, argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}

	if programType != "" {
		whereClause += fmt.Sprintf(" AND p.program_type ILIKE $%d", argIdx)
		args = append(args, "%"+programType+"%")
		argIdx++
	}

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM company_csr_programs p
		JOIN companies c ON c.id = p.company_id
		%s;
	`, whereClause)

	var total int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count company_csr_programs: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT 
			p.id::text, p.company_id::text, p.name, p.description, p.program_type,
			p.start_date::text, p.end_date::text, COALESCE(p.status, 'ACTIVE'), p.budget_amount::double precision, p.partner_ngo, p.impact_summary,
			p.created_at::text, p.updated_at::text,
			c.name as company_name, c.ticker as company_ticker
		FROM company_csr_programs p
		JOIN companies c ON c.id = p.company_id
		%s
		ORDER BY p.created_at DESC
		LIMIT $%d OFFSET $%d;
	`, whereClause, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query company_csr_programs: %w", err)
	}
	defer rows.Close()

	var programs []CSRProgramWithCompany
	for rows.Next() {
		var item CSRProgramWithCompany
		err := rows.Scan(
			&item.ID, &item.CompanyID, &item.Name, &item.Description, &item.ProgramType,
			&item.StartDate, &item.EndDate, &item.Status, &item.BudgetAmount, &item.PartnerNGO, &item.ImpactSummary,
			&item.CreatedAt, &item.UpdatedAt,
			&item.CompanyName, &item.CompanyTicker,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan CSRProgramWithCompany row: %w", err)
		}
		programs = append(programs, item)
	}

	return programs, total, nil
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
