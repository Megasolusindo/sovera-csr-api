package model

import "time"

// Role categories for key persons
const (
	RoleCategoryCSRLead        = "csr_lead"
	RoleCategoryCorpSec        = "corp_sec"
	RoleCategoryCLevel         = "c_level"
	RoleCategoryFoundationHead = "foundation_head"
)

// Social signal platforms
const (
	PlatformLinkedIn  = "linkedin"
	PlatformTwitter   = "twitter"
	PlatformInstagram = "instagram"
	PlatformNewsWeb   = "news_web"
)

// CompanyKeyPerson represents an executive or CSR contact person for a company
type CompanyKeyPerson struct {
	ID              string    `json:"id"`
	CompanyID       string    `json:"company_id"`
	FullName        string    `json:"full_name"`
	NormalizedName  string    `json:"normalized_name"`
	CurrentTitle    *string   `json:"current_title,omitempty"`
	RoleCategory    *string   `json:"role_category,omitempty"`
	LinkedInURL     *string   `json:"linkedin_url,omitempty"`
	TwitterHandle   *string   `json:"twitter_handle,omitempty"`
	InstagramHandle *string   `json:"instagram_handle,omitempty"`
	IsDecisionMaker bool      `json:"is_decision_maker"`
	IsMonitored     bool      `json:"is_monitored"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// KeyPersonSocialSignal represents a social post or news signal associated with a key person
type KeyPersonSocialSignal struct {
	ID              string     `json:"id"`
	PersonID        *string    `json:"person_id,omitempty"`
	CompanyID       string     `json:"company_id"`
	Platform        string     `json:"platform"`
	PostURL         *string    `json:"post_url,omitempty"`
	PostText        string     `json:"post_text"`
	PostedAt        *time.Time `json:"posted_at,omitempty"`
	MatchedKeywords []string   `json:"matched_keywords"`
	Sentiment       *string    `json:"sentiment,omitempty"`
	AISummary       *string    `json:"ai_summary,omitempty"`
	IsActionable    bool       `json:"is_actionable"`
	CreatedAt       time.Time  `json:"created_at"`
}
