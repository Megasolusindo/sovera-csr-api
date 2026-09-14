package handler

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"sovera-core-api/internal/service"
)

type FaspayWebhookHandler struct {
	subService *service.SubscriptionService
}

func NewFaspayWebhookHandler(subService *service.SubscriptionService) *FaspayWebhookHandler {
	return &FaspayWebhookHandler{subService: subService}
}

type FaspayWebhookPayload struct {
	// SNAP BI Callback Fields
	OriginalPartnerReferenceNo string `json:"originalPartnerReferenceNo,omitempty"`
	VirtualAccountNo           string `json:"virtualAccountNo,omitempty"`
	PaymentRequestId           string `json:"paymentRequestId,omitempty"`
	PaymentFlagStatus          string `json:"paymentFlagStatus,omitempty"` // "00": Success

	// Legacy / Standard Fields
	MerchantID        string `json:"merchant_id,omitempty"`
	OrderID           string `json:"order_id,omitempty"`
	StatusCode        string `json:"status_code,omitempty"`
	TransactionStatus string `json:"transaction_status,omitempty"` // 'SETTLEMENT', 'PAID', 'EXPIRED', 'FAILED'
	GrossAmount       string `json:"gross_amount,omitempty"`
	PaymentType       string `json:"payment_type,omitempty"`
	Signature         string `json:"signature,omitempty"`
}

// HandleFaspayWebhook processes POST /api/v1/webhooks/faspay
func (h *FaspayWebhookHandler) HandleFaspayWebhook(c *fiber.Ctx) error {
	var payload FaspayWebhookPayload
	if err := json.Unmarshal(c.Body(), &payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"responseCode": "4002700", "responseMessage": "Failed to parse Faspay webhook JSON payload: " + err.Error(),
			"status": "error", "message": "Failed to parse Faspay webhook JSON payload: " + err.Error(),
		})
	}

	orderID := payload.OrderID
	if orderID == "" {
		orderID = payload.OriginalPartnerReferenceNo
	}

	if orderID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"responseCode": "4002700", "responseMessage": "order_id or originalPartnerReferenceNo is required",
			"status": "error", "message": "order_id or originalPartnerReferenceNo is required",
		})
	}

	paymentType := payload.PaymentType
	if paymentType == "" {
		paymentType = "faspay_snap_va"
	}

	isSuccess := payload.TransactionStatus == "SETTLEMENT" ||
		payload.TransactionStatus == "PAID" ||
		payload.StatusCode == "200" ||
		payload.PaymentFlagStatus == "00"

	if isSuccess {
		if err := h.subService.ProcessFaspayWebhook(c.Context(), orderID, paymentType, payload); err != nil {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
				"responseCode": "5002700", "responseMessage": err.Error(),
				"status": "error", "message": err.Error(),
			})
		}
	}

	return c.JSON(fiber.Map{
		"responseCode":    "2002700",
		"responseMessage": "Successful",
		"status":          "ok",
		"message":         "Faspay payment notification processed successfully",
	})
}
