package esgextractor

import (
	"context"
	"fmt"
	"log"

	"sovera-core-api/internal/model"
)

type ESGPDFExtractionResult struct {
	CompanyESGProfile *model.CompanyESGProfile
	CSRBudgetAmount   float64
	ReportingYear     int
	MaterialTopics    []string
	SDGAlignment      map[string]interface{}
}

// ProcessSustainabilityReportPDF accepts multi-page PDF text or markdown content, parses reporting year,
// budget tables, material ESG topics, and aligns SDGs with company profiles.
func (e *ESGExtractor) ProcessSustainabilityReportPDF(
	ctx context.Context,
	rawPDFText string,
	markdownPDFContent string,
	companyName string,
	companyID *string,
) (*ESGPDFExtractionResult, error) {

	log.Printf("[ESG PDF Extractor] Parsing Sustainability Report PDF for [%s]...", companyName)

	bestContent := e.normalizer.SelectBestContent(rawPDFText, markdownPDFContent)
	if bestContent == "" {
		return nil, fmt.Errorf("cannot process PDF extraction on empty content")
	}

	// 1. Process standard ESG profile extraction via Gemini AI
	profile, err := e.ProcessESGExtraction(ctx, rawPDFText, markdownPDFContent, companyName, "", companyID)
	if err != nil {
		return nil, fmt.Errorf("failed during ESG profile LLM extraction: %w", err)
	}

	result := &ESGPDFExtractionResult{
		CompanyESGProfile: profile,
		ReportingYear:     int(profile.ReportingYear),
		SDGAlignment:      profile.SDGAlignment,
	}

	log.Printf("[ESG PDF Extractor] Completed extraction for [%s]. Reporting Year [%d], ESG Rating [%v]", companyName, result.ReportingYear, profile.ESGRating)
	return result, nil
}
