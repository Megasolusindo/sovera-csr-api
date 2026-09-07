package main

import (
	"context"
	"log"

	"sovera-core-api/internal/config"
	"sovera-core-api/internal/repository"
	"sovera-core-api/internal/service/companyenricher"
	"sovera-core-api/internal/service/crawler"
)

func main() {
	log.Println("=== SOVERA COMPANY WEBSITE ENRICHMENT AGENT ===")
	cfg := config.LoadConfig()

	ctx := context.Background()
	dbPool, err := repository.InitDBPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Fatal: Could not connect to database: %v", err)
	}
	defer dbPool.Close()

	companyRepo := repository.NewCompanyRepository(dbPool)
	dispatcher := crawler.NewDispatcher(cfg)
	enricher := companyenricher.NewEnricherService(companyRepo, dispatcher)

	log.Println("Starting automated discovery & verification for companies with missing website...")
	count, err := enricher.EnrichMissingWebsites(ctx, 500)
	if err != nil {
		log.Fatalf("Error running company website enrichment: %v", err)
	}

	log.Printf("Successfully completed enrichment agent run. Total verified websites updated: %d", count)
}
