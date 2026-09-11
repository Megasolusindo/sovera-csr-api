package main

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"sovera-core-api/internal/queue"
	"sovera-core-api/internal/repository"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:ResulteW212%23@10.10.29.177:5432/sovera?sslmode=disable"
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	companyRepo := repository.NewCompanyRepository(pool)
	crawlerRepo := repository.NewCrawlerRepository(pool)

	idxWorker := queue.NewIDXCompanyWorker(companyRepo, crawlerRepo)

	log.Println("🚀 Running Live BEI / IDX Company Ingestion Worker...")
	if err := idxWorker.ProcessIDXSyncTask(ctx, nil); err != nil {
		log.Fatalf("❌ IDX Ingestion Worker Error: %v", err)
	}

	log.Println("✅ Live BEI / IDX Ingestion Worker completed successfully!")
}
