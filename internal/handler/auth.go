package handler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"sovera-core-api/internal/model"
	"sovera-core-api/internal/repository"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

// AuthHandler handles authentication endpoints: register, login, me.
type AuthHandler struct {
	userRepo  *repository.UserRepository
	jwtSecret string
	rdb         *redis.Client
}

func NewAuthHandler(userRepo *repository.UserRepository, jwtSecret string, rdb *redis.Client) *AuthHandler {
	return &AuthHandler{userRepo: userRepo, jwtSecret: jwtSecret, rdb: rdb}
}

type RegisterPayload struct {
	OrgID    string `json:"org_id"`
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

// LoginPayload is the request body for POST /auth/login
type LoginPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Register godoc
// POST /api/v1/auth/register
// Creates a new user within an existing organization.
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var payload RegisterPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false, "error": "INVALID_BODY", "message": err.Error(),
		})
	}

	if payload.Email == "" || payload.Password == "" || payload.OrgID == "" || payload.FullName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false, "error": "MISSING_FIELDS",
			"message": "org_id, email, password, and full_name are required",
		})
	}

	if len(payload.Password) < 8 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false, "error": "WEAK_PASSWORD",
			"message": "Password must be at least 8 characters long",
		})
	}

	// Register endpoint only allows self-registration as FUNDRAISER;
	// ORG_ADMIN and DIRECTOR roles must be assigned by an existing admin
	role := model.RoleFundraiser

	// Hash password using bcrypt
	hash, err := bcrypt.GenerateFromPassword([]byte(payload.Password), 12)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false, "error": "HASH_FAILED", "message": "Failed to hash password",
		})
	}

	newUser := model.User{
		OrgID:        payload.OrgID,
		Email:        payload.Email,
		PasswordHash: string(hash),
		FullName:     payload.FullName,
		Role:         role,
	}

	created, err := h.userRepo.Create(c.Context(), newUser)
	if err != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"success": false, "error": "EMAIL_TAKEN", "message": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"id":        created.ID,
			"org_id":    created.OrgID,
			"email":     created.Email,
			"full_name": created.FullName,
			"role":      created.Role,
		},
	})
}

// Login godoc
// POST /api/v1/auth/login
// Validates credentials and returns a signed JWT token.
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var payload LoginPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false, "error": "INVALID_BODY", "message": err.Error(),
		})
	}

	user, err := h.userRepo.FindUserWithTenantByEmail(c.Context(), payload.Email)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false, "error": "INVALID_CREDENTIALS",
			"message": "Email atau password salah",
		})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(payload.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false, "error": "INVALID_CREDENTIALS",
			"message": "Email atau password salah",
		})
	}

	// Generate JWT with RBAC & Tenant claims
	claims := jwt.MapClaims{
		"sub":         user.ID,
		"org_id":      user.OrgID,
		"email":       user.Email,
		"role":        string(user.Role),
		"tenant_type": string(user.TenantType),
		"jti":         uuid.New().String(), // JWT ID for token revocation
		"exp":         time.Now().Add(24 * time.Hour).Unix(),
		"iat":         time.Now().Unix(),
	}
	if user.CompanyID != nil {
		claims["company_id"] = *user.CompanyID
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(h.jwtSecret))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false, "error": "TOKEN_SIGN_FAILED", "message": "Failed to sign token",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"token":   tokenString,
		"user": fiber.Map{
			"id":          user.ID,
			"org_id":      user.OrgID,
			"org_name":    user.OrgName,
			"email":       user.Email,
			"full_name":   user.FullName,
			"role":        user.Role,
			"tenant_type": user.TenantType,
			"company_id":  user.CompanyID,
		},
	})
}

// Me godoc
// GET /api/v1/auth/me
// Returns the authenticated user's profile (requires JWT).
func (h *AuthHandler) Me(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	user, err := h.userRepo.FindUserWithTenantByID(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false, "error": "USER_NOT_FOUND", "message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"id":          user.ID,
			"org_id":      user.OrgID,
			"org_name":    user.OrgName,
			"email":       user.Email,
			"full_name":   user.FullName,
			"role":        user.Role,
			"tenant_type": user.TenantType,
			"company_id":  user.CompanyID,
			"is_active":   user.IsActive,
			"created_at":  user.CreatedAt,
		},
	})
}

// Logout godoc
// POST /api/v1/auth/logout
// Invalidates client auth token and completes operator logout session.
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "MISSING_TOKEN",
			"message": "Authorization header is required",
		})
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
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
		return []byte(h.jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "INVALID_TOKEN",
			"message": "Invalid or expired token",
		})
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "INVALID_CLAIMS",
			"message": "Unable to parse JWT claims",
		})
	}

	jti, _ := claims["jti"].(string)
	if jti != "" {
		if h.rdb != nil {
			ctx := context.Background()
			h.rdb.SAdd(ctx, "revoked_tokens", jti)
			// Set expiry on the JTI so the blacklist doesn't grow indefinitely
			h.rdb.Expire(ctx, "revoked_tokens", 24*time.Hour)
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Sesi otorisasi admin berhasil diakhiri.",
	})
}
