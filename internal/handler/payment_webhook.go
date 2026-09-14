package handler

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"sovera-core-api/internal/payment/midtrans"
	"sovera-core-api/internal/service"
)

type PaymentWebhookHandler struct {
	subService *service.SubscriptionService
}

func NewPaymentWebhookHandler(subService *service.SubscriptionService) *PaymentWebhookHandler {
	return &PaymentWebhookHandler{subService: subService}
}

// HandleMidtransWebhook processes POST /api/v1/webhooks/midtrans
func (h *PaymentWebhookHandler) HandleMidtransWebhook(c *fiber.Ctx) error {
	var notif midtrans.WebhookNotification
	if err := json.Unmarshal(c.Body(), &notif); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "error", "message": "Failed to parse webhook JSON payload: " + err.Error(),
		})
	}

	if err := h.subService.ProcessMidtransWebhook(c.Context(), &notif); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"status": "error", "message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status": "ok", "message": "Midtrans payment notification processed successfully",
	})
}
