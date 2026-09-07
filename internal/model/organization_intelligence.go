package model

import "time"

// Source represents scraped or web sources referenced for intelligence data.
type Source struct {
	ID          string     `json:"id" db:"id"`
	SourceType  string     `json:"source_type" db:"source_type"`
	Name        string     `json:"name" db:"name"`
	URL         *string    `json:"url,omitempty" db:"url"`
	Domain      *string    `json:"domain,omitempty" db:"domain"`
	Title       *string    `json:"title,omitempty" db:"title"`
	PublishedAt *time.Time `json:"published_at,omitempty" db:"published_at"`
	AccessedAt  time.Time  `json:"accessed_at" db:"accessed_at"`
	Confidence  float64    `json:"confidence" db:"confidence"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// OrganizationProfile stores intelligence metadata for an organization.
type OrganizationProfile struct {
	ID                     string     `json:"id" db:"id"`
	OrgID                  string     `json:"org_id" db:"org_id"`
	Description            *string    `json:"description,omitempty" db:"description"`
	CoverageScope          *string    `json:"coverage_scope,omitempty" db:"coverage_scope"`
	HasCSRPartnerships     bool       `json:"has_csr_partnerships" db:"has_csr_partnerships"`
	HasCorporatePartnership bool      `json:"has_corporate_partnership" db:"has_corporate_partnership"`
	HasGrantProgram        bool       `json:"has_grant_program" db:"has_grant_program"`
	PartnershipsPageURL    *string    `json:"partnerships_page_url,omitempty" db:"partnerships_page_url"`
	Confidence             float64    `json:"confidence" db:"confidence"`
	LastVerifiedAt         *time.Time `json:"last_verified_at,omitempty" db:"last_verified_at"`
	CreatedAt              time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at" db:"updated_at"`
}

// OrganizationFocus represents a link between an organization and a CSR focus area.
type OrganizationFocus struct {
	ID         string    `json:"id" db:"id"`
	OrgID      string    `json:"org_id" db:"org_id"`
	FocusID    string    `json:"focus_id" db:"focus_id"`
	Priority   *string   `json:"priority,omitempty" db:"priority"`
	Confidence float64   `json:"confidence" db:"confidence"`
	SourceID   *string   `json:"source_id,omitempty" db:"source_id"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// OrganizationProgram represents a social/CSR program run or managed by an organization.
type OrganizationProgram struct {
	ID                     string    `json:"id" db:"id"`
	OrgID                  string    `json:"org_id" db:"org_id"`
	Name                   string    `json:"name" db:"name"`
	Description            *string   `json:"description,omitempty" db:"description"`
	ProgramType            *string   `json:"program_type,omitempty" db:"program_type"`
	Status                 string    `json:"status" db:"status"`
	LocationScope          *string   `json:"location_scope,omitempty" db:"location_scope"`
	BeneficiaryDescription *string   `json:"beneficiary_description,omitempty" db:"beneficiary_description"`
	SourceID               *string   `json:"source_id,omitempty" db:"source_id"`
	Confidence             float64   `json:"confidence" db:"confidence"`
	CreatedAt              time.Time `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time `json:"updated_at" db:"updated_at"`
}

// OrganizationPartnership represents a partnership between an organization and a corporate company.
type OrganizationPartnership struct {
	ID              string     `json:"id" db:"id"`
	OrgID           string     `json:"org_id" db:"org_id"`
	CompanyID       string     `json:"company_id" db:"company_id"`
	PartnershipType *string    `json:"partnership_type,omitempty" db:"partnership_type"`
	ProgramName     *string    `json:"program_name,omitempty" db:"program_name"`
	Description     *string    `json:"description,omitempty" db:"description"`
	StartDate       *time.Time `json:"start_date,omitempty" db:"start_date"`
	EndDate         *time.Time `json:"end_date,omitempty" db:"end_date"`
	SourceID        *string    `json:"source_id,omitempty" db:"source_id"`
	Confidence      float64    `json:"confidence" db:"confidence"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}

// OrganizationSignal represents media/press signals detected for an organization.
type OrganizationSignal struct {
	ID          string    `json:"id" db:"id"`
	OrgID       string    `json:"org_id" db:"org_id"`
	SignalPress *string   `json:"signal_press,omitempty" db:"signal_press"`
	SignalExt   *string   `json:"signal_ext,omitempty" db:"signal_ext"`
	SourceID    *string   `json:"source_id,omitempty" db:"source_id"`
	Confidence  float64   `json:"confidence" db:"confidence"`
	DetectedAt  time.Time `json:"detected_at" db:"detected_at"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}
