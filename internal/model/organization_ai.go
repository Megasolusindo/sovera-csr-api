package model

import (
	"time"

	"github.com/google/uuid"
)

// OrganizationAIConversation represents a private user chat thread within an organization
type OrganizationAIConversation struct {
	ID        uuid.UUID `json:"id" db:"id"`
	OrgID     uuid.UUID `json:"org_id" db:"org_id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Title     string    `json:"title" db:"title"`
	Status    string    `json:"status" db:"status"` // "active", "archived"
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// ToolCallInfo records detail of a tool call executed during LLM reasoning
type ToolCallInfo struct {
	ToolName  string                 `json:"tool_name"`
	Arguments map[string]interface{} `json:"arguments"`
	Output    interface{}            `json:"output,omitempty"`
}

// OrganizationAIChatLog records audit and context window message pairs
type OrganizationAIChatLog struct {
	ID             uuid.UUID      `json:"id" db:"id"`
	OrgID          uuid.UUID      `json:"org_id" db:"org_id"`
	UserID         uuid.UUID      `json:"user_id" db:"user_id"`
	ConversationID uuid.UUID      `json:"conversation_id" db:"conversation_id"`
	Message        string         `json:"message" db:"message"`
	Reply          string         `json:"reply" db:"reply"`
	ToolsCalled    []ToolCallInfo `json:"tools_called" db:"tools_called"`
	LatencyMS      int            `json:"latency_ms" db:"latency_ms"`
	ModelName      string         `json:"model_name" db:"model_name"`
	CreatedAt      time.Time      `json:"created_at" db:"created_at"`
}

// OrganizationAIChatRequest DTO for incoming user chat messages
type OrganizationAIChatRequest struct {
	Message        string     `json:"message"`
	ConversationID *uuid.UUID `json:"conversation_id,omitempty"`
}

// OrganizationAIChatResponse DTO for chat assistant response
type OrganizationAIChatResponse struct {
	Reply          string    `json:"reply"`
	ConversationID uuid.UUID `json:"conversation_id"`
	ToolsCalled    []string  `json:"tools_called,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
}

// OrganizationProgramInfo DTO for tool response of organization social/CSR programs
type OrganizationProgramInfo struct {
	ID                  string   `json:"id"`
	OrgID               string   `json:"org_id"`
	Title               string   `json:"title"`
	Description         string   `json:"description"`
	PrimaryCluster      string   `json:"primary_cluster"`
	TargetSDGs          []string `json:"target_sdgs"`
	AsnafCategory       string   `json:"asnaf_category,omitempty"`
	ESGPillar           string   `json:"esg_pillar"`
	TargetBeneficiaries string   `json:"target_beneficiaries"`
	CreatedAt           string   `json:"created_at"`
}

