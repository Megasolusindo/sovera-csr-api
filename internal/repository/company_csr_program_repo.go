package repository

import (
	"context"
	"fmt"
	"strconv"
	"strings"

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
	Visibility    string   `json:"visibility"`
}

// ListAllPrograms returns a paginated list of all CSR programs across companies.
// By default only public and curated programs are returned; pass visibilityFilter to narrow results.
// Pass "ALL" to bypass visibility filtering (requires superadmin at the route level).
// When authorizedOrgID is provided, private programs are restricted to the matching verified corporate organization.
func (r *CompanyCSRProgramRepository) ListAllPrograms(ctx context.Context, limit, offset int, search, programType, visibilityFilter, authorizedOrgID string) ([]CSRProgramWithCompany, int, error) {
	if r.pool == nil {
		return nil, 0, fmt.Errorf("database pool is nil")
	}

	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	// Visibility filter: default to public+curated for explore/catalog
	if visibilityFilter != "ALL" {
		if visibilityFilter == "" {
			visibilityFilter = "public,curated"
		}
		visList := strings.Split(visibilityFilter, ",")
		placeholders := make([]string, len(visList))
		for i, v := range visList {
			placeholders[i] = "$" + strconv.Itoa(argIdx+i)
			visList[i] = strings.TrimSpace(v)
		}
		whereClause += " AND p.visibility IN (" + strings.Join(placeholders, ",") + ")"
		for _, v := range visList {
			args = append(args, v)
		}
		argIdx += len(visList)
	}

	if authorizedOrgID != "" && strings.Contains(strings.ToLower(visibilityFilter), "private") {
		whereClause += ` AND (p.visibility <> 'private' OR EXISTS (
			SELECT 1
			FROM organizations o
			WHERE o.id::text = $` + strconv.Itoa(argIdx) + `
			  AND o.is_verified = TRUE
			  AND o.type = 'CORPORATE'
			  AND o.company_id::text = p.company_id::text
		))`
		args = append(args, authorizedOrgID)
		argIdx++
	}

	if search != "" {
		whereClause += " AND (p.name ILIKE $" + strconv.Itoa(argIdx) + " OR c.name ILIKE $" + strconv.Itoa(argIdx) + " OR p.partner_ngo ILIKE $" + strconv.Itoa(argIdx) + " OR p.program_type ILIKE $" + strconv.Itoa(argIdx) + ")"
		args = append(args, "%"+search+"%")
		argIdx++
	}

	if programType != "" {
		whereClause += " AND p.program_type ILIKE $" + strconv.Itoa(argIdx)
		args = append(args, "%"+programType+"%")
		argIdx++
	}

	countQuery := `
		SELECT COUNT(*)
		FROM company_enriched_programs p
		JOIN companies c ON c.id = p.company_id
		` + whereClause + `;`

	var total int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count company_enriched_programs: %w", err)
	}

	query := `
		SELECT 
			p.id::text, p.company_id::text, p.name, p.description, p.program_type,
			p.start_date::text, p.end_date::text, COALESCE(p.status, 'ACTIVE'), p.budget_amount::double precision, p.partner_ngo, p.impact_summary,
			p.created_at::text, p.updated_at::text, COALESCE(p.visibility, 'public'),
			c.name as company_name, c.ticker as company_ticker
		FROM company_enriched_programs p
		JOIN companies c ON c.id = p.company_id
		` + whereClause + `
		ORDER BY p.created_at DESC
		LIMIT $` + strconv.Itoa(argIdx) + ` OFFSET $` + strconv.Itoa(argIdx+1) + `;`

	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query company_enriched_programs: %w", err)
	}
	defer rows.Close()

	var programs []CSRProgramWithCompany
	for rows.Next() {
		var item CSRProgramWithCompany
		err := rows.Scan(
			&item.ID, &item.CompanyID, &item.Name, &item.Description, &item.ProgramType,
			&item.StartDate, &item.EndDate, &item.Status, &item.BudgetAmount, &item.PartnerNGO, &item.ImpactSummary,
			&item.CreatedAt, &item.UpdatedAt, &item.Visibility,
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
			created_at, updated_at, COALESCE(visibility, 'public')
		FROM company_enriched_programs
		WHERE company_id::text = $1
		ORDER BY created_at DESC;
	`

	rows, err := r.pool.Query(ctx, query, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to query company_enriched_programs: %w", err)
	}
	defer rows.Close()

	var programs []model.CompanyCSRProgram
	for rows.Next() {
		var p model.CompanyCSRProgram
		err := rows.Scan(
			&p.ID, &p.CompanyID, &p.Name, &p.Description, &p.ProgramType,
			&p.StartDate, &p.EndDate, &p.Status, &p.BudgetAmount, &p.PartnerNGO, &p.ImpactSummary,
			&p.CreatedAt, &p.UpdatedAt, &p.Visibility,
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
			created_at, updated_at, COALESCE(visibility, 'public')
		FROM company_enriched_programs
		WHERE id::text = $1
		LIMIT 1;
	`

	var p model.CompanyCSRProgram
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.CompanyID, &p.Name, &p.Description, &p.ProgramType,
		&p.StartDate, &p.EndDate, &p.Status, &p.BudgetAmount, &p.PartnerNGO, &p.ImpactSummary,
		&p.CreatedAt, &p.UpdatedAt, &p.Visibility,
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

// CanManageCSRProgram reports whether the verified corporate organization linked to orgID owns the program.
func (r *CompanyCSRProgramRepository) CanManageCSRProgram(ctx context.Context, id, orgID string) (bool, error) {
	if r.pool == nil {
		return false, fmt.Errorf("database pool is nil")
	}

	var canManage bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM company_enriched_programs p
			JOIN organizations o ON o.company_id = p.company_id
			WHERE p.id::text = $1
			  AND o.id::text = $2
			  AND o.is_verified = TRUE
			  AND o.type = 'CORPORATE'
		)
	`, id, orgID).Scan(&canManage)
	if err != nil {
		return false, fmt.Errorf("failed to check program ownership: %w", err)
	}

	return canManage, nil
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

	// Normalize visibility to a valid default
	visibility := strings.TrimSpace(strings.ToLower(p.Visibility))
	if visibility == "" {
		visibility = "public"
	}

	query := `
		INSERT INTO company_enriched_programs (
			company_id, name, description, program_type,
			start_date, end_date, status, budget_amount, partner_ngo, impact_summary, visibility,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, COALESCE(NULLIF($7, ''), 'ACTIVE'), $8, $9, $10, $11, NOW(), NOW()
		)
		RETURNING 
			id::text, company_id::text, name, description, program_type,
			start_date, end_date, status, budget_amount, partner_ngo, impact_summary,
			created_at, updated_at, COALESCE(visibility, 'public');
	`

	var created model.CompanyCSRProgram
	err = tx.QueryRow(ctx, query,
		p.CompanyID, p.Name, p.Description, p.ProgramType,
		p.StartDate, p.EndDate, p.Status, p.BudgetAmount, p.PartnerNGO, p.ImpactSummary, visibility,
	).Scan(
		&created.ID, &created.CompanyID, &created.Name, &created.Description, &created.ProgramType,
		&created.StartDate, &created.EndDate, &created.Status, &created.BudgetAmount, &created.PartnerNGO, &created.ImpactSummary,
		&created.CreatedAt, &created.UpdatedAt, &created.Visibility,
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

	query := `DELETE FROM company_enriched_programs WHERE id::text = $1;`
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

// UpdateVisibility sets the visibility level for a CSR program.
// Allowed values: public, curated, private.
func (r *CompanyCSRProgramRepository) UpdateVisibility(ctx context.Context, id, visibility string) (*model.CompanyCSRProgram, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	visibility = strings.TrimSpace(strings.ToLower(visibility))
	if visibility != "public" && visibility != "curated" && visibility != "private" {
		return nil, fmt.Errorf("invalid visibility value: %s (allowed: public, curated, private)", visibility)
	}

	query := `
		UPDATE company_enriched_programs
		SET visibility = $2, updated_at = NOW()
		WHERE id::text = $1
		RETURNING
			id::text, company_id::text, name, description, program_type,
			start_date, end_date, status, budget_amount, partner_ngo, impact_summary,
			created_at, updated_at, COALESCE(visibility, 'public');
	`

	var p model.CompanyCSRProgram
	err := r.pool.QueryRow(ctx, query, id, visibility).Scan(
		&p.ID, &p.CompanyID, &p.Name, &p.Description, &p.ProgramType,
		&p.StartDate, &p.EndDate, &p.Status, &p.BudgetAmount, &p.PartnerNGO, &p.ImpactSummary,
		&p.CreatedAt, &p.UpdatedAt, &p.Visibility,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update program visibility: %w", err)
	}

	focuses, err := r.getProgramFocuses(ctx, p.ID)
	if err == nil {
		p.Focuses = focuses
	}

	return &p, nil
}
