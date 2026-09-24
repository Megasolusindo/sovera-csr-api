package handler

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"sovera-core-api/internal/model"
	"sovera-core-api/internal/repository"
)

type AIAgentHandler struct {
	dbPool        *pgxpool.Pool
	aiRepo        *repository.AIAgentRepository
	companyRepo   *repository.CompanyRepository
	watchlistRepo *repository.AIWatchlistRepository
}

func NewAIAgentHandler(dbPool *pgxpool.Pool) *AIAgentHandler {
	var aiRepo *repository.AIAgentRepository
	var companyRepo *repository.CompanyRepository
	var watchlistRepo *repository.AIWatchlistRepository

	if dbPool != nil {
		aiRepo = repository.NewAIAgentRepository(dbPool)
		companyRepo = repository.NewCompanyRepository(dbPool)
		watchlistRepo = repository.NewAIWatchlistRepository(dbPool)
	}

	return &AIAgentHandler{
		dbPool:        dbPool,
		aiRepo:        aiRepo,
		companyRepo:   companyRepo,
		watchlistRepo: watchlistRepo,
	}
}

// SearchCompanies handles GET /api/v1/ai/companies/search
func (h *AIAgentHandler) SearchCompanies(c *fiber.Ctx) error {
	queryStr := strings.TrimSpace(c.Query("q"))
	if queryStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "MISSING_QUERY",
			"message": "Query parameter 'q' is required for company search",
		})
	}

	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	if h.companyRepo == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "DATABASE_UNAVAILABLE",
			"message": "Company repository is not connected",
		})
	}

	companies, total, err := h.companyRepo.ListCompanies(c.Context(), limit, offset, queryStr, "", "", "", "")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "SEARCH_FAILED",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":   true,
		"data":      companies,
		"count":     len(companies),
		"total":     total,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// SearchCorporateData handles GET /api/v1/ai/search?q=...
func (h *AIAgentHandler) SearchCorporateData(c *fiber.Ctx) error {
	queryStr := strings.TrimSpace(c.Query("q"))
	if queryStr == "" {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": true,
			"count":   0,
			"data":    []interface{}{},
		})
	}

	if h.dbPool == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "DATABASE_UNAVAILABLE",
		})
	}

	searchPattern := "%" + queryStr + "%"

	type SignalResult struct {
		ID              string   `json:"id"`
		CompanyName     string   `json:"company_name"`
		Summary         string   `json:"summary"`
		Pillar          string   `json:"pillar"`
		TargetRegions   []string `json:"target_regions"`
		EstimatedBudget float64  `json:"estimated_budget"`
		SourceURL       string   `json:"source_url"`
		CreatedAt       string   `json:"created_at"`
	}

	results := []SignalResult{}

	// 1. Query public_corporate_signals
	sqlQuery := `
		SELECT 
			id::text, company_name, COALESCE(summary, ''), COALESCE(extracted_pillar, ''), 
			COALESCE(target_regions, '{}'), COALESCE(estimated_budget_signal, 0), COALESCE(source_url, ''), created_at
		FROM public_corporate_signals
		WHERE 
			company_name ILIKE $1 
			OR summary ILIKE $1 
			OR extracted_pillar ILIKE $1 
			OR activity_focus ILIKE $1
			OR array_to_string(target_regions, ' ') ILIKE $1
		ORDER BY created_at DESC
		LIMIT 20;
	`

	rows, err := h.dbPool.Query(c.Context(), sqlQuery, searchPattern)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var r SignalResult
			var createdAt time.Time
			if err := rows.Scan(&r.ID, &r.CompanyName, &r.Summary, &r.Pillar, &r.TargetRegions, &r.EstimatedBudget, &r.SourceURL, &createdAt); err == nil {
				r.CreatedAt = createdAt.Format(time.RFC3339)
				results = append(results, r)
			}
		}
	}

		// 2. Query company_enriched_programs if results < 10
		if len(results) < 10 {
			progQuery := `
			SELECT 
				p.id::text, c.name, COALESCE(p.name, ''), 				COALESCE(p.impact_summary, ''), 
				'', COALESCE(p.budget_amount, 0), COALESCE(c.website, '')
			FROM company_enriched_programs p
			JOIN companies c ON c.id = p.company_id
			WHERE 
				c.name ILIKE $1 
				OR p.name ILIKE $1 
				OR p.impact_summary ILIKE $1 
			LIMIT 10;
		`
		progRows, progErr := h.dbPool.Query(c.Context(), progQuery, searchPattern)
		if progErr == nil {
			defer progRows.Close()
			for progRows.Next() {
				var id, compName, progName, desc, loc, web string
				var budget float64
				if err := progRows.Scan(&id, &compName, &progName, &desc, &loc, &web); err == nil {
					summary := fmt.Sprintf("Program: %s. %s", progName, desc)
					results = append(results, SignalResult{
						ID:              id,
						CompanyName:     compName,
						Summary:         summary,
						Pillar:          "CSR Program",
						TargetRegions:   []string{loc},
						EstimatedBudget: budget,
						SourceURL:       web,
						CreatedAt:       time.Now().Format(time.RFC3339),
					})
				}
			}
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"query":   queryStr,
		"count":   len(results),
		"data":    results,
	})
}

