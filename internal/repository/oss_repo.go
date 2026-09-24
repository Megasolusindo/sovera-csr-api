package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"sovera-core-api/internal/model"
)

type OSSRepository struct {
	pool *pgxpool.Pool
}

func NewOSSRepository(pool *pgxpool.Pool) *OSSRepository {
	return &OSSRepository{pool: pool}
}

// GetByNIB retrieves an OSS NIB registration by exact 13-digit NIB number.
func (r *OSSRepository) GetByNIB(ctx context.Context, nib string) (*model.OSSNIBRegistration, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	query := `
		SELECT 
			o.id::text, o.company_id::text, o.nib, o.business_name, o.legal_entity_type,
			o.kbli_code, k.title AS kbli_title, o.risk_level, o.investment_status,
			o.province, o.regency_city, o.district, o.address, o.license_status,
			o.issued_date, o.verified_at, o.created_at, o.updated_at
		FROM public.oss_nib_registrations o
		LEFT JOIN public.kbli_reference k ON k.code = o.kbli_code
		WHERE o.nib = $1
		LIMIT 1;
	`

	var item model.OSSNIBRegistration
	err := r.pool.QueryRow(ctx, query, nib).Scan(
		&item.ID, &item.CompanyID, &item.NIB, &item.BusinessName, &item.LegalEntityType,
		&item.KBLICode, &item.KBLITitle, &item.RiskLevel, &item.InvestmentStatus,
		&item.Province, &item.RegencyCity, &item.District, &item.Address, &item.LicenseStatus,
		&item.IssuedDate, &item.VerifiedAt, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("OSS NIB record '%s' not found: %w", nib, err)
	}

	return &item, nil
}

// Search queries OSS NIB registrations by keyword, KBLI code, investment status (PMDN/PMA), or province.
func (r *OSSRepository) Search(ctx context.Context, searchQuery, kbliCode, investmentStatus, province string, limit int) ([]model.OSSNIBRegistration, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	if limit <= 0 {
		limit = 20
	}

	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if searchQuery != "" {
		whereClause += fmt.Sprintf(" AND (o.business_name ILIKE $%d OR o.nib ILIKE $%d OR o.address ILIKE $%d)", argIdx, argIdx, argIdx)
		args = append(args, "%"+strings.TrimSpace(searchQuery)+"%")
		argIdx++
	}

	if kbliCode != "" {
		whereClause += fmt.Sprintf(" AND o.kbli_code = $%d", argIdx)
		args = append(args, kbliCode)
		argIdx++
	}

	if investmentStatus != "" {
		whereClause += fmt.Sprintf(" AND o.investment_status = $%d", argIdx)
		args = append(args, strings.ToUpper(investmentStatus))
		argIdx++
	}

	if province != "" {
		whereClause += fmt.Sprintf(" AND o.province ILIKE $%d", argIdx)
		args = append(args, "%"+strings.TrimSpace(province)+"%")
		argIdx++
	}

	query := fmt.Sprintf(`
		SELECT 
			o.id::text, o.company_id::text, o.nib, o.business_name, o.legal_entity_type,
			o.kbli_code, k.title AS kbli_title, o.risk_level, o.investment_status,
			o.province, o.regency_city, o.district, o.address, o.license_status,
			o.issued_date, o.verified_at, o.created_at, o.updated_at
		FROM public.oss_nib_registrations o
		LEFT JOIN public.kbli_reference k ON k.code = o.kbli_code
		%s
		ORDER BY o.created_at DESC
		LIMIT $%d;
	`, whereClause, argIdx)

	args = append(args, limit)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search OSS NIB registrations: %w", err)
	}
	defer rows.Close()

	var result []model.OSSNIBRegistration
	for rows.Next() {
		var item model.OSSNIBRegistration
		err := rows.Scan(
			&item.ID, &item.CompanyID, &item.NIB, &item.BusinessName, &item.LegalEntityType,
			&item.KBLICode, &item.KBLITitle, &item.RiskLevel, &item.InvestmentStatus,
			&item.Province, &item.RegencyCity, &item.District, &item.Address, &item.LicenseStatus,
			&item.IssuedDate, &item.VerifiedAt, &item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan OSS NIB row: %w", err)
		}
		result = append(result, item)
	}

	return result, nil
}

