package ai

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	ErrMaxStepsExceeded = errors.New("research agent sandbox: hard cap max 5 reasoning steps exceeded")
	ErrMaxToolsExceeded = errors.New("research agent sandbox: hard cap max 10 tool calls exceeded")
	ErrTenantAccessDeny = errors.New("research agent sandbox: access to tenant context or secrets strictly forbidden")
)

const (
	MaxReasoningSteps = 5
	MaxToolCalls      = 10
)

type ResearchFindingPayload struct {
	SourceURL   string    `json:"source_url"`
	RawContent  string    `json:"raw_content"`
	DataStatus  string    `json:"data_status"` // Always "UNTRUSTED_DATA"
	StepsUsed   int       `json:"steps_used"`
	ToolsUsed   int       `json:"tools_used"`
	Discovered  time.Time `json:"discovered_at"`
}

type ResearchAgent struct {
	fetcher     *SecureFetcher
	costMonitor *CostMonitor
}

func NewResearchAgent(fetcher *SecureFetcher, costMonitor *CostMonitor) *ResearchAgent {
	if fetcher == nil {
		fetcher = NewSecureFetcher(15 * time.Second)
	}
	return &ResearchAgent{
		fetcher:     fetcher,
		costMonitor: costMonitor,
	}
}

// RunGlobalResearch executes a research task in full GLOBAL scope isolation
func (ra *ResearchAgent) RunGlobalResearch(ctx context.Context, stc *SignedTaskContext, targetURL string) (*ResearchFindingPayload, error) {
	// 1. Hard Isolation Check: Scope must be ScopeGlobal
	if stc.ScopeType != ScopeGlobal {
		return nil, fmt.Errorf("%w: scope must be ScopeGlobal, got %s", ErrTenantAccessDeny, stc.ScopeType)
	}
	if stc.OrgID != "" {
		return nil, fmt.Errorf("%w: orgID must be empty for global research agent", ErrTenantAccessDeny)
	}

	// 2. Track Circuit Breaker cost
	if ra.costMonitor != nil {
		if err := ra.costMonitor.CheckLimit(stc.TaskID, ""); err != nil {
			return nil, err
		}
	}

	stepCount := 0
	toolCount := 0

	// Step 1: Initialize reasoning
	stepCount++
	if stepCount > MaxReasoningSteps {
		return nil, ErrMaxStepsExceeded
	}

	// Step 2: Execute Web Fetcher Tool
	toolCount++
	if toolCount > MaxToolCalls {
		return nil, ErrMaxToolsExceeded
	}

	content, err := ra.fetcher.FetchUntrustedContent(ctx, targetURL)
	if err != nil {
		return nil, fmt.Errorf("research fetch tool failed: %w", err)
	}

	stepCount++

	// Record estimated cost ($0.005 per fetch)
	if ra.costMonitor != nil {
		_ = ra.costMonitor.TrackUsage(stc.TaskID, "", 0.005)
	}

	return &ResearchFindingPayload{
		SourceURL:  targetURL,
		RawContent: content,
		DataStatus: "UNTRUSTED_DATA", // Mark untrusted data explicitly
		StepsUsed:  stepCount,
		ToolsUsed:  toolCount,
		Discovered: time.Now().UTC(),
	}, nil
}
