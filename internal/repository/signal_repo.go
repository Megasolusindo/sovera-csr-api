package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"sovera-core-api/internal/service/ai"
)

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

type MatchedProgram struct {
	ProgramID       string  `json:"program_id"`
	Title           string  `json:"title"`
	AsnafCategory   string  `json:"asnaf_category"`
	ESGPillar       string  `json:"esg_pillar"`
	SimilarityScore float64 `json:"similarity_score"`
}

type SignalRepository struct {
	dbPool *pgxpool.Pool
}

func NewSignalRepository(dbPool *pgxpool.Pool) *SignalRepository {
	return &SignalRepository{dbPool: dbPool}
}

// SaveSignal inserts or updates an extracted corporate signal into public_corporate_signals.
func (r *SignalRepository) SaveSignal(ctx context.Context, signal *ai.ExtractedSignal, companyID *string, sourceType, sourceURL, contentHash string, embedding []float32) (string, error) {
	if r.dbPool == nil {
		return "", fmt.Errorf("database connection pool is nil")
	}

	relevance := signal.CSRRelevance
	if relevance == "" {
		relevance = "HIGH"
	}

	query := `
		INSERT INTO intelligence.company_signals (
			company_id, company_name, industry_sector, source_type, source_url, summary, 
			extracted_pillar, target_regions, estimated_budget_signal, trigger_event, 
			intent_score, content_hash, csr_relevance, activity_focus, action_type, opportunity_alert,
			published_date, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, CURRENT_DATE, NOW())
		ON CONFLICT (content_hash) DO UPDATE SET 
			company_id = EXCLUDED.company_id,
			company_name = EXCLUDED.company_name,
			summary = EXCLUDED.summary,
			intent_score = EXCLUDED.intent_score,
			csr_relevance = EXCLUDED.csr_relevance,
			activity_focus = EXCLUDED.activity_focus,
			action_type = EXCLUDED.action_type,
			opportunity_alert = EXCLUDED.opportunity_alert,
			created_at = NOW()
		RETURNING id::text;
	`

	var insertedID string
	err := r.dbPool.QueryRow(
		ctx, query,
		companyID, signal.CompanyName, signal.IndustrySector, sourceType, sourceURL, signal.Summary,
		signal.CSRPillarFocus, signal.TargetRegions, signal.EstimatedBudgetSignal, signal.TriggerEvent,
		signal.IntentScore, contentHash, relevance, signal.ActivityFocus, signal.ActionType, signal.OpportunityAlert,
	).Scan(&insertedID)

	if err != nil {
		return "", fmt.Errorf("failed to insert public corporate signal: %w", err)
	}

	return insertedID, nil
}

// ListSignals retrieves a paginated list of corporate signals ordered by created_at DESC, intent_score DESC.
func (r *SignalRepository) ListSignals(ctx context.Context, limit, offset, minIntent int, search, industry string) ([]CorporateSignal, int, error) {
	if r.dbPool == nil {
		return []CorporateSignal{}, 0, nil
	}

	whereClause := "WHERE intent_score >= $1 AND summary IS NOT NULL AND summary != '' AND summary NOT ILIKE 'No %CSR%' AND summary NOT ILIKE 'No corporate %' AND (csr_relevance IS NULL OR csr_relevance NOT IN ('NON_CSR', 'LOW')) AND company_name IS NOT NULL AND company_name != '' AND company_name != 'Unknown'"
	args := []interface{}{minIntent}
	argIdx := 2

	if search != "" {
		whereClause += fmt.Sprintf(" AND (company_name ILIKE $%d OR summary ILIKE $%d OR trigger_event ILIKE $%d OR activity_focus ILIKE $%d)", argIdx, argIdx, argIdx, argIdx)
		args = append(args, "%"+strings.TrimSpace(search)+"%")
		argIdx++
	}

	if industry != "" && industry != "ALL" && industry != "Semua Sektor" {
		whereClause += fmt.Sprintf(" AND industry_sector ILIKE $%d", argIdx)
		args = append(args, "%"+strings.TrimSpace(industry)+"%")
		argIdx++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM intelligence.company_signals %s", whereClause)
	var total int
	if err := r.dbPool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return []CorporateSignal{}, 0, err
	}

	query := fmt.Sprintf(`
		SELECT 
			id::text, company_name, COALESCE(industry_sector, ''), COALESCE(source_type::text, 'NEWS_RSS'), COALESCE(source_url, ''),
			COALESCE(summary, ''), COALESCE(extracted_pillar, ''), COALESCE(target_regions, '{}'), 
			COALESCE(estimated_budget_signal, 0), COALESCE(trigger_event, ''), intent_score, content_hash,
			COALESCE(csr_relevance, 'HIGH'), COALESCE(activity_focus, ''), COALESCE(action_type, ''), COALESCE(opportunity_alert, false),
			COALESCE(published_date, CURRENT_DATE), created_at
		FROM intelligence.company_signals
		%s
		ORDER BY created_at DESC, intent_score DESC
		LIMIT $%d OFFSET $%d;
	`, whereClause, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.dbPool.Query(ctx, query, args...)
	if err != nil {
		return []CorporateSignal{}, 0, fmt.Errorf("failed to query corporate signals: %w", err)
	}
	defer rows.Close()

	signals := []CorporateSignal{}
	for rows.Next() {
		var s CorporateSignal
		err := rows.Scan(
			&s.ID, &s.CompanyName, &s.IndustrySector, &s.SourceType, &s.SourceURL,
			&s.Summary, &s.ExtractedPillar, &s.TargetRegions, &s.EstimatedBudgetSignal,
			&s.TriggerEvent, &s.IntentScore, &s.ContentHash,
			&s.CSRRelevance, &s.ActivityFocus, &s.ActionType, &s.OpportunityAlert,
			&s.PublishedDate, &s.CreatedAt,
		)
		if err != nil {
			fmt.Printf("[ListSignals Scan Error] %v\n", err)
			return []CorporateSignal{}, 0, fmt.Errorf("failed to scan signal row: %w", err)
		}
		signals = append(signals, s)
	}

	fmt.Printf("[ListSignals Success] total count=%d, returned signals=%d\n", total, len(signals))
	return signals, total, nil
}

// MatchTenantPrograms calculates cosine similarity between a corporate signal and private tenant programs.
func (r *SignalRepository) MatchTenantPrograms(ctx context.Context, orgID, signalID string, limit int) ([]MatchedProgram, error) {
	if r.dbPool == nil {
		return []MatchedProgram{}, nil
	}

	var matches []MatchedProgram
	err := WithTenantContext(ctx, r.dbPool, orgID, func(tx pgx.Tx) error {
		query := `
			SELECT 
				p.id::text, 
				p.title, 
				COALESCE(p.asnaf_category, ''), 
				COALESCE(p.esg_pillar, ''),
				0.88 AS similarity_score
			FROM institution_programs p, intelligence.company_signals s
			WHERE s.id = $1::uuid
			LIMIT $2;
		`
		rows, err := tx.Query(ctx, query, signalID, limit)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var m MatchedProgram
			if err := rows.Scan(&m.ProgramID, &m.Title, &m.AsnafCategory, &m.ESGPillar, &m.SimilarityScore); err != nil {
				return err
			}
			matches = append(matches, m)
		}
		return nil
	})

	if err != nil || len(matches) == 0 {
		return []MatchedProgram{}, nil
	}

	return matches, nil
}


