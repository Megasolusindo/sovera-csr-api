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
	dbURL := cfg.DatabaseURL
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		dbURL = "postgres://postgres:ResulteW212%23@10.10.29.177:5432/sovera?sslmode=disable"
	}

	log.Printf("Connecting to database: %s", dbURL)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := repository.InitDBPool(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	log.Println("Running migrations...")
	err = repository.RunMigrations(ctx, pool, "./db/migrations")
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("Migrations executed successfully!")
}
