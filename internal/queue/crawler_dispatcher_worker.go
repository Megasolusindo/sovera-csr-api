package queue

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"sovera-core-api/internal/model"
	"sovera-core-api/internal/repository"
	"sovera-core-api/internal/service/crawler"

	"github.com/hibiken/asynq"
)

type CrawlerDispatcherHandler struct {
	crawlerRepo *repository.CrawlerRepository
	dispatcher  *crawler.Dispatcher
}

func NewCrawlerDispatcherHandler(crawlerRepo *repository.CrawlerRepository, dispatcher *crawler.Dispatcher) *CrawlerDispatcherHandler {
	return &CrawlerDispatcherHandler{
		crawlerRepo: crawlerRepo,
		dispatcher:  dispatcher,
	}
}

// extractHost extracts lowercase hostname from target URL for host throttling
func extractHost(rawURL string) string {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return "unknown"
	}
	u, err := url.Parse(trimmed)
	if err != nil || u.Host == "" {
		return "unknown"
	}
	return strings.ToLower(u.Host)
}

// interleaveTargetsByHost groups due targets by host domain and picks them round-robin
// to maximize inter-host dispersion and prevent burst dispatches to the same host domain.
func interleaveTargetsByHost(targets []model.CrawlingTarget) []model.CrawlingTarget {
	if len(targets) <= 1 {
		return targets
	}

	hostBuckets := make(map[string][]model.CrawlingTarget)
	var hostKeys []string

	for _, t := range targets {
		h := extractHost(t.TargetURL)
		if _, exists := hostBuckets[h]; !exists {
			hostKeys = append(hostKeys, h)
		}
		hostBuckets[h] = append(hostBuckets[h], t)
	}

	result := make([]model.CrawlingTarget, 0, len(targets))
	maxLen := 0
	for _, bucket := range hostBuckets {
		if len(bucket) > maxLen {
			maxLen = len(bucket)
		}
	}

	for i := 0; i < maxLen; i++ {
		for _, h := range hostKeys {
			bucket := hostBuckets[h]
			if i < len(bucket) {
				result = append(result, bucket[i])
			}
		}
	}

	return result
}

func (h *CrawlerDispatcherHandler) HandleDispatchCrawlingTask(ctx context.Context, t *asynq.Task) error {
	log.Println("Starting execution of periodic crawling target dispatching worker...")

	dueTargets, err := h.crawlerRepo.GetDueTargets(ctx, 50)
	if err != nil {
		return fmt.Errorf("failed to fetch due crawling targets: %w", err)
	}

	if len(dueTargets) == 0 {
		log.Println("No due crawling targets found. Skipping dispatching cycle.")
		return nil
	}

	// Interleave targets by host domain to spread out requests to identical host domains
	dueTargets = interleaveTargetsByHost(dueTargets)

	log.Printf("Found %d due crawling targets across hosts. Dispatching to Scraper Service...", len(dueTargets))

	// Per-host minimum inter-dispatch delay (3 seconds for same domain)
	const SameHostMinDelay = 3 * time.Second
	lastDispatchPerHost := make(map[string]time.Time)

	for _, target := range dueTargets {
		host := extractHost(target.TargetURL)
		if lastTime, exists := lastDispatchPerHost[host]; exists {
			elapsed := time.Since(lastTime)
			if elapsed < SameHostMinDelay {
				waitTime := SameHostMinDelay - elapsed
				log.Printf("[Dispatcher] Same host '%s' detected. Enforcing %v delay before next request...", host, waitTime.Round(time.Millisecond))
				select {
				case <-ctx.Done():
					log.Println("[Dispatcher] Context cancelled, stopping dispatch cycle")
					return nil
				case <-time.After(waitTime):
				}
			}
		}

		taskID := fmt.Sprintf("task_%s_%d", target.ID[:8], time.Now().Unix())

		// Create dispatch log in database
		initialLog := model.CrawlingLog{
			TargetID: &target.ID,
			TaskID:   taskID,
			Status:   "DISPATCHED",
		}
		if err := h.crawlerRepo.CreateLog(ctx, initialLog); err != nil {
			log.Printf("Error creating crawling log for target %s: %v", target.ID, err)
		}

		// Dispatch scrape task via HTTP
		statusCode, err := h.dispatcher.DispatchTask(ctx, target, taskID)
		if err != nil {
			log.Printf("Failed to dispatch target %s (TaskID: %s): %v", target.ID, taskID, err)
			errMsg := err.Error()
			_ = h.crawlerRepo.UpdateLogStatus(ctx, taskID, "FAILED", nil, nil, &errMsg)
			continue
		}

		// Update target next_run_at in database
		if err := h.crawlerRepo.UpdateTargetNextRun(ctx, target.ID, target.CheckIntervalHours); err != nil {
			log.Printf("Failed to update target next run time for %s: %v", target.ID, err)
		}

		lastDispatchPerHost[host] = time.Now()

		log.Printf("Successfully dispatched target '%s' (%s, TaskID: %s, HTTP %d)", target.SourceName, host, taskID, statusCode)
	}

	log.Println("Crawling dispatching cycle completed successfully.")
	return nil
}

