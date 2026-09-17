package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// VerifyHMAC creates a Fiber middleware that validates incoming webhooks using multi-layer verification.
func VerifyHMAC(secretKey string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Fail closed if secret key is not configured
		if secretKey == "" {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"error":   "WEBHOOK_NOT_CONFIGURED",
				"message": "Webhook secret key is not configured on the server",
			})
		}

		// 1. Check Query Parameter Token (?secret=... or ?token=...)
		queryToken := c.Query("secret")
		if queryToken == "" {
			queryToken = c.Query("token")
		}
		if queryToken != "" && queryToken == secretKey {
			return c.Next()
		}

		// 2. Check Direct Secret Header (X-Webhook-Secret)
		webhookSecretHeader := c.Get("X-Webhook-Secret")
		if webhookSecretHeader != "" && webhookSecretHeader == secretKey {
			return c.Next()
		}

		// 3. Check Authorization Bearer Header (Authorization: Bearer <secretKey>)
		authHeader := c.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			bearerToken := strings.TrimPrefix(authHeader, "Bearer ")
			if bearerToken == secretKey {
				return c.Next()
			}
		}

		// 4. Check HMAC SHA-256 Signature (X-Hub-Signature-256: sha256=<hex_hash>)
		signatureHeader := c.Get("X-Hub-Signature-256")
		if signatureHeader != "" {
			parts := strings.SplitN(signatureHeader, "=", 2)
			if len(parts) == 2 && parts[0] == "sha256" {
				expectedHashHex := parts[1]
				mac := hmac.New(sha256.New, []byte(secretKey))
				mac.Write(c.Body())
				computedHashHex := hex.EncodeToString(mac.Sum(nil))

				if subtle.ConstantTimeCompare([]byte(computedHashHex), []byte(expectedHashHex)) == 1 {
					return c.Next()
				}
			}
		}

		log.Printf("[Webhook Auth Warning] 401 Unauthorized: Multi-layer verification failed for request from IP %s (Path: %s)", c.IP(), c.Path())
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "UNAUTHORIZED_WEBHOOK",
			"message": "Webhook authentication failed. Provide a valid token via query parameter (?secret=), X-Webhook-Secret header, Authorization header, or X-Hub-Signature-256.",
		})
	}
}
