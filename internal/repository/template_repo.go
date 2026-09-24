package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TenantTemplate struct {
	ID                string `json:"id"`
	OrgID             string `json:"org_id"`
	DocxS3Key         string `json:"docx_s3_key"`
	PptxS3Key         string `json:"pptx_s3_key"`
	BrandPrimaryColor string `json:"brand_primary_color"`
	LogoS3Key         string `json:"logo_s3_key"`
}

type TemplateRepository struct {
	dbPool *pgxpool.Pool
}

func NewTemplateRepository(dbPool *pgxpool.Pool) *TemplateRepository {
	return &TemplateRepository{dbPool: dbPool}
}

func (r *TemplateRepository) GetTenantTemplate(ctx context.Context, orgID string) (*TenantTemplate, error) {
	if r.dbPool == nil {
		return &TenantTemplate{
			OrgID:             orgID,
			BrandPrimaryColor: "#047857",
		}, nil
	}

	var t TenantTemplate
	err := WithTenantContext(ctx, r.dbPool, orgID, func(tx pgx.Tx) error {
		query := `
			SELECT id::text, org_id::text, COALESCE(docx_s3_key, ''), COALESCE(pptx_s3_key, ''), COALESCE(brand_primary_color, '#047857'), COALESCE(logo_s3_key, '')
			FROM public.tenant_templates
			WHERE org_id = $1::uuid;
		`
		return tx.QueryRow(ctx, query, orgID).Scan(
			&t.ID, &t.OrgID, &t.DocxS3Key, &t.PptxS3Key, &t.BrandPrimaryColor, &t.LogoS3Key,
		)
	})

	if err != nil {
		return &TenantTemplate{
			OrgID:             orgID,
			BrandPrimaryColor: "#047857",
		}, nil
	}

	return &t, nil
}

func (r *TemplateRepository) SaveTenantTemplateKey(ctx context.Context, orgID, fileType, s3Key string) error {
	if r.dbPool == nil {
		return nil
	}

	return WithTenantContext(ctx, r.dbPool, orgID, func(tx pgx.Tx) error {
		col := "pptx_s3_key"
		if fileType == "docx" {
			col = "docx_s3_key"
		}
		query := fmt.Sprintf(`
			INSERT INTO public.tenant_templates (org_id, %s, updated_at)
			VALUES ($1::uuid, $2, NOW())
			ON CONFLICT (org_id) DO UPDATE SET %s = EXCLUDED.%s, updated_at = NOW();
		`, col, col, col)
		_, err := tx.Exec(ctx, query, orgID, s3Key)
		return err
	})
}

func (r *TemplateRepository) ClearTenantTemplateKey(ctx context.Context, orgID, fileType string) error {
	if r.dbPool == nil {
		return nil
	}

	return WithTenantContext(ctx, r.dbPool, orgID, func(tx pgx.Tx) error {
		col := "pptx_s3_key"
		if fileType == "docx" {
			col = "docx_s3_key"
		}
		query := fmt.Sprintf(`
			UPDATE public.tenant_templates 
			SET %s = NULL, updated_at = NOW()
			WHERE org_id = $1::uuid;
		`, col)
		_, err := tx.Exec(ctx, query, orgID)
		return err
	})
}
