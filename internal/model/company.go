package model

import "time"

type Company struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	LegalName       *string   `json:"legal_name,omitempty"`
	Slug            string    `json:"slug"`
	IndustryID      *string   `json:"industry_id,omitempty"`
	IndustrySector  string    `json:"industry_sector"`
	CompanyType     string    `json:"company_type"`
	Website         *string   `json:"website,omitempty"`
	Phone           *string   `json:"phone,omitempty"`
	LinkedinURL     *string   `json:"linkedin_url,omitempty"`
	LinkedinStatus  *string   `json:"linkedin_status,omitempty"`
	InstagramURL    *string   `json:"instagram_url,omitempty"`
	InstagramStatus *string   `json:"instagram_status,omitempty"`
	FacebookURL     *string   `json:"facebook_url,omitempty"`
	FacebookStatus  *string   `json:"facebook_status,omitempty"`
	YoutubeURL      *string   `json:"youtube_url,omitempty"`
	YoutubeStatus   *string   `json:"youtube_status,omitempty"`
	Headquarters    *string   `json:"headquarters,omitempty"`
	EmployeeRange   *string   `json:"employee_range,omitempty"`
	RevenueRange    *string   `json:"revenue_range,omitempty"`
	IsPublic        bool      `json:"is_public"`
	Ticker          *string   `json:"ticker,omitempty"`
	ParentCompanyID *string   `json:"parent_company_id,omitempty"`
	PriorityTier    string    `json:"priority_tier"`
	CSRCategory     string    `json:"csr_category"`
	PartnerNGO      *string   `json:"partner_ngo,omitempty"`
	AliasKeywords   []string  `json:"alias_keywords"`
	IsClaimed       bool      `json:"is_claimed"`
	CorporateDomain *string   `json:"corporate_domain,omitempty"`
	ClaimedByTenantID *string `json:"claimed_by_tenant_id,omitempty"`
	AHUNumber       *string   `json:"ahu_number,omitempty"`
	NIB             *string   `json:"nib,omitempty"`
	KBLICode        *string   `json:"kbli_code,omitempty"`
	LegalEntityType *string   `json:"legal_entity_type,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type CompanyDetail struct {
	Company
	CSRProfile  *CompanyCSRProfile `json:"csr_profile,omitempty"`
	TargetCount int                `json:"target_count"`
	SignalCount int                `json:"signal_count"`
	TotalBudget float64            `json:"total_budget_signal"`
}

type CorporateHierarchy struct {
	Parent            *CompanyDetail  `json:"parent,omitempty"`
	Company           CompanyDetail   `json:"company"`
	Subsidiaries      []CompanyDetail `json:"subsidiaries"`
	TotalSubsidiaries int             `json:"total_subsidiaries"`
}
