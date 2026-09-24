package repository

import (
	"context"
	"fmt"
	"log"
	"strconv"
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

func (r *CompanyRepository) GetDBPool() *pgxpool.Pool {
	return r.pool
}


type CompanyStats struct {
	TotalCount    int `json:"total_count"`
	VerifiedCount int `json:"verified_count"`
	TbkBumnCount  int `json:"tbk_bumn_count"`
}

type CompanyLinkedInStats struct {
	TotalCompanies  int `json:"total_companies"`
	HasLinkedIn     int `json:"has_linkedin"`
	ValidCount      int `json:"valid_count"`
	InvalidCount    int `json:"invalid_count"`
	UnverifiedCount int `json:"unverified_count"`
}

type CompanyInstagramStats struct {
	TotalCompanies  int `json:"total_companies"`
	HasInstagram    int `json:"has_instagram"`
	ValidCount      int `json:"valid_count"`
	InvalidCount    int `json:"invalid_count"`
	UnverifiedCount int `json:"unverified_count"`
}

type CompanyFacebookStats struct {
	TotalCompanies  int `json:"total_companies"`
	HasFacebook     int `json:"has_facebook"`
	ValidCount      int `json:"valid_count"`
	InvalidCount    int `json:"invalid_count"`
	UnverifiedCount int `json:"unverified_count"`
}

type CompanyYoutubeStats struct {
	TotalCompanies  int `json:"total_companies"`
	HasYoutube      int `json:"has_youtube"`
	ValidCount      int `json:"valid_count"`
	InvalidCount    int `json:"invalid_count"`
	UnverifiedCount int `json:"unverified_count"`
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

func (r *CompanyRepository) GetLinkedInValidationStats(ctx context.Context) (*CompanyLinkedInStats, error) {
	if r.pool == nil {
		return &CompanyLinkedInStats{}, nil
	}
	var stats CompanyLinkedInStats
	err := r.pool.QueryRow(ctx, `
		SELECT 
			COUNT(*),
			COUNT(*) FILTER (WHERE linkedin_url IS NOT NULL AND linkedin_url <> ''),
			COUNT(*) FILTER (WHERE linkedin_status = 'VALID'),
			COUNT(*) FILTER (WHERE linkedin_status = 'INVALID'),
			COUNT(*) FILTER (WHERE linkedin_url IS NOT NULL AND linkedin_url <> '' AND (linkedin_status IS NULL OR linkedin_status = 'UNVERIFIED'))
		FROM companies;
	`).Scan(&stats.TotalCompanies, &stats.HasLinkedIn, &stats.ValidCount, &stats.InvalidCount, &stats.UnverifiedCount)
	if err != nil {
		return &CompanyLinkedInStats{}, err
	}
	return &stats, nil
}

func (r *CompanyRepository) UpdateCompanyLinkedInStatus(ctx context.Context, id string, status string, lastError string) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE companies
		SET linkedin_status = $1,
			linkedin_verified_at = NOW(),
			linkedin_last_error = $2,
			updated_at = NOW()
		WHERE id::text = $3
	`, status, lastError, id)
	return err
}

func (r *CompanyRepository) GetInstagramValidationStats(ctx context.Context) (*CompanyInstagramStats, error) {
	if r.pool == nil {
		return &CompanyInstagramStats{}, nil
	}
	var stats CompanyInstagramStats
	err := r.pool.QueryRow(ctx, `
		SELECT 
			COUNT(*),
			COUNT(*) FILTER (WHERE instagram_url IS NOT NULL AND instagram_url <> ''),
			COUNT(*) FILTER (WHERE instagram_status = 'VALID'),
			COUNT(*) FILTER (WHERE instagram_status = 'INVALID'),
			COUNT(*) FILTER (WHERE instagram_url IS NOT NULL AND instagram_url <> '' AND (instagram_status IS NULL OR instagram_status = 'UNVERIFIED'))
		FROM companies;
	`).Scan(&stats.TotalCompanies, &stats.HasInstagram, &stats.ValidCount, &stats.InvalidCount, &stats.UnverifiedCount)
	if err != nil {
		return &CompanyInstagramStats{}, err
	}
	return &stats, nil
}

func (r *CompanyRepository) UpdateCompanyInstagramStatus(ctx context.Context, id string, status string, lastError string) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE company.companies
		SET instagram_status = $1,
			instagram_verified_at = NOW(),
			instagram_last_error = $2,
			updated_at = NOW()
		WHERE id::text = $3
	`, status, lastError, id)
	return err
}

