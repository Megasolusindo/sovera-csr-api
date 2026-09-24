package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"sovera-core-api/internal/model"
)

type AIAgentRepository struct {
	dbPool *pgxpool.Pool
}

func NewAIAgentRepository(dbPool *pgxpool.Pool) *AIAgentRepository {
	return &AIAgentRepository{dbPool: dbPool}
}

func HashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// ValidateToken checks if the provided Bearer token is valid, active, and unexpired.
func (r *AIAgentRepository) ValidateToken(ctx context.Context, rawToken string) (*model.AIAgentCredential, error) {
	if r.dbPool == nil {
		return nil, fmt.Errorf("database pool is not initialized")
	}

	keyHash := HashAPIKey(rawToken)

	query := `
		SELECT id, agent_name, api_key_hash, scopes, is_active, expires_at, last_used_at, created_at, updated_at
		FROM ai_agent_credentials
		WHERE api_key_hash = $1 AND is_active = true;
	`

	var agent model.AIAgentCredential
	err := r.dbPool.QueryRow(ctx, query, keyHash).Scan(
		&agent.ID,
		&agent.AgentName,
		&agent.APIKeyHash,
		&agent.Scopes,
		&agent.IsActive,
		&agent.ExpiresAt,
		&agent.LastUsedAt,
		&agent.CreatedAt,
		&agent.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("invalid or inactive AI agent credentials")
		}
		return nil, fmt.Errorf("database query error: %w", err)
	}

	if agent.ExpiresAt != nil && time.Now().After(*agent.ExpiresAt) {
		return nil, fmt.Errorf("AI agent credentials expired")
	}

	// Update last_used_at timestamp asynchronously
	go func() {
		updateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = r.dbPool.Exec(updateCtx, `UPDATE ai_agent_credentials SET last_used_at = NOW() WHERE id = $1;`, agent.ID)
	}()

	return &agent, nil
}

// CreateAuditLog records immutable audit log for AI actions
func (r *AIAgentRepository) CreateAuditLog(ctx context.Context, audit model.AIAuditLog) error {
	if r.dbPool == nil {
		return nil
	}

	payloadJSON, _ := json.Marshal(audit.Payload)

	query := `
		INSERT INTO ai_audit_logs (agent_name, action, endpoint, request_id, target_type, target_id, payload, status, client_ip, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW());
	`

	_, err := r.dbPool.Exec(ctx, query,
		audit.AgentName,
		audit.Action,
		audit.Endpoint,
		audit.RequestID,
		audit.TargetType,
		audit.TargetID,
		payloadJSON,
		audit.Status,
		audit.ClientIP,
	)

	if err != nil {
		log.Printf("Warning: Failed to write AI audit log: %v", err)
	}

	return err
}

// SubmitFinding records a research finding from OpenClaw into the review queue
func (r *AIAgentRepository) SubmitFinding(ctx context.Context, finding model.AIResearchFinding) (*model.AIResearchFinding, error) {
	if r.dbPool == nil {
		return nil, fmt.Errorf("database pool not initialized")
	}

	evidenceJSON, _ := json.Marshal(finding.EvidenceData)

	query := `
		INSERT INTO ai_research_findings (
			company_id, company_name, finding_type, title, summary,
			source_url, source_name, source_type, published_at, discovered_at,
			confidence_score, evidence_data, status, idempotency_key, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, NOW(),
			$10, $11, 'pending_review', $12, NOW(), NOW()
		)
		ON CONFLICT (idempotency_key) DO UPDATE
		SET updated_at = NOW()
		RETURNING id, company_id, company_name, finding_type, title, summary, source_url, source_name, source_type, published_at, discovered_at, confidence_score, evidence_data, status, idempotency_key, created_at, updated_at;
	`

	var result model.AIResearchFinding
	var evidenceBytes []byte

	err := r.dbPool.QueryRow(ctx, query,
		finding.CompanyID,
		finding.CompanyName,
		finding.FindingType,
		finding.Title,
		finding.Summary,
		finding.SourceURL,
		finding.SourceName,
		finding.SourceType,
		finding.PublishedAt,
		finding.ConfidenceScore,
		evidenceJSON,
		finding.IdempotencyKey,
	).Scan(
		&result.ID,
		&result.CompanyID,
		&result.CompanyName,
		&result.FindingType,
		&result.Title,
		&result.Summary,
		&result.SourceURL,
		&result.SourceName,
		&result.SourceType,
		&result.PublishedAt,
		&result.DiscoveredAt,
		&result.ConfidenceScore,
		&evidenceBytes,
		&result.Status,
		&result.IdempotencyKey,
		&result.CreatedAt,
		&result.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to insert research finding: %w", err)
	}

	if len(evidenceBytes) > 0 {
		_ = json.Unmarshal(evidenceBytes, &result.EvidenceData)
	}

	// Auto-mirror finding to intelligence.company_signals for live /signals dashboard display
	r.mirrorFindingToSignal(ctx, result)

	return &result, nil
}

