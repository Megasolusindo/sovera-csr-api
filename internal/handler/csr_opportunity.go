package handler

import (
	"strconv"
	"time"

	"sovera-core-api/internal/model"
	"sovera-core-api/internal/repository"

	"github.com/gofiber/fiber/v2"
)

type CSROpportunityHandler struct {
	oppRepo *repository.CSROpportunityRepository
}

func NewCSROpportunityHandler(oppRepo *repository.CSROpportunityRepository) *CSROpportunityHandler {
	return &CSROpportunityHandler{oppRepo: oppRepo}
}

type CreateOpportunityPayload struct {
	CompanyID      string  `json:"company_id"`
	Title          string  `json:"title"`
	Description    string  `json:"description"`
	Category       string  `json:"category"` // Education, Health, Environment, Economic, Social
	TargetLocation string  `json:"target_location"`
	BudgetAmount   float64 `json:"budget_amount"`
	OpenUntilDays  int     `json:"open_until_days"`
}

// CreateOpportunity handles Corporate CSR RFP/Grant creation
func (h *CSROpportunityHandler) CreateOpportunity(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	orgID, _ := c.Locals("org_id").(string)
	companyIDLocal, _ := c.Locals("company_id").(string)

	var payload CreateOpportunityPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false, "error": "INVALID_BODY", "message": err.Error(),
		})
	}

	companyID := payload.CompanyID
	if companyID == "" {
		companyID = companyIDLocal
	}

	if companyID == "" || payload.Title == "" || payload.Category == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false, "error": "MISSING_FIELDS", "message": "company_id, title, and category are required",
		})
	}

	var openUntil *time.Time
	if payload.OpenUntilDays > 0 {
		t := time.Now().AddDate(0, 0, payload.OpenUntilDays)
		openUntil = &t
	}

	opp := model.CSROpportunity{
		CompanyID:       companyID,
		TenantID:        orgID,
		Title:           payload.Title,
		Description:     payload.Description,
		Category:        payload.Category,
		TargetLocation:  payload.TargetLocation,
		BudgetAmount:    payload.BudgetAmount,
		OpenUntil:       openUntil,
		Status:          "OPEN",
		CreatedByUserID: &userID,
	}

	created, err := h.oppRepo.Create(c.Context(), opp)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false, "error": "CREATE_FAILED", "message": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    created,
		"message": "Program penyaluran CSR Opportunity berhasil diterbitkan!",
	})
}

// ListOpportunities returns open CSR opportunities for NGOs to explore
func (h *CSROpportunityHandler) ListOpportunities(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("limit", "20"))
	category := c.Query("category")
	search := c.Query("search")

	items, total, err := h.oppRepo.ListPublic(c.Context(), page, pageSize, category, search)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false, "error": "QUERY_FAILED", "message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    items,
		"meta": fiber.Map{
			"page":       page,
			"page_size":  pageSize,
			"total":      total,
			"total_page": (total + pageSize - 1) / pageSize,
		},
	})
}
