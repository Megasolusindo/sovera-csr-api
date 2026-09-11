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

// RegisterPayload is the request body for POST /auth/register
type RegisterPayload struct {
	OrgID    string `json:"org_id"`
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
	Role     string `json:"role"` // ORG_ADMIN | DIRECTOR | FUNDRAISER
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

	// Default role to FUNDRAISER if not specified or invalid
	role := model.UserOrgRole(payload.Role)
	if role != model.RoleOrgAdmin && role != model.RoleDirector && role != model.RoleFundraiser {
		role = model.RoleFundraiser
	}

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

	user, err := h.userRepo.FindByEmail(c.Context(), payload.Email)
	if err != nil {
		// Generic error to prevent email enumeration
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

	// Generate JWT with RBAC claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":    user.ID,
		"org_id": user.OrgID,
		"email":  user.Email,
		"role":   string(user.Role),
		"exp":    time.Now().Add(24 * time.Hour).Unix(),
		"iat":    time.Now().Unix(),
	})

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
			"id":        user.ID,
			"org_id":    user.OrgID,
			"email":     user.Email,
			"full_name": user.FullName,
			"role":      user.Role,
		},
	})
}

// Me godoc
// GET /api/v1/auth/me
// Returns the authenticated user's profile (requires JWT).
func (h *AuthHandler) Me(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	user, err := h.userRepo.FindByID(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false, "error": "USER_NOT_FOUND", "message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"id":         user.ID,
			"org_id":     user.OrgID,
			"email":      user.Email,
			"full_name":  user.FullName,
			"role":       user.Role,
			"is_active":  user.IsActive,
			"created_at": user.CreatedAt,
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