// ParseEstimatedBudgetFromText extracts monetary values (in IDR) from raw text (e.g. "Rp 15 Miliar", "25 Milyar", "500 Juta", "Rp 15.000.000.000")
func ParseEstimatedBudgetFromText(text string) float64 {
	cleanText := strings.TrimSpace(text)
	if cleanText == "" {
		return 0
	}

	// Pattern 1: Multiplier words ("15 Miliar", "Rp 25 Milyar", "500 Juta", "Rp 2,5 M", "USD 10 Million")
	reMultiplier := regexp.MustCompile(`(?i)(?:rp\.?|\$|usd)?\s*([\d]+(?:[\.,][\d]+)?)\s*(triliun|trillion|miliar|milyar|billion|juta|jt|million|m|t|b)\b`)
	matches := reMultiplier.FindAllStringSubmatch(cleanText, -1)
	for _, m := range matches {
		if len(m) >= 3 {
			rawNumStr := strings.ReplaceAll(m[1], ",", ".")
			num, err := strconv.ParseFloat(rawNumStr, 64)
			if err == nil && num > 0 {
				unitStr := strings.ToLower(m[2])
				multiplier := 1.0
				switch unitStr {
				case "triliun", "trillion", "t":
					multiplier = 1e12
				case "miliar", "milyar", "billion", "m":
					multiplier = 1e9
				case "juta", "jt", "million":
					multiplier = 1e6
				}
				result := num * multiplier
				if result >= 100000 && result <= 1e14 {
					return result
				}
			}
		}
	}

	// Pattern 2: Explicit formatted full numbers e.g. "Rp 15.000.000.000" or "Rp 15,000,000,000"
	reFullDigits := regexp.MustCompile(`(?i)(?:rp\.?|\$|usd)\s*([\d\.\,]{7,})`)
	digitsMatches := reFullDigits.FindAllStringSubmatch(cleanText, -1)
	for _, m := range digitsMatches {
		if len(m) >= 2 {
			rawNumStr := m[1]
			cleanDigits := strings.ReplaceAll(rawNumStr, ".", "")
			cleanDigits = strings.ReplaceAll(cleanDigits, ",", "")
			num, err := strconv.ParseFloat(cleanDigits, 64)
			if err == nil && num >= 100000 && num <= 1e14 {
				return num
			}
		}
	}

	return 0
}

