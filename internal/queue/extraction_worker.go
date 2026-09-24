package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/hibiken/asynq"

	"sovera-core-api/internal/repository"
	"sovera-core-api/internal/service/ai"
	"sovera-core-api/internal/service/entityresolver"
	"sovera-core-api/internal/service/esgextractor"
	"sovera-core-api/internal/service/normalizer"
)

type ExtractionWorker struct {
	geminiService  *ai.GeminiService
	signalRepo     *repository.SignalRepository
	normalizer     *normalizer.Normalizer
	entityResolver *entityresolver.EntityResolver
	esgExtractor   *esgextractor.ESGExtractor
	tokenLogRepo   *repository.TokenLogRepository
}

func NewExtractionWorker(
	geminiService *ai.GeminiService,
	signalRepo *repository.SignalRepository,
	norm *normalizer.Normalizer,
	resolver *entityresolver.EntityResolver,
	esgExt *esgextractor.ESGExtractor,
	tokenLogRepo *repository.TokenLogRepository,
) *ExtractionWorker {
	if norm == nil {
		norm = normalizer.NewNormalizer()
	}
	return &ExtractionWorker{
		geminiService:  geminiService,
		signalRepo:     signalRepo,
		normalizer:     norm,
		entityResolver: resolver,
		esgExtractor:   esgExt,
		tokenLogRepo:   tokenLogRepo,
	}
}

