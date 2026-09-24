package handler

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	"sovera-core-api/internal/pkg/telegram"
	"sovera-core-api/internal/queue"
	"sovera-core-api/internal/repository"
	"sovera-core-api/internal/service/normalizer"
)

type WebhookHandler struct {
	dbPool           *pgxpool.Pool
	asynqClient      *asynq.Client
	crawlerRepo      *repository.CrawlerRepository
	normalizer       *normalizer.Normalizer
	telegramNotifier *telegram.Notifier
}

func NewWebhookHandler(dbPool *pgxpool.Pool, asynqClient *asynq.Client, norm *normalizer.Normalizer, telegramNotifier *telegram.Notifier) *WebhookHandler {
	var crawlerRepo *repository.CrawlerRepository
	if dbPool != nil {
		crawlerRepo = repository.NewCrawlerRepository(dbPool)
	}
	if norm == nil {
		norm = normalizer.NewNormalizer()
	}
	return &WebhookHandler{
		dbPool:           dbPool,
		asynqClient:      asynqClient,
		crawlerRepo:      crawlerRepo,
		normalizer:       norm,
		telegramNotifier: telegramNotifier,
	}
}

type CrawlerPayloadData struct {
	RawText         string `json:"raw_text,omitempty"`
	MarkdownContent string `json:"markdown_content,omitempty"`
	Text            string `json:"text,omitempty"`
	Content         string `json:"content,omitempty"`
	RawData         struct {
		RawText         string `json:"raw_text,omitempty"`
		MarkdownContent string `json:"markdown_content,omitempty"`
		Text            string `json:"text,omitempty"`
		Content         string `json:"content,omitempty"`
	} `json:"raw_data,omitempty"`
}

type DiscoveredItem struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet,omitempty"`
	Rank    int    `json:"rank,omitempty"`
}

// CrawlerPayload represents the webhook callback payload from the web scraper service.
type CrawlerPayload struct {
	TaskID          string              `json:"task_id"`
	JobID           string              `json:"job_id,omitempty"`
	TargetID        string              `json:"target_id,omitempty"`
	HTTPStatusCode  int                 `json:"http_status_code,omitempty"` // e.g. 200, 404, 403, 500
	Status          string              `json:"status,omitempty"`           // COMPLETED, FAILED
	ErrorMessage    string              `json:"error_message,omitempty"`
	SourceType      string              `json:"source_type"`
	TargetType      string              `json:"target_type,omitempty"`
	SourceURL       string              `json:"source_url"`
	AuthorOrAccount string              `json:"author_or_account,omitempty"`
	PublishedDate   string              `json:"published_date,omitempty"`
	RawText         string              `json:"raw_text,omitempty"`
	MarkdownContent string              `json:"markdown_content,omitempty"`
	ExecutionTimeMs int                 `json:"execution_time_ms,omitempty"`
	Query           string              `json:"query,omitempty"`
	DiscoveredItems []DiscoveredItem    `json:"discovered_items,omitempty"`
	Data            *CrawlerPayloadData `json:"data,omitempty"`
}

