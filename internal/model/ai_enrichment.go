package model

import "time"

// EnrichCompanyRequest DTO for POST /api/v1/ai/companies/:id/enrich
type EnrichCompanyRequest struct {
	Website           *string  `json:"website,omitempty"`
	Phone             *string  `json:"phone,omitempty"`
	Headquarters      *string  `json:"headquarters,omitempty"`
	IndustrySector    *string  `json:"industry_sector,omitempty"`
	EmployeeRange     *string  `json:"employee_range,omitempty"`
	RevenueRange      *string  `json:"revenue_range,omitempty"`
	CSRCategory       *string  `json:"csr_category,omitempty"`
	PriorityTier      *string  `json:"priority_tier,omitempty"`
	CSRFocuses        []string `json:"csr_focuses,omitempty"`
	ESGMaterialTopics []string `json:"esg_material_topics,omitempty"`
	PartnerNGO        *string  `json:"partner_ngo,omitempty"`
	AliasKeywords     []string `json:"alias_keywords,omitempty"`
}

// CompanyEnrichmentResult contains summary of fields enriched by OpenClaw
type CompanyEnrichmentResult struct {
	CompanyID      string    `json:"company_id"`
	CompanyName    string    `json:"company_name"`
	FieldsEnriched []string  `json:"fields_enriched"`
	EnrichedAt     time.Time `json:"enriched_at"`
}
