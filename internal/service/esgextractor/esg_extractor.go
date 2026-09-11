package esgextractor

import (
	"context"
	"fmt"
	"log"
	"time"

	"sovera-core-api/internal/model"
	"sovera-core-api/internal/repository"
	"sovera-core-api/internal/service/ai"
	"sovera-core-api/internal/service/entityresolver"
	"sovera-core-api/internal/service/normalizer"
)

type ESGExtractor struct {
	geminiService  *ai.GeminiService
	esgRepo        *repository.ESGProfileRepository
	normalizer     *normalizer.Normalizer
	entityResolver *entityresolver.EntityResolver
}

func NewESGExtractor(
	geminiService *ai.GeminiService,
	esgRepo *repository.ESGProfileRepository,
	norm *normalizer.Normalizer,
	resolver *entityresolver.EntityResolver,
) *ESGExtractor {
	if norm == nil {
		norm = normalizer.NewNormalizer()
	}
	return &ESGExtractor{
		geminiService:  geminiService,
		esgRepo:        esgRepo,
		normalizer:     norm,
		entityResolver: resolver,
	}
}

// ProcessESGExtraction cleans raw text, extracts ESG parameters via Gemini LLM, resolves the company ID if needed,
// and upserts the ESG profile to company_esg_profiles database.
func (e *ESGExtractor) ProcessESGExtraction(
	ctx context.Context,
	rawText, markdownContent string,
	companyName, industry string,
	companyID *string,
) (*model.CompanyESGProfile, error) {

	bestContent := e.normalizer.SelectBestContent(rawText, markdownContent)
	if bestContent == "" {
		return nil, fmt.Errorf("cannot process ESG extraction on empty content")
	}

	// 1. Resolve Company Master ID if missing
	targetCompanyID := companyID
	if (targetCompanyID == nil || *targetCompanyID == "") && e.entityResolver != nil && companyName != "" {
		resolved, err := e.entityResolver.ResolveCompany(ctx, companyName, industry)
		if err == nil && resolved != nil {
			targetCompanyID = resolved.CompanyID
		}
	}

	if targetCompanyID == nil || *targetCompanyID == "" {
		return nil, fmt.Errorf("company entity could not be resolved for ESG extraction")
	}

	// 2. Perform ESG Extraction via Gemini LLM
	extracted, err := e.geminiService.ExtractESGProfile(ctx, bestContent)
	if err != nil {
		return nil, fmt.Errorf("failed to extract ESG metrics via Gemini: %w", err)
	}

	reportingYear := extracted.ReportingYear
	if reportingYear == 0 {
		reportingYear = int16(time.Now().Year() - 1)
	}

	now := time.Now()
	esgProfile := &model.CompanyESGProfile{
		CompanyID:              *targetCompanyID,
		ReportingYear:          reportingYear,
		ReportDate:             &now,
		OverallScore:           &extracted.OverallScore,
		EnvironmentalScore:     &extracted.EnvironmentalScore,
		SocialScore:            &extracted.SocialScore,
		GovernanceScore:        &extracted.GovernanceScore,
		ESGRating:              &extracted.ESGRating,
		SustainabilityStrategy: &extracted.SustainabilityStrategy,
		SDGAlignment:           extracted.SDGAlignment,
		Confidence:             &extracted.Confidence,
	}

	// 3. Upsert ESG Profile to Database if Repo is available
	if e.esgRepo == nil {
		return nil, fmt.Errorf("esg repository not initialized")
	}

	upsertedProfile, err := e.esgRepo.UpsertProfile(ctx, esgProfile, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to persist ESG profile: %w", err)
	}

	log.Printf("[ESGExtractor] Successfully upserted ESG Profile [%s] Year [%d] Rating [%s]", upsertedProfile.ID, upsertedProfile.ReportingYear, *upsertedProfile.ESGRating)
	return upsertedProfile, nil
}
