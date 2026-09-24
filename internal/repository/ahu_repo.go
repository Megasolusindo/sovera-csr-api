package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"sovera-core-api/internal/model"
)

type AHURepository struct {
	pool *pgxpool.Pool
}

func NewAHURepository(pool *pgxpool.Pool) *AHURepository {
	return &AHURepository{pool: pool}
}

// GetByAHUNumber retrieves an AHU corporate registration by exact SK number.
func (r *AHURepository) GetByAHUNumber(ctx context.Context, ahuNumber string) (*model.AHURegistration, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	query := `
		SELECT 
			id::text, company_id::text, company_name, legal_name, ahu_number, legal_entity_type,
			deed_number, deed_date, notary_name, status, headquarters, capital_amount,
			verified_at, created_at, updated_at
		FROM public.ahu_registrations
		WHERE ahu_number = $1
		LIMIT 1;
	`

	var item model.AHURegistration
	err := r.pool.QueryRow(ctx, query, ahuNumber).Scan(
		&item.ID, &item.CompanyID, &item.CompanyName, &item.LegalName, &item.AHUNumber, &item.LegalEntityType,
		&item.DeedNumber, &item.DeedDate, &item.NotaryName, &item.Status, &item.Headquarters, &item.CapitalAmount,
		&item.VerifiedAt, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("AHU record with number '%s' not found: %w", ahuNumber, err)
	}

	return &item, nil
}

// SearchByName searches AHU corporate registry entries by company name or legal name.
func (r *AHURepository) SearchByName(ctx context.Context, name string, limit int) ([]model.AHURegistration, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	if limit <= 0 {
		limit = 20
	}

	pattern := "%" + strings.TrimSpace(name) + "%"
	query := `
		SELECT 
			id::text, company_id::text, company_name, legal_name, ahu_number, legal_entity_type,
			deed_number, deed_date, notary_name, status, headquarters, capital_amount,
			verified_at, created_at, updated_at
		FROM public.ahu_registrations
		WHERE company_name ILIKE $1 OR legal_name ILIKE $1 OR ahu_number ILIKE $1
		ORDER BY company_name ASC
		LIMIT $2;
	`

	rows, err := r.pool.Query(ctx, query, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search AHU registrations: %w", err)
	}
	defer rows.Close()

	var result []model.AHURegistration
	for rows.Next() {
		var item model.AHURegistration
		err := rows.Scan(
			&item.ID, &item.CompanyID, &item.CompanyName, &item.LegalName, &item.AHUNumber, &item.LegalEntityType,
			&item.DeedNumber, &item.DeedDate, &item.NotaryName, &item.Status, &item.Headquarters, &item.CapitalAmount,
			&item.VerifiedAt, &item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan AHU row: %w", err)
		}
		result = append(result, item)
	}

	return result, nil
}

