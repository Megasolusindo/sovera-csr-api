package middleware

import (
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"sovera-core-api/internal/model"
	"sovera-core-api/internal/repository"
)

// AIAgentAuthMiddleware verifies Authorization: Bearer <AI_AGENT_TOKEN> for /api/v1/ai routes
func AIAgentAuthMiddleware(dbPool *pgxpool.Pool, requiredScope string) fiber.Handler {
	agentRepo := repository.NewAIAgentRepository(dbPool)

	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "UNAUTHORIZED",
				"message": "Missing Authorization header. Expected Bearer token.",
			})
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "INVALID_AUTH_HEADER",
				"message": "Invalid Authorization header format. Expected 'Bearer <token>'.",
			})
		}

		token := strings.TrimSpace(parts[1])
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "EMPTY_TOKEN",
				"message": "Bearer token string cannot be empty.",
			})
		}

		// Validate token against ai_agent_credentials table
		agent, err := agentRepo.ValidateToken(c.Context(), token)
		if err != nil {
			log.Printf("[AI Auth Warning] Validation failed from IP %s: %v", c.IP(), err)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "INVALID_AGENT_TOKEN",
				"message": err.Error(),
			})
		}

		// Check required scope permission if specified
		if requiredScope != "" {
			hasScope := false
			for _, s := range agent.Scopes {
				if s == requiredScope || s == "admin:all" {
					hasScope = true;
					break
				}
			}
			if !hasScope {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"success": false,
					"error":   "INSUFFICIENT_SCOPE",
					"message": "AI Agent credentials lack required scope: " + requiredScope,
				})
			}
		}

		// Inject AI agent into Fiber context
		c.Locals("ai_agent", agent)
		c.Locals("agent_name", agent.AgentName)

		// Audit Log asynchronous dispatch
		requestID := c.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		go func(agentName, action, endpoint, reqID, clientIP string) {
			_ = agentRepo.CreateAuditLog(c.Context(), model.AIAuditLog{
				AgentName: agentName,
				Action:    action,
				Endpoint:  endpoint,
				RequestID: reqID,
				Status:    "SUCCESS",
				ClientIP:  clientIP,
			})
		}(agent.AgentName, c.Method()+" "+c.Path(), c.OriginalURL(), requestID, c.IP())

		return c.Next()
	}
}
