package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"sovera-core-api/internal/model"
	"sovera-core-api/internal/repository"
)

type AHUHandler struct {
	repo *repository.AHURepository
}

func NewAHUHandler(repo *repository.AHURepository) *AHUHandler {
	return &AHUHandler{repo: repo}
}

// GetAHUByNumber handles GET /api/v1/ahu/:ahu_number
func (h *AHUHandler) GetAHUByNumber(c *fiber.Ctx) error {
	ahuNumber := c.Params("ahu_number")
	if ahuNumber == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "AHU number is required"})
	}

	record, err := h.repo.GetByAHUNumber(c.UserContext(), ahuNumber)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"data": record})
}

// SearchAHUByName handles GET /api/v1/ahu/search?q=...
func (h *AHUHandler) SearchAHUByName(c *fiber.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "search query parameter 'q' is required"})
	}

	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	records, err := h.repo.SearchByName(c.UserContext(), query, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"data":  records,
		"total": len(records),
	})
}

// ResolveEntity handles POST /api/v1/ahu/resolve
func (h *AHUHandler) ResolveEntity(c *fiber.Ctx) error {
	var req model.AHUEntityResolutionRequest
	if err := c.BodyParser(&req); err != nil || req.RawName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body: raw_name is required"})
	}

	result, err := h.repo.ResolveEntity(c.UserContext(), req.RawName)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"data": result})
}

// UpsertAHU handles POST /api/v1/ahu
func (h *AHUHandler) UpsertAHU(c *fiber.Ctx) error {
	var reg model.AHURegistration
	if err := c.BodyParser(&reg); err != nil || reg.AHUNumber == "" || reg.CompanyName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body: ahu_number and company_name are required"})
	}

	upserted, err := h.repo.UpsertAHU(c.UserContext(), &reg)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success",
		"data":   upserted,
	})
}

// LinkCompanyAHU handles POST /api/v1/companies/:id/ahu
func (h *AHUHandler) LinkCompanyAHU(c *fiber.Ctx) error {
	companyID := c.Params("id")
	if companyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Company ID is required"})
	}

	var req struct {
		AHUNumber string `json:"ahu_number"`
	}

	if err := c.BodyParser(&req); err != nil || req.AHUNumber == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body: ahu_number is required"})
	}

	err := h.repo.LinkCompanyAHU(c.UserContext(), companyID, req.AHUNumber)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Company AHU registration linked successfully",
	})
}
