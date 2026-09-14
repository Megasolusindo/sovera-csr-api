package handler

import (
	"github.com/gofiber/fiber/v2"
	"sovera-core-api/internal/model"
	"sovera-core-api/internal/service"
)

type SubscriptionHandler struct {
	subService *service.SubscriptionService
}

func NewSubscriptionHandler(subService *service.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{subService: subService}
}

// ListPlans handles GET /api/v1/subscription/plans
func (h *SubscriptionHandler) ListPlans(c *fiber.Ctx) error {
	plans, err := h.subService.ListPlans(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false, "error": "FETCH_PLANS_FAILED", "message": err.Error(),
		})
	}
	return c.JSON(fiber.Map{
		"success": true,
		"data":    plans,
	})
}

// UpsertPlan handles POST /api/v1/admin/plans & POST /api/v1/subscription/plans
func (h *SubscriptionHandler) UpsertPlan(c *fiber.Ctx) error {
	var req struct {
		Code          string  `json:"code"`
		Name          string  `json:"name"`
		Description   string  `json:"description"`
		TargetPersona string  `json:"target_persona"`
		PriceMonthly  float64 `json:"price_monthly"`
		PriceYearly   float64 `json:"price_yearly"`
		CrawlQuota    int     `json:"crawl_quota"`
		AIQueryQuota  int     `json:"ai_query_quota"`
		MaxUserSeats  int     `json:"max_user_seats"`
		IsActive      *bool   `json:"is_active"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false, "error": "INVALID_BODY", "message": err.Error(),
		})
	}

	if req.Code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false, "error": "MISSING_CODE", "message": "code parameter is required",
		})
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	plan := model.SubscriptionPlan{
		Code:          req.Code,
		Name:          req.Name,
		Description:   req.Description,
		TargetPersona: req.TargetPersona,
		PriceMonthly:  req.PriceMonthly,
		PriceYearly:   req.PriceYearly,
		CrawlQuota:    req.CrawlQuota,
		AIQueryQuota:  req.AIQueryQuota,
		MaxUserSeats:  req.MaxUserSeats,
		IsActive:      isActive,
	}

	if err := h.subService.UpsertPlan(c.Context(), plan); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false, "error": "UPSERT_PLAN_FAILED", "message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Subscription plan upserted successfully",
		"data":    plan,
	})
}

// GetSubscription handles GET /api/v1/subscription/me
func (h *SubscriptionHandler) GetSubscription(c *fiber.Ctx) error {
	orgID, ok := c.Locals("org_id").(string)
	if !ok || orgID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false, "error": "UNAUTHORIZED", "message": "Missing org_id in context",
		})
	}

	usage, err := h.subService.GetSubscriptionUsage(c.Context(), orgID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false, "error": "FETCH_SUBSCRIPTION_FAILED", "message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    usage,
	})
}

// Checkout handles POST /api/v1/subscription/checkout
func (h *SubscriptionHandler) Checkout(c *fiber.Ctx) error {
	orgID, ok := c.Locals("org_id").(string)
	if !ok || orgID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false, "error": "UNAUTHORIZED", "message": "Missing org_id in context",
		})
	}

	var req service.CheckoutRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false, "error": "INVALID_BODY", "message": err.Error(),
		})
	}

	if req.PlanID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false, "error": "MISSING_PLAN_ID", "message": "plan_id is required",
		})
	}

	resp, err := h.subService.Checkout(c.Context(), orgID, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false, "error": "CHECKOUT_FAILED", "message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    resp,
	})
}

// ListInvoices handles GET /api/v1/subscription/invoices
func (h *SubscriptionHandler) ListInvoices(c *fiber.Ctx) error {
	orgID, ok := c.Locals("org_id").(string)
	if !ok || orgID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false, "error": "UNAUTHORIZED", "message": "Missing org_id in context",
		})
	}

	invoices, err := h.subService.ListInvoices(c.Context(), orgID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false, "error": "FETCH_INVOICES_FAILED", "message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    invoices,
	})
}
