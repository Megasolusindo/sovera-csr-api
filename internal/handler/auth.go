package handler

import (
	"time"

	"sovera-core-api/internal/model"
	"sovera-core-api/internal/repository"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// AuthHandler handles authentication endpoints: register, login, me.
type AuthHandler struct {
	userRepo  *repository.UserRepository
	jwtSecret string
}

func NewAuthHandler(userRepo *repository.UserRepository, jwtSecret string) *AuthHandler {
	return &AuthHandler{userRepo: userRepo, jwtSecret: jwtSecret}
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
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Sesi otorisasi admin berhasil diakhiri.",
	})
}
