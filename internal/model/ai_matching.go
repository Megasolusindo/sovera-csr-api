package model

import "time"

// AIMatchingRequest DTO for POST /api/v1/ai/matching
type AIMatchingRequest struct {
	ProgramTitle        string   `json:"program_title"`
	CSRCategory         string   `json:"csr_category"`         // e.g. "Pendidikan", "Kesehatan", "Lingkungan", "UMKM"
	TargetLocation      string   `json:"target_location"`      // e.g. "Jawa Barat", "DKI Jakarta", "National"
	TargetBeneficiaries string   `json:"target_beneficiaries"` // e.g. "Siswa Kurang Mampu", "Petani"
	EstimatedBudget     float64  `json:"estimated_budget"`     // e.g. 500000000
	Keywords            []string `json:"keywords,omitempty"`
	Limit               int      `json:"limit,omitempty"`
}

// CompanyMatchResult represents a ranked corporate prospect match
type CompanyMatchResult struct {
	CompanyID      string   `json:"company_id"`
	CompanyName    string   `json:"company_name"`
	IndustrySector string   `json:"industry_sector"`
	PriorityTier   string   `json:"priority_tier"`
	MatchScore     float64  `json:"match_score"` // 0.0 to 100.0
	MatchGrade     string   `json:"match_grade"` // "HIGHLY_RECOMMENDED", "GOOD_MATCH", "POTENTIAL"
	MatchRationale string   `json:"match_rationale"`
	KeyFocusAreas  []string `json:"key_focus_areas"`
	Website        string   `json:"website,omitempty"`
}

// AIMatchingResponse DTO returned by matching API
type AIMatchingResponse struct {
	ProgramTitle     string               `json:"program_title"`
	CSRCategory      string               `json:"csr_category"`
	MatchesFound     int                  `json:"matches_found"`
	CorporateMatches []CompanyMatchResult `json:"corporate_matches"`
	EvaluatedAt      time.Time            `json:"evaluated_at"`
}
