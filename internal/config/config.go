package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port              string
	Environment       string
	DatabaseURL       string
	RedisURL          string
	WebhookSecretKey  string
	JWTSecret         string
	AIAPIKey          string
	ScraperServiceURL string
	ScraperAPIKey     string
	WebhookURL        string
	S3Endpoint        string
	S3Bucket          string
	S3Region          string
	S3AccessKey       string
	S3SecretKey       string
	StorageLocalDir   string
}

func LoadConfig() *Config {
	// Attempt to load .env file if available
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	return &Config{
		Port:              getEnv("PORT", "4000"),
		Environment:       getEnv("NODE_ENV", "development"),
		DatabaseURL:       getEnv("DATABASE_URL", "postgresql://postgres:postgres@localhost:5432/sovera_db?sslmode=disable"),
		RedisURL:          getEnv("REDIS_URL", "localhost:6379"),
		WebhookSecretKey:  getEnv("WEBHOOK_SECRET_KEY", "super_secret_crawler_key_123"),
		JWTSecret:         getEnv("JWT_SECRET", "super_secret_jwt_key_enterprise"),
		AIAPIKey:          getEnv("AI_API_KEY", ""),
		ScraperServiceURL: getEnv("SCRAPER_SERVICE_URL", "https://api-scraper.megasolusindo.com/api/v1/scrape-tasks"),
		ScraperAPIKey:     getEnv("SCRAPER_API_KEY", "change-me"),
		WebhookURL:        getEnv("WEBHOOK_URL", "http://host.docker.internal:4000/api/v1/webhooks/crawler?secret=super_secret_crawler_key_123"),
		S3Endpoint:        getEnv("S3_ENDPOINT", "http://10.10.29.177:9000"),
		S3Bucket:          getEnv("S3_BUCKET", "sovera-templates"),
		S3Region:          getEnv("S3_REGION", "us-east-1"),
		S3AccessKey:       getEnv("S3_ACCESS_KEY", "minioadmin"),
		S3SecretKey:       getEnv("S3_SECRET_KEY", "minioadmin"),
		StorageLocalDir:   getEnv("STORAGE_LOCAL_DIR", "/tmp/sovera_storage"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return fallback
}
