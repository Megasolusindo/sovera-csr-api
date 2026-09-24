package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/redis/go-redis/v9"

	"github.com/hibiken/asynq"

	"sovera-core-api/internal/config"
	"sovera-core-api/internal/handler"
	"sovera-core-api/internal/middleware"
	"sovera-core-api/internal/payment"
	"sovera-core-api/internal/payment/midtrans"
	"sovera-core-api/internal/pkg/serper"
	"sovera-core-api/internal/pkg/telegram"
	"sovera-core-api/internal/queue"
	"sovera-core-api/internal/repository"
	"sovera-core-api/internal/service"
	"sovera-core-api/internal/service/ai"
	"sovera-core-api/internal/service/exporter"
	"sovera-core-api/internal/service/keyperson"
	"sovera-core-api/internal/service/normalizer"
	"sovera-core-api/internal/service/storage"
	"sovera-core-api/internal/service/urlverifier"
)

func main() {
	// 1. Load Environment Configuration
	cfg := config.LoadConfig()
	missing := cfg.Validate()
	if len(missing) > 0 {
		log.Fatalf("FATAL: Required environment variables are not set: %v. Please configure them before starting the server.", missing)
	}

	isProd := cfg.Environment != "development"
	log.Printf("Starting Sovera Core API Server in [%s] mode...", cfg.Environment)

	// 2. Initialize PostgreSQL DB Pool with pgx/v5
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbPool, err := repository.InitDBPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Printf("Warning: Database pool initialization failed: %v", err)
	} else {
		log.Println("PostgreSQL connection pool initialized successfully.")
		defer dbPool.Close()

		// Run SQL DDL Migrations
		if err := repository.RunMigrations(context.Background(), dbPool, "db/migrations"); err != nil {
			log.Printf("Warning: Automatic migration run failed: %v", err)
		} else {
			log.Println("Database DDL migrations executed successfully.")
		}
	}

	// 3. Initialize Asynq Redis Client
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.RedisURL})
	defer asynqClient.Close()

	// 4. Initialize Redis client for token blacklist
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisURL,
	})
	defer rdb.Close()

	// 4. Initialize Services & Repositories
	midtransClient := midtrans.NewClient(cfg.MidtransServerKey, cfg.MidtransClientKey, cfg.MidtransIsProduction)
	paymentGateway := payment.NewGatewayFactory(cfg)
	geminiService := ai.NewGeminiService(cfg.AIAPIKey)
	docExporter := exporter.NewDocumentExporter()
	textNormalizer := normalizer.NewNormalizer()
	storageService := storage.NewStorageService()
	telegramNotifier := telegram.NewNotifier(cfg.TelegramBotToken, cfg.TelegramChatID)
	serperMonitor := serper.InitDefaultMonitor(cfg.SerperAPIKey, telegramNotifier)
	go func() {
		time.Sleep(2 * time.Second)
		_, _ = serperMonitor.CheckAndNotify(context.Background())
	}()

	signalRepo := repository.NewSignalRepository(dbPool)
	programRepo := repository.NewProgramRepository(dbPool)
	dealRepo := repository.NewDealRepository(dbPool)
	userRepo := repository.NewUserRepository(dbPool)
	companyRepo := repository.NewCompanyRepository(dbPool)
	companyCSRProgramRepo := repository.NewCompanyCSRProgramRepository(dbPool)
	esgProfileRepo := repository.NewESGProfileRepository(dbPool)
	templateRepo := repository.NewTemplateRepository(dbPool)
	tokenLogRepo := repository.NewTokenLogRepository(dbPool)
	orgRepo := repository.NewOrganizationRepository(dbPool)
	crawlerRepo := repository.NewCrawlerRepository(dbPool)
	keyPersonRepo := repository.NewKeyPersonRepository(dbPool)
	keyPersonService := keyperson.NewKeyPersonService(keyPersonRepo)

	subRepo := repository.NewSubscriptionRepository(dbPool)
	subService := service.NewSubscriptionService(subRepo, midtransClient, paymentGateway)

	orgAIRepo := repository.NewOrganizationAIRepository(dbPool)
	orgAIChatService := ai.NewOrganizationAIChatService(cfg.AIAPIKey, orgAIRepo)

	claimRepo := repository.NewCompanyClaimRepository(dbPool)
	claimHandler := handler.NewCompanyClaimHandler(claimRepo, companyRepo, telegramNotifier)

	oppRepo := repository.NewCSROpportunityRepository(dbPool)
	oppHandler := handler.NewCSROpportunityHandler(oppRepo)

	proposalRepo := repository.NewProposalRepository(dbPool)
	proposalHandler := handler.NewProposalHandler(proposalRepo)

	// 5. Create Rate Limiter Stores for Tenant Control
	aiRateLimiterStore := middleware.NewRateLimiterStore()
	generalRateLimiterStore := middleware.NewRateLimiterStore()
	ipRateLimiterStore := middleware.NewIPRateLimiterStore()
	ipLimit := middleware.IPRateLimit(ipRateLimiterStore, 60, 1*time.Minute, "Public API")
	loginLimit := middleware.IPRateLimit(ipRateLimiterStore, 10, 1*time.Minute, "Login attempts")
	registerLimit := middleware.IPRateLimit(ipRateLimiterStore, 5, 1*time.Minute, "Registration attempts")

	// 6. Create Fiber Web Application
	app := fiber.New(fiber.Config{
		AppName:      "Sovera Core API & Intelligence Engine v1.0",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
		ErrorHandler: sanitizedErrorHandler(isProd),
		BodyLimit:    10 * 1024 * 1024, // 10MB max request body
	})

	// 7. Global Middlewares (CORS MUST BE FIRST)
	app.Use(cors.New(cors.Config{
		AllowOrigins:     getCORSOrigins(cfg),
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Requested-With",
		AllowMethods:     "GET, POST, HEAD, PUT, DELETE, PATCH, OPTIONS",
		AllowCredentials: false,
	}))
	app.Use(func(c *fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("X-XSS-Protection", "1; mode=block")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Set("Permissions-Policy", "geolocation=(), camera=(), microphone=()")
		if isProd {
			c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			c.Set("Content-Security-Policy", "default-src 'self'")
		}
		return c.Next()
	})
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))

	// 8. Register Handlers & Controllers
	healthHandler := handler.NewHealthHandler(dbPool)
	webhookHandler := handler.NewWebhookHandler(dbPool, asynqClient, textNormalizer, telegramNotifier)
	signalHandler := handler.NewSignalHandler(signalRepo)
	programHandler := handler.NewProgramHandler(programRepo, geminiService)
	dealHandler := handler.NewDealHandler(dealRepo, programRepo, signalRepo, templateRepo, tokenLogRepo, storageService, geminiService, docExporter)
	authHandler := handler.NewAuthHandler(userRepo, cfg.JWTSecret, rdb)
	linkedinVerifier := urlverifier.NewLinkedInVerifier()
	companyHandler := handler.NewCompanyHandler(companyRepo, companyCSRProgramRepo, linkedinVerifier)
	templateHandler := handler.NewTemplateHandler(templateRepo, tokenLogRepo, storageService)
	adminHandler := handler.NewAdminHandler(orgRepo, userRepo, crawlerRepo, tokenLogRepo, esgProfileRepo, telegramNotifier, dbPool)
	aiAgentHandler := handler.NewAIAgentHandler(dbPool)
	orgAIChatHandler := handler.NewOrganizationAIChatHandler(orgAIChatService, orgAIRepo)
	urlVerifierHandler := handler.NewURLVerifierHandler(linkedinVerifier)
	keyPersonHandler := handler.NewKeyPersonHandler(keyPersonService)
	subHandler := handler.NewSubscriptionHandler(subService)
	paymentWebhookHandler := handler.NewPaymentWebhookHandler(subService, paymentGateway)
	faspayWebhookHandler := handler.NewFaspayWebhookHandler(subService, paymentGateway)
	intelligenceRepo := repository.NewIntelligenceRepository(dbPool)
	intelligenceHandler := handler.NewIntelligenceHandler(intelligenceRepo)

	// Seed default OpenClaw AI Agent credential if DB pool is ready
	if dbPool != nil && cfg.OpenClawAgentToken != "" {
		aiRepo := repository.NewAIAgentRepository(dbPool)
		if err := aiRepo.SeedAgentCredential(context.Background(), "openclaw-research-agent", cfg.OpenClawAgentToken, []string{"research:create", "company:read", "csr_program:create"}); err != nil {
			log.Printf("[AI Agent Seed Error] %v", err)
		}
		if err := aiRepo.SeedAgentCredential(context.Background(), "openclaw-agent-legacy", "openclaw_agent_live_key_998877665544", []string{"research:create", "company:read", "csr_program:create"}); err != nil {
			log.Printf("[AI Agent Seed Error Legacy] %v", err)
		}
	}

	// Root & Health check routes (public)
	app.Get("/health", healthHandler.HealthCheck)

	apiV1 := app.Group("/api/v1")
	apiV1.Get("/health", healthHandler.HealthCheck)
	apiV1.Post("/webhooks/midtrans", paymentWebhookHandler.HandleMidtransWebhook)
	apiV1.Post("/webhooks/faspay", faspayWebhookHandler.HandleFaspayWebhook)
	apiV1.Get("/subscription/plans", subHandler.ListPlans)

	aiToolsHandler := handler.NewAIToolsHandler(dbPool)

	// ─── OpenClaw AI Agent Dedicated API Namespace (/api/v1/ai/*) ────────────
	aiGroup := apiV1.Group("/ai", middleware.AIAgentAuthMiddleware(dbPool, ""))
	aiGroup.Get("/companies/search", aiAgentHandler.SearchCompanies)
	aiGroup.Get("/companies/:id", aiAgentHandler.GetCompanyDetails)
	aiGroup.Post("/research/findings", aiAgentHandler.SubmitResearchFinding)
	aiGroup.Get("/research/findings", aiAgentHandler.ListResearchFindings)
	aiGroup.Post("/csr-programs", aiAgentHandler.SubmitResearchFinding)
	aiGroup.Get("/watchlist", aiAgentHandler.GetActiveWatchlist)
	aiGroup.Post("/watchlist", aiAgentHandler.AddWatchlistCompany)
	aiGroup.Post("/companies/:id/monitor", aiAgentHandler.ToggleCompanyWatchlist)
	aiGroup.Post("/companies/:id/enrich", aiAgentHandler.EnrichCompany)
	aiGroup.Post("/matching", aiAgentHandler.MatchProgram)
	aiGroup.Get("/search", aiAgentHandler.SearchCorporateData)

	// Granular Tool API Surface (§4.4)
	toolsGroup := aiGroup.Group("/tools")
	toolsGroup.Post("/search_companies", aiToolsHandler.SearchCompanies)
	toolsGroup.Post("/get_csr_signals", aiToolsHandler.GetCSRSignals)
	toolsGroup.Post("/match_opportunity", aiToolsHandler.MatchOpportunity)
	toolsGroup.Post("/send_session_alert", aiToolsHandler.SendSessionAlert)

	// ─── Dashboard AI Chat Dedicated API Namespace (/api/v1/tenant-ai/* & /api/v1/org-ai/*) ───
	tenantAIGuard := middleware.OrgAIAuthMiddleware(cfg.JWTSecret)
	dashboardChatLimit := middleware.TenantRateLimit(aiRateLimiterStore, 20, 1*time.Minute, "Dashboard AI Chat Assistant")

	tenantAIGroup := apiV1.Group("/tenant-ai", tenantAIGuard, dashboardChatLimit)
	tenantAIGroup.Post("/chat", orgAIChatHandler.PostChat)
	tenantAIGroup.Get("/search", orgAIChatHandler.SearchSignals)
	tenantAIGroup.Get("/companies/search", orgAIChatHandler.SearchCompanies)
	tenantAIGroup.Get("/watchlist", orgAIChatHandler.GetWatchlist)
	tenantAIGroup.Get("/conversations", orgAIChatHandler.ListConversations)
	tenantAIGroup.Get("/conversations/:id/messages", orgAIChatHandler.GetConversationMessages)
	tenantAIGroup.Delete("/conversations/:id", orgAIChatHandler.ArchiveConversation)
	tenantAIGroup.Get("/admin/conversations", orgAIChatHandler.AdminListConversations)
	tenantAIGroup.Get("/admin/conversations/:id/messages", orgAIChatHandler.AdminGetConversationMessages)

	orgAIGroup := apiV1.Group("/org-ai", tenantAIGuard, dashboardChatLimit)
	orgAIGroup.Post("/chat", orgAIChatHandler.PostChat)
	orgAIGroup.Get("/search", orgAIChatHandler.SearchSignals)
	orgAIGroup.Get("/companies/search", orgAIChatHandler.SearchCompanies)
	orgAIGroup.Get("/watchlist", orgAIChatHandler.GetWatchlist)
	orgAIGroup.Get("/conversations", orgAIChatHandler.ListConversations)
	orgAIGroup.Get("/conversations/:id/messages", orgAIChatHandler.GetConversationMessages)
	orgAIGroup.Delete("/conversations/:id", orgAIChatHandler.ArchiveConversation)
	orgAIGroup.Get("/admin/conversations", orgAIChatHandler.AdminListConversations)
	orgAIGroup.Get("/admin/conversations/:id/messages", orgAIChatHandler.AdminGetConversationMessages)

	// ─── Auth Routes (PUBLIC — rate-limited to prevent brute force) ─────────────────
	auth := apiV1.Group("/auth")
	auth.Post("/register", registerLimit, authHandler.Register)
	auth.Post("/login", loginLimit, authHandler.Login)
	auth.Post("/logout", authHandler.Logout)
	auth.Get("/me", middleware.AuthenticateJWT(cfg.JWTSecret, rdb), authHandler.Me)

	// ─── Webhook Ingestion (Protected by HMAC Verification) ──────────────────
	apiV1.Post(
		"/webhooks/crawler",
		middleware.VerifyHMAC(cfg.WebhookSecretKey),
		webhookHandler.HandleCrawlerWebhook,
	)

	// ─── JWT & AI Agent Protected Routes ──────────────────────────────────────
	jwtGuard := middleware.AuthenticateJWTOrAgentKey(cfg.JWTSecret, dbPool, rdb)
	tenantGeneralLimit := middleware.TenantRateLimit(generalRateLimiterStore, 120, 1*time.Minute, "API General")
	tenantAILimit := middleware.TenantRateLimit(aiRateLimiterStore, 10, 1*time.Minute, "Generasi AI Proposal & Pitch")

	// Platform Admin Console — Full System Control & Analytics (requires SUPERADMIN role)
	adminGroup := apiV1.Group("/admin", jwtGuard, middleware.RequireRole("SUPERADMIN"))
	adminGroup.Get("/plans", subHandler.ListPlans)
	adminGroup.Post("/plans", subHandler.UpsertPlan)
	adminGroup.Get("/organizations", adminHandler.ListOrganizations)
	adminGroup.Post("/organizations", adminHandler.CreateOrganization)
	adminGroup.Put("/organizations/:id", adminHandler.UpdateOrganization)
	adminGroup.Patch("/organizations/:id", adminHandler.UpdateOrganization)
	adminGroup.Get("/users", adminHandler.ListUsers)
	adminGroup.Get("/scraping-jobs", adminHandler.ListScrapingJobs)
	adminGroup.Post("/scraping-jobs", adminHandler.CreateScrapingJob)
	adminGroup.Get("/sources", adminHandler.ListSources)
	adminGroup.Post("/sources", adminHandler.CreateScrapingJob)
	adminGroup.Get("/documents", adminHandler.ListDocuments)
	adminGroup.Get("/esg-intelligence", adminHandler.GetESGIntelligence)
	adminGroup.Get("/analytics", adminHandler.GetAnalytics)
	adminGroup.Get("/ai-metering", adminHandler.GetAIMetering)
	adminGroup.Get("/crawler-errors", adminHandler.GetCrawlerErrors)
	adminGroup.Post("/crawler-errors/reset", adminHandler.ResetCrawlerErrors)
	adminGroup.Get("/ai/findings", adminHandler.ListAIFindings)
	adminGroup.Post("/ai/findings/:id/review", adminHandler.ReviewAIFinding)
	adminGroup.Post("/ai/trigger-research", adminHandler.TriggerOpenClawResearch)
	adminGroup.Post("/ai/trigger-enrichment", adminHandler.TriggerOpenClawEnrichment)
	adminGroup.Post("/ai/trigger-matching", adminHandler.TriggerOpenClawMatching)
	adminGroup.Get("/ai/watchlist", adminHandler.ListAIWatchlist)
	adminGroup.Delete("/ai/watchlist/:id", adminHandler.RemoveAIWatchlist)
	adminGroup.Post("/ai/watchlist", aiAgentHandler.ToggleCompanyWatchlist)
	adminGroup.Post("/notifications/telegram-test", adminHandler.TestTelegramNotification)
	adminGroup.Post("/idx-sync", adminHandler.TriggerIDXSync)
	adminGroup.Post("/dedup", adminHandler.TriggerDeduplication)
	adminGroup.Post("/health-check", func(c *fiber.Ctx) error {
		task, err := queue.NewURLHealthCheckTask()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.RedisURL})
		defer asynqClient.Close()
		info, err := asynqClient.Enqueue(task)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{
			"message": "URL health check task enqueued",
			"task_id": info.ID,
		})
	})

	// Admin Claim Verification
	adminGroup.Post("/claims/:id/verify", claimHandler.VerifyClaim)

	// Two-Sided Platform Corporate Claims, Opportunities & Proposals
	apiV1.Post("/corporate/claims", jwtGuard, claimHandler.SubmitClaim)
	apiV1.Post("/corporate/opportunities", jwtGuard, oppHandler.CreateOpportunity)
	apiV1.Get("/public/opportunities", oppHandler.ListOpportunities)
	apiV1.Get("/opportunities", oppHandler.ListOpportunities)

	apiV1.Post("/proposals", jwtGuard, proposalHandler.SubmitProposal)
	apiV1.Get("/proposals/sent", jwtGuard, proposalHandler.ListSentProposals)
	apiV1.Get("/corporate/proposals", jwtGuard, proposalHandler.ListIncomingProposals)
	apiV1.Patch("/corporate/proposals/:id/status", jwtGuard, proposalHandler.UpdateProposalStatus)

	// Companies Directory & CSR Master Programs & Web Sources & ESG — semua role / public browsing
	apiV1.Get("/stats", jwtGuard, middleware.RequireRole("SUPERADMIN"), adminHandler.GetAnalytics)
	apiV1.Get("/companies", ipLimit, companyHandler.ListCompanies)
	apiV1.Post("/companies", jwtGuard, middleware.RequireRole("CORP_ADMIN", "CSR_MANAGER", "SUPERADMIN", "ORG_ADMIN"), companyHandler.CreateCompany)
	apiV1.Get("/companies/csr-programs", ipLimit, middleware.RequireVisibilityAccess(cfg.JWTSecret), companyHandler.ListCSRPrograms)
	apiV1.Get("/csr-programs", ipLimit, middleware.RequireVisibilityAccess(cfg.JWTSecret), companyHandler.ListCSRPrograms)
	apiV1.Patch("/companies/csr-programs/:id/visibility", jwtGuard, middleware.RequireRole("CORP_ADMIN", "CSR_MANAGER", "SUPERADMIN", "ORG_ADMIN"), companyHandler.UpdateCSRProgramVisibility)
	apiV1.Get("/sources", jwtGuard, adminHandler.ListSources)
	apiV1.Post("/sources", jwtGuard, middleware.RequireRole("SUPERADMIN", "ORG_ADMIN"), adminHandler.CreateScrapingJob)
	apiV1.Get("/documents", jwtGuard, adminHandler.ListDocuments)
	apiV1.Get("/scraping-jobs", jwtGuard, adminHandler.ListScrapingJobs)
	apiV1.Post("/scraping-jobs", jwtGuard, middleware.RequireRole("SUPERADMIN", "ORG_ADMIN"), adminHandler.CreateScrapingJob)
	apiV1.Get("/esg-intelligence", jwtGuard, adminHandler.GetESGIntelligence)

	// Key Person & Social Signals (must be placed before /companies/:id)
	apiV1.Get("/companies/:id/key-persons", jwtGuard, keyPersonHandler.ListKeyPersons)
	apiV1.Post("/companies/:id/key-persons", jwtGuard, middleware.RequireRole("CORP_ADMIN", "CSR_MANAGER", "SUPERADMIN", "ORG_ADMIN"), keyPersonHandler.CreateKeyPerson)
	apiV1.Get("/companies/:id/key-person-signals", jwtGuard, keyPersonHandler.ListSocialSignals)
	apiV1.Post("/companies/:id/key-person-signals", jwtGuard, middleware.RequireRole("CORP_ADMIN", "CSR_MANAGER", "SUPERADMIN", "ORG_ADMIN"), keyPersonHandler.IngestSocialSignal)
	apiV1.Put("/key-persons/:id", jwtGuard, middleware.RequireRole("CORP_ADMIN", "CSR_MANAGER", "SUPERADMIN", "ORG_ADMIN"), keyPersonHandler.UpdateKeyPerson)
	apiV1.Delete("/key-persons/:id", jwtGuard, middleware.RequireRole("CORP_ADMIN", "CSR_MANAGER", "SUPERADMIN", "ORG_ADMIN"), keyPersonHandler.DeleteKeyPerson)

	apiV1.Get("/companies/:id", companyHandler.GetCompany)
	apiV1.Put("/companies/:id", jwtGuard, middleware.RequireRole("CORP_ADMIN", "CSR_MANAGER", "SUPERADMIN", "ORG_ADMIN"), companyHandler.UpdateCompany)
	apiV1.Patch("/companies/:id", jwtGuard, middleware.RequireRole("CORP_ADMIN", "CSR_MANAGER", "SUPERADMIN", "ORG_ADMIN"), companyHandler.UpdateCompany)

	apiV1.Post("/url/verify-linkedin", jwtGuard, urlVerifierHandler.VerifyLinkedInURL)
	apiV1.Get("/url/verify-linkedin", urlVerifierHandler.VerifyLinkedInURL)
	apiV1.Post("/url/verify-linkedin-batch", jwtGuard, companyHandler.TriggerBatchLinkedInVerification)
	apiV1.Post("/url/discover-linkedin-batch", jwtGuard, companyHandler.TriggerBatchLinkedInDiscovery)
	apiV1.Get("/url/linkedin-stats", jwtGuard, companyHandler.GetLinkedInStats)

	apiV1.Post("/url/verify-instagram", jwtGuard, urlVerifierHandler.VerifyInstagramURL)
	apiV1.Get("/url/verify-instagram", urlVerifierHandler.VerifyInstagramURL)
	apiV1.Post("/url/verify-instagram-batch", jwtGuard, companyHandler.TriggerBatchInstagramVerification)
	apiV1.Post("/url/discover-instagram-batch", jwtGuard, companyHandler.TriggerBatchInstagramDiscovery)
	apiV1.Get("/url/instagram-stats", jwtGuard, companyHandler.GetInstagramStats)

	apiV1.Post("/url/verify-facebook", jwtGuard, companyHandler.VerifyFacebookURL)
	apiV1.Get("/url/verify-facebook", jwtGuard, companyHandler.VerifyFacebookURL)
	apiV1.Post("/url/verify-facebook-batch", jwtGuard, companyHandler.TriggerBatchFacebookVerification)
	apiV1.Get("/url/facebook-stats", jwtGuard, companyHandler.GetFacebookStats)

	apiV1.Post("/url/verify-youtube", jwtGuard, companyHandler.VerifyYoutubeURL)
	apiV1.Get("/url/verify-youtube", jwtGuard, companyHandler.VerifyYoutubeURL)
	apiV1.Post("/url/verify-youtube-batch", jwtGuard, companyHandler.TriggerBatchYoutubeVerification)
	apiV1.Get("/url/youtube-stats", jwtGuard, companyHandler.GetYoutubeStats)




	// Corporate Intelligence Feeds — semua role
	apiV1.Get("/signals", jwtGuard, tenantGeneralLimit, signalHandler.ListSignals)
	apiV1.Get("/signals/:id/match-programs", jwtGuard, tenantGeneralLimit, signalHandler.MatchPrograms)

	// CSR Intelligence Workspace (/api/v1/intelligence/*)
	intelligenceGroup := apiV1.Group("/intelligence", jwtGuard)
	intelligenceGroup.Get("/overview", intelligenceHandler.GetOverview)
	intelligenceGroup.Get("/organizations", intelligenceHandler.ListOrganizations)
	intelligenceGroup.Get("/programs", intelligenceHandler.ListPrograms)
	intelligenceGroup.Get("/trends", intelligenceHandler.GetTrends)
	intelligenceGroup.Post("/saved", intelligenceHandler.SaveItem)

	// Institution Programs — GET: semua role | POST, PUT, DELETE: ORG_ADMIN & DIRECTOR only
	apiV1.Get("/programs", jwtGuard, tenantGeneralLimit, programHandler.ListPrograms)
	apiV1.Post("/programs", jwtGuard, tenantGeneralLimit,
		middleware.RequireRole("ORG_ADMIN", "DIRECTOR"),
		programHandler.CreateProgram,
	)
	apiV1.Put("/programs/:id", jwtGuard, tenantGeneralLimit,
		middleware.RequireRole("ORG_ADMIN", "DIRECTOR"),
		programHandler.UpdateProgram,
	)
	apiV1.Delete("/programs/:id", jwtGuard, tenantGeneralLimit,
		middleware.RequireRole("ORG_ADMIN", "DIRECTOR"),
		programHandler.DeleteProgram,
	)


	// Deal Pipeline & Proposal Studio — semua role (ownership filter via org_id RLS & AI Rate Limiting)
	apiV1.Get("/deals", jwtGuard, tenantGeneralLimit, dealHandler.ListDeals)
	apiV1.Post("/deals", jwtGuard, tenantGeneralLimit, dealHandler.CreateDeal)
	apiV1.Patch("/deals/:id/stage", jwtGuard, tenantGeneralLimit, dealHandler.UpdateStage)
	apiV1.Post("/deals/:id/generate-pitch", jwtGuard, tenantAILimit, dealHandler.GeneratePitch)
	apiV1.Post("/deals/:id/export", jwtGuard, tenantGeneralLimit, dealHandler.ExportProposal)

	// Tenant Subscriptions & Billing
	apiV1.Get("/subscription/plans", subHandler.ListPlans)
	apiV1.Get("/subscription/me", jwtGuard, subHandler.GetSubscription)
	apiV1.Post("/subscription/checkout", jwtGuard, subHandler.Checkout)
	apiV1.Get("/subscription/invoices", jwtGuard, subHandler.ListInvoices)

	// Master Template & Token Usage Management — ORG_ADMIN & DIRECTOR only
	apiV1.Get("/settings/templates", jwtGuard, templateHandler.GetTemplates)
	apiV1.Post("/settings/templates/upload", jwtGuard, templateHandler.UploadTemplate)
	apiV1.Delete("/settings/templates/:type", jwtGuard, templateHandler.ResetTemplate)
	apiV1.Get("/settings/token-usage", jwtGuard, templateHandler.GetTokenUsage)

	// 8. Graceful Shutdown Handler
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		addr := fmt.Sprintf(":%s", cfg.Port)
		log.Printf("Sovera API Listening on http://localhost%s/api/v1", addr)
		if err := app.Listen(addr); err != nil {
			log.Fatalf("Fiber server error: %v", err)
		}
	}()

	<-shutdownChan
	log.Println("Gracefully shutting down Sovera API server...")

	_ = app.Shutdown()
	log.Println("Server stopped cleanly.")
}

func getCORSOrigins(cfg *config.Config) string {
	origins := getEnv("CORS_ALLOWED_ORIGINS", "")
	if origins != "" {
		return origins
	}
	return "*"
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func sanitizedErrorHandler(isProd bool) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		if isProd {
			log.Printf("INTERNAL ERROR: %v [path=%s]", err, c.Path())
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"error":   "INTERNAL_SERVER_ERROR",
				"message": "An internal server error occurred",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "INTERNAL_SERVER_ERROR",
			"message": err.Error(),
		})
	}
}
