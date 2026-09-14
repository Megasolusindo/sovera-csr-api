package middleware

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestTenantRateLimit_EnforcesQuota(t *testing.T) {
	app := fiber.New()
	store := NewRateLimiterStore()

	// Rate limit: 2 requests / minute per tenant
	limiter := TenantRateLimit(store, 2, 1*time.Minute, "Pemicu Generasi AI Proposal")

	app.Post("/test-ai", func(c *fiber.Ctx) error {
		c.Locals("org_id", "org-tenant-123")
		return c.Next()
	}, limiter, func(c *fiber.Ctx) error {
		return c.SendString("AI Generation Success")
	})

	// Request 1: Should pass (1/2)
	req1 := httptest.NewRequest("POST", "/test-ai", nil)
	resp1, err := app.Test(req1)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp1.StatusCode)

	// Request 2: Should pass (2/2)
	req2 := httptest.NewRequest("POST", "/test-ai", nil)
	resp2, err := app.Test(req2)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp2.StatusCode)

	// Request 3: Exceeds limit (3/2) -> Should return 429 Too Many Requests
	req3 := httptest.NewRequest("POST", "/test-ai", nil)
	resp3, err := app.Test(req3)
	assert.NoError(t, err)
	assert.Equal(t, 429, resp3.StatusCode)
	assert.NotEmpty(t, resp3.Header.Get("Retry-After"))
}
