package handler

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"sovera-core-api/internal/model"
	"sovera-core-api/internal/service/ai"
)

type OrganizationAIChatHandler struct {
	chatService *ai.OrganizationAIChatService
	repo        ai.OrganizationAIRepositoryInterface
}

func NewOrganizationAIChatHandler(chatService *ai.OrganizationAIChatService, repo ai.OrganizationAIRepositoryInterface) *OrganizationAIChatHandler {
	return &OrganizationAIChatHandler{
		chatService: chatService,
		repo:        repo,
	}
}

// PostChat processes incoming user chat queries for the organization assistant
func (h *OrganizationAIChatHandler) PostChat(c *fiber.Ctx) error {
	orgIDStr, _ := c.Locals("org_id").(string)
	userIDStr, _ := c.Locals("user_id").(string)
	role, _ := c.Locals("role").(string)

	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error":   "TENANT_CONTEXT_MISSING",
			"message": "Valid org_id is required in session token context",
		})
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error":   "USER_CONTEXT_MISSING",
			"message": "Valid user_id is required in session token context",
		})
	}

	var req model.OrganizationAIChatRequest
	if err := c.BodyParser(&req); err != nil || strings.TrimSpace(req.Message) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "VALIDATION_ERROR",
			"message": "Field 'message' cannot be empty",
		})
	}

	resp, err := h.chatService.ProcessChatMessage(c.Context(), orgID, userID, role, req.Message, req.ConversationID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "LLM_ERROR",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    resp,
		"timestamp": time.Now(),
	})
}

// SearchSignals handles org-filtered corporate signal searches
func (h *OrganizationAIChatHandler) SearchSignals(c *fiber.Ctx) error {
	orgIDStr, _ := c.Locals("org_id").(string)
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error":   "TENANT_CONTEXT_MISSING",
		})
	}

	queryStr := c.Query("q")
	if queryStr == "" {
		queryStr = c.Query("query")
	}

	signals, err := h.repo.SearchSignalsByOrg(c.Context(), orgID, queryStr)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    signals,
		"count":   len(signals),
	})
}

// SearchCompanies handles public company master data search
func (h *OrganizationAIChatHandler) SearchCompanies(c *fiber.Ctx) error {
	queryStr := c.Query("q")
	if queryStr == "" {
		queryStr = c.Query("query")
	}

	companies, err := h.repo.SearchCompaniesPublic(c.Context(), queryStr)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    companies,
		"count":   len(companies),
	})
}

// GetWatchlist lists company watchlist for the organization
func (h *OrganizationAIChatHandler) GetWatchlist(c *fiber.Ctx) error {
	orgIDStr, _ := c.Locals("org_id").(string)
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error":   "TENANT_CONTEXT_MISSING",
		})
	}

	watchlist, err := h.repo.ListWatchlistByOrg(c.Context(), orgID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    watchlist,
		"count":   len(watchlist),
	})
}

// ListConversations lists active chat threads for the logged-in user
func (h *OrganizationAIChatHandler) ListConversations(c *fiber.Ctx) error {
	orgIDStr, _ := c.Locals("org_id").(string)
	userIDStr, _ := c.Locals("user_id").(string)

	orgID, err1 := uuid.Parse(orgIDStr)
	userID, err2 := uuid.Parse(userIDStr)
	if err1 != nil || err2 != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error":   "CONTEXT_MISSING",
		})
	}

	convs, err := h.repo.ListConversationsByUser(c.Context(), orgID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    convs,
		"count":   len(convs),
	})
}

// GetConversationMessages returns full history of a thread owned by the user
func (h *OrganizationAIChatHandler) GetConversationMessages(c *fiber.Ctx) error {
	orgIDStr, _ := c.Locals("org_id").(string)
	userIDStr, _ := c.Locals("user_id").(string)

	orgID, err1 := uuid.Parse(orgIDStr)
	userID, err2 := uuid.Parse(userIDStr)
	convID, err3 := uuid.Parse(c.Params("id"))

	if err1 != nil || err2 != nil || err3 != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "INVALID_PARAMETER",
		})
	}

	logs, err := h.repo.GetConversationHistory(c.Context(), orgID, userID, convID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "CONVERSATION_NOT_FOUND",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    logs,
		"count":   len(logs),
	})
}

// ArchiveConversation soft-deletes a conversation thread
func (h *OrganizationAIChatHandler) ArchiveConversation(c *fiber.Ctx) error {
	orgIDStr, _ := c.Locals("org_id").(string)
	userIDStr, _ := c.Locals("user_id").(string)

	orgID, err1 := uuid.Parse(orgIDStr)
	userID, err2 := uuid.Parse(userIDStr)
	convID, err3 := uuid.Parse(c.Params("id"))

	if err1 != nil || err2 != nil || err3 != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "INVALID_PARAMETER",
		})
	}

	if err := h.repo.ArchiveConversation(c.Context(), orgID, userID, convID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Conversation archived successfully",
	})
}

// AdminListConversations provides read-only overview of all org chat threads for ORG_ADMIN
func (h *OrganizationAIChatHandler) AdminListConversations(c *fiber.Ctx) error {
	role, _ := c.Locals("role").(string)
	if !strings.EqualFold(role, "ORG_ADMIN") && !strings.EqualFold(role, "DIRECTOR") && !strings.EqualFold(role, "ADMIN") {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error":   "FORBIDDEN",
			"message": "Only organization administrators can view tenant-wide chat logs",
		})
	}

	orgIDStr, _ := c.Locals("org_id").(string)
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error":   "TENANT_CONTEXT_MISSING",
		})
	}

	convs, err := h.repo.ListConversationsByOrgAdmin(c.Context(), orgID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    convs,
		"count":   len(convs),
	})
}

// AdminGetConversationMessages provides read-only message history of an org thread for ORG_ADMIN
func (h *OrganizationAIChatHandler) AdminGetConversationMessages(c *fiber.Ctx) error {
	role, _ := c.Locals("role").(string)
	if !strings.EqualFold(role, "ORG_ADMIN") && !strings.EqualFold(role, "DIRECTOR") && !strings.EqualFold(role, "ADMIN") {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error":   "FORBIDDEN",
			"message": "Only organization administrators can view tenant-wide chat logs",
		})
	}

	orgIDStr, _ := c.Locals("org_id").(string)
	orgID, err1 := uuid.Parse(orgIDStr)
	convID, err2 := uuid.Parse(c.Params("id"))

	if err1 != nil || err2 != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "INVALID_PARAMETER",
		})
	}

	logs, err := h.repo.GetAdminConversationHistory(c.Context(), orgID, convID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    logs,
		"count":   len(logs),
	})
}
