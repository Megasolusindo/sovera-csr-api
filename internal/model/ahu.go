package model

import "time"

type AHURegistration struct {
	ID              string     `json:"id"`
	CompanyID       *string    `json:"company_id,omitempty"`
	CompanyName     string     `json:"company_name"`
	LegalName       string     `json:"legal_name"`
	AHUNumber       string     `json:"ahu_number"`
	LegalEntityType string     `json:"legal_entity_type"`
	DeedNumber      *string    `json:"deed_number,omitempty"`
	DeedDate        *time.Time `json:"deed_date,omitempty"`
	NotaryName      *string    `json:"notary_name,omitempty"`
	Status          string     `json:"status"`
	Headquarters    *string    `json:"headquarters,omitempty"`
	CapitalAmount   *float64   `json:"capital_amount,omitempty"`
	VerifiedAt      time.Time  `json:"verified_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type AHUEntityResolutionRequest struct {
	RawName        string `json:"raw_name"`
	IndustrySector string `json:"industry_sector,omitempty"`
}

type AHUEntityResolutionResult struct {
	CanonicalName string  `json:"canonical_name"`
	LegalName     string  `json:"legal_name"`
	AHUNumber     string  `json:"ahu_number"`
	CompanyID     *string `json:"company_id,omitempty"`
	MatchScore    float64 `json:"match_score"`
	IsMatched     bool    `json:"is_matched"`
}
