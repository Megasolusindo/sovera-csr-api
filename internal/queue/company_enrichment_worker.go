package queue

import (
	"context"
	"fmt"
	"log"

	"github.com/hibiken/asynq"

	"sovera-core-api/internal/service/companyenricher"
)

const (
	TypeEnrichMissingWebsites  = "task:enrich_missing_websites"
	QueueEnrichMissingWebsites = "company_enrichment"
)

type CompanyEnrichmentWorker struct {
	enricher *companyenricher.EnricherService
}

func NewCompanyEnrichmentWorker(enricher *companyenricher.EnricherService) *CompanyEnrichmentWorker {
	return &CompanyEnrichmentWorker{enricher: enricher}
}

func NewEnrichMissingWebsitesTask() (*asynq.Task, error) {
	return asynq.NewTask(TypeEnrichMissingWebsites, nil, asynq.Queue(QueueEnrichMissingWebsites)), nil
}

func (w *CompanyEnrichmentWorker) HandleEnrichMissingWebsites(ctx context.Context, task *asynq.Task) error {
	log.Println("[Worker] Starting scheduled background job: Enrich Missing Company Websites...")
	if w.enricher == nil {
		return fmt.Errorf("enricher service is nil")
	}

	count, err := w.enricher.EnrichMissingWebsites(ctx, 100)
	if err != nil {
		log.Printf("[Worker] Error during website enrichment job: %v", err)
		return err
	}

	log.Printf("[Worker] Successfully enriched %d missing company websites.", count)
	return nil
}
