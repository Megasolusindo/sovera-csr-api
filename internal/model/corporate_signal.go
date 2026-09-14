package model

import "time"

// CorporateSignal represents extracted corporate intelligence signals
type CorporateSignal struct {
	ID                    string    `json:"id"`
	CompanyName           string    `json:"company_name"`
	IndustrySector        string    `json:"industry_sector"`
	SourceType            string    `json:"source_type"`
	SourceURL             string    `json:"source_url"`
	Summary               string    `json:"summary"`
	ExtractedPillar       string    `json:"extracted_pillar"`
	TargetRegions         []string  `json:"target_regions"`
	EstimatedBudgetSignal float64   `json:"estimated_budget_signal"`
	TriggerEvent          string    `json:"trigger_event"`
	IntentScore           int       `json:"intent_score"`
	ContentHash           string    `json:"content_hash"`
	CSRRelevance          string    `json:"csr_relevance"`
	ActivityFocus         string    `json:"activity_focus"`
	ActionType            string    `json:"action_type"`
	OpportunityAlert      bool      `json:"opportunity_alert"`
	PublishedDate         time.Time `json:"published_date"`
	CreatedAt             time.Time `json:"created_at"`
}