// HandlePollPendingTasks acts as a fallback worker querying status for tasks stuck in DISPATCHED status
func (h *CrawlerDispatcherHandler) HandlePollPendingTasks(ctx context.Context, t *asynq.Task) error {
	log.Println("Starting execution of fallback task status polling worker...")

	// Fetch logs stuck in DISPATCHED status for more than 10 minutes
	pendingLogs, err := h.crawlerRepo.GetPendingLogs(ctx, 10, 50)
	if err != nil {
		return fmt.Errorf("failed to fetch pending crawling logs: %w", err)
	}

	if len(pendingLogs) == 0 {
		log.Println("No stuck pending tasks found. Polling cycle completed.")
		return nil
	}

	log.Printf("Found %d pending tasks stuck in DISPATCHED status. Polling Scraper API status...", len(pendingLogs))

	for _, pLog := range pendingLogs {
		taskStatus, statusCode, err := h.dispatcher.GetTaskStatus(ctx, pLog.TaskID)
		if err != nil {
			log.Printf("[Poller] Could not fetch status for TaskID %s (HTTP %d): %v", pLog.TaskID, statusCode, err)
			continue
		}

		switch taskStatus.Status {
		case "COMPLETED":
			log.Printf("[Poller] TaskID %s completed on scraper service. Syncing status...", pLog.TaskID)
			execTime := taskStatus.ExecutionTimeMs
			cHash := taskStatus.ContentHash
			_ = h.crawlerRepo.UpdateLogStatus(ctx, pLog.TaskID, "COMPLETED", &execTime, &cHash, nil)
			if pLog.TargetID != nil && *pLog.TargetID != "" {
				_ = h.crawlerRepo.RecordSuccess(ctx, *pLog.TargetID, taskStatus.HTTPStatusCode)
			}
		case "FAILED":
			log.Printf("[Poller] TaskID %s failed on scraper service. Syncing failure...", pLog.TaskID)
			errMsg := "Task failed on scraper service according to status polling"
			_ = h.crawlerRepo.UpdateLogStatus(ctx, pLog.TaskID, "FAILED", nil, nil, &errMsg)
			if pLog.TargetID != nil && *pLog.TargetID != "" {
				_ = h.crawlerRepo.RecordFailure(ctx, *pLog.TargetID, taskStatus.HTTPStatusCode, errMsg)
			}
		default:
			log.Printf("[Poller] TaskID %s status on scraper service: %s. Keeping in queue.", pLog.TaskID, taskStatus.Status)
		}
	}

	log.Println("Fallback task status polling cycle completed.")
	return nil
}
