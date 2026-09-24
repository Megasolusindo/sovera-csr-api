package main

import (
	"context"
	"log"
	"time"

	"github.com/hibiken/asynq"

	"sovera-core-api/internal/config"
	"sovera-core-api/internal/pkg/telegram"
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
	openclawWorker := queue.NewOpenClawWorker()
	healthCheckWorker := queue.NewURLHealthCheckWorker(dbPool)
	companyLinkedInWorker := queue.NewCompanyLinkedInWorker(dbPool)
	companyInstagramWorker := queue.NewCompanyInstagramWorker(dbPool)
	telegramNotifier := telegram.NewNotifier(cfg.TelegramBotToken, cfg.TelegramChatID)
	serperQuotaWorker := queue.NewSerperQuotaWorker(cfg.SerperAPIKey, telegramNotifier)
	companyLinkedInDiscoveryWorker := queue.NewCompanyLinkedInDiscoveryWorker(dbPool, cfg.SerperAPIKey)
	companyInstagramDiscoveryWorker := queue.NewCompanyInstagramDiscoveryWorker(dbPool, cfg.SerperAPIKey)
	companyContactDiscoveryWorker := queue.NewCompanyContactDiscoveryWorker(dbPool, cfg.SerperAPIKey)

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

	// Schedule task:company_linkedin_batch_check every 12 hours (cron: "0 */12 * * *")
	linkedInBatchTask, err := queue.NewCompanyLinkedInBatchCheckTask()
	if err == nil {
		startupClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.RedisURL})
		if info, enqueueErr := startupClient.Enqueue(linkedInBatchTask); enqueueErr != nil {
			log.Printf("Notice: Could not enqueue instant startup LinkedIn batch task: %v", enqueueErr)
		} else {
			log.Printf("Enqueued instant startup LinkedIn batch check to Redis queue (TaskID: %s)", info.ID)
		}
		startupClient.Close()

		if entryID, err := scheduler.Register("0 */12 * * *", linkedInBatchTask); err != nil {
			log.Printf("Warning: Could not register company LinkedIn batch check cron: %v", err)
		} else {
			log.Printf("Registered company LinkedIn batch check cron with entry ID: %s", entryID)
		}
	}

	// Schedule task:company_linkedin_discovery_batch every 12 hours (cron: "0 */12 * * *")
	linkedInDiscoveryTask, err := queue.NewCompanyLinkedInDiscoveryBatchTask()
	if err == nil {
		startupClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.RedisURL})
		if info, enqueueErr := startupClient.Enqueue(linkedInDiscoveryTask); enqueueErr != nil {
			log.Printf("Notice: Could not enqueue instant startup LinkedIn discovery task: %v", enqueueErr)
		} else {
			log.Printf("Enqueued instant startup LinkedIn discovery sweep to Redis queue (TaskID: %s)", info.ID)
		}
		startupClient.Close()

		if entryID, err := scheduler.Register("0 */12 * * *", linkedInDiscoveryTask); err != nil {
			log.Printf("Warning: Could not register company LinkedIn discovery cron: %v", err)
		} else {
			log.Printf("Registered company LinkedIn discovery cron with entry ID: %s", entryID)
		}
	}

	// Schedule task:company_instagram_batch_check every 12 hours (cron: "0 */12 * * *")
	instagramBatchTask, err := queue.NewCompanyInstagramBatchCheckTask()
	if err == nil {
		startupClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.RedisURL})
		if info, enqueueErr := startupClient.Enqueue(instagramBatchTask); enqueueErr != nil {
			log.Printf("Notice: Could not enqueue instant startup Instagram batch task: %v", enqueueErr)
		} else {
			log.Printf("Enqueued instant startup Instagram batch check to Redis queue (TaskID: %s)", info.ID)
		}
		startupClient.Close()

		if entryID, err := scheduler.Register("0 */12 * * *", instagramBatchTask); err != nil {
			log.Printf("Warning: Could not register company Instagram batch check cron: %v", err)
		} else {
			log.Printf("Registered company Instagram batch check cron with entry ID: %s", entryID)
		}
	}

	// Schedule task:company_instagram_discovery_batch every 12 hours (cron: "0 */12 * * *")
	instagramDiscoveryTask, err := queue.NewCompanyInstagramDiscoveryBatchTask()
	if err == nil {
		startupClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.RedisURL})
		if info, enqueueErr := startupClient.Enqueue(instagramDiscoveryTask); enqueueErr != nil {
			log.Printf("Notice: Could not enqueue instant startup Instagram discovery task: %v", enqueueErr)
		} else {
			log.Printf("Enqueued instant startup Instagram discovery sweep to Redis queue (TaskID: %s)", info.ID)
		}
		startupClient.Close()

		if entryID, err := scheduler.Register("0 */12 * * *", instagramDiscoveryTask); err != nil {
			log.Printf("Warning: Could not register company Instagram discovery cron: %v", err)
		} else {
			log.Printf("Registered company Instagram discovery cron with entry ID: %s", entryID)
		}
	}

	// Schedule task:company_contact_discovery every 6 hours (cron: "0 */6 * * *")
	contactDiscoveryTask, err := queue.NewCompanyContactDiscoveryBatchTask()
	if err == nil {
		startupClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.RedisURL})
		if info, enqueueErr := startupClient.Enqueue(contactDiscoveryTask); enqueueErr != nil {
			log.Printf("Notice: Could not enqueue instant startup contact discovery task: %v", enqueueErr)
		} else {
			log.Printf("Enqueued instant startup company contact discovery sweep to Redis queue (TaskID: %s)", info.ID)
		}
		startupClient.Close()

		if entryID, err := scheduler.Register("0 */6 * * *", contactDiscoveryTask); err != nil {
			log.Printf("Warning: Could not register company contact discovery cron: %v", err)
		} else {
			log.Printf("Registered company contact discovery cron with entry ID: %s", entryID)
		}
	}

	// Serper quota monitoring moved 100% to web-scraper discovery engine


	go func() {
		fbWorker := queue.NewCompanyFacebookWorker(dbPool)
		_ = fbWorker.HandleCompanyFacebookBatchCheck(context.Background(), nil)
	}()

	go func() {
		ytWorker := queue.NewCompanyYoutubeWorker(dbPool)
		_ = ytWorker.HandleCompanyYoutubeBatchCheck(context.Background(), nil)
	}()

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
				queue.QueueDispatchCrawling:             10,
				queue.QueuePollPendingTasks:             5,
				queue.QueueEnrichMissingWebsites:        5,
				queue.QueueURLHealthCheck:               2,
				queue.QueueCompanyLinkedInBatchCheck:    2,
				queue.QueueCompanyLinkedInDiscoveryBatch:  2,
				queue.QueueCompanyInstagramBatchCheck:   2,
				queue.QueueCompanyInstagramDiscoveryBatch: 2,
				queue.QueueCompanyContactDiscoveryBatch:   2,
				queue.QueueSerperQuota:                  2,
				queue.QueueRawIngestion:                 10,
				queue.QueueLLMExtraction:                5,
				queue.QueueESGExtraction:                5,
				queue.QueueProposalGeneration:           3,
				queue.QueueOpenClawResearch:          2,
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
	mux.HandleFunc(queue.TypeOpenClawResearch, openclawWorker.HandleOpenClawResearchTask)
	mux.HandleFunc(queue.TypeURLHealthCheck, healthCheckWorker.HandleURLHealthCheck)
	mux.HandleFunc(queue.TypeCompanyLinkedInBatchCheck, companyLinkedInWorker.HandleCompanyLinkedInBatchCheck)
	mux.HandleFunc(queue.TypeCompanyLinkedInDiscoveryBatch, companyLinkedInDiscoveryWorker.HandleCompanyLinkedInDiscoveryBatch)
	mux.HandleFunc(queue.TypeCompanyInstagramBatchCheck, companyInstagramWorker.HandleCompanyInstagramBatchCheck)
	mux.HandleFunc(queue.TypeCompanyInstagramDiscoveryBatch, companyInstagramDiscoveryWorker.HandleCompanyInstagramDiscoveryBatch)
	mux.HandleFunc(queue.TypeCompanyContactDiscoveryBatch, companyContactDiscoveryWorker.HandleCompanyContactDiscoveryBatch)
	mux.HandleFunc(queue.TypeSerperQuotaCheck, serperQuotaWorker.HandleSerperQuotaCheck)


	log.Println("Asynq Worker server listening for queue jobs...")
	if err := srv.Run(mux); err != nil {
		log.Fatalf("Could not run Asynq worker server: %v", err)
	}
}
