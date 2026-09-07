package handler

import (
	"io"
	"strings"

	"github.com/gofiber/fiber/v2"
	"sovera-core-api/internal/repository"
	"sovera-core-api/internal/service/storage"
)

type TemplateHandler struct {
	repo         *repository.TemplateRepository
	tokenLogRepo *repository.TokenLogRepository
	storage      storage.StorageService
}

func NewTemplateHandler(repo *repository.TemplateRepository, tokenLogRepo *repository.TokenLogRepository, storage storage.StorageService) *TemplateHandler {
	return &TemplateHandler{
		repo:         repo,
		tokenLogRepo: tokenLogRepo,
		storage:      storage,
	}
}

func (h *TemplateHandler) GetTokenUsage(c *fiber.Ctx) error {
	orgID, ok := c.Locals("org_id").(string)
	if !ok || orgID == "" {
		orgID = "77123aaa-8819-4c12-99a1-00123456789a"
	}

	summary, err := h.tokenLogRepo.GetTenantTokenSummary(c.Context(), orgID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "FETCH_FAILED",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    summary,
	})
}

func (h *TemplateHandler) GetTemplates(c *fiber.Ctx) error {
	orgID, ok := c.Locals("org_id").(string)
	if !ok || orgID == "" {
		orgID = "77123aaa-8819-4c12-99a1-00123456789a"
	}

	template, err := h.repo.GetTenantTemplate(c.Context(), orgID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "FETCH_FAILED",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    template,
	})
}

func (h *TemplateHandler) UploadTemplate(c *fiber.Ctx) error {
	orgID, ok := c.Locals("org_id").(string)
	if !ok || orgID == "" {
		orgID = "77123aaa-8819-4c12-99a1-00123456789a"
	}

	fileType := strings.ToLower(c.Query("type", "pptx"))
	if fileType != "pptx" && fileType != "docx" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "INVALID_TYPE",
			"message": "Template type must be either pptx or docx",
		})
	}

	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "NO_FILE",
			"message": "Please attach a valid .pptx or .docx template file",
		})
	}

	src, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	defer src.Close()

	content, err := io.ReadAll(src)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	s3Key, err := h.storage.SaveTemplate(c.Context(), orgID, fileType, content)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "STORAGE_ERROR",
			"message": err.Error(),
		})
	}

	if err := h.repo.SaveTenantTemplateKey(c.Context(), orgID, fileType, s3Key); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "DB_ERROR",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Master template uploaded successfully",
		"s3_key":  s3Key,
	})
}

func (h *TemplateHandler) ResetTemplate(c *fiber.Ctx) error {
	orgID, ok := c.Locals("org_id").(string)
	if !ok || orgID == "" {
		orgID = "77123aaa-8819-4c12-99a1-00123456789a"
	}

	fileType := strings.ToLower(c.Params("type"))
	if fileType != "pptx" && fileType != "docx" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "INVALID_TYPE",
			"message": "Template type must be either pptx or docx",
		})
	}

	if err := h.repo.ClearTenantTemplateKey(c.Context(), orgID, fileType); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "RESET_FAILED",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Master template reset to system default",
	})
}
