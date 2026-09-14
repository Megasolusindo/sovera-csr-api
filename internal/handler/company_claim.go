package handler

import (
	"fmt"
	"strings"

	"sovera-core-api/internal/model"
	"sovera-core-api/internal/pkg/telegram"
	"sovera-core-api/internal/repository"

	"github.com/gofiber/fiber/v2"
)

type CompanyClaimHandler struct {
	claimRepo   *repository.CompanyClaimRepository
	companyRepo *repository.CompanyRepository
	notifier    *telegram.Notifier
}

func NewCompanyClaimHandler(claimRepo *repository.CompanyClaimRepository, companyRepo *repository.CompanyRepository, notifier *telegram.Notifier) *CompanyClaimHandler {
	return &CompanyClaimHandler{
		claimRepo:   claimRepo,
		companyRepo: companyRepo,
		notifier:    notifier,
	}
}

type ClaimPayload struct {
	CompanyID        string `json:"company_id"`
	WorkEmail        string `json:"work_email"`
	DocumentProofURL string `json:"document_proof_url"`
}

// SubmitClaim handles corporate profile verification requests (Tier 1 Instant Email Domain vs Tier 2 Upload)
func (h *CompanyClaimHandler) SubmitClaim(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	orgID, _ := c.Locals("org_id").(string)

	if userID == "" || orgID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false, "error": "UNAUTHORIZED", "message": "Authentication required",
		})
	}

	var payload ClaimPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false, "error": "INVALID_BODY", "message": err.Error(),
		})
	}

	if payload.CompanyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false, "error": "MISSING_COMPANY_ID", "message": "company_id is required",
		})
	}

	// Retrieve Target Company
	comp, err := h.companyRepo.GetCompanyByID(c.Context(), payload.CompanyID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false, "error": "COMPANY_NOT_FOUND", "message": "Perusahaan tidak ditemukan",
		})
	}

	if comp.IsClaimed {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"success": false, "error": "ALREADY_CLAIMED", "message": "Profil perusahaan ini sudah diklaim oleh entitas terverifikasi",
		})
	}

	// Tier 1: Check work email domain match
	var method model.ClaimMethod = model.ClaimMethodDocumentUpload
	var status model.ClaimStatus = model.ClaimStatusPending

	emailParts := strings.Split(payload.WorkEmail, "@")
	if len(emailParts) == 2 {
		userDomain := strings.ToLower(emailParts[1])
		compDomain := ""
		if comp.CorporateDomain != nil {
			compDomain = strings.ToLower(*comp.CorporateDomain)
		}
		if compDomain == "" && comp.Website != nil && *comp.Website != "" {
			compDomain = strings.ToLower(*comp.Website)
			compDomain = strings.TrimPrefix(compDomain, "https://")
			compDomain = strings.TrimPrefix(compDomain, "http://")
			compDomain = strings.TrimPrefix(compDomain, "www.")
			compDomain = strings.Split(compDomain, "/")[0]
		}

		if userDomain != "" && compDomain != "" && strings.Contains(compDomain, userDomain) && !strings.Contains(userDomain, "gmail") && !strings.Contains(userDomain, "yahoo") {
			method = model.ClaimMethodCorporateEmail
			status = model.ClaimStatusApproved
		}
	}

	claimRecord := model.CompanyClaim{
		CompanyID:         payload.CompanyID,
		TenantID:          orgID,
		RequestedByUserID: userID,
		WorkEmail:         payload.WorkEmail,
		DocumentProofURL:  payload.DocumentProofURL,
		Method:            method,
		Status:            status,
	}

	if status == model.ClaimStatusApproved {
		claimRecord.VerificationNotes = "Tier 1 Instant Verification: Domain Match"
	} else {
		claimRecord.VerificationNotes = "Tier 2 Document Upload: Pending Admin Verification"
	}

	created, err := h.claimRepo.Create(c.Context(), claimRecord)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false, "error": "CLAIM_FAILED", "message": err.Error(),
		})
	}

	// If Auto-Approved, trigger status update immediately
	if status == model.ClaimStatusApproved {
		_, _ = h.claimRepo.UpdateStatus(c.Context(), created.ID, model.ClaimStatusApproved, userID, "Auto approved via Tier 1 Email Domain Matching")
	} else {
		// Notify Telegram Admin for 1-Click Verification
		msg := fmt.Sprintf(
			"<b>PENGAJUAN KLAIM PERUSAHAAN BARU</b>\n\n"+
				"• <b>Perusahaan</b>: %s\n"+
				"• <b>Work Email</b>: %s\n"+
				"• <b>Dokumen Bukti</b>: %s\n"+
				"• <b>Claim ID</b>: <code>%s</code>\n\n"+
				"Balas <code>/approve_claim_%s</code> atau <code>/reject_claim_%s</code> untuk memverifikasi.",
			comp.Name, payload.WorkEmail, payload.DocumentProofURL, created.ID, created.ID, created.ID,
		)
		_ = h.notifier.SendNotification(c.Context(), msg)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    created,
		"message": map[bool]string{
			true:  "Profil perusahaan berhasil diklaim secara instan!",
			false: "Pengajuan klaim berhasil dikirim dan sedang diverifikasi oleh admin platform.",
		}[status == model.ClaimStatusApproved],
	})
}

type VerifyClaimPayload struct {
	Status model.ClaimStatus `json:"status"` // APPROVED | REJECTED
	Notes  string            `json:"notes"`
}

// VerifyClaim allows superadmins to approve/reject claims
func (h *CompanyClaimHandler) VerifyClaim(c *fiber.Ctx) error {
	claimID := c.Params("id")
	verifierUserID, _ := c.Locals("user_id").(string)

	var payload VerifyClaimPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false, "error": "INVALID_BODY", "message": err.Error(),
		})
	}

	updated, err := h.claimRepo.UpdateStatus(c.Context(), claimID, payload.Status, verifierUserID, payload.Notes)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false, "error": "VERIFY_FAILED", "message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    updated,
		"message": "Status klaim profil perusahaan berhasil diperbarui.",
	})
}
