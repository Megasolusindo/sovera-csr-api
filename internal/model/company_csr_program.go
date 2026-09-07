package model

import "time"

type CompanyCSRProgram struct {
	ID            string     `json:"id"`
	CompanyID     string     `json:"company_id"`
	Name          string     `json:"name"`
	Description   *string    `json:"description,omitempty"`
	ProgramType   *string    `json:"program_type,omitempty"`
	StartDate     *time.Time `json:"start_date,omitempty"`
	EndDate       *time.Time `json:"end_date,omitempty"`
	Status        string     `json:"status"`
	BudgetAmount  *float64   `json:"budget_amount,omitempty"`
	PartnerNGO    *string    `json:"partner_ngo,omitempty"`
	ImpactSummary *string    `json:"impact_summary,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`

	Focuses []CSRFocus `json:"focuses,omitempty"`
}

type CompanyCSRProgramFocus struct {
	ProgramID string    `json:"program_id"`
	FocusID   string    `json:"focus_id"`
	CreatedAt time.Time `json:"created_at"`
}