// GetCompanyDetails handles GET /api/v1/ai/companies/:id
func (h *AIAgentHandler) GetCompanyDetails(c *fiber.Ctx) error {
	companyIDStr := strings.TrimSpace(c.Params("id"))
	if companyIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "MISSING_ID",
			"message": "Company ID or slug parameter is required",
		})
	}

	if h.companyRepo == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "DATABASE_UNAVAILABLE",
		})
	}

	company, err := h.companyRepo.GetCompanyByID(c.Context(), companyIDStr)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "COMPANY_NOT_FOUND",
			"message": fmt.Sprintf("Company '%s' not found", companyIDStr),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    company,
	})
}

// SubmitResearchFinding handles POST /api/v1/ai/research/findings
func (h *AIAgentHandler) SubmitResearchFinding(c *fiber.Ctx) error {
	var req model.SubmitFindingRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "INVALID_PAYLOAD",
			"message": "Failed to parse request JSON body",
		})
	}

	// Validation 1: Title and Summary are required
	if strings.TrimSpace(req.Title) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "MISSING_TITLE",
			"message": "Finding title is mandatory",
		})
	}

	// Validation 2: Source URL is mandatory according to PRD Rules (Zero Mocking & Evidence Only)
	sourceURL := strings.TrimSpace(req.SourceURL)
	if sourceURL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "MISSING_SOURCE_URL",
			"message": "PRD Mandate: Every research finding MUST include a valid source_url evidence link.",
		})
	}

	parsedURL, urlErr := url.Parse(sourceURL)
	if urlErr != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "INVALID_SOURCE_URL",
			"message": "source_url must be a valid HTTP/HTTPS URL",
		})
	}

	// Validation 3: Confidence Score Range Check
	if req.ConfidenceScore <= 0 || req.ConfidenceScore > 1.0 {
		if req.ConfidenceScore <= 0 {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
				"success": false,
				"error":   "LOW_CONFIDENCE_REJECTED",
				"message": "PRD Rule: Findings with confidence_score <= 0 are rejected.",
			})
		}
		req.ConfidenceScore = 0.85
	}

	if req.SourceName == "" {
		req.SourceName = parsedURL.Host
	}

	if req.SourceType == "" {
		req.SourceType = "news"
	}

	if req.FindingType == "" {
		req.FindingType = "csr_program"
	}

	// Idempotency Key computation if not provided
	if req.IdempotencyKey == "" {
		req.IdempotencyKey = fmt.Sprintf("finding_%s_%s", req.FindingType, repository.HashAPIKey(req.Title+req.SourceURL)[:16])
	}

	var companyUUID *uuid.UUID
	if req.CompanyID != nil && *req.CompanyID != "" {
		if parsedUUID, err := uuid.Parse(*req.CompanyID); err == nil {
			companyUUID = &parsedUUID
		}
	}

	var pubTime *time.Time
	if req.PublishedAt != nil && *req.PublishedAt != "" {
		if t, err := time.Parse(time.RFC3339, *req.PublishedAt); err == nil {
			pubTime = &t
		} else if t, err := time.Parse("2006-01-02", *req.PublishedAt); err == nil {
			pubTime = &t
		}
	}

	if h.aiRepo == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "DATABASE_UNAVAILABLE",
		})
	}

	finding := model.AIResearchFinding{
		CompanyID:       companyUUID,
		CompanyName:     req.CompanyName,
		FindingType:     req.FindingType,
		Title:           req.Title,
		Summary:         req.Summary,
		SourceURL:       req.SourceURL,
		SourceName:      req.SourceName,
		SourceType:      req.SourceType,
		PublishedAt:     pubTime,
		ConfidenceScore: req.ConfidenceScore,
		EvidenceData:    req.EvidenceData,
		IdempotencyKey:  req.IdempotencyKey,
	}

	result, err := h.aiRepo.SubmitFinding(c.Context(), finding)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "SUBMISSION_FAILED",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success":   true,
		"message":   "Research finding submitted successfully and queued for admin review",
		"data":      result,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// ListResearchFindings handles GET /api/v1/ai/research/findings