// UpsertNIB inserts or updates an OSS NIB registration record and auto-links company_id if matched.
func (r *OSSRepository) UpsertNIB(ctx context.Context, reg *model.OSSNIBRegistration) (*model.OSSNIBRegistration, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Resolve company_id by exact or fuzzy name match if missing
	if reg.CompanyID == nil || *reg.CompanyID == "" {
		var compID string
		err := tx.QueryRow(ctx, `
			SELECT id::text FROM company.companies
			WHERE LOWER(TRIM(name)) = LOWER(TRIM($1))
			   OR LOWER(TRIM(legal_name)) = LOWER(TRIM($1))
			   OR LOWER(TRIM(name)) ILIKE '%' || LOWER(TRIM($1)) || '%'
			LIMIT 1;
		`, reg.BusinessName).Scan(&compID)
		if err == nil && compID != "" {
			reg.CompanyID = &compID
		}
	}

	query := `
		INSERT INTO public.oss_nib_registrations (
			company_id, nib, business_name, legal_entity_type, kbli_code, risk_level,
			investment_status, province, regency_city, district, address, license_status, issued_date, verified_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW(), NOW()
		)
		ON CONFLICT (nib) DO UPDATE SET
			company_id = COALESCE(EXCLUDED.company_id, public.oss_nib_registrations.company_id),
			business_name = EXCLUDED.business_name,
			legal_entity_type = EXCLUDED.legal_entity_type,
			kbli_code = COALESCE(EXCLUDED.kbli_code, public.oss_nib_registrations.kbli_code),
			risk_level = EXCLUDED.risk_level,
			investment_status = EXCLUDED.investment_status,
			province = COALESCE(EXCLUDED.province, public.oss_nib_registrations.province),
			regency_city = COALESCE(EXCLUDED.regency_city, public.oss_nib_registrations.regency_city),
			district = COALESCE(EXCLUDED.district, public.oss_nib_registrations.district),
			address = COALESCE(EXCLUDED.address, public.oss_nib_registrations.address),
			license_status = EXCLUDED.license_status,
			issued_date = COALESCE(EXCLUDED.issued_date, public.oss_nib_registrations.issued_date),
			updated_at = NOW()
		RETURNING id::text, company_id::text, verified_at, created_at, updated_at;
	`

	var compIDNullable *string
	err = tx.QueryRow(ctx, query,
		reg.CompanyID, reg.NIB, reg.BusinessName, reg.LegalEntityType, reg.KBLICode, reg.RiskLevel,
		reg.InvestmentStatus, reg.Province, reg.RegencyCity, reg.District, reg.Address, reg.LicenseStatus, reg.IssuedDate,
	).Scan(&reg.ID, &compIDNullable, &reg.VerifiedAt, &reg.CreatedAt, &reg.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to upsert OSS NIB registration: %w", err)
	}

	reg.CompanyID = compIDNullable

	// Update company.companies with NIB, KBLI, and HQ address if linked
	if reg.CompanyID != nil && *reg.CompanyID != "" {
		_, _ = tx.Exec(ctx, `
			UPDATE company.companies SET
				nib = $1,
				kbli_code = COALESCE(NULLIF(kbli_code, ''), $2),
				headquarters = COALESCE(NULLIF(headquarters, ''), $3),
				updated_at = NOW()
			WHERE id = $4::uuid;
		`, reg.NIB, reg.KBLICode, reg.Address, *reg.CompanyID)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit tx: %w", err)
	}

	return reg, nil
}

// LinkCompanyNIB explicitly links an OSS 13-digit NIB to a company.
func (r *OSSRepository) LinkCompanyNIB(ctx context.Context, companyID, nib string) error {
	if r.pool == nil {
		return fmt.Errorf("database pool is nil")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE public.oss_nib_registrations
		SET company_id = $1::uuid, updated_at = NOW()
		WHERE nib = $2;
	`, companyID, nib)
	if err != nil {
		return fmt.Errorf("failed to update OSS NIB company_id: %w", err)
	}

	var kbliCode *string
	_ = tx.QueryRow(ctx, "SELECT kbli_code FROM public.oss_nib_registrations WHERE nib = $1", nib).Scan(&kbliCode)

	_, err = tx.Exec(ctx, `
		UPDATE company.companies
		SET nib = $1,
		    kbli_code = COALESCE(NULLIF(kbli_code, ''), $2),
		    updated_at = NOW()
		WHERE id = $3::uuid;
	`, nib, kbliCode, companyID)
	if err != nil {
		return fmt.Errorf("failed to update company nib and kbli_code: %w", err)
	}

	return tx.Commit(ctx)
}
