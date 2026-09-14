package model

import (
	"time"

	"github.com/google/uuid"
)

// AICompanyWatchlist represents a company being actively monitored by OpenClaw
type AICompanyWatchlist struct {
	ID                  uuid.UUID  `json:"id" db:"id"`
	CompanyID           uuid.UUID  `json:"company_id" db:"company_id"`
	CompanyName         string     `json:"company_name" db:"company_name"`
	MonitoringKeywords  []string   `json:"monitoring_keywords" db:"monitoring_keywords"`
	CheckIntervalHours int        `json:"check_interval_hours" db:"check_interval_hours"`
	LastMonitoredAt     *time.Time `json:"last_monitored_at,omitempty" db:"last_monitored_at"`
	IsActive            bool       `json:"is_active" db:"is_active"`
	CreatedAt           time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at" db:"updated_at"`
}

// ToggleWatchlistRequest DTO for adding/updating company watchlist settings
type ToggleWatchlistRequest struct {
	CompanyID           string   `json:"company_id"`
	MonitoringKeywords  []string `json:"monitoring_keywords,omitempty"`
	CheckIntervalHours int      `json:"check_interval_hours,omitempty"`
	IsActive            *bool    `json:"is_active,omitempty"`
}
