package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// RequireVisibilityAccess enforces JWT authentication when the request asks for
// private programs or bypasses visibility filtering.
func RequireVisibilityAccess(secretKey string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		visibility := strings.ToLower(strings.TrimSpace(c.Query("visibility", "")))
		if visibility == "" {
			return c.Next()
		}

		if strings.EqualFold(visibility, "ALL") {
			return AuthenticateJWT(secretKey)(c)
		}

		for _, value := range strings.Split(visibility, ",") {
			if strings.EqualFold(strings.TrimSpace(value), "private") {
				return AuthenticateJWT(secretKey)(c)
			}
		}

		return c.Next()
	}
}

// RequireRole returns a Fiber middleware that enforces role-based access control.
// It reads the "role" claim previously set by AuthenticateJWT and checks against allowedRoles.
// Usage: RequireRole("ORG_ADMIN", "DIRECTOR")
func RequireRole(allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRole, ok := c.Locals("role").(string)
		if !ok || userRole == "" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"error":   "MISSING_ROLE",
				"message": "Klaim role tidak ditemukan pada token. Pastikan Anda telah login.",
			})
		}

		for _, allowed := range allowedRoles {
			if allowed == userRole {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error":   "INSUFFICIENT_PERMISSIONS",
			"message": "Anda tidak memiliki wewenang untuk tindakan ini.",
		})
	}
}
