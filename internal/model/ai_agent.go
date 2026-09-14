package model

import (
	"time"

	"github.com/google/uuid"
)

// AIAgentCredential represents authenticated OpenClaw AI Agents
type AIAgentCredential struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	AgentName  string     `json:"agent_name" db:"agent_name"`
	APIKeyHash string     `json:"-" db:"api_key_hash"`
	Scopes     []string   `json:"scopes" db:"scopes"`
	IsActive   bool       `json:"is_active" db:"is_active"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty" db:"last_used_at"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
}

// AIAuditLog records immutable AI agent actions for accountability
type AIAuditLog struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	AgentName  string     `json:"agent_name" db:"agent_name"`
	Action     string     `json:"action" db:"action"`
	Endpoint   string     `json:"endpoint" db:"endpoint"`
	RequestID  string     `json:"request_id,omitempty" db:"request_id"`
	TargetType string     `json:"target_type,omitempty" db:"target_type"`
	TargetID   string     `json:"target_id,omitempty" db:"target_id"`
	Payload    interface{} `json:"payload,omitempty" db:"payload"`
	Status     string     `json:"status" db:"status"`
	ClientIP   string     `json:"client_ip,omitempty" db:"client_ip"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
}

// AIResearchFinding represents a CSR finding submitted by OpenClaw entering review queue
type AIResearchFinding struct {
	ID              uuid.UUID              `json:"id" db:"id"`
	CompanyID       *uuid.UUID             `json:"company_id,omitempty" db:"company_id"`
	CompanyName     string                 `json:"company_name,omitempty" db:"company_name"`
	FindingType     string                 `json:"finding_type" db:"finding_type"` // e.g., "csr_program", "csr_policy", "partnership"
	Title           string                 `json:"title" db:"title"`
	Summary         string                 `json:"summary,omitempty" db:"summary"`
	SourceURL       string                 `json:"source_url" db:"source_url"`
	SourceName      string                 `json:"source_name" db:"source_name"`
	SourceType      string                 `json:"source_type" db:"source_type"` // e.g., "news", "company_website", "report"
	PublishedAt     *time.Time             `json:"published_at,omitempty" db:"published_at"`
	DiscoveredAt    time.Time              `json:"discovered_at" db:"discovered_at"`
	ConfidenceScore float64                `json:"confidence_score" db:"confidence_score"` // 0.0 to 1.0
	EvidenceData    map[string]interface{} `json:"evidence_data,omitempty" db:"evidence_data"`
	Status          string                 `json:"status" db:"status"` // "pending_review", "approved", "rejected"
	IdempotencyKey  string                 `json:"idempotency_key,omitempty" db:"idempotency_key"`
	ReviewedBy      *uuid.UUID             `json:"reviewed_by,omitempty" db:"reviewed_by"`
	ReviewedAt      *time.Time             `json:"reviewed_at,omitempty" db:"reviewed_at"`
	ReviewNotes     string                 `json:"review_notes,omitempty" db:"review_notes"`
	CreatedAt       time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at" db:"updated_at"`
}

// SubmitFindingRequest DTO for POST /api/v1/ai/research/findings
type SubmitFindingRequest struct {
	CompanyID       *string                `json:"company_id,omitempty"`
	CompanyName     string                 `json:"company_name,omitempty"`
	FindingType     string                 `json:"finding_type"`
	Title           string                 `json:"title"`
	Summary         string                 `json:"summary"`
	SourceURL       string                 `json:"source_url"`
	SourceName      string                 `json:"source_name"`
	SourceType      string                 `json:"source_type"`
	PublishedAt     *string                `json:"published_at,omitempty"`
	ConfidenceScore float64                `json:"confidence_score"`
	EvidenceData    map[string]interface{} `json:"evidence_data,omitempty"`
	IdempotencyKey  string                 `json:"idempotency_key,omitempty"`
}

// ReviewFindingRequest DTO for POST /api/v1/admin/ai/findings/:id/review
type ReviewFindingRequest struct {
	Action      string `json:"action"` // "approve" or "reject"
	ReviewNotes string `json:"review_notes,omitempty"`
}
