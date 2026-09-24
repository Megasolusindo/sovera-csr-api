package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"sovera-core-api/internal/repository"
)

type KBLIHandler struct {
	repo *repository.KBLIRepository
}

func NewKBLIHandler(repo *repository.KBLIRepository) *KBLIHandler {
	return &KBLIHandler{repo: repo}
}

// ListKBLIReferences handles GET /api/v1/kbli
func (h *KBLIHandler) ListKBLIReferences(c *fiber.Ctx) error {
	category := c.Query("category")
	relevance := c.Query("relevance")

	kbliList, err := h.repo.GetAll(c.UserContext(), category, relevance)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"data":  kbliList,
		"total": len(kbliList),
	})
}

// SearchKBLI handles GET /api/v1/kbli/search?q=...
func (h *KBLIHandler) SearchKBLI(c *fiber.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "search query parameter 'q' is required"})
	}

	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	kbliList, err := h.repo.Search(c.UserContext(), query, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"data":  kbliList,
		"total": len(kbliList),
	})
}

// GetKBLIByCode handles GET /api/v1/kbli/:code
func (h *KBLIHandler) GetKBLIByCode(c *fiber.Ctx) error {
	code := c.Params("code")
	if code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "KBLI code is required"})
	}

	kbli, err := h.repo.GetByCode(c.UserContext(), code)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"data": kbli})
}

// LinkCompanyKBLI handles POST /api/v1/companies/:id/kbli
func (h *KBLIHandler) LinkCompanyKBLI(c *fiber.Ctx) error {
	companyID := c.Params("id")
	if companyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Company ID is required"})
	}

	var req struct {
		KBLICode string `json:"kbli_code"`
	}

	if err := c.BodyParser(&req); err != nil || req.KBLICode == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body: kbli_code is required"})
	}

	err := h.repo.LinkCompanyKBLI(c.UserContext(), companyID, req.KBLICode)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Company KBLI code linked successfully",
	})
}
