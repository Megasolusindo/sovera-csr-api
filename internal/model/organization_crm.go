package model

import "time"

// OrganizationProspect represents sales funnel and lead scoring data for an organization target.
type OrganizationProspect struct {
	ID            string     `json:"id" db:"id"`
	OrgID         string     `json:"org_id" db:"org_id"`
	SalesStatus   string     `json:"sales_status" db:"sales_status"`
	LeadScore     float64    `json:"lead_score" db:"lead_score"`
	Source        *string    `json:"source,omitempty" db:"source"`
	AssignedTo    *string    `json:"assigned_to,omitempty" db:"assigned_to"`
	FirstContactAt *time.Time `json:"first_contact_at,omitempty" db:"first_contact_at"`
	LastContactAt  *time.Time `json:"last_contact_at,omitempty" db:"last_contact_at"`
	NextFollowupAt *time.Time `json:"next_followup_at,omitempty" db:"next_followup_at"`
	ConvertedAt    *time.Time `json:"converted_at,omitempty" db:"converted_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// CRMContact represents personnel contacts within an organization.
type CRMContact struct {
	ID        string    `json:"id" db:"id"`
	OrgID     string    `json:"org_id" db:"org_id"`
	Name      string    `json:"name" db:"name"`
	Position  *string   `json:"position,omitempty" db:"position"`
	Email     *string   `json:"email,omitempty" db:"email"`
	Phone     *string   `json:"phone,omitempty" db:"phone"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// CRMActivity represents interaction logs with an organization contact.
type CRMActivity struct {
	ID           string    `json:"id" db:"id"`
	OrgID        string    `json:"org_id" db:"org_id"`
	ContactID    *string   `json:"contact_id,omitempty" db:"contact_id"`
	ActivityType string    `json:"activity_type" db:"activity_type"`
	Notes        *string   `json:"notes,omitempty" db:"notes"`
	ActivityAt   time.Time `json:"activity_at" db:"activity_at"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}
