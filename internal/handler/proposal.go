package handler

import (
	"strconv"

	"sovera-core-api/internal/model"
	"sovera-core-api/internal/repository"

	"github.com/gofiber/fiber/v2"
)

type ProposalHandler struct {
	proposalRepo *repository.ProposalRepository
}

func NewProposalHandler(proposalRepo *repository.ProposalRepository) *ProposalHandler {
	return &ProposalHandler{proposalRepo: proposalRepo}
}

type SubmitProposalPayload struct {
	OpportunityID   *string `json:"opportunity_id"`
	NGOProgramID    *string `json:"ngo_program_id"`
	CompanyID       string  `json:"company_id"`
	Title           string  `json:"title"`
	Summary         string  `json:"summary"`
	ProposalFileURL string  `json:"proposal_file_url"`
	BudgetRequested float64 `json:"budget_requested"`
}

// SubmitProposal handles NGO proposal submissions to Corporate Opportunities
func (h *ProposalHandler) SubmitProposal(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	orgID, _ := c.Locals("org_id").(string)

	var payload SubmitProposalPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false, "error": "INVALID_BODY", "message": err.Error(),
		})
	}

	if payload.CompanyID == "" || payload.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false, "error": "MISSING_FIELDS", "message": "company_id and title are required",
		})
	}

	p := model.Proposal{
		OpportunityID:     payload.OpportunityID,
		NGOProgramID:      payload.NGOProgramID,
		OrgTenantID:       orgID,
		CompanyID:         payload.CompanyID,
		Title:             payload.Title,
		Summary:           payload.Summary,
		ProposalFileURL:   payload.ProposalFileURL,
		BudgetRequested:   payload.BudgetRequested,
		Status:            model.ProposalStatusSubmitted,
		SubmittedByUserID: &userID,
	}

	created, err := h.proposalRepo.Create(c.Context(), p)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false, "error": "SUBMIT_FAILED", "message": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    created,
		"message": "Proposal berhasil dikirim ke pihak Korporasi!",
	})
}

// ListSentProposals returns proposals submitted by the current NGO tenant
func (h *ProposalHandler) ListSentProposals(c *fiber.Ctx) error {
	orgID, _ := c.Locals("org_id").(string)
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("limit", "20"))

	items, total, err := h.proposalRepo.ListByOrgTenant(c.Context(), orgID, page, pageSize)
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

// ListIncomingProposals returns proposals received by the current Corporate tenant
func (h *ProposalHandler) ListIncomingProposals(c *fiber.Ctx) error {
	companyID, _ := c.Locals("company_id").(string)
	if companyID == "" {
		companyID = c.Query("company_id")
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("limit", "20"))

	items, total, err := h.proposalRepo.ListByCorpTenant(c.Context(), companyID, page, pageSize)
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

type UpdateProposalStatusPayload struct {
	Status        model.ProposalStatus `json:"status"` // UNDER_REVIEW | MEETING | ACCEPTED | REJECTED
	ReviewerNotes string               `json:"reviewer_notes"`
}

// UpdateProposalStatus allows Corporate CSR Managers to update proposal review status
func (h *ProposalHandler) UpdateProposalStatus(c *fiber.Ctx) error {
	proposalID := c.Params("id")

	var payload UpdateProposalStatusPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false, "error": "INVALID_BODY", "message": err.Error(),
		})
	}

	updated, err := h.proposalRepo.UpdateStatus(c.Context(), proposalID, payload.Status, payload.ReviewerNotes)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false, "error": "UPDATE_FAILED", "message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    updated,
		"message": "Status review proposal berhasil diperbarui.",
	})
}
