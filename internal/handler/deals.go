package handler

import (
	"fmt"

	"github.com/gofiber/fiber/v2"

	"sovera-core-api/internal/repository"
	"sovera-core-api/internal/service/ai"
	"sovera-core-api/internal/service/exporter"
	"sovera-core-api/internal/service/storage"
)

type DealHandler struct {
	dealRepo      *repository.DealRepository
	programRepo   *repository.ProgramRepository
	signalRepo    *repository.SignalRepository
	templateRepo  *repository.TemplateRepository
	tokenLogRepo  *repository.TokenLogRepository
	storage       storage.StorageService
	geminiService *ai.GeminiService
	exporter      *exporter.DocumentExporter
}

func NewDealHandler(
	dealRepo *repository.DealRepository,
	programRepo *repository.ProgramRepository,
	signalRepo *repository.SignalRepository,
	templateRepo *repository.TemplateRepository,
	tokenLogRepo *repository.TokenLogRepository,
	storage storage.StorageService,
	geminiService *ai.GeminiService,
	exporter *exporter.DocumentExporter,
) *DealHandler {
	return &DealHandler{
		dealRepo:      dealRepo,
		programRepo:   programRepo,
		signalRepo:    signalRepo,
		templateRepo:  templateRepo,
		tokenLogRepo:  tokenLogRepo,
		storage:       storage,
		geminiService: geminiService,
		exporter:      exporter,
	}
}

type CreateDealPayload struct {
	SignalID        string  `json:"signal_id"`
	CompanyName     string  `json:"company_name"`
	TargetProgramID string  `json:"target_program_id"`
	EstimatedValue  float64 `json:"estimated_value"`
}

type UpdateStagePayload struct {
	DealStage string `json:"deal_stage"`
}

type GeneratePitchPayload struct {
	Tone        string `json:"tone"`
	CustomNotes string `json:"custom_notes"`
}

func (h *DealHandler) ListDeals(c *fiber.Ctx) error {
	orgID, ok := c.Locals("org_id").(string)
	if !ok || orgID == "" {
		orgID = "77123aaa-8819-4c12-99a1-00123456789a"
	}

	deals, err := h.dealRepo.ListDeals(c.Context(), orgID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "QUERY_FAILED",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": deals,
	})
}

func (h *DealHandler) CreateDeal(c *fiber.Ctx) error {
	orgID, ok := c.Locals("org_id").(string)
	if !ok || orgID == "" {
		orgID = "77123aaa-8819-4c12-99a1-00123456789a"
	}

	var payload CreateDealPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "INVALID_PAYLOAD",
			"message": "Failed to parse deal payload",
		})
	}

	if payload.CompanyName == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"error":   "MISSING_COMPANY_NAME",
			"message": "company_name is required",
		})
	}

	deal, err := h.dealRepo.CreateDeal(
		c.Context(), orgID, payload.SignalID, payload.CompanyName,
		payload.TargetProgramID, payload.EstimatedValue,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "CREATE_FAILED",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(deal)
}

func (h *DealHandler) UpdateStage(c *fiber.Ctx) error {
	dealID := c.Params("id")
	orgID, ok := c.Locals("org_id").(string)
	if !ok || orgID == "" {
		orgID = "77123aaa-8819-4c12-99a1-00123456789a"
	}

	var payload UpdateStagePayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "INVALID_PAYLOAD",
			"message": "Failed to parse stage payload",
		})
	}

	deal, err := h.dealRepo.UpdateDealStage(c.Context(), orgID, dealID, payload.DealStage)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "UPDATE_FAILED",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(deal)
}

func (h *DealHandler) GeneratePitch(c *fiber.Ctx) error {
	dealID := c.Params("id")
	orgID, ok := c.Locals("org_id").(string)
	if !ok || orgID == "" {
		orgID = "77123aaa-8819-4c12-99a1-00123456789a"
	}

	var payload GeneratePitchPayload
	_ = c.BodyParser(&payload)

	deal, err := h.dealRepo.GetDealByID(c.Context(), orgID, dealID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "DEAL_NOT_FOUND",
			"message": "Deal record not found",
		})
	}

	programTitle := "Program Beasiswa Generasi Digital 3T"
	programDesc := "Program penyediaan sarana komputer dan beasiswa digital"
	if deal.TargetProgramID != "" {
		if prog, err := h.programRepo.GetProgramByID(c.Context(), orgID, deal.TargetProgramID); err == nil {
			programTitle = prog.Title
			programDesc = prog.Description
		}
	}

	result, err := h.geminiService.GeneratePitchStrategy(
		c.Context(), dealID, deal.CompanyName, "Alokasi TJSL Pendidikan Digital",
		programTitle, programDesc, payload.Tone, payload.CustomNotes,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "AI_GENERATION_FAILED",
			"message": err.Error(),
		})
	}

	// Persist pitch strategy outputs to deal_pipelines record
	_ = h.dealRepo.UpdateDealPitch(c.Context(), orgID, dealID, result.Icebreaker, result.ProposalMarkdown)

	// Log AI token usage per tenant
	_ = h.tokenLogRepo.LogUsage(c.Context(), orgID, dealID, "PITCH_STRATEGY", "gemini-1.5-flash", result.PromptTokens, result.CompletionTokens)

	return c.Status(fiber.StatusOK).JSON(result)
}