// mirrorFindingToSignal mirrors an AI research finding into intelligence.company_signals with string truncation safeguards and budget extraction
func (r *AIAgentRepository) mirrorFindingToSignal(ctx context.Context, f model.AIResearchFinding) {
	if r.dbPool == nil {
		return
	}

	companyID := f.CompanyID
	cleanCompName := strings.TrimSpace(f.CompanyName)
	if cleanCompName == "" || cleanCompName == "Unknown" {
		cleanCompName = "Perusahaan Indonesia"
	}

	if companyID == nil && cleanCompName != "" && cleanCompName != "Perusahaan Indonesia" {
		var resolvedID uuid.UUID
		cleanKeyword := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(cleanCompName), "pt ", ""), "tbk", ""))
		err := r.dbPool.QueryRow(ctx, `
			SELECT id FROM company.companies 
			WHERE LOWER(TRIM(name)) = LOWER(TRIM($1)) 
			   OR name ILIKE $2 
			   OR slug = $3 
			LIMIT 1
		`, cleanCompName, "%"+cleanKeyword+"%", cleanKeyword).Scan(&resolvedID)
		if err == nil {
			companyID = &resolvedID
		}
	}

	sourceType := f.SourceType
	if sourceType == "" {
		sourceType = "OPENCLAW_AI"
	}
	if len(sourceType) > 50 {
		sourceType = sourceType[:50]
	}

	summary := strings.TrimSpace(f.Summary)
	if summary == "" {
		summary = strings.TrimSpace(f.Title)
	}

	pillar := strings.TrimSpace(f.FindingType)
	if pillar == "" {
		pillar = "Pendidikan & Keberlanjutan"
	}
	if len(pillar) > 100 {
		pillar = pillar[:100]
	}

	activityFocus := strings.TrimSpace(f.Title)
	if activityFocus == "" {
		activityFocus = "Inisiatif Program CSR"
	}
	if len(activityFocus) > 100 {
		activityFocus = activityFocus[:100]
	}

	intentScore := int(f.ConfidenceScore * 100)
	if intentScore < 50 {
		intentScore = 85
	}

	rawHashInput := f.SourceURL + "_" + f.Title
	if strings.TrimSpace(rawHashInput) == "_" {
		rawHashInput = f.IdempotencyKey
	}
	hashBytes := sha256.Sum256([]byte(rawHashInput))
	contentHash := hex.EncodeToString(hashBytes[:16])

	pubDate := time.Now()
	if f.PublishedAt != nil && !f.PublishedAt.IsZero() {
		pubDate = *f.PublishedAt
	}

	compNameTrunc := cleanCompName
	if len(compNameTrunc) > 255 {
		compNameTrunc = compNameTrunc[:255]
	}

	industrySector := strings.TrimSpace(f.SourceName)
	if industrySector == "" {
		industrySector = "Sosial & Lingkungan"
	}
	if len(industrySector) > 255 {
		industrySector = industrySector[:255]
	}

	estimatedBudget := 0.0
	if evBudget, ok := f.EvidenceData["estimated_budget"].(float64); ok && evBudget > 0 {
		estimatedBudget = evBudget
	} else if evBudgetSignal, ok := f.EvidenceData["estimated_budget_signal"].(float64); ok && evBudgetSignal > 0 {
		estimatedBudget = evBudgetSignal
	} else {
		estimatedBudget = ParseEstimatedBudgetFromText(summary + " " + f.Title)
	}

	query := `
		WITH val AS (
			SELECT 
				$1::uuid AS company_id, 
				$2::varchar(255) AS company_name, 
				$3::varchar(255) AS industry_sector, 
				$4::varchar(50) AS source_type, 
				$5::text AS source_url, 
				$6::text AS summary, 
				$7::varchar(100) AS extracted_pillar, 
				$8::text AS trigger_event, 
				$9::integer AS intent_score, 
				$10::varchar(100) AS content_hash, 
				$11::varchar(100) AS activity_focus, 
				$12::date AS published_date,
				$13::numeric(18,2) AS estimated_budget_signal
		)
		INSERT INTO intelligence.company_signals (
			company_id, company_name, industry_sector, source_type, source_url, 
			summary, extracted_pillar, target_regions, estimated_budget_signal, 
			trigger_event, intent_score, content_hash, csr_relevance, activity_focus, 
			action_type, opportunity_alert, published_date, created_at
		) 
		SELECT 
			val.company_id, val.company_name, val.industry_sector, val.source_type, val.source_url,
			val.summary, val.extracted_pillar, ARRAY['Nasional'], val.estimated_budget_signal,
			val.trigger_event, val.intent_score, val.content_hash, 'HIGH', val.activity_focus,
			'PROGRAM_LAUNCH', true, val.published_date, NOW()
		FROM val
		WHERE NOT EXISTS (
			SELECT 1 FROM intelligence.company_signals cs
			WHERE (cs.source_url = val.source_url AND val.source_url IS NOT NULL AND val.source_url != '') 
			   OR cs.content_hash = val.content_hash
		);
	`

	_, err := r.dbPool.Exec(ctx, query,
		companyID,
		compNameTrunc,
		industrySector,
		sourceType,
		f.SourceURL,
		summary,
		pillar,
		f.Title,
		intentScore,
		contentHash,
		activityFocus,
		pubDate,
		estimatedBudget,
	)

	if err != nil {
		log.Printf("Warning: Failed to mirror AI research finding to company_signals: %v", err)
	} else if estimatedBudget > 0 {
		_, _ = r.dbPool.Exec(ctx, `
			UPDATE intelligence.company_signals
			SET estimated_budget_signal = $1
			WHERE ((source_url = $2 AND source_url IS NOT NULL AND source_url != '') OR content_hash = $3)
			  AND (estimated_budget_signal IS NULL OR estimated_budget_signal = 0);
		`, estimatedBudget, f.SourceURL, contentHash)
	}
}