// UpsertAHU inserts or updates an AHU corporate registration entry and updates company.companies if matched.
func (r *AHURepository) UpsertAHU(ctx context.Context, reg *model.AHURegistration) (*model.AHURegistration, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Attempt company_id resolution if missing
	if reg.CompanyID == nil || *reg.CompanyID == "" {
		var compID string
		err := tx.QueryRow(ctx, `
			SELECT id::text FROM company.companies
			WHERE LOWER(TRIM(name)) = LOWER(TRIM($1))
			   OR LOWER(TRIM(legal_name)) = LOWER(TRIM($2))
			   OR LOWER(TRIM(name)) ILIKE '%' || LOWER(TRIM($1)) || '%'
			LIMIT 1;
		`, reg.CompanyName, reg.LegalName).Scan(&compID)
		if err == nil && compID != "" {
			reg.CompanyID = &compID
		}
	}

	query := `
		INSERT INTO public.ahu_registrations (
			company_id, company_name, legal_name, ahu_number, legal_entity_type,
			deed_number, deed_date, notary_name, status, headquarters, capital_amount, verified_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW()
		)
		ON CONFLICT (ahu_number) DO UPDATE SET
			company_id = COALESCE(EXCLUDED.company_id, public.ahu_registrations.company_id),
			company_name = EXCLUDED.company_name,
			legal_name = EXCLUDED.legal_name,
			legal_entity_type = EXCLUDED.legal_entity_type,
			deed_number = COALESCE(EXCLUDED.deed_number, public.ahu_registrations.deed_number),
			deed_date = COALESCE(EXCLUDED.deed_date, public.ahu_registrations.deed_date),
			notary_name = COALESCE(EXCLUDED.notary_name, public.ahu_registrations.notary_name),
			status = EXCLUDED.status,
			headquarters = COALESCE(EXCLUDED.headquarters, public.ahu_registrations.headquarters),
			capital_amount = COALESCE(EXCLUDED.capital_amount, public.ahu_registrations.capital_amount),
			updated_at = NOW()
		RETURNING id::text, company_id::text, verified_at, created_at, updated_at;
	`

	var compIDNullable *string
	err = tx.QueryRow(ctx, query,
		reg.CompanyID, reg.CompanyName, reg.LegalName, reg.AHUNumber, reg.LegalEntityType,
		reg.DeedNumber, reg.DeedDate, reg.NotaryName, reg.Status, reg.Headquarters, reg.CapitalAmount,
	).Scan(&reg.ID, &compIDNullable, &reg.VerifiedAt, &reg.CreatedAt, &reg.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to upsert AHU registration: %w", err)
	}

	reg.CompanyID = compIDNullable

	// If linked to a company, update company.companies legal fields
	if reg.CompanyID != nil && *reg.CompanyID != "" {
		_, _ = tx.Exec(ctx, `
			UPDATE company.companies SET
				ahu_number = $1,
				legal_name = COALESCE(NULLIF(legal_name, ''), $2),
				legal_entity_type = COALESCE(NULLIF(legal_entity_type, ''), $3),
				headquarters = COALESCE(NULLIF(headquarters, ''), $4),
				updated_at = NOW()
			WHERE id = $5::uuid;
		`, reg.AHUNumber, reg.LegalName, reg.LegalEntityType, reg.Headquarters, *reg.CompanyID)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return reg, nil
}

// ResolveEntity performs entity resolution by matching an arbitrary corporate name against the official AHU registry and master companies.
func (r *AHURepository) ResolveEntity(ctx context.Context, rawName string) (*model.AHUEntityResolutionResult, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	cleanName := strings.TrimSpace(rawName)
	if cleanName == "" {
		return nil, fmt.Errorf("empty raw company name")
	}

	// 1. Try exact AHU registry match
	var res model.AHUEntityResolutionResult
	queryAHU := `
		SELECT company_name, legal_name, ahu_number, company_id::text
		FROM public.ahu_registrations
		WHERE LOWER(TRIM(company_name)) = LOWER(TRIM($1))
		   OR LOWER(TRIM(legal_name)) = LOWER(TRIM($1))
		LIMIT 1;
	`
	var compID *string
	err := r.pool.QueryRow(ctx, queryAHU, cleanName).Scan(&res.CanonicalName, &res.LegalName, &res.AHUNumber, &compID)
	if err == nil {
		res.CompanyID = compID
		res.MatchScore = 1.0
		res.IsMatched = true
		return &res, nil
	}

	// 2. Try fuzzy / ILIKE company master table match
	queryComp := `
		SELECT c.name, COALESCE(c.legal_name, c.name), COALESCE(c.ahu_number, ''), c.id::text
		FROM company.companies c
		WHERE LOWER(TRIM(c.name)) = LOWER(TRIM($1))
		   OR LOWER(TRIM(c.legal_name)) = LOWER(TRIM($1))
		   OR LOWER(TRIM(c.slug)) = LOWER(TRIM($1))
		LIMIT 1;
	`
	err = r.pool.QueryRow(ctx, queryComp, cleanName).Scan(&res.CanonicalName, &res.LegalName, &res.AHUNumber, &compID)
	if err == nil {
		res.CompanyID = compID
		res.MatchScore = 0.95
		res.IsMatched = true
		return &res, nil
	}

	// 3. Fallback: Normalize PT / TBK prefix/suffix
	normalized := cleanName
	normalized = strings.ReplaceAll(normalized, "PT.", "PT")
	normalized = strings.ReplaceAll(normalized, "Tbk.", "Tbk")
	normalized = strings.TrimSpace(normalized)

	queryFuzzy := `
		SELECT c.name, COALESCE(c.legal_name, c.name), COALESCE(c.ahu_number, ''), c.id::text
		FROM company.companies c
		WHERE c.name ILIKE '%' || $1 || '%' OR c.legal_name ILIKE '%' || $1 || '%'
		ORDER BY LENGTH(c.name) ASC
		LIMIT 1;
	`
	err = r.pool.QueryRow(ctx, queryFuzzy, normalized).Scan(&res.CanonicalName, &res.LegalName, &res.AHUNumber, &compID)
	if err == nil {
		res.CompanyID = compID
		res.MatchScore = 0.80
		res.IsMatched = true
		return &res, nil
	}

	return &model.AHUEntityResolutionResult{
		CanonicalName: cleanName,
		LegalName:     cleanName,
		AHUNumber:     "",
		CompanyID:     nil,
		MatchScore:    0.0,
		IsMatched:     false,
	}, nil
}

// LinkCompanyAHU explicitly links an AHU SK Kemenkumham number to a company.
func (r *AHURepository) LinkCompanyAHU(ctx context.Context, companyID, ahuNumber string) error {
	if r.pool == nil {
		return fmt.Errorf("database pool is nil")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE public.ahu_registrations
		SET company_id = $1::uuid, updated_at = NOW()
		WHERE ahu_number = $2;
	`, companyID, ahuNumber)
	if err != nil {
		return fmt.Errorf("failed to update AHU registration company_id: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE company.companies
		SET ahu_number = $1, updated_at = NOW()
		WHERE id = $2::uuid;
	`, ahuNumber, companyID)
	if err != nil {
		return fmt.Errorf("failed to update company ahu_number: %w", err)
	}

	return tx.Commit(ctx)
}