func (r *CompanyRepository) GetFacebookValidationStats(ctx context.Context) (*CompanyFacebookStats, error) {
	if r.pool == nil {
		return &CompanyFacebookStats{}, nil
	}
	var stats CompanyFacebookStats
	err := r.pool.QueryRow(ctx, `
		SELECT 
			COUNT(*),
			COUNT(*) FILTER (WHERE facebook_url IS NOT NULL AND facebook_url <> ''),
			COUNT(*) FILTER (WHERE facebook_status = 'VALID'),
			COUNT(*) FILTER (WHERE facebook_status = 'INVALID'),
			COUNT(*) FILTER (WHERE facebook_url IS NOT NULL AND facebook_url <> '' AND (facebook_status IS NULL OR facebook_status = 'UNVERIFIED'))
		FROM company.companies;
	`).Scan(&stats.TotalCompanies, &stats.HasFacebook, &stats.ValidCount, &stats.InvalidCount, &stats.UnverifiedCount)
	if err != nil {
		return &CompanyFacebookStats{}, err
	}
	return &stats, nil
}

func (r *CompanyRepository) UpdateCompanyFacebookStatus(ctx context.Context, id string, status string, lastError string) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE company.companies
		SET facebook_status = $1,
			facebook_verified_at = NOW(),
			facebook_last_error = $2,
			updated_at = NOW()
		WHERE id::text = $3
	`, status, lastError, id)
	return err
}

func (r *CompanyRepository) GetYoutubeValidationStats(ctx context.Context) (*CompanyYoutubeStats, error) {
	if r.pool == nil {
		return &CompanyYoutubeStats{}, nil
	}
	var stats CompanyYoutubeStats
	err := r.pool.QueryRow(ctx, `
		SELECT 
			COUNT(*),
			COUNT(*) FILTER (WHERE youtube_url IS NOT NULL AND youtube_url <> ''),
			COUNT(*) FILTER (WHERE youtube_status = 'VALID'),
			COUNT(*) FILTER (WHERE youtube_status = 'INVALID'),
			COUNT(*) FILTER (WHERE youtube_url IS NOT NULL AND youtube_url <> '' AND (youtube_status IS NULL OR youtube_status = 'UNVERIFIED'))
		FROM company.companies;
	`).Scan(&stats.TotalCompanies, &stats.HasYoutube, &stats.ValidCount, &stats.InvalidCount, &stats.UnverifiedCount)
	if err != nil {
		return &CompanyYoutubeStats{}, err
	}
	return &stats, nil
}

func (r *CompanyRepository) UpdateCompanyYoutubeStatus(ctx context.Context, id string, status string, lastError string) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE company.companies
		SET youtube_status = $1,
			youtube_verified_at = NOW(),
			youtube_last_error = $2,
			updated_at = NOW()
		WHERE id::text = $3
	`, status, lastError, id)
	return err
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
		whereClause += " AND (c.name ILIKE $" + strconv.Itoa(argIdx) + " OR c.legal_name ILIKE $" + strconv.Itoa(argIdx) + " OR c.slug ILIKE $" + strconv.Itoa(argIdx) + " OR c.ticker ILIKE $" + strconv.Itoa(argIdx) + " OR c.id::text ILIKE $" + strconv.Itoa(argIdx) + " OR array_to_string(c.alias_keywords, ' ') ILIKE $" + strconv.Itoa(argIdx) + ")"
		args = append(args, "%"+strings.TrimSpace(search)+"%")
		argIdx++
	}

	if sector != "" && sector != "ALL" {
		whereClause += " AND c.industry_sector ILIKE $" + strconv.Itoa(argIdx)
		args = append(args, "%"+sector+"%")
		argIdx++
	}

	if companyType != "" && companyType != "ALL" {
		whereClause += " AND UPPER(c.company_type) = UPPER($" + strconv.Itoa(argIdx) + ")"
		args = append(args, companyType)
		argIdx++
	}

	if priorityTier != "" && priorityTier != "ALL" {
		whereClause += " AND c.priority_tier = $" + strconv.Itoa(argIdx)
		args = append(args, priorityTier)
		argIdx++
	}

	if verificationStatus == "VERIFIED" {
		whereClause += " AND (c.website IS NOT NULL AND c.website <> '')"
	} else if verificationStatus == "PENDING" {
		whereClause += " AND (c.website IS NULL OR c.website = '')"
	}

	countQuery := "SELECT COUNT(*) FROM company.companies c " + whereClause
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count companies: %w", err)
	}

	query := `
		SELECT 
			c.id::text, c.name, c.legal_name, c.slug, c.industry_id, c.industry_sector,
			c.company_type, c.website, c.linkedin_url, c.linkedin_status, c.instagram_url, c.instagram_status, c.facebook_url, c.facebook_status, c.youtube_url, c.youtube_status, c.headquarters,
			c.employee_range, c.revenue_range, c.is_public, c.ticker,
			c.parent_company_id::text, COALESCE(c.priority_tier, 'TIER_3'), COALESCE(c.csr_category, 'POTENSIAL'), COALESCE(c.partner_ngo, (SELECT p.csr_department_name FROM company_csr_profiles p WHERE p.company_id = c.id LIMIT 1)), COALESCE(c.alias_keywords, '{}'), c.created_at, c.updated_at,
			(SELECT COUNT(*) FROM crawling_targets t WHERE t.company_id = c.id) AS target_count,
			(SELECT COUNT(*) FROM intelligence.company_signals s WHERE s.company_id = c.id OR s.company_name ILIKE c.name) AS signal_count,
			COALESCE((SELECT SUM(s.estimated_budget_signal) FROM intelligence.company_signals s WHERE s.company_id = c.id OR s.company_name ILIKE c.name), 0) AS total_budget
		FROM company.companies c
		` + whereClause + `
		ORDER BY signal_count DESC, c.priority_tier ASC, c.name ASC
		LIMIT $` + strconv.Itoa(argIdx) + ` OFFSET $` + strconv.Itoa(argIdx+1) + `;`

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
			&cd.CompanyType, &cd.Website, &cd.LinkedinURL, &cd.LinkedinStatus, &cd.InstagramURL, &cd.InstagramStatus, &cd.FacebookURL, &cd.FacebookStatus, &cd.YoutubeURL, &cd.YoutubeStatus, &cd.Headquarters,
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
			c.company_type, c.website, c.linkedin_url, c.linkedin_status, c.instagram_url, c.instagram_status, c.facebook_url, c.facebook_status, c.youtube_url, c.youtube_status, c.headquarters,
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
		&cd.CompanyType, &cd.Website, &cd.LinkedinURL, &cd.LinkedinStatus, &cd.InstagramURL, &cd.InstagramStatus, &cd.FacebookURL, &cd.FacebookStatus, &cd.YoutubeURL, &cd.YoutubeStatus, &cd.Headquarters,
		&cd.EmployeeRange, &cd.RevenueRange, &cd.IsPublic, &cd.Ticker,
		&cd.ParentCompanyID, &cd.PriorityTier, &cd.CSRCategory, &cd.PartnerNGO, &cd.AliasKeywords, &cd.CreatedAt, &cd.UpdatedAt,
		&cd.TargetCount, &cd.SignalCount, &cd.TotalBudget,
	)
	if err != nil {
		return nil, fmt.Errorf("company not found: %w", err)
	}

	return &cd, nil
}

