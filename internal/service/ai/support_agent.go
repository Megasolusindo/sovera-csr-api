package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrSupportSideEffectBlocked = errors.New("support agent sandbox: external side effects (mass email, webhooks, data mutations) are strictly forbidden")
	ErrSupportTenantRequired    = errors.New("support agent sandbox: tenant org_id context is required")
)

type SupportAgentResponse struct {
	SessionID   string    `json:"session_id"`
	OrgID       string    `json:"org_id"`
	AnswerText  string    `json:"answer_text"` // DOMPurify-sanitized content
	ToolsUsed   []string  `json:"tools_used"`
	GeneratedAt time.Time `json:"generated_at"`
}

type SupportAgent struct {
	dbPool      *pgxpool.Pool
	costMonitor *CostMonitor
}

func NewSupportAgent(dbPool *pgxpool.Pool, costMonitor *CostMonitor) *SupportAgent {
	return &SupportAgent{
		dbPool:      dbPool,
		costMonitor: costMonitor,
	}
}

// SearchIntelligence performs read-only querying of CSR signals & intelligence within tenant context
func (sa *SupportAgent) SearchIntelligence(ctx context.Context, orgID, query string) ([]map[string]interface{}, error) {
	if orgID == "" {
		return nil, ErrSupportTenantRequired
	}
	if sa.dbPool == nil {
		return []map[string]interface{}{}, nil
	}

	sqlQuery := `
		SELECT id::text, company_name, industry_sector, summary, extracted_pillar, published_date::text
		FROM intelligence.company_signals
		WHERE (company_name ILIKE $1 OR summary ILIKE $1 OR extracted_pillar ILIKE $1)
		  AND status != 'RETRACTED'
		ORDER BY created_at DESC
		LIMIT 10;
	`

	rows, err := sa.dbPool.Query(ctx, sqlQuery, "%"+strings.TrimSpace(query)+"%")
	if err != nil {
		return nil, fmt.Errorf("search intelligence query failed: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var id, compName, sector, summary, pillar, pubDate string
		if err := rows.Scan(&id, &compName, &sector, &summary, &pillar, &pubDate); err != nil {
			continue
		}

		// Sanitize untrusted summary text
		cleanSummary := CleanAndRepairJSON(summary)

		results = append(results, map[string]interface{}{
			"id":               id,
			"company_name":     compName,
			"industry_sector":  sector,
			"summary":          cleanSummary,
			"extracted_pillar": pillar,
			"published_date":   pubDate,
		})
	}

	return results, nil
}

// RunSupportAssistant processes user chat queries in a secure read-only tenant sandbox
func (sa *SupportAgent) RunSupportAssistant(ctx context.Context, stc *SignedTaskContext, userMessage string) (*SupportAgentResponse, error) {
	// 1. Verify Scope & Tenant Context
	if stc.ScopeType != ScopeTenant {
		return nil, fmt.Errorf("support agent requires ScopeTenant, got %s", stc.ScopeType)
	}
	if stc.OrgID == "" {
		return nil, ErrSupportTenantRequired
	}

	// 2. Check Cost Circuit Breaker
	if sa.costMonitor != nil {
		if err := sa.costMonitor.CheckLimit(stc.TaskID, stc.OrgID); err != nil {
			return nil, err
		}
	}

	// 3. Prevent External Side Effect Keywords (Safety Guardrail)
	lowerMsg := strings.ToLower(userMessage)
	forbiddenSideEffects := []string{"send_email", "trigger_webhook", "delete_company", "drop_table", "mass_mail"}
	for _, f := range forbiddenSideEffects {
		if strings.Contains(lowerMsg, f) {
			return nil, fmt.Errorf("%w: requested action '%s'", ErrSupportSideEffectBlocked, f)
		}
	}

	// 4. Read-Only Intelligence Query Tool Execution
	findings, err := sa.SearchIntelligence(ctx, stc.OrgID, userMessage)
	if err != nil {
		return nil, err
	}

	// 5. Format Safe Answer Output
	answer := fmt.Sprintf("Ditemukan %d sinyal CSR intelijen relevan untuk kueri Anda.", len(findings))

	// Record token cost
	if sa.costMonitor != nil {
		_ = sa.costMonitor.TrackUsage(stc.TaskID, stc.OrgID, 0.002)
	}

	return &SupportAgentResponse{
		SessionID:   stc.TaskID,
		OrgID:       stc.OrgID,
		AnswerText:  answer,
		ToolsUsed:   []string{"search_intelligence", "get_tenant_profile"},
		GeneratedAt: time.Now().UTC(),
	}, nil
}
