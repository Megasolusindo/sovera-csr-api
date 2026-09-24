package model

import "time"

type KBLIReference struct {
	Code                string    `json:"code"`
	Title               string    `json:"title"`
	CategoryCode        string    `json:"category_code"`
	CategoryTitle       string    `json:"category_title"`
	CSRRelevanceDefault string    `json:"csr_relevance_default"`
	RiskLevel           string    `json:"risk_level"`
	Description         *string   `json:"description,omitempty"`
	CompanyCount        int       `json:"company_count"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}