// FindBySlugOrAlias looks up a company by exact slug, normalized name match, or matching alias keywords.
func (r *CompanyRepository) FindBySlugOrAlias(ctx context.Context, slug, rawName string) (*model.Company, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	trimmedName := strings.TrimSpace(rawName)

	query := `
		SELECT id::text, name, legal_name, slug, industry_id, industry_sector, company_type,
		       website, linkedin_url, instagram_url, facebook_url, youtube_url, headquarters, employee_range, revenue_range,
		       is_public, ticker, parent_company_id::text, COALESCE(alias_keywords, '{}'), created_at, updated_at
		FROM company.companies
		WHERE slug = $1 OR LOWER(TRIM(name)) = LOWER(TRIM($2)) OR name ILIKE $3 OR $2 ILIKE ANY(alias_keywords)
		LIMIT 1;
	`

	var c model.Company
	err := r.pool.QueryRow(ctx, query, slug, trimmedName, "%"+trimmedName+"%").Scan(
		&c.ID, &c.Name, &c.LegalName, &c.Slug, &c.IndustryID, &c.IndustrySector, &c.CompanyType,
		&c.Website, &c.LinkedinURL, &c.InstagramURL, &c.FacebookURL, &c.YoutubeURL, &c.Headquarters, &c.EmployeeRange, &c.RevenueRange,
		&c.IsPublic, &c.Ticker, &c.ParentCompanyID, &c.AliasKeywords, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

// CreateCompany provisions a new master company record with strict name-level deduplication.
func (r *CompanyRepository) CreateCompany(ctx context.Context, c model.Company) (*model.Company, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	upperName := strings.TrimSpace(strings.ToUpper(c.Name))
	if strings.HasPrefix(upperName, "CV ") || strings.HasPrefix(upperName, "CV.") {
		return nil, fmt.Errorf("CV entities are excluded from corporate directory")
	}

	c.Name = strings.TrimRight(strings.TrimSpace(c.Name), ".")

	slug := c.Slug
	if slug == "" {
		slug = strings.ToLower(strings.ReplaceAll(c.Name, " ", "-"))
	}

	legalName := c.Name
	if c.LegalName != nil && *c.LegalName != "" {
		legalName = strings.TrimRight(strings.TrimSpace(*c.LegalName), ".")
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
			name, legal_name, slug, industry_sector, company_type, website, linkedin_url, headquarters, ticker, is_public, priority_tier, alias_keywords, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW(), NOW())
		ON CONFLICT (LOWER(TRIM(name))) DO UPDATE SET 
			website = COALESCE(NULLIF(EXCLUDED.website, ''), company.companies.website),
			linkedin_url = COALESCE(NULLIF(EXCLUDED.linkedin_url, ''), company.companies.linkedin_url),
			headquarters = COALESCE(NULLIF(EXCLUDED.headquarters, ''), company.companies.headquarters),
			ticker = COALESCE(NULLIF(EXCLUDED.ticker, ''), company.companies.ticker),
			priority_tier = EXCLUDED.priority_tier,
			updated_at = NOW()
		RETURNING id::text, created_at, updated_at;
	`

	err := r.pool.QueryRow(
		ctx, query,
		c.Name, legalName, slug, c.IndustrySector, companyType, c.Website, c.LinkedinURL, c.Headquarters, c.Ticker, isPublic, priorityTier, c.AliasKeywords,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create company record: %w", err)
	}

	return &c, nil
}

// UpdateCompany updates existing fields of a company entity.
func (r *CompanyRepository) UpdateCompany(ctx context.Context, id string, c model.Company) (*model.CompanyDetail, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	query := `
		UPDATE company.companies SET
			name = CASE 
				WHEN $1::text IS NOT NULL AND $1::text <> '' AND NOT EXISTS (
					SELECT 1 FROM company.companies other WHERE LOWER(TRIM(other.name)) = LOWER(TRIM($1::text)) AND other.id <> company.companies.id
				) THEN $1::text 
				ELSE name 
			END,
			legal_name = COALESCE($2, legal_name),
			industry_sector = COALESCE(NULLIF($3, ''), industry_sector),
			company_type = COALESCE(NULLIF($4, ''), company_type),
			website = COALESCE($5, website),
			linkedin_url = COALESCE($6, linkedin_url),
			linkedin_status = COALESCE(NULLIF($7, ''), linkedin_status),
			instagram_url = COALESCE($8, instagram_url),
			instagram_status = COALESCE(NULLIF($9, ''), instagram_status),
			facebook_url = COALESCE($10, facebook_url),
			facebook_status = COALESCE(NULLIF($11, ''), facebook_status),
			youtube_url = COALESCE($12, youtube_url),
			youtube_status = COALESCE(NULLIF($13, ''), youtube_status),
			headquarters = COALESCE($14, headquarters),
			phone = COALESCE($15, phone),
			ticker = COALESCE($16, ticker),
			priority_tier = COALESCE(NULLIF($17, ''), priority_tier),
			updated_at = NOW()
		WHERE id::text = $18 OR slug = $18
		RETURNING id::text;
	`

	var updatedID string
	err := r.pool.QueryRow(
		ctx, query,
		c.Name, c.LegalName, c.IndustrySector, c.CompanyType,
		c.Website, c.LinkedinURL, c.LinkedinStatus,
		c.InstagramURL, c.InstagramStatus,
		c.FacebookURL, c.FacebookStatus,
		c.YoutubeURL, c.YoutubeStatus,
		c.Headquarters, c.Phone, c.Ticker, c.PriorityTier, id,
	).Scan(&updatedID)

	if err != nil {
		return nil, fmt.Errorf("failed to update company record: %w", err)
	}

	return r.GetCompanyByID(ctx, updatedID)
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



