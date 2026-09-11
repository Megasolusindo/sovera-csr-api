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
	"github.com/hibiken/asynq"

	"sovera-core-api/internal/config"
	"sovera-core-api/internal/handler"
	"sovera-core-api/internal/middleware"
	"sovera-core-api/internal/queue"
	"sovera-core-api/internal/repository"
	"sovera-core-api/internal/service/ai"
	"sovera-core-api/internal/service/exporter"
	"sovera-core-api/internal/service/normalizer"
	"sovera-core-api/internal/service/storage"
)

func main() {
	// 1. Load Environment Configuration
	cfg := config.LoadConfig()
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

	// 4. Initialize Services & Repositories
	geminiService := ai.NewGeminiService(cfg.AIAPIKey)
	docExporter := exporter.NewDocumentExporter()
	textNormalizer := normalizer.NewNormalizer()
	storageService := storage.NewStorageService()

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

	// 5. Create Fiber Web Application
	app := fiber.New(fiber.Config{
		AppName:      "Sovera Core API & Intelligence Engine v1.0",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	})

	// 6. Global Middlewares (CORS MUST BE FIRST)
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Requested-With",
		AllowMethods:     "GET, POST, HEAD, PUT, DELETE, PATCH, OPTIONS",
		AllowCredentials: false,
	}))
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))

	// 7. Register Handlers & Controllers
	healthHandler := handler.NewHealthHandler(dbPool)
	webhookHandler := handler.NewWebhookHandler(dbPool, asynqClient, textNormalizer)
	signalHandler := handler.NewSignalHandler(signalRepo)
	programHandler := handler.NewProgramHandler(programRepo, geminiService)
	dealHandler := handler.NewDealHandler(dealRepo, programRepo, signalRepo, templateRepo, tokenLogRepo, storageService, geminiService, docExporter)
	authHandler := handler.NewAuthHandler(userRepo, cfg.JWTSecret)
	companyHandler := handler.NewCompanyHandler(companyRepo, companyCSRProgramRepo)
	templateHandler := handler.NewTemplateHandler(templateRepo, tokenLogRepo, storageService)
	adminHandler := handler.NewAdminHandler(orgRepo, userRepo, crawlerRepo, tokenLogRepo, esgProfileRepo, dbPool)

	// Root & Health check routes (public)
	app.Get("/health", healthHandler.HealthCheck)

	apiV1 := app.Group("/api/v1")
	apiV1.Get("/health", healthHandler.HealthCheck)

	// ─── Auth Routes (PUBLIC — no JWT required) ───────────────────────────────
	auth := apiV1.Group("/auth")
	auth.Post("/register", authHandler.Register)
	auth.Post("/login", authHandler.Login)
	auth.Post("/logout", authHandler.Logout)
	auth.Get("/me", middleware.AuthenticateJWT(cfg.JWTSecret), authHandler.Me)

	// ─── Webhook Ingestion (Protected by HMAC Verification) ──────────────────
	apiV1.Post(
		"/webhooks/crawler",
		middleware.VerifyHMAC(cfg.WebhookSecretKey),
		webhookHandler.HandleCrawlerWebhook,
	)

	// ─── JWT-Protected Routes ─────────────────────────────────────────────────
	jwtGuard := middleware.AuthenticateJWT(cfg.JWTSecret)

	// Platform Admin Console — Full System Control & Analytics
	adminGroup := apiV1.Group("/admin")
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
	adminGroup.Post("/idx-sync", adminHandler.TriggerIDXSync)
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

	// Companies Directory & CSR Master Programs & Web Sources & ESG — semua role / public browsing
	apiV1.Get("/stats", adminHandler.GetAnalytics)
	apiV1.Get("/companies", companyHandler.ListCompanies)
	apiV1.Post("/companies", companyHandler.CreateCompany)
	apiV1.Get("/companies/csr-programs", companyHandler.ListCSRPrograms)
	apiV1.Get("/csr-programs", companyHandler.ListCSRPrograms)
	apiV1.Get("/sources", adminHandler.ListSources)
	apiV1.Post("/sources", adminHandler.CreateScrapingJob)
	apiV1.Get("/documents", adminHandler.ListDocuments)
	apiV1.Get("/scraping-jobs", adminHandler.ListScrapingJobs)
	apiV1.Post("/scraping-jobs", adminHandler.CreateScrapingJob)
	apiV1.Get("/esg-intelligence", adminHandler.GetESGIntelligence)
	apiV1.Get("/companies/:id", companyHandler.GetCompany)


	// Corporate Intelligence Feeds — semua role
	apiV1.Get("/signals", jwtGuard, signalHandler.ListSignals)
	apiV1.Get("/signals/:id/match-programs", jwtGuard, signalHandler.MatchPrograms)

	// Institution Programs — GET: semua role | POST: ORG_ADMIN & DIRECTOR only
	apiV1.Get("/programs", jwtGuard, programHandler.ListPrograms)
	apiV1.Post("/programs", jwtGuard,
		middleware.RequireRole("ORG_ADMIN", "DIRECTOR"),
		programHandler.CreateProgram,
	)

	// Deal Pipeline & Proposal Studio — semua role (ownership filter via org_id RLS)
	apiV1.Get("/deals", jwtGuard, dealHandler.ListDeals)
	apiV1.Post("/deals", jwtGuard, dealHandler.CreateDeal)
	apiV1.Patch("/deals/:id/stage", jwtGuard, dealHandler.UpdateStage)
	apiV1.Post("/deals/:id/generate-pitch", jwtGuard, dealHandler.GeneratePitch)
	apiV1.Post("/deals/:id/export", jwtGuard, dealHandler.ExportProposal)

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
