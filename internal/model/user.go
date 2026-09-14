package model

import "time"

// UserOrgRole defines the RBAC roles available within a tenant organization.
type UserOrgRole string

const (
	RoleOrgAdmin   UserOrgRole = "ORG_ADMIN"
	RoleDirector   UserOrgRole = "DIRECTOR"
	RoleFundraiser UserOrgRole = "FUNDRAISER"

	// Corporate Roles
	RoleCorpAdmin  UserOrgRole = "CORP_ADMIN"
	RoleCSRManager UserOrgRole = "CSR_MANAGER"
	RoleReviewer   UserOrgRole = "REVIEWER"

	// Platform Superadmin Role
	RoleSuperadmin UserOrgRole = "SUPERADMIN"
)

// User represents an authenticated user belonging to a tenant organization.
type User struct {
	ID           string      `json:"id" db:"id"`
	OrgID        string      `json:"org_id" db:"org_id"`
	Email        string      `json:"email" db:"email"`
	PasswordHash string      `json:"-" db:"password_hash"`
	FullName     string      `json:"full_name" db:"full_name"`
	Role         UserOrgRole `json:"role" db:"role"`
	IsActive     bool        `json:"is_active" db:"is_active"`
	CreatedAt    time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at" db:"updated_at"`
}
