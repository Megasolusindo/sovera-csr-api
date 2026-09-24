package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"sovera-core-api/internal/model"
	"sovera-core-api/internal/repository"
)

type OSSHandler struct {
	repo *repository.OSSRepository
}

func NewOSSHandler(repo *repository.OSSRepository) *OSSHandler {
	return &OSSHandler{repo: repo}
}

// GetByNIB handles GET /api/v1/oss/:nib
func (h *OSSHandler) GetByNIB(c *fiber.Ctx) error {
	nib := c.Params("nib")
	if nib == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "NIB number is required"})
	}

	record, err := h.repo.GetByNIB(c.UserContext(), nib)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"data": record})
}

// Search handles GET /api/v1/oss/search?q=...&kbli=...&investment_status=...&province=...
func (h *OSSHandler) Search(c *fiber.Ctx) error {
	query := c.Query("q")
	kbli := c.Query("kbli")
	investmentStatus := c.Query("investment_status")
	province := c.Query("province")
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	records, err := h.repo.Search(c.UserContext(), query, kbli, investmentStatus, province, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"data":  records,
		"total": len(records),
	})
}

// UpsertNIB handles POST /api/v1/oss
func (h *OSSHandler) UpsertNIB(c *fiber.Ctx) error {
	var reg model.OSSNIBRegistration
	if err := c.BodyParser(&reg); err != nil || reg.NIB == "" || reg.BusinessName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body: nib and business_name are required"})
	}

	upserted, err := h.repo.UpsertNIB(c.UserContext(), &reg)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success",
		"data":   upserted,
	})
}

// LinkCompanyNIB handles POST /api/v1/companies/:id/nib
func (h *OSSHandler) LinkCompanyNIB(c *fiber.Ctx) error {
	companyID := c.Params("id")
	if companyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Company ID is required"})
	}

	var req struct {
		NIB string `json:"nib"`
	}

	if err := c.BodyParser(&req); err != nil || req.NIB == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body: nib is required"})
	}

	err := h.repo.LinkCompanyNIB(c.UserContext(), companyID, req.NIB)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Company NIB registration linked successfully",
	})
}
