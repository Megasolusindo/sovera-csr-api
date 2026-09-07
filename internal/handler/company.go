package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"sovera-core-api/internal/repository"
)

type CompanyHandler struct {
	repo *repository.CompanyRepository
}

func NewCompanyHandler(repo *repository.CompanyRepository) *CompanyHandler {
	return &CompanyHandler{repo: repo}
}

// ListCompanies returns a paginated list of companies.
func (h *CompanyHandler) ListCompanies(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	search := c.Query("search", "")
	sector := c.Query("sector", "")
	verificationStatus := c.Query("verification_status", "")

	companies, total, err := h.repo.ListCompanies(c.Context(), limit, offset, search, sector, verificationStatus)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "QUERY_FAILED",
			"message": err.Error(),
		})
	}

	stats, _ := h.repo.GetCompanyStats(c.Context())

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": companies,
		"pagination": fiber.Map{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
		"stats": stats,
	})
}

// GetCompany returns details for a single company by ID or Slug.
func (h *CompanyHandler) GetCompany(c *fiber.Ctx) error {
	idOrSlug := c.Params("id")
	company, err := h.repo.GetCompanyByID(c.Context(), idOrSlug)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "COMPANY_NOT_FOUND",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": company,
	})
}
