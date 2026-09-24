package handler

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"sovera-core-api/internal/model"
	"sovera-core-api/internal/pkg/phoneverifier"
	pkgurlverifier "sovera-core-api/internal/pkg/urlverifier"
	"sovera-core-api/internal/queue"
	"sovera-core-api/internal/repository"
	"sovera-core-api/internal/service/urlverifier"
)


type CompanyHandler struct {
	repo             *repository.CompanyRepository
	programRepo      *repository.CompanyCSRProgramRepository
	linkedinVerifier *urlverifier.LinkedInVerifier
}

func NewCompanyHandler(repo *repository.CompanyRepository, programRepo *repository.CompanyCSRProgramRepository, linkedinVerifier *urlverifier.LinkedInVerifier) *CompanyHandler {
	if linkedinVerifier == nil {
		linkedinVerifier = urlverifier.NewLinkedInVerifier()
	}
	return &CompanyHandler{
		repo:             repo,
		programRepo:      programRepo,
		linkedinVerifier: linkedinVerifier,
	}
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

// ListCSRPrograms returns a paginated list of corporate CSR programs.
// Visibility filter:
//   - empty/default: returns public + curated (explore catalog)
//   - "public": only public programs
//   - "curated": only curated programs
//   - "private": only private programs owned by the authenticated organization
//   - "ALL": bypass visibility filter, superadmin only
func (h *CompanyHandler) ListCSRPrograms(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	search := c.Query("search", "")
	programType := c.Query("pillar", "")
	visibility := c.Query("visibility", "")

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

	role, _ := c.Locals("role").(string)
	orgID, _ := c.Locals("org_id").(string)
	authorizedOrgID := orgID
	if strings.EqualFold(role, "SUPERADMIN") {
		authorizedOrgID = ""
	}

	if strings.EqualFold(strings.TrimSpace(visibility), "ALL") && !strings.EqualFold(role, "SUPERADMIN") {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error":   "INSUFFICIENT_PERMISSIONS",
			"message": "Visibility bypass hanya tersedia untuk superadmin",
		})
	}

	containsPrivate := false
	for _, value := range strings.Split(visibility, ",") {
		if strings.EqualFold(strings.TrimSpace(value), "private") {
			containsPrivate = true
			break
		}
	}
	if containsPrivate && orgID == "" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error":   "AUTH_REQUIRED",
			"message": "Program private memerlukan autentikasi dan kepemilikan organisasi",
		})
	}

	programs, total, err := h.programRepo.ListAllPrograms(c.Context(), limit, offset, search, programType, visibility, authorizedOrgID)
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

