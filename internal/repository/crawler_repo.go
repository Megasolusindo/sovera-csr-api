package repository

import (
	"context"
	"fmt"
	"time"

	"sovera-core-api/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CrawlerRepository struct {
	pool *pgxpool.Pool
}

func NewCrawlerRepository(pool *pgxpool.Pool) *CrawlerRepository {
	return &CrawlerRepository{pool: pool}
}

// GetDueTargets retrieves active crawling targets that are due for scraping.
// Ignores targets that have been disabled due to dead links (404).
func (r *CrawlerRepository) GetDueTargets(ctx context.Context, limit int) ([]model.CrawlingTarget, error) {
	query := `
		SELECT id, company_id::text, source_name, source_type, target_url, check_interval_hours, 
		       last_scraped_at, next_run_at, is_active, consecutive_failures,
		       last_http_status, last_error_message, health_status, created_at, updated_at
		FROM crawling_targets
		WHERE is_active = TRUE AND health_status != 'DISABLED_DEAD_LINK' AND next_run_at <= $1
		ORDER BY next_run_at ASC
		LIMIT $2;
	`
	rows, err := r.pool.Query(ctx, query, time.Now(), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query due targets: %w", err)
	}
	defer rows.Close()

	var targets []model.CrawlingTarget
	for rows.Next() {
		var t model.CrawlingTarget
		err := rows.Scan(
			&t.ID, &t.CompanyID, &t.SourceName, &t.SourceType, &t.TargetURL, &t.CheckIntervalHours,
			&t.LastScrapedAt, &t.NextRunAt, &t.IsActive, &t.ConsecutiveFailures,
			&t.LastHTTPStatus, &t.LastErrorMsg, &t.HealthStatus, &t.CreatedAt, &t.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan crawling target: %w", err)
		}
		targets = append(targets, t)
	}
	return targets, nil
}

// UpdateTargetNextRun updates last_scraped_at and schedules next_run_at
func (r *CrawlerRepository) UpdateTargetNextRun(ctx context.Context, targetID string, intervalHours int) error {
	query := `
		UPDATE crawling_targets
		SET last_scraped_at = NOW(),
		    next_run_at = NOW() + MAKE_INTERVAL(hours => $2),
		    updated_at = NOW()
		WHERE id = $1;
	`
	_, err := r.pool.Exec(ctx, query, targetID, intervalHours)
	if err != nil {
		return fmt.Errorf("failed to update target next run: %w", err)
	}
	return nil
}

// RecordSuccess resets consecutive_failures and sets health_status to HEALTHY
func (r *CrawlerRepository) RecordSuccess(ctx context.Context, targetID string, httpStatusCode int) error {
	query := `
		UPDATE crawling_targets
		SET consecutive_failures = 0,
		    last_http_status = $2,
		    last_error_message = NULL,
		    health_status = 'HEALTHY',
		    updated_at = NOW()
		WHERE id = $1;
	`
	_, err := r.pool.Exec(ctx, query, targetID, httpStatusCode)
	if err != nil {
		return fmt.Errorf("failed to record crawling target success: %w", err)
	}
	return nil
}

// RecordFailure increments consecutive_failures and trips the circuit breaker if 404 or max retries exceeded.
func (r *CrawlerRepository) RecordFailure(ctx context.Context, targetID string, httpStatusCode int, errorMsg string) error {
	// Circuit breaker condition:
	// - If httpStatusCode is 404 (Not Found) or 410 (Gone), immediately mark as DISABLED_DEAD_LINK and set is_active = FALSE.
	// - If consecutive_failures + 1 >= 5, also set health_status = 'DISABLED_DEAD_LINK' and is_active = FALSE.
	// - Otherwise, set health_status = 'DEGRADED'.
	query := `
		UPDATE crawling_targets
		SET consecutive_failures = consecutive_failures + 1,
		    last_http_status = $2,
		    last_error_message = $3,
		    health_status = CASE 
		        WHEN $2 IN (404, 410) OR consecutive_failures + 1 >= 5 THEN 'DISABLED_DEAD_LINK'
		        ELSE 'DEGRADED'
		    END,
		    is_active = CASE 
		        WHEN $2 IN (404, 410) OR consecutive_failures + 1 >= 5 THEN FALSE
		        ELSE is_active
		    END,
		    updated_at = NOW()
		WHERE id = $1;
	`
	_, err := r.pool.Exec(ctx, query, targetID, httpStatusCode, errorMsg)
	if err != nil {
		return fmt.Errorf("failed to record crawling target failure: %w", err)
	}
	return nil
}

// CreateLog records a task dispatch attempt
func (r *CrawlerRepository) CreateLog(ctx context.Context, log model.CrawlingLog) error {
	query := `
		INSERT INTO crawling_logs (target_id, task_id, status, http_status_code, error_message, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		ON CONFLICT (task_id) DO UPDATE 
		SET status = EXCLUDED.status, updated_at = NOW();
	`
	_, err := r.pool.Exec(ctx, query, log.TargetID, log.TaskID, log.Status, log.HTTPStatusCode, log.ErrorMessage)
	if err != nil {
		return fmt.Errorf("failed to insert crawling log: %w", err)
	}
	return nil
}

// UpdateLogStatus updates log status when callback arrives
func (r *CrawlerRepository) UpdateLogStatus(ctx context.Context, taskID, status string, execTimeMs *int, contentHash *string, errMsg *string) error {
	query := `
		UPDATE crawling_logs
		SET status = $2,
		    execution_time_ms = COALESCE($3, execution_time_ms),
		    content_hash = COALESCE($4, content_hash),
		    error_message = COALESCE($5, error_message),
		    updated_at = NOW()
		WHERE task_id = $1;
	`
	_, err := r.pool.Exec(ctx, query, taskID, status, execTimeMs, contentHash, errMsg)
	if err != nil {
		return fmt.Errorf("failed to update crawling log status: %w", err)
	}
	return nil
}

// GetPendingLogs retrieves dispatched logs that haven't received a callback within pendingMinutes.
func (r *CrawlerRepository) GetPendingLogs(ctx context.Context, pendingMinutes int, limit int) ([]model.CrawlingLog, error) {
	query := `
		SELECT id, target_id::text, task_id, status, http_status_code, error_message, execution_time_ms, content_hash, created_at, updated_at
		FROM crawling_logs
		WHERE status = 'DISPATCHED' AND updated_at <= NOW() - MAKE_INTERVAL(mins => $1)
		ORDER BY updated_at ASC
		LIMIT $2;
	`
	rows, err := r.pool.Query(ctx, query, pendingMinutes, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending crawling logs: %w", err)
	}
	defer rows.Close()

	var logs []model.CrawlingLog
	for rows.Next() {
		var l model.CrawlingLog
		err := rows.Scan(
			&l.ID, &l.TargetID, &l.TaskID, &l.Status, &l.HTTPStatusCode, &l.ErrorMessage, &l.ExecutionTimeMs, &l.ContentHash, &l.CreatedAt, &l.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan crawling log: %w", err)
		}
		logs = append(logs, l)
	}
	return logs, nil
}
