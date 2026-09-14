package middleware

import (
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

type tenantWindow struct {
	mu          sync.Mutex
	count       int
	windowStart time.Time
}

type RateLimiterStore struct {
	mu      sync.RWMutex
	tenants map[string]*tenantWindow
}

func NewRateLimiterStore() *RateLimiterStore {
	store := &RateLimiterStore{
		tenants: make(map[string]*tenantWindow),
	}

	// Background cleanup of inactive tenant windows every 10 minutes
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		for range ticker.C {
			store.mu.Lock()
			now := time.Now()
			for orgID, tw := range store.tenants {
				tw.mu.Lock()
				if now.Sub(tw.windowStart) > 15*time.Minute {
					delete(store.tenants, orgID)
				}
				tw.mu.Unlock()
			}
			store.mu.Unlock()
		}
	}()

	return store
}

// TenantRateLimit creates a Fiber middleware enforcing rate limits per tenant (org_id).
// limit: maximum allowed requests per tenant inside windowDuration
// windowDuration: time window duration (e.g. 1 minute)
// resourceName: descriptive name for the resource (e.g. "panggilan AI Gemini Proposal")
func TenantRateLimit(store *RateLimiterStore, limit int, windowDuration time.Duration, resourceName string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		orgID, ok := c.Locals("org_id").(string)
		if !ok || orgID == "" {
			orgID = "anonymous"
		}

		store.mu.Lock()
		tw, exists := store.tenants[orgID]
		if !exists {
			tw = &tenantWindow{
				windowStart: time.Now(),
				count:       0,
			}
			store.tenants[orgID] = tw
		}
		store.mu.Unlock()

		tw.mu.Lock()
		now := time.Now()
		if now.Sub(tw.windowStart) > windowDuration {
			tw.windowStart = now
			tw.count = 0
		}

		if tw.count >= limit {
			resetInSeconds := int((windowDuration - now.Sub(tw.windowStart)).Seconds())
			if resetInSeconds < 1 {
				resetInSeconds = 1
			}
			tw.mu.Unlock()

			c.Set("Retry-After", fmt.Sprintf("%d", resetInSeconds))
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"success": false,
				"error":   "TENANT_RATE_LIMIT_EXCEEDED",
				"message": fmt.Sprintf("Batas kuota %s per tenant (%d req/%v) telah tercapai. Silakan coba lagi dalam %d detik.", resourceName, limit, windowDuration, resetInSeconds),
			})
		}

		tw.count++
		tw.mu.Unlock()

		return c.Next()
	}
}