// UpdateCSRProgramVisibility handles PATCH /api/v1/companies/csr-programs/:id/visibility.
// Only CORP_ADMIN, CSR_MANAGER, or SUPERADMIN can change visibility.
// Non-superadmin changes are restricted to programs owned by their verified corporate organization.
func (h *CompanyHandler) UpdateCSRProgramVisibility(c *fiber.Ctx) error {
	id := c.Params("id")
	if strings.TrimSpace(id) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "MISSING_ID",
			"message": "Program ID wajib disertakan",
		})
	}

	role, _ := c.Locals("role").(string)
	orgID, _ := c.Locals("org_id").(string)
	if !strings.EqualFold(role, "CORP_ADMIN") && !strings.EqualFold(role, "CSR_MANAGER") && !strings.EqualFold(role, "SUPERADMIN") {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error":   "INSUFFICIENT_PERMISSIONS",
			"message": "Hanya CORP_ADMIN, CSR_MANAGER, atau SUPERADMIN yang dapat mengubah visibility",
		})
	}

	if !strings.EqualFold(role, "SUPERADMIN") {
		canManage, err := h.programRepo.CanManageCSRProgram(c.Context(), id, orgID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"error":   "OWNERSHIP_CHECK_FAILED",
				"message": err.Error(),
			})
		}
		if !canManage {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"error":   "INSUFFICIENT_PERMISSIONS",
				"message": "Anda tidak memiliki wewenang untuk program ini",
			})
		}
	}

	var payload struct {
		Visibility string `json:"visibility"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "INVALID_BODY",
			"message": err.Error(),
		})
	}

	visibility := strings.ToLower(strings.TrimSpace(payload.Visibility))
	if visibility != "public" && visibility != "curated" && visibility != "private" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "INVALID_VISIBILITY",
			"message": "visibility harus salah satu dari: public, curated, private",
		})
	}

	updated, err := h.programRepo.UpdateVisibility(c.Context(), id, visibility)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "UPDATE_FAILED",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    updated,
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
	LinkedinURL     string   `json:"linkedin_url,omitempty"`
	LinkedinStatus  string   `json:"linkedin_status,omitempty"`
	InstagramURL    string   `json:"instagram_url,omitempty"`
	InstagramStatus string   `json:"instagram_status,omitempty"`
	FacebookURL     string   `json:"facebook_url,omitempty"`
	FacebookStatus  string   `json:"facebook_status,omitempty"`
	YoutubeURL      string   `json:"youtube_url,omitempty"`
	YoutubeStatus   string   `json:"youtube_status,omitempty"`
	Headquarters    string   `json:"headquarters,omitempty"`
	Phone           string   `json:"phone,omitempty"`
	ContactPhone    string   `json:"contact_phone,omitempty"`
	Ticker          string   `json:"ticker,omitempty"`
	PriorityTier    string   `json:"priority_tier,omitempty"`
	AliasKeywords   []string `json:"alias_keywords,omitempty"`
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
		ok, normalized, err := pkgurlverifier.DefaultVerifier.VerifyWebsite(c.UserContext(), payload.Website)
		if !ok {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"error":   "INVALID_WEBSITE_URL",
				"message": fmt.Sprintf("Website URL '%s' tidak dapat diakses atau dead link: %v. Mohon pastikan situs web dapat dijangkau.", payload.Website, err),
			})
		}
		website = &normalized
	}

	var linkedinURL *string
	if strings.TrimSpace(payload.LinkedinURL) != "" {
		res := h.linkedinVerifier.ValidateLinkedInURL(c.UserContext(), payload.LinkedinURL)
		if !res.IsValid {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"error":   "INVALID_LINKEDIN_URL",
				"message": fmt.Sprintf("LinkedIn URL '%s' tidak valid atau tidak ditemukan: %s. Mohon pastikan link profil LinkedIn benar.", payload.LinkedinURL, res.Reason),
			})
		}
		linkedinURL = &res.CanonicalURL
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

	var instagramURL *string
	if payload.InstagramURL != "" {
		instagramURL = &payload.InstagramURL
	}
	var facebookURL *string
	if payload.FacebookURL != "" {
		facebookURL = &payload.FacebookURL
	}
	var youtubeURL *string
	if payload.YoutubeURL != "" {
		youtubeURL = &payload.YoutubeURL
	}
	var headquarters *string
	if payload.Headquarters != "" {
		headquarters = &payload.Headquarters
	}

	comp := model.Company{
		Name:           payload.Name,
		LegalName:      legalName,
		IndustrySector: payload.IndustrySector,
		CompanyType:    payload.CompanyType,
		Website:        website,
		LinkedinURL:    linkedinURL,
		InstagramURL:   instagramURL,
		FacebookURL:    facebookURL,
		YoutubeURL:     youtubeURL,
		Headquarters:   headquarters,
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

// UpdateCompany handles PUT / PATCH /api/v1/companies/:id to update an existing company.
func (h *CompanyHandler) UpdateCompany(c *fiber.Ctx) error {
	id := c.Params("id")
	if strings.TrimSpace(id) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "MISSING_ID",
			"message": "ID Perusahaan wajib disertakan",
		})
	}

	var payload CreateCompanyPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "INVALID_BODY",
			"message": err.Error(),
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
		ok, normalized, err := pkgurlverifier.DefaultVerifier.VerifyWebsite(c.UserContext(), payload.Website)
		if !ok {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"error":   "INVALID_WEBSITE_URL",
				"message": fmt.Sprintf("Website URL '%s' tidak dapat diakses atau dead link: %v. Mohon pastikan situs web dapat dijangkau.", payload.Website, err),
			})
		}
		website = &normalized
	}

	var linkedinURL *string
	if strings.TrimSpace(payload.LinkedinURL) != "" {
		res := h.linkedinVerifier.ValidateLinkedInURL(c.UserContext(), payload.LinkedinURL)
		if !res.IsValid {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"error":   "INVALID_LINKEDIN_URL",
				"message": fmt.Sprintf("LinkedIn URL '%s' tidak valid atau tidak ditemukan: %s. Mohon pastikan link profil LinkedIn benar.", payload.LinkedinURL, res.Reason),
			})
		}
		linkedinURL = &res.CanonicalURL
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
				"message": fmt.Sprintf("Format nomor telepon perusahaan '%s' tidak valid: %v.", rawPhone, err),
			})
		}
		phone = &normalized
	}

	var instagramURL *string
	if payload.InstagramURL != "" {
		instagramURL = &payload.InstagramURL
	}
	var facebookURL *string
	if payload.FacebookURL != "" {
		facebookURL = &payload.FacebookURL
	}
	var youtubeURL *string
	if payload.YoutubeURL != "" {
		youtubeURL = &payload.YoutubeURL
	}
	var linkedinStatus *string
	if payload.LinkedinStatus != "" {
		linkedinStatus = &payload.LinkedinStatus
	}
	var instagramStatus *string
	if payload.InstagramStatus != "" {
		instagramStatus = &payload.InstagramStatus
	}
	var facebookStatus *string
	if payload.FacebookStatus != "" {
		facebookStatus = &payload.FacebookStatus
	}
	var youtubeStatus *string
	if payload.YoutubeStatus != "" {
		youtubeStatus = &payload.YoutubeStatus
	}
	var headquarters *string
	if payload.Headquarters != "" {
		headquarters = &payload.Headquarters
	}

	comp := model.Company{
		Name:            payload.Name,
		LegalName:       legalName,
		IndustrySector:  payload.IndustrySector,
		CompanyType:     payload.CompanyType,
		Website:         website,
		LinkedinURL:     linkedinURL,
		LinkedinStatus:  linkedinStatus,
		InstagramURL:    instagramURL,
		InstagramStatus: instagramStatus,
		FacebookURL:     facebookURL,
		FacebookStatus:  facebookStatus,
		YoutubeURL:      youtubeURL,
		YoutubeStatus:   youtubeStatus,
		Headquarters:    headquarters,
		Phone:           phone,
		Ticker:          ticker,
		PriorityTier:    payload.PriorityTier,
	}

	updated, err := h.repo.UpdateCompany(c.Context(), id, comp)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "UPDATE_FAILED",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    updated,
	})
}

// GetLinkedInStats handles GET /api/v1/url/linkedin-stats
func (h *CompanyHandler) GetLinkedInStats(c *fiber.Ctx) error {
	stats, err := h.repo.GetLinkedInValidationStats(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "STATS_FAILED",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    stats,
	})
}

// TriggerBatchLinkedInVerification handles POST /api/v1/url/verify-linkedin-batch
func (h *CompanyHandler) TriggerBatchLinkedInVerification(c *fiber.Ctx) error {
	go func() {
		if h.repo != nil && h.repo.GetDBPool() != nil {
			_, _ = h.repo.GetDBPool().Exec(context.Background(), `
				UPDATE companies 
				SET linkedin_verified_at = NULL, linkedin_status = 'UNVERIFIED' 
				WHERE linkedin_url IS NOT NULL AND linkedin_url <> ''
			`)
		}
		worker := queue.NewCompanyLinkedInWorker(h.repo.GetDBPool())
		_ = worker.HandleCompanyLinkedInBatchCheck(context.Background(), nil)
	}()

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"success": true,
		"message": "Batch company LinkedIn URL verification task triggered in background",
	})
}

// VerifyInstagramURL handles POST /api/v1/url/verify-instagram
func (h *CompanyHandler) VerifyInstagramURL(c *fiber.Ctx) error {
	var payload struct {
		URL string `json:"url"`
	}
	if err := c.BodyParser(&payload); err != nil || payload.URL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "INVALID_INPUT",
			"message": "Request body must contain 'url'",
		})
	}

	verifier := urlverifier.NewInstagramVerifier()
	res := verifier.ValidateInstagramURL(c.Context(), payload.URL)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    res,
	})
}

// GetInstagramStats handles GET /api/v1/url/instagram-stats
func (h *CompanyHandler) GetInstagramStats(c *fiber.Ctx) error {
	stats, err := h.repo.GetInstagramValidationStats(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "STATS_FAILED",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    stats,
	})
}

// TriggerBatchInstagramVerification handles POST /api/v1/url/verify-instagram-batch
func (h *CompanyHandler) TriggerBatchInstagramVerification(c *fiber.Ctx) error {
	go func() {
		if h.repo != nil && h.repo.GetDBPool() != nil {
			_, _ = h.repo.GetDBPool().Exec(context.Background(), `
				UPDATE companies 
				SET instagram_verified_at = NULL, instagram_status = 'UNVERIFIED' 
				WHERE instagram_url IS NOT NULL AND instagram_url <> ''
			`)
		}
		worker := queue.NewCompanyInstagramWorker(h.repo.GetDBPool())
		_ = worker.HandleCompanyInstagramBatchCheck(context.Background(), nil)
	}()

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"success": true,
		"message": "Batch company Instagram URL verification task triggered in background",
	})
}

// TriggerBatchLinkedInDiscovery handles POST /api/v1/url/discover-linkedin-batch
func (h *CompanyHandler) TriggerBatchLinkedInDiscovery(c *fiber.Ctx) error {
	serperKey := c.Query("serper_key", "")
	if serperKey == "" {
		serperKey = os.Getenv("SERPER_API_KEY")
	}
	worker := queue.NewCompanyLinkedInDiscoveryWorker(h.repo.GetDBPool(), serperKey)
	go func() {
		_ = worker.HandleCompanyLinkedInDiscoveryBatch(context.Background(), nil)
	}()

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"success": true,
		"message": "Batch company LinkedIn discovery task triggered in background",
	})
}

// TriggerBatchInstagramDiscovery handles POST /api/v1/url/discover-instagram-batch
func (h *CompanyHandler) TriggerBatchInstagramDiscovery(c *fiber.Ctx) error {
	serperKey := c.Query("serper_key", "")
	if serperKey == "" {
		serperKey = os.Getenv("SERPER_API_KEY")
	}
	worker := queue.NewCompanyInstagramDiscoveryWorker(h.repo.GetDBPool(), serperKey)
	go func() {
		_ = worker.HandleCompanyInstagramDiscoveryBatch(context.Background(), nil)
	}()

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"success": true,
		"message": "Batch company Instagram discovery task triggered in background",
	})
}

// VerifyFacebookURL handles POST /api/v1/url/verify-facebook
func (h *CompanyHandler) VerifyFacebookURL(c *fiber.Ctx) error {
	var payload struct {
		URL string `json:"url"`
	}
	if err := c.BodyParser(&payload); err != nil || payload.URL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "INVALID_INPUT",
			"message": "Request body must contain 'url'",
		})
	}

	verifier := urlverifier.NewFacebookVerifier()
	res := verifier.ValidateFacebookURL(c.Context(), payload.URL)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    res,
	})
}

// GetFacebookStats handles GET /api/v1/url/facebook-stats
func (h *CompanyHandler) GetFacebookStats(c *fiber.Ctx) error {
	stats, err := h.repo.GetFacebookValidationStats(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "STATS_FAILED",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    stats,
	})
}

// TriggerBatchFacebookVerification handles POST /api/v1/url/verify-facebook-batch
func (h *CompanyHandler) TriggerBatchFacebookVerification(c *fiber.Ctx) error {
	go func() {
		if h.repo != nil && h.repo.GetDBPool() != nil {
			_, _ = h.repo.GetDBPool().Exec(context.Background(), `
				UPDATE company.companies 
				SET facebook_verified_at = NULL, facebook_status = 'UNVERIFIED' 
				WHERE facebook_url IS NOT NULL AND facebook_url <> ''
			`)
		}
		worker := queue.NewCompanyFacebookWorker(h.repo.GetDBPool())
		_ = worker.HandleCompanyFacebookBatchCheck(context.Background(), nil)
	}()

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"success": true,
		"message": "Batch company Facebook URL verification task triggered in background",
	})
}

// VerifyYoutubeURL handles POST /api/v1/url/verify-youtube
func (h *CompanyHandler) VerifyYoutubeURL(c *fiber.Ctx) error {
	var payload struct {
		URL string `json:"url"`
	}
	if err := c.BodyParser(&payload); err != nil || payload.URL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "INVALID_INPUT",
			"message": "Request body must contain 'url'",
		})
	}

	verifier := urlverifier.NewYoutubeVerifier()
	res := verifier.ValidateYoutubeURL(c.Context(), payload.URL)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    res,
	})
}

// GetYoutubeStats handles GET /api/v1/url/youtube-stats
func (h *CompanyHandler) GetYoutubeStats(c *fiber.Ctx) error {
	stats, err := h.repo.GetYoutubeValidationStats(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "STATS_FAILED",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    stats,
	})
}

// TriggerBatchYoutubeVerification handles POST /api/v1/url/verify-youtube-batch
func (h *CompanyHandler) TriggerBatchYoutubeVerification(c *fiber.Ctx) error {
	go func() {
		if h.repo != nil && h.repo.GetDBPool() != nil {
			_, _ = h.repo.GetDBPool().Exec(context.Background(), `
				UPDATE company.companies 
				SET youtube_verified_at = NULL, youtube_status = 'UNVERIFIED' 
				WHERE youtube_url IS NOT NULL AND youtube_url <> ''
			`)
		}
		worker := queue.NewCompanyYoutubeWorker(h.repo.GetDBPool())
		_ = worker.HandleCompanyYoutubeBatchCheck(context.Background(), nil)
	}()

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"success": true,
		"message": "Batch company YouTube URL verification task triggered in background",
	})
}



