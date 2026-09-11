package main

import (
	"context"
	"log"
	"time"

	"github.com/hibiken/asynq"

	"sovera-core-api/internal/config"
	"sovera-core-api/internal/queue"
	"sovera-core-api/internal/repository"
	"sovera-core-api/internal/service/ai"
	"sovera-core-api/internal/service/companyenricher"
	"sovera-core-api/internal/service/crawler"
	"sovera-core-api/internal/service/entityresolver"
	"sovera-core-api/internal/service/esgextractor"
	"sovera-core-api/internal/service/normalizer"
)

func main() {
	cfg := config.LoadConfig()
	log.Printf("Starting Sovera Background Worker connected to Redis [%s]...", cfg.RedisURL)

	// 1. Database Pool Connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbPool, err := repository.InitDBPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Printf("Warning: Worker running without live DB pool: %v", err)
	} else {
		defer dbPool.Close()
	}

	// 2. Services & Worker Dependencies
	geminiService := ai.NewGeminiService(cfg.AIAPIKey)
	textNormalizer := normalizer.NewNormalizer()
	companyRepo := repository.NewCompanyRepository(dbPool)
	entityResolver := entityresolver.NewEntityResolver(companyRepo)
	signalRepo := repository.NewSignalRepository(dbPool)
	esgRepo := repository.NewESGProfileRepository(dbPool)
	esgExtractor := esgextractor.NewESGExtractor(geminiService, esgRepo, textNormalizer, entityResolver)
	crawlerRepo := repository.NewCrawlerRepository(dbPool)
	dispatcher := crawler.NewDispatcher(cfg)
	companyEnricher := companyenricher.NewEnricherService(companyRepo, dispatcher)

	tokenLogRepo := repository.NewTokenLogRepository(dbPool)
	extractionWorker := queue.NewExtractionWorker(geminiService, signalRepo, textNormalizer, entityResolver, esgExtractor, tokenLogRepo)
	dispatcherWorker := queue.NewCrawlerDispatcherHandler(crawlerRepo, dispatcher)
	enrichmentWorker := queue.NewCompanyEnrichmentWorker(companyEnricher)
	healthCheckWorker := queue.NewURLHealthCheckWorker(dbPool)

	// 3. Asynq Scheduler for Periodic Tasks
	scheduler := asynq.NewScheduler(
		asynq.RedisClientOpt{Addr: cfg.RedisURL},
		&asynq.SchedulerOpts{},
	)

	// Schedule task:enrich_missing_websites every 6 hours (cron: "0 */6 * * *")
	enrichTask, err := queue.NewEnrichMissingWebsitesTask()
	if err == nil {
		if entryID, err := scheduler.Register("0 */6 * * *", enrichTask); err != nil {
			log.Printf("Warning: Could not register company website enrichment cron: %v", err)
		} else {
			log.Printf("Registered company website enrichment cron with entry ID: %s", entryID)
		}
	}

	// Schedule task:dispatch_crawling
	dispatchTask, err := queue.NewDispatchCrawlingTask()
	if err == nil {
		// Instant startup sweep: Enqueue 1x dispatch task immediately on worker launch/restart
		startupClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.RedisURL})
		if info, enqueueErr := startupClient.Enqueue(dispatchTask); enqueueErr != nil {
			log.Printf("Notice: Could not enqueue instant startup crawling task: %v", enqueueErr)
		} else {
			log.Printf("Enqueued instant startup crawling target check to Redis queue (TaskID: %s)", info.ID)
		}
		startupClient.Close()

		// Register periodic cron to run every 2 minutes (cron: "*/2 * * * *")
		if entryID, err := scheduler.Register("*/2 * * * *", dispatchTask); err != nil {
			log.Printf("Warning: Could not register periodic crawling dispatch cron: %v", err)
		} else {
			log.Printf("Registered periodic crawling dispatch cron with entry ID: %s", entryID)
		}
	}

	// Schedule task:poll_pending_tasks every 15 minutes
	pollTask, err := queue.NewPollPendingTasksTask()
	if err == nil {
		if entryID, err := scheduler.Register("*/15 * * * *", pollTask); err != nil {
			log.Printf("Warning: Could not register fallback polling cron: %v", err)
		} else {
			log.Printf("Registered fallback task polling cron with entry ID: %s", entryID)
		}
	}

	// Schedule task:url_health_check every 5 minutes (cron: "*/5 * * * *")
	healthCheckTask, err := queue.NewURLHealthCheckTask()
	if err == nil {
		if entryID, err := scheduler.Register("*/5 * * * *", healthCheckTask); err != nil {
			log.Printf("Warning: Could not register URL health check cron: %v", err)
		} else {
			log.Printf("Registered URL health check cron with entry ID: %s", entryID)
		}
	}

	go func() {
		if err := scheduler.Run(); err != nil {
			log.Printf("Scheduler error: %v", err)
		}
	}()

	// 4. Asynq Server Setup
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: cfg.RedisURL},
		asynq.Config{
			Concurrency: 10,
			Queues: map[string]int{
				queue.QueueDispatchCrawling:      10,
				queue.QueuePollPendingTasks:      5,
				queue.QueueEnrichMissingWebsites: 5,
				queue.QueueURLHealthCheck:        2,
				queue.QueueRawIngestion:          10,
				queue.QueueLLMExtraction:         5,
				queue.QueueESGExtraction:         5,
				queue.QueueProposalGeneration:    3,
			},
		},
	)

	mux := asynq.NewServeMux()

	// Register Asynq task handlers
	mux.HandleFunc(queue.TypeDispatchCrawling, dispatcherWorker.HandleDispatchCrawlingTask)
	mux.HandleFunc(queue.TypePollPendingTasks, dispatcherWorker.HandlePollPendingTasks)
	mux.HandleFunc(queue.TypeEnrichMissingWebsites, enrichmentWorker.HandleEnrichMissingWebsites)
	mux.HandleFunc(queue.TypeLLMExtraction, extractionWorker.ProcessExtractionTask)
	mux.HandleFunc(queue.TypeESGExtraction, extractionWorker.ProcessESGTask)
	mux.HandleFunc(queue.TypeURLHealthCheck, healthCheckWorker.HandleURLHealthCheck)

	log.Println("Asynq Worker server listening for queue jobs...")
	if err := srv.Run(mux); err != nil {
		log.Fatalf("Could not run Asynq worker server: %v", err)
	}
}
