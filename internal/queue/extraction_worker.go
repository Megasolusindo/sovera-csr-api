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
}

func NewExtractionWorker(
	geminiService *ai.GeminiService,
	signalRepo *repository.SignalRepository,
	norm *normalizer.Normalizer,
	resolver *entityresolver.EntityResolver,
	esgExt *esgextractor.ESGExtractor,
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
		return fmt.Errorf("LLM signal extraction failed: %w", err)
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

	// 5. Persist Extracted Signal, company_id & Vector to Database
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
