package repository

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"sovera-core-api/internal/model"
	"sovera-core-api/internal/pkg/phoneverifier"
	"sovera-core-api/internal/pkg/urlverifier"
)

type CompanyRepository struct {
	pool *pgxpool.Pool
}

func NewCompanyRepository(pool *pgxpool.Pool) *CompanyRepository {
	return &CompanyRepository{pool: pool}
}

type CompanyStats struct {
	TotalCount    int `json:"total_count"`
	VerifiedCount int `json:"verified_count"`
	TbkBumnCount  int `json:"tbk_bumn_count"`
}

func (r *CompanyRepository) GetCompanyStats(ctx context.Context) (*CompanyStats, error) {
	if r.pool == nil {
		return &CompanyStats{TotalCount: 0, VerifiedCount: 0, TbkBumnCount: 0}, nil
	}
	var stats CompanyStats
	err := r.pool.QueryRow(ctx, `
		SELECT 
			COUNT(*),
			COUNT(*) FILTER (WHERE website IS NOT NULL AND website <> ''),
			COUNT(*) FILTER (WHERE is_public = true OR company_type IN ('BUMN', 'SWASTA_TBK'))
		FROM company.companies;
	`).Scan(&stats.TotalCount, &stats.VerifiedCount, &stats.TbkBumnCount)
	if err != nil {
		return &CompanyStats{TotalCount: 0, VerifiedCount: 0, TbkBumnCount: 0}, nil
	}
	return &stats, nil
}

