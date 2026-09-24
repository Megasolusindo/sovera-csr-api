package handler

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"sovera-core-api/internal/model"
	"sovera-core-api/internal/repository"
)

type AIToolsHandler struct {
	dbPool       *pgxpool.Pool
	companyRepo  *repository.CompanyRepository
	signalRepo   *repository.SignalRepository
	aiRepo       *repository.AIAgentRepository
}

func NewAIToolsHandler(dbPool *pgxpool.Pool) *AIToolsHandler {
	return &AIToolsHandler{
		dbPool:      dbPool,
		companyRepo: repository.NewCompanyRepository(dbPool),
		signalRepo:  repository.NewSignalRepository(dbPool),
		aiRepo:       repository.NewAIAgentRepository(dbPool),
	}
}

// SearchCompanies handles POST /api/v1/ai/tools/search_companies (§4.4)
func (h *AIToolsHandler) SearchCompanies(c *fiber.Ctx) error {
	var req struct {
		Query  string `json:"query"`
		Sector string `json:"sector"`
		Limit  int    `json:"limit"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request payload"})
	}

	if req.Limit <= 0 {
		req.Limit = 10
	}

	companies, _, err := h.companyRepo.ListCompanies(c.Context(), req.Limit, 0, req.Query, req.Sector, "", "", "")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"tool_name": "search_companies",
		"status":    "SUCCESS",
		"count":     len(companies),
		"companies": companies,
	})
}

// GetCSRSignals handles POST /api/v1/ai/tools/get_csr_signals (§4.4)
func (h *AIToolsHandler) GetCSRSignals(c *fiber.Ctx) error {
	var req struct {
		Industry  string `json:"industry"`
		MinIntent int    `json:"min_intent"`
		Limit     int    `json:"limit"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request payload"})
	}

	if req.Limit <= 0 {
		req.Limit = 10
	}

	signals, total, err := h.signalRepo.ListSignals(c.Context(), req.Limit, 0, req.MinIntent, "", req.Industry)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"tool_name": "get_csr_signals",
		"status":    "SUCCESS",
		"total":     total,
		"signals":   signals,
	})
}

// MatchOpportunity handles POST /api/v1/ai/tools/match_opportunity (§4.4)
func (h *AIToolsHandler) MatchOpportunity(c *fiber.Ctx) error {
	var req model.AIMatchingRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request payload"})
	}

	resp, err := h.aiRepo.MatchCompaniesForProgram(c.Context(), req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"tool_name": "match_opportunity",
		"status":    "SUCCESS",
		"result":    resp,
	})
}

// SendSessionAlert handles POST /api/v1/ai/tools/send_session_alert (§4.4)
func (h *AIToolsHandler) SendSessionAlert(c *fiber.Ctx) error {
	var req struct {
		SessionID string `json:"session_id"`
		AlertType string `json:"alert_type"`
		Message   string `json:"message"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request payload"})
	}

	log.Printf("[AIToolsHandler] Session Alert sent to session %s [%s]: %s", req.SessionID, req.AlertType, req.Message)

	return c.JSON(fiber.Map{
		"tool_name":  "send_session_alert",
		"status":     "DELIVERED",
		"session_id": req.SessionID,
	})
}