// ListFindings queries research findings with filtering for Admin Review Console
func (r *AIAgentRepository) ListFindings(ctx context.Context, status string, limit, offset int) ([]model.AIResearchFinding, int, error) {
	if r.dbPool == nil {
		return []model.AIResearchFinding{}, 0, nil
	}

	if limit <= 0 {
		limit = 20
	}

	countQuery := `SELECT COUNT(*) FROM ai_research_findings WHERE ($1 = '' OR status = $1);`
	var total int
	err := r.dbPool.QueryRow(ctx, countQuery, status).Scan(&total)
	if err != nil {
		return []model.AIResearchFinding{}, 0, err
	}

	query := `
		SELECT id, company_id, COALESCE(company_name, ''), finding_type, title, COALESCE(summary, ''),
		       source_url, source_name, source_type, published_at, discovered_at,
		       confidence_score, evidence_data, status, COALESCE(idempotency_key, ''), reviewed_by,
		       reviewed_at, COALESCE(review_notes, ''), created_at, updated_at
		FROM ai_research_findings
		WHERE ($1 = '' OR status = $1)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3;
	`

	rows, err := r.dbPool.Query(ctx, query, status, limit, offset)
	if err != nil {
		return []model.AIResearchFinding{}, 0, err
	}
	defer rows.Close()

	list := make([]model.AIResearchFinding, 0)
	for rows.Next() {
		var f model.AIResearchFinding
		var evidenceBytes []byte
		err := rows.Scan(
			&f.ID,
			&f.CompanyID,
			&f.CompanyName,
			&f.FindingType,
			&f.Title,
			&f.Summary,
			&f.SourceURL,
			&f.SourceName,
			&f.SourceType,
			&f.PublishedAt,
			&f.DiscoveredAt,
			&f.ConfidenceScore,
			&evidenceBytes,
			&f.Status,
			&f.IdempotencyKey,
			&f.ReviewedBy,
			&f.ReviewedAt,
			&f.ReviewNotes,
			&f.CreatedAt,
			&f.UpdatedAt,
		)
		if err != nil {
			log.Printf("Scan error in ListFindings: %v", err)
			continue
		}
		if len(evidenceBytes) > 0 {
			_ = json.Unmarshal(evidenceBytes, &f.EvidenceData)
		}
		list = append(list, f)
	}

	return list, total, nil
}

