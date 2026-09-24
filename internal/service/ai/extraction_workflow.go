package ai

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

type ExtractedClaim struct {
	SourceID        string    `json:"source_id"`
	CompanyName     string    `json:"company_name"`
	ClaimType       string    `json:"claim_type"` // e.g. FUNDING_SIGNAL, PROGRAM_LAUNCH, POLICY_CHANGE
	Title           string    `json:"title"`
	Summary         string    `json:"summary"`
	Quote           string    `json:"quote"`
	EstimatedBudget float64   `json:"estimated_budget"`
	ProgramYear     int       `json:"program_year"`
	IdempotencyKey  string    `json:"idempotency_key"`
	ExtractedAt     time.Time `json:"extracted_at"`
}

// ComputeFlexibleIdempotencyKey computes sha256(source_id + task_type + pipeline_version + prompt_version) per §18
func ComputeFlexibleIdempotencyKey(sourceID, taskType, pipelineVersion, promptVersion string) string {
	raw := fmt.Sprintf("%s:%s:%s:%s",
		strings.TrimSpace(sourceID),
		strings.TrimSpace(taskType),
		strings.TrimSpace(pipelineVersion),
		strings.TrimSpace(promptVersion),
	)
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}

type ExtractionWorkflow struct {
	pipelineVersion string
	promptVersion   string
}

func NewExtractionWorkflow(pipelineVersion, promptVersion string) *ExtractionWorkflow {
	if pipelineVersion == "" {
		pipelineVersion = "v1.2.0"
	}
	if promptVersion == "" {
		promptVersion = "p1.0"
	}
	return &ExtractionWorkflow{
		pipelineVersion: pipelineVersion,
		promptVersion:   promptVersion,
	}
}

// ProcessExtraction constructs extracted claim with flexible idempotency key
func (ew *ExtractionWorkflow) ProcessExtraction(sourceID, companyName, claimType, title, summary, quote string, budget float64, year int) *ExtractedClaim {
	taskType := "CLAIM_EXTRACTION_" + strings.ToUpper(claimType)
	idempotencyKey := ComputeFlexibleIdempotencyKey(sourceID, taskType, ew.pipelineVersion, ew.promptVersion)

	return &ExtractedClaim{
		SourceID:        sourceID,
		CompanyName:     companyName,
		ClaimType:       claimType,
		Title:           title,
		Summary:         summary,
		Quote:           quote,
		EstimatedBudget: budget,
		ProgramYear:     year,
		IdempotencyKey:  idempotencyKey,
		ExtractedAt:     time.Now().UTC(),
	}
}
