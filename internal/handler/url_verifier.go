package handler

import (
	"context"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"sovera-core-api/internal/service/urlverifier"
)

type URLVerifierHandler struct {
	linkedinVerifier  *urlverifier.LinkedInVerifier
	instagramVerifier *urlverifier.InstagramVerifier
}

func NewURLVerifierHandler(linkedinVerifier *urlverifier.LinkedInVerifier) *URLVerifierHandler {
	if linkedinVerifier == nil {
		linkedinVerifier = urlverifier.NewLinkedInVerifier()
	}
	return &URLVerifierHandler{
		linkedinVerifier:  linkedinVerifier,
		instagramVerifier: urlverifier.NewInstagramVerifier(),
	}
}

type VerifyLinkedInRequest struct {
	URL string `json:"url" xml:"url" form:"url"`
}

// VerifyLinkedInURL handles verification of LinkedIn profile, company, school, job, or article links.
// POST /api/v1/url/verify-linkedin OR GET /api/v1/url/verify-linkedin?url=...
func (h *URLVerifierHandler) VerifyLinkedInURL(c *fiber.Ctx) error {
	var req VerifyLinkedInRequest

	if c.Method() == fiber.MethodPost {
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"error":   "INVALID_PAYLOAD",
				"message": "Malformed JSON payload",
			})
		}
	}

	// Fallback to query param or form param
	if strings.TrimSpace(req.URL) == "" {
		req.URL = c.Query("url")
	}

	rawURL := strings.TrimSpace(req.URL)
	if rawURL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "MISSING_URL",
			"message": "URL parameter 'url' is required",
		})
	}

	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	result := h.linkedinVerifier.ValidateLinkedInURL(ctx, rawURL)

	statusCode := fiber.StatusOK
	if !result.IsValid && result.HealthStatus == urlverifier.StatusDisabledDeadLink {
		statusCode = fiber.StatusUnprocessableEntity
	}

	return c.Status(statusCode).JSON(fiber.Map{
		"success": result.IsValid,
		"data":    result,
	})
}

// VerifyInstagramURL handles verification of Instagram profile, post, reel, or story links.
// POST /api/v1/url/verify-instagram OR GET /api/v1/url/verify-instagram?url=...
func (h *URLVerifierHandler) VerifyInstagramURL(c *fiber.Ctx) error {
	var req VerifyLinkedInRequest

	if c.Method() == fiber.MethodPost {
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"error":   "INVALID_PAYLOAD",
				"message": "Malformed JSON payload",
			})
		}
	}

	if strings.TrimSpace(req.URL) == "" {
		req.URL = c.Query("url")
	}

	rawURL := strings.TrimSpace(req.URL)
	if rawURL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "MISSING_URL",
			"message": "URL parameter 'url' is required",
		})
	}

	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	result := h.instagramVerifier.ValidateInstagramURL(ctx, rawURL)

	statusCode := fiber.StatusOK
	if !result.IsValid && result.HealthStatus == urlverifier.StatusDisabledDeadLink {
		statusCode = fiber.StatusUnprocessableEntity
	}

	return c.Status(statusCode).JSON(fiber.Map{
		"success": result.IsValid,
		"data":    result,
	})
}
