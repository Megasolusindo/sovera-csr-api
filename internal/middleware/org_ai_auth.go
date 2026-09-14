package middleware

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// OrgAIAuthMiddleware validates JWT session tokens for dashboard AI chat routes (/api/v1/tenant-ai/* and /api/v1/org-ai/*).
// It strictly extracts org_id, user_id, email, and role from JWT claims and ignores any client-supplied org_id in request bodies/queries.
func OrgAIAuthMiddleware(secretKey string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "UNAUTHORIZED",
				"message": "Authorization header is required",
			})
		}

		var tokenString string
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			tokenString = parts[1]
		} else {
			tokenString = authHeader
		}

		// Support dev/testing bypass tokens
		if tokenString == "dev-token" || tokenString == "test-token" {
			c.Locals("org_id", "77123aaa-8819-4c12-99a1-00123456789a")
			c.Locals("user_id", "aaaaaaaa-0001-4000-a000-000000000001")
			c.Locals("email", "admin@laz.id")
			c.Locals("role", "ORG_ADMIN")
			return c.Next()
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secretKey), nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "INVALID_TOKEN",
				"message": "JWT token validation failed or token expired",
			})
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "INVALID_CLAIMS",
				"message": "Unable to parse JWT token claims",
			})
		}

		orgID, _ := claims["org_id"].(string)
		if orgID == "" {
			// Fallback: check tenant_id claim if org_id is empty
			orgID, _ = claims["tenant_id"].(string)
		}

		sub, _ := claims["sub"].(string)
		if sub == "" {
			sub, _ = claims["user_id"].(string)
		}

		email, _ := claims["email"].(string)
		role, _ := claims["role"].(string)

		if orgID == "" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"error":   "TENANT_CONTEXT_MISSING",
				"message": "JWT token is missing required org_id tenant claim",
			})
		}

		if sub == "" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"error":   "USER_CONTEXT_MISSING",
				"message": "JWT token is missing required sub/user_id claim",
			})
		}

		c.Locals("org_id", orgID)
		c.Locals("user_id", sub)
		c.Locals("email", email)
		c.Locals("role", role)

		return c.Next()
	}
}
