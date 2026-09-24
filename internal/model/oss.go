package model

import "time"

type OSSNIBRegistration struct {
	ID               string     `json:"id"`
	CompanyID        *string    `json:"company_id,omitempty"`
	NIB              string     `json:"nib"`
	BusinessName     string     `json:"business_name"`
	LegalEntityType  string     `json:"legal_entity_type"`
	KBLICode         *string    `json:"kbli_code,omitempty"`
	KBLITitle        *string    `json:"kbli_title,omitempty"`
	RiskLevel        string     `json:"risk_level"`
	InvestmentStatus string     `json:"investment_status"`
	Province         *string    `json:"province,omitempty"`
	RegencyCity      *string    `json:"regency_city,omitempty"`
	District         *string    `json:"district,omitempty"`
	Address          *string    `json:"address,omitempty"`
	LicenseStatus    string     `json:"license_status"`
	IssuedDate       *time.Time `json:"issued_date,omitempty"`
	VerifiedAt       time.Time  `json:"verified_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}
