package handler

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"

	"sovera-core-api/internal/repository"
	"sovera-core-api/internal/service/ai"
)

type ProgramHandler struct {
	programRepo   *repository.ProgramRepository
	geminiService *ai.GeminiService
}

func NewProgramHandler(programRepo *repository.ProgramRepository, geminiService *ai.GeminiService) *ProgramHandler {
	return &ProgramHandler{
		programRepo:   programRepo,
		geminiService: geminiService,
	}
}

type CreateProgramPayload struct {
	Title               string   `json:"title"`
	Description         string   `json:"description"`
	PrimaryCluster      string   `json:"primary_cluster"`
	TargetSDGs          []string `json:"target_sdgs"`
	AsnafCategory       string   `json:"asnaf_category"`
	ESGPillar           string   `json:"esg_pillar"`
	TargetBeneficiaries string   `json:"target_beneficiaries"`
}

func (h *ProgramHandler) ListPrograms(c *fiber.Ctx) error {
	orgID, ok := c.Locals("org_id").(string)
	if !ok || orgID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "UNAUTHORIZED",
			"message": "Organization context is required",
		})
	}

	programs, err := h.programRepo.ListPrograms(c.Context(), orgID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "QUERY_FAILED",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": programs,
	})
}

func (h *ProgramHandler) CreateProgram(c *fiber.Ctx) error {
	orgID, ok := c.Locals("org_id").(string)
	if !ok || orgID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "UNAUTHORIZED",
			"message": "Organization context is required",
		})
	}

	var payload CreateProgramPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "INVALID_PAYLOAD",
			"message": "Failed to parse program payload",
		})
	}

	if payload.Title == "" || payload.Description == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"error":   "MISSING_FIELDS",
			"message": "Title and description are required",
		})
	}

	if payload.PrimaryCluster == "" {
		payload.PrimaryCluster = "COMMUNITY_DEVELOPMENT"
	}
	if payload.ESGPillar == "" {
		payload.ESGPillar = "SOCIAL"
	}

	// Auto-generate vector embedding (1536 dim) for program text
	textToEmbed := fmt.Sprintf("%s %s %s %s %s %s", payload.Title, payload.Description, payload.PrimaryCluster, strings.Join(payload.TargetSDGs, " "), payload.AsnafCategory, payload.ESGPillar)
	embedding, err := h.geminiService.GenerateEmbedding(c.Context(), textToEmbed)
	if err != nil {
		embedding = make([]float32, 1536)
	}

	prog, err := h.programRepo.CreateProgram(
		c.Context(), orgID, payload.Title, payload.Description,
		payload.PrimaryCluster, payload.TargetSDGs, payload.AsnafCategory, payload.ESGPillar, payload.TargetBeneficiaries, embedding,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "CREATE_FAILED",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(prog)
}