func (h *DealHandler) ExportProposal(c *fiber.Ctx) error {
	dealID := c.Params("id")
	orgID, ok := c.Locals("org_id").(string)
	if !ok || orgID == "" {
		orgID = "77123aaa-8819-4c12-99a1-00123456789a"
	}

	format := c.Query("format", "docx")

	deal, err := h.dealRepo.GetDealByID(c.Context(), orgID, dealID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "DEAL_NOT_FOUND",
			"message": "Deal record not found",
		})
	}

	content := deal.GeneratedProposal
	if content == "" {
		content = fmt.Sprintf("# PROPOSAL KEMITRAAN STRATEGIS\n\n## Korporasi: %s\n\nRingkasan draf proposal kemitraan institusional.", deal.CompanyName)
	}

	// Check if tenant has uploaded a custom master template in S3/Storage
	tenantTpl, _ := h.templateRepo.GetTenantTemplate(c.Context(), orgID)

	replacements := map[string]string{
		"COMPANY_NAME":     deal.CompanyName,
		"TARGET_PROGRAM":   deal.TargetProgramID,
		"ESTIMATED_VALUE":  fmt.Sprintf("Rp %.2f Miliar", deal.EstimatedValue/1000000000.0),
		"ICEBREAKER_TEXT":  deal.GeneratedIcebreaker,
		"PROPOSAL_BODY":    content,
		"SLIDE_1_TITLE":    "Judul & Alignment Nilai Syariah & ESG",
		"SLIDE_1_CONTENT":  fmt.Sprintf("Kemitraan Strategis: %s x LAZ Peduli Ummat - Akselerasi Pendidikan Vokasi Syariah & Digitalisasi 3T (SDG 4 & SDG 9).", deal.CompanyName),
		"SLIDE_2_TITLE":    "Tantangan Sosial & Urgensi Intervensi",
		"SLIDE_2_CONTENT":  "Senjang digital di 50 pesantren 3T & kebutuhan 500 talenta muda berdaya saing global berbasis nilai keislaman.",
		"SLIDE_3_TITLE":    "Solusi Program & Metrik Keberhasilan (OKRs)",
		"SLIDE_3_CONTENT":  "Beasiswa Penuh 3 Tahun, Penyediaan Laptop & Pelatihan TI Syariah. Target: 100% Lulusan Terserap Kerja dalam 6 Bulan.",
		"SLIDE_4_TITLE":    "Rancangan Anggaran Biaya (RAB) & Efisiensi",
		"SLIDE_4_CONTENT":  fmt.Sprintf("Estimasi Nilai Investasi Sosial: Rp %.2f Miliar. Alokasi 85%% Penyaluran Langsung, 15%% Pendampingan Asnaf & Evaluasi Dampak.", deal.EstimatedValue/1000000000.0),
		"SLIDE_5_TITLE":    "Tata Kelola, Akuntabilitas & Pelaporan POJK 51",
		"SLIDE_5_CONTENT":  "Audit Keuangan Publik Beropini WTP & Laporan Akuntabilitas Dampak Sosial untuk Lampiran Sustainability Report Emiten.",
	}

	if format == "pdf" {
		pdfBytes, filename, err := h.exporter.GeneratePDF(deal.CompanyName, "Proposal Kemitraan", content)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		c.Set("Content-Type", "application/pdf")
		c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		return c.Send(pdfBytes)
	}

	if format == "pptx" {
		// 1. Try Custom S3/Storage Master PPTX Template
		if tenantTpl != nil && tenantTpl.PptxS3Key != "" {
			if tplBytes, err := h.storage.GetTemplate(c.Context(), tenantTpl.PptxS3Key); err == nil && len(tplBytes) > 0 {
				if substitutedBytes, err := h.exporter.SubstitutePlaceholders(tplBytes, replacements); err == nil {
					c.Set("Content-Type", "application/vnd.openxmlformats-officedocument.presentationml.presentation")
					c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"Pitch_Deck_CSR_%s.pptx\"", deal.CompanyName))
					return c.Send(substitutedBytes)
				}
			}
		}

		// 2. Fallback to System Default Master PPTX Generator
		slides := []exporter.SlideData{
			{SlideNumber: 1, Title: replacements["SLIDE_1_TITLE"], Content: replacements["SLIDE_1_CONTENT"]},
			{SlideNumber: 2, Title: replacements["SLIDE_2_TITLE"], Content: replacements["SLIDE_2_CONTENT"]},
			{SlideNumber: 3, Title: replacements["SLIDE_3_TITLE"], Content: replacements["SLIDE_3_CONTENT"]},
			{SlideNumber: 4, Title: replacements["SLIDE_4_TITLE"], Content: replacements["SLIDE_4_CONTENT"]},
			{SlideNumber: 5, Title: replacements["SLIDE_5_TITLE"], Content: replacements["SLIDE_5_CONTENT"]},
		}
		pptxBytes, filename, err := h.exporter.GeneratePPTX(deal.CompanyName, "Pitch Deck CSR", slides)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		c.Set("Content-Type", "application/vnd.openxmlformats-officedocument.presentationml.presentation")
		c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		return c.Send(pptxBytes)
	}

	// Default format DOCX
	if tenantTpl != nil && tenantTpl.DocxS3Key != "" {
		if tplBytes, err := h.storage.GetTemplate(c.Context(), tenantTpl.DocxS3Key); err == nil && len(tplBytes) > 0 {
			if substitutedBytes, err := h.exporter.SubstitutePlaceholders(tplBytes, replacements); err == nil {
				c.Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
				c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"Proposal_CSR_%s.docx\"", deal.CompanyName))
				return c.Send(substitutedBytes)
			}
		}
	}

	docxBytes, filename, err := h.exporter.GenerateDOCX(deal.CompanyName, "Proposal Kemitraan", content)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	c.Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	return c.Send(docxBytes)
}
