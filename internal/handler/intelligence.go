package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"sovera-core-api/internal/repository"
)

type IntelligenceHandler struct {
	repo *repository.IntelligenceRepository
}

func NewIntelligenceHandler(repo *repository.IntelligenceRepository) *IntelligenceHandler {
	return &IntelligenceHandler{repo: repo}
}

// GetOverview handles GET /api/v1/intelligence/overview
func (h *IntelligenceHandler) GetOverview(c *fiber.Ctx) error {
	orgID, ok := c.Locals("org_id").(string)
	if !ok || orgID == "" {
		orgID = "99999999-9999-4000-a000-000000000001"
	}

	stats, err := h.repo.GetOverviewKPIs(c.Context(), orgID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "OVERVIEW_FAILED",
			"message": err.Error(),
		})
	}

	trends, _ := h.repo.GetCSRTrends(c.Context(), 30)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"stats":  stats,
			"trends": trends,
		},
	})
}

// ListOrganizations handles GET /api/v1/intelligence/organizations
func (h *IntelligenceHandler) ListOrganizations(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	search := c.Query("search", "")
	pillar := c.Query("pillar", "")
	region := c.Query("region", "")

	orgs, total, err := h.repo.ListRecommendedOrganizations(c.Context(), limit, offset, search, pillar, region)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "QUERY_FAILED",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    orgs,
		"pagination": fiber.Map{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// ListPrograms handles GET /api/v1/intelligence/programs
func (h *IntelligenceHandler) ListPrograms(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	search := c.Query("search", "")
	pillar := c.Query("pillar", "")

	programs, total, err := h.repo.ListIntelligencePrograms(c.Context(), limit, offset, search, pillar)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "QUERY_FAILED",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    programs,
		"pagination": fiber.Map{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetTrends handles GET /api/v1/intelligence/trends
func (h *IntelligenceHandler) GetTrends(c *fiber.Ctx) error {
	trends, err := h.repo.GetCSRTrends(c.Context(), 30)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "TRENDS_FAILED",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    trends,
	})
}

type SaveItemPayload struct {
	ItemType string `json:"item_type"` // ORGANIZATION | PROGRAM | OPPORTUNITY | SIGNAL
	ItemID   string `json:"item_id"`
	Notes    string `json:"notes"`
}

// SaveItem handles POST /api/v1/intelligence/saved
func (h *IntelligenceHandler) SaveItem(c *fiber.Ctx) error {
	orgID, ok := c.Locals("org_id").(string)
	if !ok || orgID == "" {
		orgID = "99999999-9999-4000-a000-000000000001"
	}

	var payload SaveItemPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false, "error": "INVALID_BODY", "message": err.Error(),
		})
	}

	if payload.ItemType == "" || payload.ItemID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false, "error": "MISSING_FIELDS", "message": "item_type and item_id are required",
		})
	}

	if err := h.repo.SaveTenantItem(c.Context(), orgID, payload.ItemType, payload.ItemID, payload.Notes); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false, "error": "SAVE_FAILED", "message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Item saved successfully",
	})
}
