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
	SerperAPIKey      string
	WebhookURL        string
	S3Endpoint        string
	S3Bucket          string
	S3Region          string
	S3AccessKey       string
	S3SecretKey       string
	StorageLocalDir   string
	TelegramBotToken  string
	TelegramChatID    string
	MidtransServerKey    string
	MidtransClientKey    string
	MidtransIsProduction bool
	ActivePaymentGateway string
	FaspayMerchantID    string
	FaspayMerchantKey   string
	FaspayIsProduction  bool
	FaspayReturnURL     string
	OpenClawAgentToken  string
}

func LoadConfig() *Config {
	// Attempt to load .env file if available
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	return &Config{
		Port:                 getEnv("PORT", "4000"),
		Environment:          getEnv("NODE_ENV", "development"),
		DatabaseURL:          getEnv("DATABASE_URL", ""),
		RedisURL:             getEnv("REDIS_URL", "localhost:6379"),
		WebhookSecretKey:     getEnv("WEBHOOK_SECRET_KEY", ""),
		JWTSecret:            getEnv("JWT_SECRET", ""),
		AIAPIKey:             getEnv("AI_API_KEY", ""),
		ScraperServiceURL:    getEnv("SCRAPER_SERVICE_URL", "https://api-scraper.megasolusindo.com/api/v1/scrape-tasks"),
		ScraperAPIKey:        getEnv("SCRAPER_API_KEY", ""),
		SerperAPIKey:         getEnv("SERPER_API_KEY", ""),
		WebhookURL:           getEnv("WEBHOOK_URL", ""),
		S3Endpoint:           getEnv("S3_ENDPOINT", "http://127.0.0.1:9000"),
		S3Bucket:             getEnv("S3_BUCKET", "sovera-templates"),
		S3Region:             getEnv("S3_REGION", "us-east-1"),
		S3AccessKey:          getEnv("S3_ACCESS_KEY", ""),
		S3SecretKey:          getEnv("S3_SECRET_KEY", ""),
		StorageLocalDir:      getEnv("STORAGE_LOCAL_DIR", "/tmp/sovera_storage"),
		TelegramBotToken:     getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramChatID:       getEnv("TELEGRAM_CHAT_ID", ""),
		MidtransServerKey:    getEnv("MIDTRANS_SERVER_KEY", ""),
		MidtransClientKey:    getEnv("MIDTRANS_CLIENT_KEY", ""),
		MidtransIsProduction: getEnv("MIDTRANS_IS_PRODUCTION", "false") == "true",
		ActivePaymentGateway: getEnv("ACTIVE_PAYMENT_GATEWAY", "midtrans"),
		FaspayMerchantID:     getEnv("FASPAY_MERCHANT_ID", ""),
		FaspayMerchantKey:    getEnv("FASPAY_MERCHANT_KEY", ""),
		FaspayIsProduction:   getEnv("FASPAY_IS_PRODUCTION", "false") == "true",
		FaspayReturnURL:      getEnv("FASPAY_RETURN_URL", "https://sovera.id/settings/billing"),
		OpenClawAgentToken:   getEnv("OPENCLAW_AGENT_TOKEN", ""),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return fallback
}

// Validate ensures all required secrets and settings are present.
// It returns a list of missing required environment variables.
func (c *Config) Validate() []string {
	var missing []string
	required := []struct {
		value    string
		envKey   string
		friendly string
	}{
		{c.DatabaseURL, "DATABASE_URL", "database connection URL"},
		{c.JWTSecret, "JWT_SECRET", "JWT signing secret"},
		{c.WebhookSecretKey, "WEBHOOK_SECRET_KEY", "webhook secret key"},
		{c.AIAPIKey, "AI_API_KEY", "AI API key"},
		{c.OpenClawAgentToken, "OPENCLAW_AGENT_TOKEN", "OpenClaw agent token"},
	}
	for _, r := range required {
		if r.value == "" {
			missing = append(missing, r.envKey)
		}
	}
	return missing
}