func (w *ExtractionWorker) ProcessExtractionTask(ctx context.Context, task *asynq.Task) error {
	var payload LLMExtractionPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal LLM extraction task payload: %w", err)
	}

	log.Printf("[Asynq Worker] Starting LLM Entity Extraction for Job [%s] Content Hash [%s]", payload.JobID, payload.ContentHash)

	textToExtract := w.normalizer.SelectBestContent(payload.RawText, payload.MarkdownContent)

	// 1. LLM Structured Entity Extraction via Gemini
	extractedSignal, err := w.geminiService.ExtractCorporateSignal(ctx, textToExtract)
	if err != nil {
		log.Printf("[Asynq Worker] LLM signal extraction failed for Job [%s]: %v", payload.JobID, err)
		return fmt.Errorf("LLM signal extraction failed: %w", err)
	}

	log.Printf("[Asynq Worker] LLM Extracted Signal: Company=%q | Sector=%q | CSRRelevance=%s | Confidence=%.2f | Summary=%q",
		extractedSignal.CompanyName, extractedSignal.IndustrySector, extractedSignal.CSRRelevance, extractedSignal.ConfidenceScore, extractedSignal.Summary)

	// Log AI token usage to crm.ai_token_logs
	if w.tokenLogRepo != nil {
		promptTokens := len(textToExtract) / 4
		if promptTokens < 50 {
			promptTokens = 50
		}
		completionTokens := 350
		_ = w.tokenLogRepo.LogUsage(ctx, "00000000-0000-0000-0000-000000000000", "", "SIGNAL_LLM_EXTRACTION", "gemini-1.5-flash", promptTokens, completionTokens)
	}

	// 2. Entity Resolution (Match or auto-provision company master)
	var companyID *string
	if w.entityResolver != nil {
		resolvedComp, resErr := w.entityResolver.ResolveCompany(ctx, extractedSignal.CompanyName, extractedSignal.IndustrySector)
		if resErr != nil {
			log.Printf("[Asynq Worker] Warning: Entity resolution failed for %s: %v", extractedSignal.CompanyName, resErr)
		} else if resolvedComp != nil {
			companyID = resolvedComp.CompanyID
			extractedSignal.CompanyName = resolvedComp.CanonicalName
			log.Printf("[Asynq Worker] Resolved Entity [%s] -> Slug [%s] (New: %t)", extractedSignal.CompanyName, resolvedComp.Slug, resolvedComp.IsNew)
		}
	}

	// 3. Automated ESG Extraction if report or ESG keywords detected
	if w.esgExtractor != nil && (payload.SourceType == "BEI_REPORT" || payload.SourceType == "PDF_REPORTS" || payload.SourceType == "PDF_DOCUMENT" || payload.SourceType == "COMPANY_ENRICHMENT" || payload.SourceType == "CSR_OPPORTUNITY_SEARCH" || strings.Contains(strings.ToUpper(textToExtract), "ESG") || strings.Contains(strings.ToLower(textToExtract), "keberlanjutan")) {
		log.Printf("[Asynq Worker] Triggering ESG Profile Extractor for company [%s]...", extractedSignal.CompanyName)
		esgProfile, esgErr := w.esgExtractor.ProcessESGExtraction(
			ctx,
			payload.RawText, payload.MarkdownContent,
			extractedSignal.CompanyName, extractedSignal.IndustrySector,
			companyID,
		)
		if esgErr != nil {
			log.Printf("[Asynq Worker] Warning: ESG profile extraction skipped: %v", esgErr)
		} else if esgProfile != nil {
			log.Printf("[Asynq Worker] ESG Extractor Completed -> Profile ID [%s], Year [%d]", esgProfile.ID, esgProfile.ReportingYear)
		}
	}

	// 4. Vector Embedding Generation (1536 dim) via Normalizer text preparation
	textToEmbed := w.normalizer.PrepareEmbeddingText(
		extractedSignal.CompanyName,
		extractedSignal.IndustrySector,
		extractedSignal.CSRPillarFocus,
		extractedSignal.Summary,
	)
	embedding, err := w.geminiService.GenerateEmbedding(ctx, textToEmbed)
	if err != nil {
		return fmt.Errorf("vector embedding generation failed: %w", err)
	}

	// 5. Skip saving if extracted signal is NON_CSR, low confidence, invalid company, or a negative noise response
	cleanSummary := strings.TrimSpace(extractedSignal.Summary)
	lowerSummary := strings.ToLower(cleanSummary)
	lowerText := strings.ToLower(textToExtract)

	// Explicit NON_CSR taxonomy or low confidence skip
	if strings.EqualFold(extractedSignal.CSRRelevance, "NON_CSR") || strings.EqualFold(extractedSignal.CSRRelevance, "LOW") {
		log.Printf("[Asynq Worker] Skipping NON_CSR/LOW relevance signal for company [%s] (Relevance: %s): '%s'", extractedSignal.CompanyName, extractedSignal.CSRRelevance, cleanSummary)
		return nil
	}

	if extractedSignal.CompanyName == "" || strings.EqualFold(extractedSignal.CompanyName, "Unknown") {
		log.Printf("[Asynq Worker] Skipping signal with empty/unknown company name")
		return nil
	}

	if extractedSignal.ConfidenceScore > 0 && extractedSignal.ConfidenceScore < 0.50 {
		log.Printf("[Asynq Worker] Skipping low confidence signal (Score: %.2f) for company [%s]", extractedSignal.ConfidenceScore, extractedSignal.CompanyName)
		return nil
	}

	// Internal employee welfare & B2B commercial agreement keyword heuristic fallback
	isInternalHRBenefit := strings.Contains(lowerSummary, "kpr karyawan") ||
		strings.Contains(lowerSummary, "kepemilikan rumah karyawan") ||
		stringsContainsAny(lowerText, "program kepemilikan rumah karyawan", "kpr karyawan", "fasilitas payroll", "pinjaman karyawan", "internal employee benefit")

	if isInternalHRBenefit {
		log.Printf("[Asynq Worker] Skipping internal HR benefit / employee welfare signal for company [%s]: '%s'", extractedSignal.CompanyName, cleanSummary)
		return nil
	}

	if cleanSummary == "" || strings.HasPrefix(lowerSummary, "no corporate csr") || strings.HasPrefix(lowerSummary, "no csr") || strings.Contains(lowerSummary, "no corporate csr information") || strings.Contains(lowerSummary, "no csr funding information") {
		log.Printf("[Asynq Worker] Skipping noise/negative signal for company [%s]: '%s'", extractedSignal.CompanyName, cleanSummary)
		return nil
	}

	// Persist Extracted Signal, company_id & Vector to Database
	signalID, err := w.signalRepo.SaveSignal(ctx, extractedSignal, companyID, payload.SourceType, payload.SourceURL, payload.ContentHash, embedding)
	if err != nil {
		return fmt.Errorf("database signal persistence failed: %w", err)
	}

	log.Printf("[Asynq Worker] Successfully saved Signal [%s] for company [%s] with intent score [%d]", signalID, extractedSignal.CompanyName, extractedSignal.IntentScore)
	return nil
}

func (w *ExtractionWorker) ProcessESGTask(ctx context.Context, task *asynq.Task) error {
	var payload ESGExtractionPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal ESG extraction task payload: %w", err)
	}

	if w.esgExtractor == nil {
		return fmt.Errorf("ESGExtractor service is uninitialized")
	}

	esgProfile, err := w.esgExtractor.ProcessESGExtraction(
		ctx,
		payload.RawText, payload.MarkdownContent,
		payload.CompanyName, payload.IndustrySector,
		payload.CompanyID,
	)
	if err != nil {
		return fmt.Errorf("ESG extraction task failed: %w", err)
	}

	log.Printf("[Asynq Worker] Successfully completed ESG Extraction Task for Company [%s], Profile ID [%s]", payload.CompanyName, esgProfile.ID)
	return nil
}

func stringsContainsAny(s string, keywords ...string) bool {
	for _, kw := range keywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}
