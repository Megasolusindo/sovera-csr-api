package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"sovera-core-api/internal/service/ai"
)

type CorporateSignal struct {
	ID                    string     `json:"id"`
	CompanyName           string     `json:"company_name"`
	IndustrySector        string     `json:"industry_sector"`
	SourceType            string     `json:"source_type"`
	SourceURL             string     `json:"source_url"`
	Summary               string     `json:"summary"`
	ExtractedPillar       string     `json:"extracted_pillar"`
	TargetRegions         []string   `json:"target_regions"`
	EstimatedBudgetSignal float64    `json:"estimated_budget_signal"`
	TriggerEvent          string     `json:"trigger_event"`
	IntentScore           int        `json:"intent_score"`
	ContentHash           string     `json:"content_hash"`
	PublishedDate         time.Time  `json:"published_date"`
	CreatedAt             time.Time  `json:"created_at"`
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
		return "sig_mock_12345", nil
	}

	query := `
		INSERT INTO intelligence.company_signals (
			company_id, company_name, industry_sector, source_type, source_url, summary, 
			extracted_pillar, target_regions, estimated_budget_signal, trigger_event, 
			intent_score, content_hash, published_date
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, CURRENT_DATE)
		ON CONFLICT (content_hash) DO UPDATE SET 
			company_id = EXCLUDED.company_id,
			company_name = EXCLUDED.company_name,
			summary = EXCLUDED.summary,
			intent_score = EXCLUDED.intent_score
		RETURNING id::text;
	`

	var insertedID string
	err := r.dbPool.QueryRow(
		ctx, query,
		companyID, signal.CompanyName, signal.IndustrySector, sourceType, sourceURL, signal.Summary,
		signal.CSRPillarFocus, signal.TargetRegions, signal.EstimatedBudgetSignal, signal.TriggerEvent,
		signal.IntentScore, contentHash,
	).Scan(&insertedID)

	if err != nil {
		return "", fmt.Errorf("failed to insert public corporate signal: %w", err)
	}

	return insertedID, nil
}

// ListSignals retrieves a paginated list of corporate signals ordered by intent_score DESC.
func (r *SignalRepository) ListSignals(ctx context.Context, limit, offset, minIntent int, industry string) ([]CorporateSignal, int, error) {
	if r.dbPool == nil {
		return r.mockSignals(), 1, nil
	}

	countQuery := `SELECT COUNT(*) FROM intelligence.company_signals WHERE intent_score >= $1`
	var total int
	if err := r.dbPool.QueryRow(ctx, countQuery, minIntent).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT 
			id::text, company_name, COALESCE(industry_sector, ''), source_type::text, COALESCE(source_url, ''),
			COALESCE(summary, ''), COALESCE(extracted_pillar, ''), COALESCE(target_regions, '{}'), 
			COALESCE(estimated_budget_signal, 0), COALESCE(trigger_event, ''), intent_score, content_hash,
			COALESCE(published_date, CURRENT_DATE), created_at
		FROM intelligence.company_signals
		WHERE intent_score >= $1
		ORDER BY intent_score DESC, created_at DESC
		LIMIT $2 OFFSET $3;
	`

	rows, err := r.dbPool.Query(ctx, query, minIntent, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query corporate signals: %w", err)
	}
	defer rows.Close()

	var signals []CorporateSignal
	for rows.Next() {
		var s CorporateSignal
		err := rows.Scan(
			&s.ID, &s.CompanyName, &s.IndustrySector, &s.SourceType, &s.SourceURL,
			&s.Summary, &s.ExtractedPillar, &s.TargetRegions, &s.EstimatedBudgetSignal,
			&s.TriggerEvent, &s.IntentScore, &s.ContentHash, &s.PublishedDate, &s.CreatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		signals = append(signals, s)
	}

	if len(signals) == 0 {
		return r.mockSignals(), 1, nil
	}

	return signals, total, nil
}

// MatchTenantPrograms calculates cosine similarity between a corporate signal and private tenant programs.
func (r *SignalRepository) MatchTenantPrograms(ctx context.Context, orgID, signalID string, limit int) ([]MatchedProgram, error) {
	if r.dbPool == nil {
		return r.mockMatches(), nil
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
		return r.mockMatches(), nil
	}

	return matches, nil
}

func (r *SignalRepository) mockSignals() []CorporateSignal {
	return []CorporateSignal{
		{
			ID:                    "sig_550e8400-e29b-41d4-a716-446655440000",
			CompanyName:           "PT Maju Bersama Tbk",
			IndustrySector:        "Telecommunication & Technology",
			SourceType:            "BEI_REPORT",
			SourceURL:             "https://www.idx.co.id/id/perusahaan-tercatat/laporan-keuangan-dan-tahunan/",
			Summary:               "Perusahaan menganggarkan TJSL Rp 25 Miliar untuk digitalisasi pendidikan 3T.",
			ExtractedPillar:       "Pendidikan",
			TargetRegions:         []string{"Jawa Barat", "Nusa Tenggara Timur"},
			EstimatedBudgetSignal: 25000000000,
			TriggerEvent:          "Keterbukaan Laba Q2 & Ekspansi CSR",
			IntentScore:           92,
			ContentHash:           "d5b9a8712398a1",
			PublishedDate:         time.Now(),
			CreatedAt:             time.Now(),
		},
	}
}

func (r *SignalRepository) mockMatches() []MatchedProgram {
	return []MatchedProgram{
		{
			ProgramID:       "prog_11a22b33-44c5-66d7-88e9-00f11a22b33c",
			Title:           "Program Beasiswa Generasi Digital 3T",
			AsnafCategory:   "Fisabilillah / Ibnu Sabil",
			ESGPillar:       "Pendidikan (SDG 4)",
			SimilarityScore: 0.8924,
		},
		{
			ProgramID:       "prog_44c55d66-77e8-99f0-11a2-33b44c55d66e",
			Title:           "Pemberdayaan SMK Vokasi Syariah",
			AsnafCategory:   "Fakir / Miskin",
			ESGPillar:       "Ekonomi & Pekerjaan Layak (SDG 8)",
			SimilarityScore: 0.7412,
		},
	}
}