// ReviewFinding updates the review status of an AI finding (approve or reject)
func (r *AIAgentRepository) ReviewFinding(ctx context.Context, findingID uuid.UUID, adminID uuid.UUID, action, notes string) (*model.AIResearchFinding, error) {
	if r.dbPool == nil {
		return nil, fmt.Errorf("database pool not initialized")
	}

	newStatus := "approved"
	if action == "reject" {
		newStatus = "rejected"
	}

	var reviewedByPtr *uuid.UUID
	if adminID != uuid.Nil {
		reviewedByPtr = &adminID
	}

	query := `
		UPDATE ai_research_findings
		SET status = $1, reviewed_by = $2, reviewed_at = NOW(), review_notes = $3, updated_at = NOW()
		WHERE id = $4
		RETURNING id, company_id, company_name, finding_type, title, summary, source_url, source_name, source_type, published_at, discovered_at, confidence_score, evidence_data, status, idempotency_key, reviewed_by, reviewed_at, review_notes, created_at, updated_at;
	`

	var f model.AIResearchFinding
	var evidenceBytes []byte

	err := r.dbPool.QueryRow(ctx, query, newStatus, reviewedByPtr, notes, findingID).Scan(
		&f.ID,
		&f.CompanyID,
		&f.CompanyName,
		&f.FindingType,
		&f.Title,
		&f.Summary,
		&f.SourceURL,
		&f.SourceName,
		&f.SourceType,
		&f.PublishedAt,
		&f.DiscoveredAt,
		&f.ConfidenceScore,
		&evidenceBytes,
		&f.Status,
		&f.IdempotencyKey,
		&f.ReviewedBy,
		&f.ReviewedAt,
		&f.ReviewNotes,
		&f.CreatedAt,
		&f.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update finding review status: %w", err)
	}

	if len(evidenceBytes) > 0 {
		_ = json.Unmarshal(evidenceBytes, &f.EvidenceData)
	}

	// If approved, promote finding to intelligence.company_signals (for /signals page) AND company_enriched_programs
	if newStatus == "approved" {
		contentHash := fmt.Sprintf("finding_%s", f.ID.String())
		sourceType := f.SourceType
		if sourceType == "" {
			sourceType = "OPENCLAW_AI_FINDING"
		}

		companyID := f.CompanyID
		if companyID == nil && strings.TrimSpace(f.CompanyName) != "" {
			var resolvedID uuid.UUID
			cleanName := strings.TrimSpace(f.CompanyName)
			cleanKeyword := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(cleanName), "pt ", ""), "tbk", ""))
			err := r.dbPool.QueryRow(ctx, `
				SELECT id FROM companies 
				WHERE LOWER(TRIM(name)) = LOWER(TRIM($1)) 
				   OR name ILIKE $2 
				   OR slug = $3 
				LIMIT 1
			`, cleanName, "%"+cleanKeyword+"%", cleanKeyword).Scan(&resolvedID)
			if err == nil {
				companyID = &resolvedID
			}
		}

		_, _ = r.dbPool.Exec(ctx, `
			INSERT INTO intelligence.company_signals (
				company_id, company_name, industry_sector, source_type, source_url, 
				summary, extracted_pillar, target_regions, estimated_budget_signal, 
				trigger_event, intent_score, content_hash, csr_relevance, activity_focus, 
				action_type, opportunity_alert, published_date, created_at
			) VALUES (
				$1, $2, 'Lingkungan & Keberlanjutan', $3, $4, 
				$5, 'Keberlanjutan', ARRAY['Nasional'], 0, 
				$6, 90, $7, 'HIGH', $6, 'PROGRAM_LAUNCH', true, CURRENT_DATE, NOW()
			)
			ON CONFLICT (content_hash) DO UPDATE SET
				company_name = EXCLUDED.company_name,
				summary = EXCLUDED.summary,
				intent_score = EXCLUDED.intent_score,
				created_at = NOW();
		`, companyID, f.CompanyName, sourceType, f.SourceURL, f.Summary, f.Title, contentHash)

		if companyID != nil {
			_, _ = r.dbPool.Exec(ctx, `
				INSERT INTO company_enriched_programs (company_id, name, description, partner_ngo, created_at, updated_at)
				VALUES ($1, $2, $3, $4, NOW(), NOW())
				ON CONFLICT DO NOTHING;
			`, companyID, f.Title, f.Summary, f.SourceName)
		}
	}


	return &f, nil
}

// SeedAgentCredential inserts default agent token if not exists
func (r *AIAgentRepository) SeedAgentCredential(ctx context.Context, agentName, rawToken string, scopes []string) error {
	if r.dbPool == nil {
		return nil
	}

	keyHash := HashAPIKey(rawToken)

	query := `
		INSERT INTO ai_agent_credentials (agent_name, api_key_hash, scopes, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, true, NOW(), NOW())
		ON CONFLICT (api_key_hash) DO UPDATE
		SET agent_name = EXCLUDED.agent_name, scopes = EXCLUDED.scopes, is_active = true, updated_at = NOW();
	`

	_, err := r.dbPool.Exec(ctx, query, agentName, keyHash, scopes)
	return err
}

