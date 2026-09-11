package entityresolver

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"sovera-core-api/internal/model"
	"sovera-core-api/internal/repository"
)

var (
	// Regex matching common corporate legal prefixes/suffixes
	legalSuffixRegex = regexp.MustCompile(`(?i)\b(PT\.?|Persero|Tbk\.?|CV\.?|Inc\.?|Corp\.?|Corporation|Limited|Ltd\.?|Group|Holdings?)\b`)
	punctuationRegex = regexp.MustCompile(`[^\w\s-]`)
	multiSpaceRegex  = regexp.MustCompile(`\s+`)
)

type ResolvedCompany struct {
	CompanyID       *string `json:"company_id"`
	CanonicalName   string  `json:"canonical_name"`
	Slug            string  `json:"slug"`
	IsNew           bool    `json:"is_new"`
	MatchConfidence float64 `json:"match_confidence"`
}

type EntityResolver struct {
	companyRepo *repository.CompanyRepository
}

func NewEntityResolver(companyRepo *repository.CompanyRepository) *EntityResolver {
	return &EntityResolver{companyRepo: companyRepo}
}

// CleanCompanyName strips legal prefixes/suffixes (e.g. PT, Tbk, Persero) and punctuation.
func (e *EntityResolver) CleanCompanyName(rawName string) string {
	if rawName == "" {
		return ""
	}

	cleaned := legalSuffixRegex.ReplaceAllString(rawName, " ")
	cleaned = punctuationRegex.ReplaceAllString(cleaned, " ")
	cleaned = multiSpaceRegex.ReplaceAllString(cleaned, " ")

	return strings.TrimSpace(cleaned)
}

// GenerateSlug generates a clean URL/DB slug from a company name (e.g. "Telkom Indonesia" -> "telkom-indonesia").
func (e *EntityResolver) GenerateSlug(rawName string) string {
	cleaned := e.CleanCompanyName(rawName)
	if cleaned == "" {
		cleaned = rawName
	}

	slug := strings.ToLower(cleaned)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = regexp.MustCompile(`-+`).ReplaceAllString(slug, "-")

	return strings.Trim(slug, "-")
}

// IsCV returns true if rawName represents a CV (Commanditaire Vennootschap) entity.
func IsCV(rawName string) bool {
	upper := strings.TrimSpace(strings.ToUpper(rawName))
	return strings.HasPrefix(upper, "CV ") || strings.HasPrefix(upper, "CV.") || regexp.MustCompile(`(?i)\bCV\.?\b`).MatchString(rawName)
}

// ResolveCompany attempts to find a matching company in companies master table.
// If none exists, it auto-provisions a new Company record.
func (e *EntityResolver) ResolveCompany(ctx context.Context, rawName, industrySector string) (*ResolvedCompany, error) {
	if IsCV(rawName) {
		return nil, fmt.Errorf("entity '%s' is a CV and excluded from corporate directory", rawName)
	}

	cleanName := e.CleanCompanyName(rawName)
	if cleanName == "" {
		cleanName = rawName
	}
	slug := e.GenerateSlug(rawName)

	if e.companyRepo == nil {
		return nil, fmt.Errorf("company repository not initialized")
	}

	// 1. Search existing company by slug, name ILIKE or alias keywords
	existing, err := e.companyRepo.FindBySlugOrAlias(ctx, slug, cleanName)
	if err == nil && existing != nil {
		return &ResolvedCompany{
			CompanyID:       &existing.ID,
			CanonicalName:   existing.Name,
			Slug:            existing.Slug,
			IsNew:           false,
			MatchConfidence: 0.95,
		}, nil
	}

	// 2. Auto-provision new company master record if not found
	canonicalName := rawName
	if !strings.HasPrefix(strings.ToUpper(rawName), "PT") {
		canonicalName = "PT " + rawName
	}

	newCompany := model.Company{
		Name:           canonicalName,
		Slug:           slug,
		IndustrySector: industrySector,
		CompanyType:    "SWASTA",
		AliasKeywords:  []string{cleanName, rawName},
	}

	created, err := e.companyRepo.CreateCompany(ctx, newCompany)
	if err != nil {
		return nil, fmt.Errorf("failed auto-provisioning resolved company: %w", err)
	}

	return &ResolvedCompany{
		CompanyID:       &created.ID,
		CanonicalName:   created.Name,
		Slug:            created.Slug,
		IsNew:           true,
		MatchConfidence: 1.0,
	}, nil
}
