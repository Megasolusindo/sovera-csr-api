package model

import "time"

// TenantType defines the type of organization entity using the platform.
type TenantType string

const (
	TenantTypeOrganization TenantType = "ORGANIZATION"
	TenantTypeCorporate    TenantType = "CORPORATE"
	TenantTypeAdmin        TenantType = "ADMIN"
)

// ClaimStatus defines the state of a corporate profile verification request.
type ClaimStatus string

const (
	ClaimStatusPending  ClaimStatus = "PENDING"
	ClaimStatusApproved ClaimStatus = "APPROVED"
	ClaimStatusRejected ClaimStatus = "REJECTED"
	ClaimStatusRevoked  ClaimStatus = "REVOKED"
)

// ClaimMethod defines how a company profile claim is verified.
type ClaimMethod string

const (
	ClaimMethodCorporateEmail ClaimMethod = "CORPORATE_EMAIL"
	ClaimMethodDocumentUpload ClaimMethod = "DOCUMENT_UPLOAD"
)

// ProposalStatus defines the workflow state of an NGO proposal to a Corporate Opportunity.
type ProposalStatus string

const (
	ProposalStatusSubmitted   ProposalStatus = "SUBMITTED"
	ProposalStatusUnderReview ProposalStatus = "UNDER_REVIEW"
	ProposalStatusMeeting     ProposalStatus = "MEETING"
	ProposalStatusAccepted    ProposalStatus = "ACCEPTED"
	ProposalStatusRejected    ProposalStatus = "REJECTED"
)

// Organization represents a registered platform tenant (NGO or Corporate).
type Organization struct {
	ID               string     `json:"id" db:"id"`
	Name             string     `json:"name" db:"name"`
	Slug             string     `json:"slug,omitempty" db:"slug"`
	Type             TenantType `json:"type" db:"type"`
	CompanyID        *string    `json:"company_id,omitempty" db:"company_id"`
	SubscriptionTier string     `json:"subscription_tier" db:"subscription_tier"`
	IsVerified       bool       `json:"is_verified" db:"is_verified"`
	LogoURL          string     `json:"logo_url,omitempty" db:"logo_url"`
	Website          string     `json:"website,omitempty" db:"website"`
	ContactName      string     `json:"contact_name,omitempty" db:"contact_name"`
	ContactEmail     string     `json:"contact_email,omitempty" db:"contact_email"`
	ContactPhone     string     `json:"contact_phone,omitempty" db:"contact_phone"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
}

// CompanyClaim represents a verification claim request for a corporate profile.
type CompanyClaim struct {
	ID                 string      `json:"id" db:"id"`
	CompanyID          string      `json:"company_id" db:"company_id"`
	TenantID           string      `json:"tenant_id" db:"tenant_id"`
	RequestedByUserID  string      `json:"requested_by_user_id" db:"requested_by_user_id"`
	WorkEmail          string      `json:"work_email,omitempty" db:"work_email"`
	DocumentProofURL   string      `json:"document_proof_url,omitempty" db:"document_proof_url"`
	Method             ClaimMethod `json:"method" db:"method"`
	Status             ClaimStatus `json:"status" db:"status"`
	VerificationNotes  string      `json:"verification_notes,omitempty" db:"verification_notes"`
	VerifiedAt         *time.Time  `json:"verified_at,omitempty" db:"verified_at"`
	VerifiedByUserID   *string     `json:"verified_by_user_id,omitempty" db:"verified_by_user_id"`
	CreatedAt          time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time   `json:"updated_at" db:"updated_at"`
}

// CSROpportunity represents a corporate RFP or grant opening published for NGOs.
type CSROpportunity struct {
	ID              string    `json:"id" db:"id"`
	CompanyID       string    `json:"company_id" db:"company_id"`
	TenantID        string    `json:"tenant_id" db:"tenant_id"`
	Title           string    `json:"title" db:"title"`
	Description     string    `json:"description,omitempty" db:"description"`
	Category        string    `json:"category" db:"category"`
	TargetLocation  string    `json:"target_location,omitempty" db:"target_location"`
	BudgetAmount    float64   `json:"budget_amount" db:"budget_amount"`
	OpenUntil       *time.Time `json:"open_until,omitempty" db:"open_until"`
	Status          string    `json:"status" db:"status"`
	CreatedByUserID *string   `json:"created_by_user_id,omitempty" db:"created_by_user_id"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// NGOProgram represents an NGO project or proposal offer.
type NGOProgram struct {
	ID                 string    `json:"id" db:"id"`
	TenantID           string    `json:"tenant_id" db:"tenant_id"`
	Title              string    `json:"title" db:"title"`
	Description        string    `json:"description,omitempty" db:"description"`
	Category           string    `json:"category" db:"category"`
	Location           string    `json:"location,omitempty" db:"location"`
	TargetBeneficiaries int       `json:"target_beneficiaries" db:"target_beneficiaries"`
	BudgetNeeded       float64   `json:"budget_needed" db:"budget_needed"`
	SDGGoals           []int     `json:"sdg_goals,omitempty" db:"sdg_goals"`
	FiqhAsnaf          string    `json:"fiqh_asnaf,omitempty" db:"fiqh_asnaf"`
	CreatedByUserID    *string   `json:"created_by_user_id,omitempty" db:"created_by_user_id"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}

// Proposal represents an NGO proposal submitted to a Corporate Opportunity.
type Proposal struct {
	ID                  string         `json:"id" db:"id"`
	OpportunityID       *string        `json:"opportunity_id,omitempty" db:"opportunity_id"`
	NGOProgramID        *string        `json:"ngo_program_id,omitempty" db:"ngo_program_id"`
	OrgTenantID         string         `json:"org_tenant_id" db:"org_tenant_id"`
	CorpTenantID        *string        `json:"corp_tenant_id,omitempty" db:"corp_tenant_id"`
	CompanyID           string         `json:"company_id" db:"company_id"`
	Title               string         `json:"title" db:"title"`
	Summary             string         `json:"summary,omitempty" db:"summary"`
	ProposalFileURL     string         `json:"proposal_file_url,omitempty" db:"proposal_file_url"`
	BudgetRequested     float64        `json:"budget_requested" db:"budget_requested"`
	Status              ProposalStatus `json:"status" db:"status"`
	ReviewerNotes       string         `json:"reviewer_notes,omitempty" db:"reviewer_notes"`
	SubmittedByUserID   *string        `json:"submitted_by_user_id,omitempty" db:"submitted_by_user_id"`
	CreatedAt           time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at" db:"updated_at"`
}

// UserInvitation represents a pending team invite.
type UserInvitation struct {
	ID               string    `json:"id" db:"id"`
	OrgID            string    `json:"org_id" db:"org_id"`
	InvitedByUserID  string    `json:"invited_by_user_id" db:"invited_by_user_id"`
	Email            string    `json:"email" db:"email"`
	Role             string    `json:"role" db:"role"`
	InviteToken      string    `json:"invite_token" db:"invite_token"`
	Status           string    `json:"status" db:"status"`
	ExpiresAt        time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}
