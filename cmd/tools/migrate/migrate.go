package main

import (
	"context"
	"log"
	"os"
	"time"

	"sovera-core-api/internal/config"
	"sovera-core-api/internal/repository"
)

func main() {
	cfg := config.LoadConfig()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dbPool, err := repository.InitDBPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	migrationFile := "db/migrations/000035_seed_corporate_subscription_plans.up.sql"
	sqlBytes, err := os.ReadFile(migrationFile)
	if err != nil {
		log.Fatalf("Failed to read migration file %s: %v", migrationFile, err)
	}

	log.Printf("Executing 000035 migration...")
	_, err = dbPool.Exec(ctx, string(sqlBytes))
	if err != nil {
		log.Fatalf("Migration 000035 failed: %v", err)
	}

	log.Println("Migration 000035 executed successfully!")
}