func (h *AIAgentHandler) ListResearchFindings(c *fiber.Ctx) error {
	status := c.Query("status", "")
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	if h.aiRepo == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "DATABASE_UNAVAILABLE",
		})
	}

	findings, total, err := h.aiRepo.ListFindings(c.Context(), status, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "QUERY_FAILED",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    findings,
		"count":   len(findings),
		"total":   total,
	})
}

// GetActiveWatchlist handles GET /api/v1/ai/watchlist
func (h *AIAgentHandler) GetActiveWatchlist(c *fiber.Ctx) error {
	if h.watchlistRepo == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "DATABASE_UNAVAILABLE",
		})
	}

	watchlist, err := h.watchlistRepo.ListActiveWatchlist(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "QUERY_FAILED",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":   true,
		"data":      watchlist,
		"count":     len(watchlist),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// AddWatchlistCompany handles POST /api/v1/ai/watchlist
func (h *AIAgentHandler) AddWatchlistCompany(c *fiber.Ctx) error {
	type AddRequest struct {
		CompanyID          string   `json:"company_id"`
		CompanyName        string   `json:"company_name"`
		MonitoringKeywords []string `json:"monitoring_keywords"`
		Keywords           []string `json:"keywords"`
		CheckIntervalHours int      `json:"check_interval_hours"`
	}

	var req AddRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "INVALID_PAYLOAD",
			"message": "Failed to parse request JSON body",
		})
	}

	companyName := strings.TrimSpace(req.CompanyName)
	companyIDStr := strings.TrimSpace(req.CompanyID)

	if companyName == "" && companyIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "MISSING_COMPANY_NAME",
			"message": "Company name or company ID is required to add to watchlist",
		})
	}

	var companyUUID uuid.UUID
	if companyIDStr != "" {
		if parsed, err := uuid.Parse(companyIDStr); err == nil {
			companyUUID = parsed
		}
	}

	if companyUUID == uuid.Nil && companyName != "" {
		if h.companyRepo != nil {
			comps, _, err := h.companyRepo.ListCompanies(c.Context(), 1, 0, companyName, "", "", "", "")
			if err == nil && len(comps) > 0 {
				companyUUID, _ = uuid.Parse(comps[0].ID)
				companyName = comps[0].Name
			} else {
				// Master company record does not exist yet -> Provision master company record
				newComp := model.Company{
					Name:         companyName,
					CompanyType:  "SWASTA",
					PriorityTier: "TIER_1",
				}
				if created, createErr := h.companyRepo.CreateCompany(c.Context(), newComp); createErr == nil && created != nil {
					companyUUID, _ = uuid.Parse(created.ID)
					companyName = created.Name
				}
			}
		}
	}

	if companyUUID == uuid.Nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "COMPANY_RESOLVE_FAILED",
			"message": "Failed to create or resolve master company record in database",
		})
	}

	keywords := req.MonitoringKeywords
	if len(keywords) == 0 {
		keywords = req.Keywords
	}
	if len(keywords) == 0 {
		keywords = []string{"CSR", "TJSL", "Keberlanjutan", "ESG", companyName}
	}

	intervalHours := req.CheckIntervalHours
	if intervalHours <= 0 {
		intervalHours = 12
	}

	item := model.AICompanyWatchlist{
		CompanyID:          companyUUID,
		CompanyName:        companyName,
		MonitoringKeywords: keywords,
		CheckIntervalHours: intervalHours,
		IsActive:           true,
	}

	if h.watchlistRepo == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "DATABASE_UNAVAILABLE",
		})
	}

	result, err := h.watchlistRepo.AddOrUpdateWatchlist(c.Context(), item)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "WATCHLIST_UPDATE_FAILED",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":   true,
		"message":   fmt.Sprintf("Company '%s' added to watchlist successfully", companyName),
		"data":      result,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// ToggleCompanyWatchlist handles POST /api/v1/ai/companies/:id/monitor