// ListCompanies retrieves a paginated list of companies with target and signal counts.
func (r *CompanyRepository) ListCompanies(ctx context.Context, limit, offset int, search, sector, verificationStatus, priorityTier, companyType string) ([]model.CompanyDetail, int, error) {
	if r.pool == nil {
		return nil, 0, nil
	}

	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if search != "" {
		whereClause += fmt.Sprintf(" AND (c.name ILIKE $%d OR c.legal_name ILIKE $%d OR c.slug ILIKE $%d OR c.ticker ILIKE $%d OR c.id::text ILIKE $%d OR array_to_string(c.alias_keywords, ' ') ILIKE $%d)", argIdx, argIdx, argIdx, argIdx, argIdx, argIdx)
		args = append(args, "%"+strings.TrimSpace(search)+"%")
		argIdx++
	}

	if sector != "" && sector != "ALL" {
		whereClause += fmt.Sprintf(" AND c.industry_sector ILIKE $%d", argIdx)
		args = append(args, "%"+sector+"%")
		argIdx++
	}

	if companyType != "" && companyType != "ALL" {
		whereClause += fmt.Sprintf(" AND UPPER(c.company_type) = UPPER($%d)", argIdx)
		args = append(args, companyType)
		argIdx++
	}

	if priorityTier != "" && priorityTier != "ALL" {
		whereClause += fmt.Sprintf(" AND c.priority_tier = $%d", argIdx)
		args = append(args, priorityTier)
		argIdx++
	}

	if verificationStatus == "VERIFIED" {
		whereClause += " AND (c.website IS NOT NULL AND c.website <> '')"
	} else if verificationStatus == "PENDING" {
		whereClause += " AND (c.website IS NULL OR c.website = '')"
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM company.companies c %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count companies: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT 
			c.id::text, c.name, c.legal_name, c.slug, c.industry_id, c.industry_sector,
			c.company_type, c.website, c.linkedin_url, c.headquarters,
			c.employee_range, c.revenue_range, c.is_public, c.ticker,
			c.parent_company_id::text, COALESCE(c.priority_tier, 'TIER_3'), COALESCE(c.csr_category, 'POTENSIAL'), COALESCE(c.partner_ngo, (SELECT p.csr_department_name FROM company_csr_profiles p WHERE p.company_id = c.id LIMIT 1)), COALESCE(c.alias_keywords, '{}'), c.created_at, c.updated_at,
			(SELECT COUNT(*) FROM crawling_targets t WHERE t.company_id = c.id) AS target_count,
			(SELECT COUNT(*) FROM intelligence.company_signals s WHERE s.company_id = c.id OR s.company_name ILIKE c.name) AS signal_count,
			COALESCE((SELECT SUM(s.estimated_budget_signal) FROM intelligence.company_signals s WHERE s.company_id = c.id OR s.company_name ILIKE c.name), 0) AS total_budget
		FROM company.companies c
		%s
		ORDER BY signal_count DESC, c.priority_tier ASC, c.name ASC
		LIMIT $%d OFFSET $%d;
	`, whereClause, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query companies: %w", err)
	}
	defer rows.Close()

	result := []model.CompanyDetail{}
	for rows.Next() {
		var cd model.CompanyDetail
		err := rows.Scan(
			&cd.ID, &cd.Name, &cd.LegalName, &cd.Slug, &cd.IndustryID, &cd.IndustrySector,
			&cd.CompanyType, &cd.Website, &cd.LinkedinURL, &cd.Headquarters,
			&cd.EmployeeRange, &cd.RevenueRange, &cd.IsPublic, &cd.Ticker,
			&cd.ParentCompanyID, &cd.PriorityTier, &cd.CSRCategory, &cd.PartnerNGO, &cd.AliasKeywords, &cd.CreatedAt, &cd.UpdatedAt,
			&cd.TargetCount, &cd.SignalCount, &cd.TotalBudget,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan company row: %w", err)
		}
		result = append(result, cd)
	}

	return result, total, nil
}

// GetCompanyByID retrieves a single company detail by ID or Slug.
func (r *CompanyRepository) GetCompanyByID(ctx context.Context, idOrSlug string) (*model.CompanyDetail, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	query := `
		SELECT 
			c.id::text, c.name, c.legal_name, c.slug, c.industry_id, c.industry_sector,
			c.company_type, c.website, c.linkedin_url, c.headquarters,
			c.employee_range, c.revenue_range, c.is_public, c.ticker,
			c.parent_company_id::text, COALESCE(c.priority_tier, 'TIER_3'), COALESCE(c.csr_category, 'POTENSIAL'), COALESCE(c.partner_ngo, (SELECT p.csr_department_name FROM company_csr_profiles p WHERE p.company_id = c.id LIMIT 1)), COALESCE(c.alias_keywords, '{}'), c.created_at, c.updated_at,
			(SELECT COUNT(*) FROM crawling_targets t WHERE t.company_id = c.id) AS target_count,
			(SELECT COUNT(*) FROM intelligence.company_signals s WHERE s.company_id = c.id OR s.company_name ILIKE c.name) AS signal_count,
			COALESCE((SELECT SUM(s.estimated_budget_signal) FROM intelligence.company_signals s WHERE s.company_id = c.id OR s.company_name ILIKE c.name), 0) AS total_budget
		FROM company.companies c
		WHERE c.id::text = $1 OR c.slug = $1
		LIMIT 1;
	`

	var cd model.CompanyDetail
	err := r.pool.QueryRow(ctx, query, idOrSlug).Scan(
		&cd.ID, &cd.Name, &cd.LegalName, &cd.Slug, &cd.IndustryID, &cd.IndustrySector,
		&cd.CompanyType, &cd.Website, &cd.LinkedinURL, &cd.Headquarters,
		&cd.EmployeeRange, &cd.RevenueRange, &cd.IsPublic, &cd.Ticker,
		&cd.ParentCompanyID, &cd.PriorityTier, &cd.CSRCategory, &cd.PartnerNGO, &cd.AliasKeywords, &cd.CreatedAt, &cd.UpdatedAt,
		&cd.TargetCount, &cd.SignalCount, &cd.TotalBudget,
	)
	if err != nil {
		return nil, fmt.Errorf("company not found: %w", err)
	}

	return &cd, nil
}

// FindBySlugOrAlias looks up a company by exact slug, name match, or matching alias keywords.
func (r *CompanyRepository) FindBySlugOrAlias(ctx context.Context, slug, rawName string) (*model.Company, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	query := `
		SELECT id::text, name, legal_name, slug, industry_id, industry_sector, company_type,
		       website, linkedin_url, headquarters, employee_range, revenue_range,
		       is_public, ticker, parent_company_id::text, COALESCE(alias_keywords, '{}'), created_at, updated_at
		FROM company.companies
		WHERE slug = $1 OR name ILIKE $2 OR $2 ILIKE ANY(alias_keywords)
		LIMIT 1;
	`

	var c model.Company
	err := r.pool.QueryRow(ctx, query, slug, "%"+rawName+"%").Scan(
		&c.ID, &c.Name, &c.LegalName, &c.Slug, &c.IndustryID, &c.IndustrySector, &c.CompanyType,
		&c.Website, &c.LinkedinURL, &c.Headquarters, &c.EmployeeRange, &c.RevenueRange,
		&c.IsPublic, &c.Ticker, &c.ParentCompanyID, &c.AliasKeywords, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

// CreateCompany provisions a new master company record.
func (r *CompanyRepository) CreateCompany(ctx context.Context, c model.Company) (*model.Company, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	upperName := strings.TrimSpace(strings.ToUpper(c.Name))
	if strings.HasPrefix(upperName, "CV ") || strings.HasPrefix(upperName, "CV.") {
		return nil, fmt.Errorf("CV entities are excluded from corporate directory")
	}

	slug := c.Slug
	if slug == "" {
		slug = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(c.Name), " ", "-"))
	}

	legalName := c.Name
	if c.LegalName != nil && *c.LegalName != "" {
		legalName = *c.LegalName
	}
	companyType := c.CompanyType
	if companyType == "" {
		companyType = "SWASTA"
	}
	priorityTier := c.PriorityTier
	if priorityTier == "" {
		priorityTier = "TIER_1"
	}
	isPublic := c.IsPublic || companyType == "SWASTA_TBK" || (c.Ticker != nil && *c.Ticker != "")

	if c.Website != nil && *c.Website != "" {
		ok, normalized, err := urlverifier.DefaultVerifier.VerifyWebsite(ctx, *c.Website)
		if !ok {
			log.Printf("[URLVerifier] Rejected invalid/unreachable website '%s' for company '%s': %v", *c.Website, c.Name, err)
			c.Website = nil
		} else {
			c.Website = &normalized
		}
	}

	if c.Phone != nil && *c.Phone != "" {
		ok, normalized, err := phoneverifier.DefaultVerifier.Verify(*c.Phone)
		if !ok {
			log.Printf("[PhoneVerifier] Rejected invalid company phone number '%s' for '%s': %v", *c.Phone, c.Name, err)
			c.Phone = nil
		} else {
			c.Phone = &normalized
		}
	}

	query := `
		INSERT INTO company.companies (
			name, legal_name, slug, industry_sector, company_type, website, ticker, is_public, priority_tier, alias_keywords, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
		ON CONFLICT (slug) DO UPDATE SET 
			name = EXCLUDED.name,
			website = COALESCE(NULLIF(EXCLUDED.website, ''), company.companies.website),
			ticker = COALESCE(NULLIF(EXCLUDED.ticker, ''), company.companies.ticker),
			priority_tier = EXCLUDED.priority_tier,
			updated_at = NOW()
		RETURNING id::text, created_at, updated_at;
	`

	err := r.pool.QueryRow(
		ctx, query,
		c.Name, legalName, slug, c.IndustrySector, companyType, c.Website, c.Ticker, isPublic, priorityTier, c.AliasKeywords,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create company record: %w", err)
	}

	return &c, nil
}

// GetCompaniesMissingWebsite fetches companies where website is NULL or empty up to limit.
func (r *CompanyRepository) GetCompaniesMissingWebsite(ctx context.Context, limit int) ([]model.Company, error) {
	if r.pool == nil {
		return nil, nil
	}

	query := `
		SELECT id::text, name, legal_name, slug, industry_sector, company_type, is_public, ticker, COALESCE(alias_keywords, '{}')
		FROM company.companies
		WHERE website IS NULL OR website = ''
		ORDER BY is_public DESC, name ASC
		LIMIT $1;
	`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query missing websites: %w", err)
	}
	defer rows.Close()

	var list []model.Company
	for rows.Next() {
		var c model.Company
		if err := rows.Scan(&c.ID, &c.Name, &c.LegalName, &c.Slug, &c.IndustrySector, &c.CompanyType, &c.IsPublic, &c.Ticker, &c.AliasKeywords); err == nil {
			list = append(list, c)
		}
	}
	return list, nil
}

// UpdateCompanyWebsite updates the verified website URL and website_source in company_csr_profiles after checking reachability.
func (r *CompanyRepository) UpdateCompanyWebsite(ctx context.Context, companyID string, website string) error {
	if r.pool == nil {
		return nil
	}

	website = strings.TrimSpace(website)
	if website == "" {
		return fmt.Errorf("empty website URL")
	}

	// Verify reachability of the website URL before persisting into corporate database
	ok, normalized, err := urlverifier.DefaultVerifier.VerifyWebsite(ctx, website)
	if !ok {
		log.Printf("[URLVerifier] Rejected website '%s' for company ID %s: %v", website, companyID, err)
		return fmt.Errorf("website URL '%s' is invalid or unreachable: %w", website, err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `UPDATE company.companies SET website = $1, updated_at = NOW() WHERE id = $2::uuid`, normalized, companyID)
	if err != nil {
		return fmt.Errorf("failed to update company website: %w", err)
	}

	csrSource := fmt.Sprintf("%s/csr", normalized)
	_, _ = tx.Exec(ctx, `
		INSERT INTO company_csr_profiles (company_id, has_csr, website_source, updated_at)
		VALUES ($1::uuid, true, $2, NOW())
		ON CONFLICT (company_id) DO UPDATE SET website_source = EXCLUDED.website_source, updated_at = NOW();
	`, companyID, csrSource)

	return tx.Commit(ctx)
}



