package model

import (
	"time"
)

type CrawlingTarget struct {
	ID                 string     `json:"id" db:"id"`
	CompanyID          *string    `json:"company_id,omitempty" db:"company_id"`
	SourceName         string     `json:"source_name" db:"source_name"`
	SourceType         string     `json:"source_type" db:"source_type"`
	TargetURL          string     `json:"target_url" db:"target_url"`
	CheckIntervalHours int        `json:"check_interval_hours" db:"check_interval_hours"`
	LastScrapedAt      *time.Time `json:"last_scraped_at,omitempty" db:"last_scraped_at"`
	NextRunAt          time.Time  `json:"next_run_at" db:"next_run_at"`
	IsActive           bool       `json:"is_active" db:"is_active"`
	ConsecutiveFailures int        `json:"consecutive_failures" db:"consecutive_failures"`
	LastHTTPStatus      *int       `json:"last_http_status,omitempty" db:"last_http_status"`
	LastErrorMsg        *string    `json:"last_error_message,omitempty" db:"last_error_message"`
	HealthStatus        string     `json:"health_status" db:"health_status"` // HEALTHY, DEGRADED, DISABLED_DEAD_LINK
	CreatedAt           time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at" db:"updated_at"`
}

type CrawlingLog struct {
	ID              string    `json:"id" db:"id"`
	TargetID        *string   `json:"target_id,omitempty" db:"target_id"`
	TaskID          string    `json:"task_id" db:"task_id"`
	Status          string    `json:"status" db:"status"` // DISPATCHED, COMPLETED, FAILED
	HTTPStatusCode  *int      `json:"http_status_code,omitempty" db:"http_status_code"`
	ErrorMessage    *string   `json:"error_message,omitempty" db:"error_message"`
	ExecutionTimeMs *int      `json:"execution_time_ms,omitempty" db:"execution_time_ms"`
	ContentHash     *string   `json:"content_hash,omitempty" db:"content_hash"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

type ScrapeTaskConfig struct {
	RenderJS      bool `json:"render_js"`
	BypassAntiBot bool `json:"bypass_anti_bot"`
	MaxPages      int  `json:"max_pages,omitempty"`
}

type ScrapeTaskPayload struct {
	TaskID       string            `json:"task_id"`
	TargetID     string            `json:"target_id,omitempty"`
	CompanyID    string            `json:"company_id,omitempty"`
	ClientOrigin string            `json:"client_origin"`
	SourceType   string            `json:"source_type"`
	TargetURL    string            `json:"target_url"`
	CallbackURL  string            `json:"callback_url"`
	Config       *ScrapeTaskConfig `json:"config,omitempty"`
}

type DiscoveryTaskPayload struct {
	TaskID       string `json:"task_id"`
	ClientOrigin string `json:"client_origin"`
	Query        string `json:"query"`
	Engine       string `json:"engine,omitempty"`
	Limit        int    `json:"limit,omitempty"`
	CallbackURL  string `json:"callback_url"`
}

type CrawlJobPayload struct {
	TaskID       string   `json:"task_id"`
	TargetID     string   `json:"target_id,omitempty"`
	ClientOrigin string   `json:"client_origin"`
	BaseURL      string   `json:"base_url"`
	URLPatterns  []string `json:"url_patterns,omitempty"`
	MaxDepth     int      `json:"max_depth,omitempty"`
	MaxPages     int      `json:"max_pages,omitempty"`
	CallbackURL  string   `json:"callback_url"`
}

type TaskAcceptedResponse struct {
	TaskID string `json:"task_id"`
	Status string `json:"status"`
}

type InspectBatchItemRequest struct {
	ReferenceID string `json:"reference_id"`
	TargetURL   string `json:"target_url"`
	KnownHash   string `json:"known_hash"`
}

type InspectBatchPayload struct {
	Items []InspectBatchItemRequest `json:"items"`
}

type InspectBatchResult struct {
	ReferenceID    string `json:"reference_id"`
	TargetURL      string `json:"target_url"`
	HTTPStatusCode int    `json:"http_status_code"`
	IsModified     bool   `json:"is_modified"`
	CurrentHash    string `json:"current_hash"`
}

type InspectBatchResponse struct {
	CheckedAt    string               `json:"checked_at"`
	TotalChecked int                  `json:"total_checked"`
	Results      []InspectBatchResult `json:"results"`
}

type DocumentInspectPayload struct {
	FileURL            string `json:"file_url"`
	KnownETag          string `json:"known_etag,omitempty"`
	KnownContentLength int64  `json:"known_content_length,omitempty"`
}

type DocumentInspectResponse struct {
	FileURL        string `json:"file_url"`
	HTTPStatusCode int    `json:"http_status_code"`
	IsModified     bool   `json:"is_modified"`
	ETag           string `json:"etag,omitempty"`
	ContentLength  int64  `json:"content_length,omitempty"`
	ContentType    string `json:"content_type,omitempty"`
	LastModified   string `json:"last_modified,omitempty"`
}

type TaskStatusResponse struct {
	TaskID          string `json:"task_id"`
	TargetID        string `json:"target_id,omitempty"`
	ClientOrigin    string `json:"client_origin,omitempty"`
	SourceType      string `json:"source_type"`
	TargetURL       string `json:"target_url"`
	CallbackURL     string `json:"callback_url,omitempty"`
	Status          string `json:"status"` // PENDING, ACCEPTED, COMPLETED, FAILED
	HTTPStatusCode  int    `json:"http_status_code,omitempty"`
	ExecutionTimeMs int    `json:"execution_time_ms,omitempty"`
	ContentHash     string `json:"content_hash,omitempty"`
	CreatedAt       string `json:"created_at,omitempty"`
	UpdatedAt       string `json:"updated_at,omitempty"`
}
