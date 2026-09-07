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
	LinkedinURL     *string   `json:"linkedin_url,omitempty"`
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