func (h *WebhookHandler) HandleCrawlerWebhook(c *fiber.Ctx) error {
	var payload CrawlerPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "INVALID_PAYLOAD",
			"message": "Failed to parse JSON body",
		})
	}

	// Fallback TaskID from JobID or TargetType from SourceType
	if payload.TaskID == "" && payload.JobID != "" {
		payload.TaskID = payload.JobID
	}
	if payload.SourceType == "" && payload.TargetType != "" {
		payload.SourceType = payload.TargetType
	}

	// Extract raw_text / markdown_content from nested Data struct if top-level is empty
	if payload.RawText == "" && payload.Data != nil {
		if payload.Data.RawText != "" {
			payload.RawText = payload.Data.RawText
		} else if payload.Data.RawData.RawText != "" {
			payload.RawText = payload.Data.RawData.RawText
		} else if payload.Data.Text != "" {
			payload.RawText = payload.Data.Text
		} else if payload.Data.Content != "" {
			payload.RawText = payload.Data.Content
		} else if payload.Data.RawData.Text != "" {
			payload.RawText = payload.Data.RawData.Text
		} else if payload.Data.RawData.Content != "" {
			payload.RawText = payload.Data.RawData.Content
		}
	}

	if payload.MarkdownContent == "" && payload.Data != nil {
		if payload.Data.MarkdownContent != "" {
			payload.MarkdownContent = payload.Data.MarkdownContent
		} else if payload.Data.RawData.MarkdownContent != "" {
			payload.MarkdownContent = payload.Data.RawData.MarkdownContent
		}
	}

	httpStatusCode := payload.HTTPStatusCode
	if httpStatusCode == 0 {
		if payload.Status == "FAILED" {
			httpStatusCode = 500
		} else {
			httpStatusCode = 200
		}
	}

	execTime := payload.ExecutionTimeMs
	if execTime == 0 {
		execTime = 1200
	}

	// ─── Handle Scraper Failure Callback ──────────────────────────────────────
	if payload.Status == "FAILED" || httpStatusCode >= 400 {
		errMsg := payload.ErrorMessage
		if errMsg == "" {
			errMsg = "Scraper returned failure HTTP status"
		}

		if h.crawlerRepo != nil {
			// 1. Log failure in crawling_logs
			if payload.TaskID != "" {
				_ = h.crawlerRepo.UpdateLogStatus(c.Context(), payload.TaskID, "FAILED", &execTime, nil, &errMsg)
			}
			// 2. Trigger Circuit Breaker in crawling_targets if target_id present
			if payload.TargetID != "" {
				if err := h.crawlerRepo.RecordFailure(c.Context(), payload.TargetID, httpStatusCode, errMsg); err != nil {
					log.Printf("Notice: Could not record failure for TargetID %s: %v", payload.TargetID, err)
				}
			}
		}

		// 3. Dispatch automated Telegram alert for critical errors (429 Rate Limit, 404 Dead Link, 500 Server Error)
		if h.telegramNotifier != nil && h.telegramNotifier.IsEnabled() {
			severity := "WARNING"
			if httpStatusCode == 429 {
				severity = "HIGH"
			} else if httpStatusCode == 404 || httpStatusCode >= 500 {
				severity = "CRITICAL"
			}

			alertTitle := fmt.Sprintf("Scraper Target Failure (HTTP %d)", httpStatusCode)
			alertDetails := fmt.Sprintf("Task ID: <code>%s</code>\nSource Type: %s\nError Log: %s", payload.TaskID, payload.SourceType, errMsg)
			go func() {
				_ = h.telegramNotifier.SendAlert(context.Background(), alertTitle, severity, alertDetails, payload.SourceURL)
			}()
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": true,
			"message": "Scraper failure recorded and target health updated",
			"status":  "FAILED",
		})
	}

	// ─── Handle Scraper Success Callback ──────────────────────────────────────
	if (payload.SourceType == "SEARCH_DISCOVERY" || payload.SourceType == "COMPANY_ENRICHMENT") && payload.RawText == "" && len(payload.DiscoveredItems) > 0 {
		var sb strings.Builder
		if payload.Query != "" {
			sb.WriteString("Search Query: " + payload.Query + "\n\n")
		}
		for _, item := range payload.DiscoveredItems {
			sb.WriteString(fmt.Sprintf("[%d] %s (%s)\n%s\n\n", item.Rank, item.Title, item.URL, item.Snippet))
		}
		payload.RawText = sb.String()

		// Auto-enrich company website if TargetID matches company ID and item rank 1 URL is valid
		if h.dbPool != nil && len(payload.DiscoveredItems) > 0 {
			topURL := strings.TrimSpace(payload.DiscoveredItems[0].URL)
			if topURL != "" && (strings.HasPrefix(topURL, "http://") || strings.HasPrefix(topURL, "https://")) {
				compRepo := repository.NewCompanyRepository(h.dbPool)
				if payload.TargetID != "" {
					_ = compRepo.UpdateCompanyWebsite(c.Context(), payload.TargetID, topURL)
				}
			}
		}
	}

	// Record success in target health (resets consecutive_failures & updates last_http_status)
	if payload.TargetID != "" && h.crawlerRepo != nil {
		if err := h.crawlerRepo.RecordSuccess(c.Context(), payload.TargetID, httpStatusCode); err != nil {
			log.Printf("Notice: Could not record success for TargetID %s: %v", payload.TargetID, err)
		}
	}

	if payload.RawText == "" && payload.MarkdownContent == "" && (payload.SourceType != "SEARCH_DISCOVERY" || len(payload.DiscoveredItems) == 0) {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"error":   "EMPTY_CONTENT",
			"message": "Payload raw_text or markdown_content cannot be empty for successful scrapes",
		})
	}

	// Calculate SHA-256 content_hash for deduplication via Normalizer
	bestContent := h.normalizer.SelectBestContent(payload.RawText, payload.MarkdownContent)
	contentHash := h.normalizer.GenerateContentHash(bestContent)

	log.Printf("[Webhook Received] TaskID: %s | SourceType: %s | URL: %s | Content: %q | Hash: %s", payload.TaskID, payload.SourceType, payload.SourceURL, bestContent, contentHash)


	jobID := "job_ingest_" + uuid.New().String()[:8]

	// Update crawling_logs for completed task
	if payload.TaskID != "" && h.crawlerRepo != nil {
		if err := h.crawlerRepo.UpdateLogStatus(c.Context(), payload.TaskID, "COMPLETED", &execTime, &contentHash, nil); err != nil {
			log.Printf("Notice: Could not update crawling_logs for TaskID %s: %v", payload.TaskID, err)
		}
	}

	// Enqueue Asynq task into Redis for LLM extraction
	taskPayload := queue.LLMExtractionPayload{
		JobID:           jobID,
		TaskID:          payload.TaskID,
		SourceType:      payload.SourceType,
		SourceURL:       payload.SourceURL,
		AuthorOrAccount: payload.AuthorOrAccount,
		PublishedDate:   payload.PublishedDate,
		RawText:         payload.RawText,
		MarkdownContent: payload.MarkdownContent,
		ContentHash:     contentHash,
	}

	if h.asynqClient != nil {
		task, err := queue.NewLLMExtractionTask(taskPayload)
		if err == nil {
			info, enqueueErr := h.asynqClient.Enqueue(task)
			if enqueueErr != nil {
				log.Printf("Failed to enqueue extraction task to Asynq: %v", enqueueErr)
			} else {
				log.Printf("Successfully enqueued task [%s] to queue [%s]", info.ID, info.Queue)
			}
		}
	}

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"success":      true,
		"message":      "Payload queued for background ingestion & LLM extraction",
		"job_id":       jobID,
		"content_hash": contentHash,
		"queued_at":    time.Now().UTC().Format(time.RFC3339),
	})
}
