package handler

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"sovera-core-api/internal/payment"
	"sovera-core-api/internal/payment/midtrans"
	"sovera-core-api/internal/service"
)

type PaymentWebhookHandler struct {
	subService    *service.SubscriptionService
	paymentGateway payment.PaymentGateway
}

func NewPaymentWebhookHandler(subService *service.SubscriptionService, paymentGateway payment.PaymentGateway) *PaymentWebhookHandler {
	return &PaymentWebhookHandler{subService: subService, paymentGateway: paymentGateway}
}

// HandleMidtransWebhook processes POST /api/v1/webhooks/midtrans
func (h *PaymentWebhookHandler) HandleMidtransWebhook(c *fiber.Ctx) error {
	var notif midtrans.WebhookNotification
	if err := json.Unmarshal(c.Body(), &notif); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "error", "message": "Failed to parse webhook JSON payload",
		})
	}

	if h.paymentGateway != nil && !h.paymentGateway.VerifyWebhookSignature(notif.OrderID, notif.StatusCode, notif.GrossAmount, notif.SignatureKey) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status": "error", "message": "Invalid Midtrans webhook signature",
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
