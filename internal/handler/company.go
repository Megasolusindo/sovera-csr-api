package handler

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"sovera-core-api/internal/model"
	"sovera-core-api/internal/pkg/phoneverifier"
	"sovera-core-api/internal/pkg/urlverifier"
	"sovera-core-api/internal/repository"
)

type CompanyHandler struct {
	repo        *repository.CompanyRepository
	programRepo *repository.CompanyCSRProgramRepository
}

func NewCompanyHandler(repo *repository.CompanyRepository, programRepo *repository.CompanyCSRProgramRepository) *CompanyHandler {
	return &CompanyHandler{repo: repo, programRepo: programRepo}
}

// ListCompanies returns a paginated list of companies.
func (h *CompanyHandler) ListCompanies(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	search := c.Query("search", c.Query("q", ""))
	sector := c.Query("sector", "")
	verificationStatus := c.Query("verification_status", "")
	priorityTier := c.Query("priority_tier", "")
	companyType := c.Query("company_type", c.Query("company_category", ""))

	companies, total, err := h.repo.ListCompanies(c.Context(), limit, offset, search, sector, verificationStatus, priorityTier, companyType)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "QUERY_FAILED",
			"message": err.Error(),
		})
	}

	stats, _ := h.repo.GetCompanyStats(c.Context())

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": companies,
		"pagination": fiber.Map{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
		"stats": stats,
	})
}

// ListCSRPrograms returns a paginated list of all corporate CSR programs.
func (h *CompanyHandler) ListCSRPrograms(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	search := c.Query("search", "")
	programType := c.Query("pillar", "")

	if h.programRepo == nil {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"data": []interface{}{},
			"pagination": fiber.Map{
				"total":  0,
				"limit":  limit,
				"offset": offset,
			},
		})
	}

	programs, total, err := h.programRepo.ListAllPrograms(c.Context(), limit, offset, search, programType)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "QUERY_FAILED",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": programs,
		"pagination": fiber.Map{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetCompany returns details for a single company by ID or Slug.
func (h *CompanyHandler) GetCompany(c *fiber.Ctx) error {
	idOrSlug := c.Params("id")
	company, err := h.repo.GetCompanyByID(c.Context(), idOrSlug)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "COMPANY_NOT_FOUND",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": company,
	})
}

// CreateCompany handles POST /api/v1/companies to insert a new company record.
type CreateCompanyPayload struct {
	Name           string   `json:"name"`
	LegalName      string   `json:"legal_name,omitempty"`
	IndustrySector string   `json:"industry_sector"`
	CompanyType    string   `json:"company_type"`
	Website        string   `json:"website,omitempty"`
	Phone          string   `json:"phone,omitempty"`
	ContactPhone   string   `json:"contact_phone,omitempty"`
	Ticker         string   `json:"ticker,omitempty"`
	PriorityTier   string   `json:"priority_tier,omitempty"`
	AliasKeywords  []string `json:"alias_keywords,omitempty"`
}

func (h *CompanyHandler) CreateCompany(c *fiber.Ctx) error {
	var payload CreateCompanyPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "INVALID_BODY",
			"message": err.Error(),
		})
	}

	if strings.TrimSpace(payload.Name) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "MISSING_NAME",
			"message": "Nama perusahaan wajib diisi",
		})
	}

	var legalName *string
	if payload.LegalName != "" {
		legalName = &payload.LegalName
	}
	var ticker *string
	if payload.Ticker != "" {
		ticker = &payload.Ticker
	}
	var website *string
	if payload.Website != "" {
		ok, normalized, err := urlverifier.DefaultVerifier.VerifyWebsite(c.UserContext(), payload.Website)
		if !ok {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"error":   "INVALID_WEBSITE_URL",
				"message": fmt.Sprintf("Website URL '%s' tidak dapat diakses atau dead link: %v. Mohon pastikan situs web dapat dijangkau.", payload.Website, err),
			})
		}
		website = &normalized
	}

	rawPhone := payload.Phone
	if rawPhone == "" {
		rawPhone = payload.ContactPhone
	}
	var phone *string
	if rawPhone != "" {
		ok, normalized, err := phoneverifier.DefaultVerifier.Verify(rawPhone)
		if !ok {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"error":   "INVALID_PHONE_NUMBER",
				"message": fmt.Sprintf("Format nomor telepon perusahaan '%s' tidak valid: %v. Mohon gunakan format nomor telepon yang benar.", rawPhone, err),
			})
		}
		phone = &normalized
	}

	comp := model.Company{
		Name:           payload.Name,
		LegalName:      legalName,
		IndustrySector: payload.IndustrySector,
		CompanyType:    payload.CompanyType,
		Website:        website,
		Phone:          phone,
		Ticker:         ticker,
		PriorityTier:   payload.PriorityTier,
		AliasKeywords:  payload.AliasKeywords,
	}

	created, err := h.repo.CreateCompany(c.Context(), comp)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "CREATE_FAILED",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    created,
	})
}
