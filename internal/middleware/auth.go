package middleware

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/golang-jwt/jwt/v5"
)

// AuthenticateJWT creates a Fiber middleware that validates JWT Bearer tokens
// and extracts tenant claims. If rdb is provided, it also checks token revocation
// via the JTI (JWT ID) blacklist stored in Redis.
func AuthenticateJWT(secretKey string, rdb ...*redis.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get Redis client from variadic parameter if provided
		var redisClient *redis.Client
		if len(rdb) > 0 && rdb[0] != nil {
			redisClient = rdb[0]
		}

		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "MISSING_TOKEN",
				"message": "Authorization header is required",
			})
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "INVALID_TOKEN_FORMAT",
				"message": "Authorization header format must be Bearer <token>",
			})
		}

		tokenString := parts[1]
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

// Check token revocation via JTI blacklist in Redis (if Redis client provided)
	if redisClient != nil {
		jti, _ := claims["jti"].(string)
		if jti != "" {
			blacklisted, err := redisClient.SIsMember(c.Context(), "revoked_tokens", jti).Result()
			if err == nil && blacklisted {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"success": false,
					"error":   "TOKEN_REVOKED",
					"message": "Token has been revoked (user logged out)",
				})
			}
		}
	}

		orgID, _ := claims["org_id"].(string)
		sub, _ := claims["sub"].(string)
		email, _ := claims["email"].(string)
		role, _ := claims["role"].(string)

		if orgID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "MISSING_ORG_ID",
				"message": "JWT token is missing required org_id tenant claim",
			})
		}

		c.Locals("org_id", orgID)
		c.Locals("user_id", sub)
		c.Locals("email", email)
		c.Locals("role", role)

		return c.Next()
	}
}