// EnrichCompanyProfile enriches company metadata from OpenClaw research
func (r *AIAgentRepository) EnrichCompanyProfile(ctx context.Context, companyID string, req model.EnrichCompanyRequest) (*model.CompanyEnrichmentResult, error) {
	if r.dbPool == nil {
		return nil, fmt.Errorf("database pool is not initialized")
	}

	companyUUID, err := uuid.Parse(companyID)
	if err != nil {
		return nil, fmt.Errorf("invalid company UUID '%s': %w", companyID, err)
	}

	var companyName string
	err = r.dbPool.QueryRow(ctx, `SELECT name FROM company.companies WHERE id = $1;`, companyUUID).Scan(&companyName)
	if err != nil {
		return nil, fmt.Errorf("company not found: %w", err)
	}

	var fieldsEnriched []string

	query := `
		UPDATE company.companies
		SET website = COALESCE(NULLIF($1, ''), website),
		    headquarters = COALESCE(NULLIF($2, ''), headquarters),
		    industry_sector = COALESCE(NULLIF($3, ''), industry_sector),
		    employee_range = COALESCE(NULLIF($4, ''), employee_range),
		    revenue_range = COALESCE(NULLIF($5, ''), revenue_range),
		    csr_category = COALESCE(NULLIF($6, ''), csr_category),
		    priority_tier = COALESCE(NULLIF($7, ''), priority_tier),
		    partner_ngo = COALESCE(NULLIF($8, ''), partner_ngo),
		    updated_at = NOW()
		WHERE id = $9;
	`

	var webStr, hqStr, sectorStr, empStr, revStr, catStr, tierStr, ngoStr string
	if req.Website != nil && *req.Website != "" {
		webStr = *req.Website
		fieldsEnriched = append(fieldsEnriched, "website")
	}
	if req.Headquarters != nil && *req.Headquarters != "" {
		hqStr = *req.Headquarters
		fieldsEnriched = append(fieldsEnriched, "headquarters")
	}
	if req.IndustrySector != nil && *req.IndustrySector != "" {
		sectorStr = *req.IndustrySector
		fieldsEnriched = append(fieldsEnriched, "industry_sector")
	}
	if req.EmployeeRange != nil && *req.EmployeeRange != "" {
		empStr = *req.EmployeeRange
		fieldsEnriched = append(fieldsEnriched, "employee_range")
	}
	if req.RevenueRange != nil && *req.RevenueRange != "" {
		revStr = *req.RevenueRange
		fieldsEnriched = append(fieldsEnriched, "revenue_range")
	}
	if req.CSRCategory != nil && *req.CSRCategory != "" {
		catStr = *req.CSRCategory
		fieldsEnriched = append(fieldsEnriched, "csr_category")
	}
	if req.PriorityTier != nil && *req.PriorityTier != "" {
		tierStr = *req.PriorityTier
		fieldsEnriched = append(fieldsEnriched, "priority_tier")
	}
	if req.PartnerNGO != nil && *req.PartnerNGO != "" {
		ngoStr = *req.PartnerNGO
		fieldsEnriched = append(fieldsEnriched, "partner_ngo")
	}

	_, err = r.dbPool.Exec(ctx, query,
		webStr, hqStr, sectorStr, empStr, revStr, catStr, tierStr, ngoStr, companyUUID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update company profile: %w", err)
	}

	if len(req.AliasKeywords) > 0 {
		_, _ = r.dbPool.Exec(ctx, `
			UPDATE company.companies
			SET alias_keywords = (
				SELECT ARRAY(SELECT DISTINCT unnest(alias_keywords || $1::text[]))
			), updated_at = NOW()
			WHERE id = $2;
		`, req.AliasKeywords, companyUUID)
		fieldsEnriched = append(fieldsEnriched, "alias_keywords")
	}

	return &model.CompanyEnrichmentResult{
		CompanyID:      companyID,
		CompanyName:    companyName,
		FieldsEnriched: fieldsEnriched,
		EnrichedAt:     time.Now().UTC(),
	}, nil
}

