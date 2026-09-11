package queue

import (
	"context"
	"fmt"
	"log"
	"time"

	"sovera-core-api/internal/service/urlverifier"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	TypeURLHealthCheck  = "task:url_health_check"
	QueueURLHealthCheck = "url-health-check-queue"
)

type URLHealthCheckWorker struct {
	dbPool   *pgxpool.Pool
	verifier *urlverifier.URLVerifier
}

func NewURLHealthCheckWorker(dbPool *pgxpool.Pool) *URLHealthCheckWorker {
	return &URLHealthCheckWorker{
		dbPool:   dbPool,
		verifier: urlverifier.NewURLVerifier(),
	}
}

func NewURLHealthCheckTask() (*asynq.Task, error) {
	return asynq.NewTask(TypeURLHealthCheck, []byte("{}"), asynq.Queue(QueueURLHealthCheck), asynq.MaxRetry(1)), nil
}

// HandleURLHealthCheck performs a batch health check on all active crawling targets.
// It verifies each URL and updates health_status, last_http_status, and consecutive_failures.
// URLs with >= 3 consecutive failures are automatically deactivated.
func (w *URLHealthCheckWorker) HandleURLHealthCheck(ctx context.Context, t *asynq.Task) error {
	if w.dbPool == nil {
		log.Println("[HealthCheck] No DB pool available, skipping health check")
		return nil
	}

	log.Println("[HealthCheck] Starting batch URL health check...")

	// Fetch all active targets that haven't been checked in the last 12 hours,
	// or have never been checked.
	rows, err := w.dbPool.Query(ctx, `
		SELECT id, target_url, consecutive_failures, health_status
		FROM crawling_targets
		WHERE is_active = true
		  AND (last_http_status IS NULL OR last_scraped_at IS NULL OR last_scraped_at < NOW() - INTERVAL '12 hours')
		ORDER BY (CASE WHEN last_http_status IS NULL THEN 0 ELSE 1 END) ASC, consecutive_failures DESC, created_at ASC
		LIMIT 100
	`)
	if err != nil {
		return fmt.Errorf("failed to query targets for health check: %w", err)
	}
	defer rows.Close()

	type target struct {
		ID                  string
		URL                 string
		ConsecutiveFailures int
		HealthStatus        string
	}

	var targets []target
	for rows.Next() {
		var t target
		if err := rows.Scan(&t.ID, &t.URL, &t.ConsecutiveFailures, &t.HealthStatus); err != nil {
			log.Printf("[HealthCheck] Error scanning row: %v", err)
			continue
		}
		targets = append(targets, t)
	}

	if len(targets) == 0 {
		log.Println("[HealthCheck] No targets to check, all recently verified.")
		return nil
	}

	log.Printf("[HealthCheck] Checking %d URLs...", len(targets))

	var checked, healthy, degraded, dead, deactivated int

	for _, tgt := range targets {
		// Rate limit: 1 check per 2 seconds to avoid overwhelming targets
		select {
		case <-ctx.Done():
			log.Println("[HealthCheck] Context cancelled, stopping early")
			return nil
		case <-time.After(2 * time.Second):
		}

		result := w.verifier.VerifyAndResolve(ctx, tgt.URL)
		checked++

		var newHealthStatus string
		var newConsecFailures int
		var isActive bool

		switch result.HealthStatus {
		case urlverifier.StatusHealthy:
			newHealthStatus = "HEALTHY"
			newConsecFailures = 0
			isActive = true
			healthy++

		case urlverifier.StatusDegraded:
			newHealthStatus = "DEGRADED"
			if result.HTTPStatus == 429 {
				newConsecFailures = tgt.ConsecutiveFailures
				isActive = true
			} else {
				newConsecFailures = tgt.ConsecutiveFailures + 1
				isActive = newConsecFailures < 3 // auto-deactivate after 3 non-429 failures
			}
			degraded++

		case urlverifier.StatusDisabledDeadLink:
			newHealthStatus = "DEAD"
			newConsecFailures = tgt.ConsecutiveFailures + 1
			isActive = false // dead links are always deactivated
			dead++
		}

		if !isActive && tgt.HealthStatus != "DEAD" {
			deactivated++
		}

		// Update the target in the database
		_, err := w.dbPool.Exec(ctx, `
			UPDATE crawling_targets
			SET health_status = $1,
				last_http_status = $2,
				consecutive_failures = $3,
				is_active = $4,
				last_error_message = $5,
				next_run_at = CASE 
					WHEN $2 = 429 THEN 
						CASE 
							WHEN $3 <= 1 THEN NOW() + INTERVAL '30 minutes'
							WHEN $3 = 2 THEN NOW() + INTERVAL '2 hours'
							WHEN $3 = 3 THEN NOW() + INTERVAL '6 hours'
							ELSE NOW() + INTERVAL '24 hours'
						END
					ELSE next_run_at
				END,
				updated_at = NOW()
			WHERE id = $6
		`, newHealthStatus, result.HTTPStatus, newConsecFailures, isActive, result.ErrorMsg, tgt.ID)

		if err != nil {
			log.Printf("[HealthCheck] Error updating target %s: %v", tgt.ID, err)
		} else {
			log.Printf("[HealthCheck] %s => %s (HTTP %d) active=%v | %s",
				tgt.URL, newHealthStatus, result.HTTPStatus, isActive, result.ErrorMsg)
		}
	}

	log.Printf("[HealthCheck] Completed: checked=%d healthy=%d degraded=%d dead=%d deactivated=%d",
		checked, healthy, degraded, dead, deactivated)

	return nil
}