func (h *AIAgentHandler) ToggleCompanyWatchlist(c *fiber.Ctx) error {
	companyIDStr := strings.TrimSpace(c.Params("id"))
	if companyIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "MISSING_ID",
			"message": "Company ID parameter is required",
		})
	}

	companyUUID, err := uuid.Parse(companyIDStr)
	if err != nil {
		if h.companyRepo != nil {
			comp, compErr := h.companyRepo.GetCompanyByID(c.Context(), companyIDStr)
			if compErr == nil && comp.ID != "" {
				companyUUID, _ = uuid.Parse(comp.ID)
			}
		}
	}

	if companyUUID == uuid.Nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "INVALID_COMPANY_ID",
			"message": "Valid Company UUID is required",
		})
	}

	var req model.ToggleWatchlistRequest
	_ = c.BodyParser(&req)

	companyName := ""
	if h.companyRepo != nil {
		comp, compErr := h.companyRepo.GetCompanyByID(c.Context(), companyUUID.String())
		if compErr == nil {
			companyName = comp.Name
		}
	}

	if companyName == "" {
		companyName = fmt.Sprintf("Company %s", companyUUID.String()[:8])
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	item := model.AICompanyWatchlist{
		CompanyID:          companyUUID,
		CompanyName:        companyName,
		MonitoringKeywords: req.MonitoringKeywords,
		CheckIntervalHours: req.CheckIntervalHours,
		IsActive:           isActive,
	}

	if h.watchlistRepo == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "DATABASE_UNAVAILABLE",
		})
	}

	result, err := h.watchlistRepo.AddOrUpdateWatchlist(c.Context(), item)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "WATCHLIST_UPDATE_FAILED",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Company watchlist setting updated successfully",
		"data":    result,
	})
}

// EnrichCompany handles POST /api/v1/ai/companies/:id/enrich
func (h *AIAgentHandler) EnrichCompany(c *fiber.Ctx) error {
	companyIDStr := strings.TrimSpace(c.Params("id"))
	if companyIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "MISSING_ID",
			"message": "Company ID parameter is required",
		})
	}

	var req model.EnrichCompanyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "INVALID_PAYLOAD",
			"message": "Failed to parse request JSON body",
		})
	}

	if req.Website != nil && *req.Website != "" {
		web := strings.TrimSpace(*req.Website)
		if !strings.HasPrefix(web, "http://") && !strings.HasPrefix(web, "https://") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"error":   "INVALID_WEBSITE_URL",
				"message": "PRD Rule: Website URL must start with http:// or https://",
			})
		}
	}

	if h.aiRepo == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "DATABASE_UNAVAILABLE",
		})
	}

	result, err := h.aiRepo.EnrichCompanyProfile(c.Context(), companyIDStr, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "ENRICHMENT_FAILED",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":   true,
		"message":   fmt.Sprintf("Company '%s' profile enriched successfully", result.CompanyName),
		"data":      result,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// MatchProgram handles POST /api/v1/ai/matching
func (h *AIAgentHandler) MatchProgram(c *fiber.Ctx) error {
	var req model.AIMatchingRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "INVALID_PAYLOAD",
			"message": "Failed to parse request JSON body",
		})
	}

	if strings.TrimSpace(req.ProgramTitle) == "" {
		req.ProgramTitle = "Proposal Program CSR Lembaga"
	}

	if h.aiRepo == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "DATABASE_UNAVAILABLE",
		})
	}

	result, err := h.aiRepo.MatchCompaniesForProgram(c.Context(), req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "MATCHING_FAILED",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":   true,
		"message":   "Program matching executed successfully",
		"data":      result,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}