// MatchCompaniesForProgram computes weighted corporate matching for institution proposals
func (r *AIAgentRepository) MatchCompaniesForProgram(ctx context.Context, req model.AIMatchingRequest) (*model.AIMatchingResponse, error) {
	if r.dbPool == nil {
		return nil, fmt.Errorf("database pool is not initialized")
	}

	limit := req.Limit
	if limit <= 0 || limit > 20 {
		limit = 10
	}

	query := `
		SELECT c.id::text, c.name, c.industry_sector, COALESCE(c.priority_tier, 'TIER_3'), COALESCE(c.csr_category, 'POTENSIAL'), COALESCE(c.website, ''), COALESCE(c.headquarters, ''), COALESCE(c.alias_keywords, '{}')
		FROM company.companies c
		ORDER BY c.priority_tier ASC, c.name ASC
		LIMIT 100;
	`

	rows, err := r.dbPool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query companies for matching: %w", err)
	}
	defer rows.Close()

	categoryLower := strings.ToLower(req.CSRCategory)
	locationLower := strings.ToLower(req.TargetLocation)

	var matches []model.CompanyMatchResult

	for rows.Next() {
		var id, name, sector, tier, category, website, hq string
		var aliases []string
		if err := rows.Scan(&id, &name, &sector, &tier, &category, &website, &hq, &aliases); err != nil {
			continue
		}

		focusScore := 20.0
		searchableText := strings.ToLower(name + " " + sector + " " + category + " " + strings.Join(aliases, " "))
		if categoryLower != "" && strings.Contains(searchableText, categoryLower) {
			focusScore += 20.0
		} else {
			for _, kw := range req.Keywords {
				if kw != "" && strings.Contains(searchableText, strings.ToLower(kw)) {
					focusScore += 15.0
					break
				}
			}
		}

		geoScore := 15.0
		hqLower := strings.ToLower(hq)
		if locationLower != "" && (strings.Contains(hqLower, locationLower) || locationLower == "national" || tier == "TIER_1") {
			geoScore = 30.0
		}

		tierScore := 10.0
		switch tier {
		case "TIER_1":
			tierScore = 30.0
		case "TIER_2":
			tierScore = 20.0
		default:
			tierScore = 10.0
		}

		totalScore := focusScore + geoScore + tierScore
		if totalScore > 100.0 {
			totalScore = 100.0
		}

		grade := "POTENTIAL"
		if totalScore >= 85.0 {
			grade = "HIGHLY_RECOMMENDED"
		} else if totalScore >= 70.0 {
			grade = "GOOD_MATCH"
		}

		rationale := fmt.Sprintf("Kesesuaian tinggi sektor %s dengan program %s (Fokus: %.0f/40, Lokasi: %.0f/30, Tier: %s).", sector, req.ProgramTitle, focusScore, geoScore, tier)

		maxAliases := len(aliases)
		if maxAliases > 2 {
			maxAliases = 2
		}
		focusAreas := []string{sector, category}
		if maxAliases > 0 {
			focusAreas = append(focusAreas, aliases[:maxAliases]...)
		}

		matches = append(matches, model.CompanyMatchResult{
			CompanyID:      id,
			CompanyName:    name,
			IndustrySector: sector,
			PriorityTier:   tier,
			MatchScore:     totalScore,
			MatchGrade:     grade,
			MatchRationale: rationale,
			KeyFocusAreas:  focusAreas,
			Website:        website,
		})
	}

	for i := 0; i < len(matches); i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[j].MatchScore > matches[i].MatchScore {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}

	if len(matches) > limit {
		matches = matches[:limit]
	}

	return &model.AIMatchingResponse{
		ProgramTitle:     req.ProgramTitle,
		CSRCategory:      req.CSRCategory,
		MatchesFound:     len(matches),
		CorporateMatches: matches,
		EvaluatedAt:      time.Now().UTC(),
	}, nil
}


