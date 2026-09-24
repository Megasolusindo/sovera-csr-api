package repository

import (
	"context"
	"fmt"
	"strconv"
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

// GetAllTargets retrieves all crawling targets for admin monitoring.
func (r *CrawlerRepository) GetAllTargets(ctx context.Context) ([]model.CrawlingTarget, error) {
	query := `
		SELECT id, company_id::text, source_name, source_type, target_url, check_interval_hours, 
		       last_scraped_at, next_run_at, is_active, consecutive_failures,
		       last_http_status, last_error_message, health_status, created_at, updated_at
		FROM crawling_targets
		ORDER BY created_at DESC
		LIMIT 100;
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all targets: %w", err)
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
	// Circuit breaker & 429 Backoff condition:
	// - If httpStatusCode is 404 or 410, immediately mark as DISABLED_DEAD_LINK and set is_active = FALSE.
	// - If non-429 failures >= 5, set health_status = 'DISABLED_DEAD_LINK' and is_active = FALSE.
	// - If 429 (Rate Limit), use Progressive Exponential Backoff matching Google's recovery windows:
	//   1st hit: 30 minutes (soft block recovery)
	//   2nd hit: 2 hours
	//   3rd hit: 6 hours
	//   4th+ hit: 24 hours (full bot block duration)
	query := `
		UPDATE crawling_targets
		SET consecutive_failures = consecutive_failures + 1,
		    last_http_status = $2,
		    last_error_message = $3,
		    health_status = CASE 
		        WHEN $2 IN (404, 410) OR ($2 != 429 AND consecutive_failures + 1 >= 5) THEN 'DISABLED_DEAD_LINK'
		        ELSE 'DEGRADED'
		    END,
		    is_active = CASE 
		        WHEN $2 IN (404, 410) OR ($2 != 429 AND consecutive_failures + 1 >= 5) THEN FALSE
		        ELSE is_active
		    END,
		    next_run_at = CASE 
		        WHEN $2 = 429 THEN 
		            CASE 
		                WHEN consecutive_failures + 1 <= 1 THEN NOW() + INTERVAL '30 minutes'
		                WHEN consecutive_failures + 1 = 2 THEN NOW() + INTERVAL '2 hours'
		                WHEN consecutive_failures + 1 = 3 THEN NOW() + INTERVAL '6 hours'
		                ELSE NOW() + INTERVAL '24 hours'
		            END
		        ELSE next_run_at
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
		ON CONFLICT (task_id, created_at) DO UPDATE
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

type SourceTargetItem struct {
	ID                  string   `json:"id"`
	CompanyID           *string  `json:"company_id,omitempty"`
	SourceName          string   `json:"source_name"`
	SourceType          string   `json:"source_type"`
	TargetURL           string   `json:"target_url"`
	CheckIntervalHours  int      `json:"check_interval_hours"`
	LastScrapedAt       *string  `json:"last_scraped_at,omitempty"`
	NextRunAt           *string  `json:"next_run_at,omitempty"`
	IsActive            bool     `json:"is_active"`
	ConsecutiveFailures int      `json:"consecutive_failures"`
	LastHTTPStatus      *int     `json:"last_http_status,omitempty"`
	LastErrorMsg        *string  `json:"last_error_message,omitempty"`
	HealthStatus        string   `json:"health_status"`
	CreatedAt           string   `json:"created_at"`
	UpdatedAt           string   `json:"updated_at"`
	CompanyName         *string  `json:"company_name,omitempty"`
}

// ListSources retrieves a paginated list of crawling targets with filters.
func (r *CrawlerRepository) ListSources(ctx context.Context, limit, offset int, search, sourceType, healthStatus string) ([]SourceTargetItem, int, error) {
	if r.pool == nil {
		return nil, 0, fmt.Errorf("database pool is nil")
	}

	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if search != "" {
		whereClause += " AND (t.source_name ILIKE $" + strconv.Itoa(argIdx) + " OR t.target_url ILIKE $" + strconv.Itoa(argIdx) + " OR c.name ILIKE $" + strconv.Itoa(argIdx) + ")"
		args = append(args, "%"+search+"%")
		argIdx++
	}

	if sourceType != "" {
		whereClause += " AND t.source_type ILIKE $" + strconv.Itoa(argIdx)
		args = append(args, "%"+sourceType+"%")
		argIdx++
	}

	if healthStatus != "" {
		whereClause += " AND t.health_status ILIKE $" + strconv.Itoa(argIdx)
		args = append(args, "%"+healthStatus+"%")
		argIdx++
	}

	countQuery := `
		SELECT COUNT(*)
		FROM crawling_targets t
		LEFT JOIN companies c ON c.id = t.company_id
		` + whereClause + `;`

	var total int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count crawling_targets: %w", err)
	}

	query := `
		SELECT 
			t.id::text, t.company_id::text, t.source_name, t.source_type, t.target_url,
			t.check_interval_hours, t.last_scraped_at::text, t.next_run_at::text,
			t.is_active, t.consecutive_failures, t.last_http_status, t.last_error_message,
			COALESCE(t.health_status, 'HEALTHY'), t.created_at::text, t.updated_at::text,
			c.name as company_name
		FROM crawling_targets t
		LEFT JOIN companies c ON c.id = t.company_id
		` + whereClause + `
		ORDER BY t.created_at DESC
		LIMIT $` + strconv.Itoa(argIdx) + ` OFFSET $` + strconv.Itoa(argIdx+1) + `;`

	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query crawling_targets: %w", err)
	}
	defer rows.Close()

	var items []SourceTargetItem
	for rows.Next() {
		var item SourceTargetItem
		err := rows.Scan(
			&item.ID, &item.CompanyID, &item.SourceName, &item.SourceType, &item.TargetURL,
			&item.CheckIntervalHours, &item.LastScrapedAt, &item.NextRunAt,
			&item.IsActive, &item.ConsecutiveFailures, &item.LastHTTPStatus, &item.LastErrorMsg,
			&item.HealthStatus, &item.CreatedAt, &item.UpdatedAt,
			&item.CompanyName,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan SourceTargetItem row: %w", err)
		}
		items = append(items, item)
	}

	return items, total, nil
}

// CreateTarget inserts a new crawling target into crawling_targets table
func (r *CrawlerRepository) CreateTarget(ctx context.Context, target model.CrawlingTarget) (*model.CrawlingTarget, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	if target.CheckIntervalHours <= 0 {
		target.CheckIntervalHours = 24
	}

	query := `
		INSERT INTO crawling_targets (
			source_name, source_type, target_url, check_interval_hours, is_active, health_status, company_id, next_run_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, 'HEALTHY', NULLIF($6, '')::uuid, NOW(), NOW(), NOW())
		ON CONFLICT (target_url) DO UPDATE SET
			company_id = COALESCE(EXCLUDED.company_id, crawling_targets.company_id),
			is_active = true,
			updated_at = NOW()
		RETURNING id, company_id::text, source_name, source_type, target_url, check_interval_hours, 
		          last_scraped_at, next_run_at, is_active, consecutive_failures,
		          last_http_status, last_error_message, health_status, created_at, updated_at;
	`

	var t model.CrawlingTarget
	var companyIDStr *string
	companyIDInput := ""
	if target.CompanyID != nil {
		companyIDInput = *target.CompanyID
	}

	err := r.pool.QueryRow(ctx, query,
		target.SourceName, target.SourceType, target.TargetURL, target.CheckIntervalHours, target.IsActive, companyIDInput,
	).Scan(
		&t.ID, &companyIDStr, &t.SourceName, &t.SourceType, &t.TargetURL, &t.CheckIntervalHours,
		&t.LastScrapedAt, &t.NextRunAt, &t.IsActive, &t.ConsecutiveFailures,
		&t.LastHTTPStatus, &t.LastErrorMsg, &t.HealthStatus, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create crawling target: %w", err)
	}

	t.CompanyID = companyIDStr
	return &t, nil
}

// GetTargetByURL fetches a crawling target by its exact target_url
func (r *CrawlerRepository) GetTargetByURL(ctx context.Context, targetURL string) (*model.CrawlingTarget, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	query := `
		SELECT id, company_id::text, source_name, source_type, target_url, check_interval_hours, 
		       last_scraped_at, next_run_at, is_active, consecutive_failures,
		       last_http_status, last_error_message, health_status, created_at, updated_at
		FROM crawling_targets
		WHERE target_url = $1
		LIMIT 1;
	`

	var t model.CrawlingTarget
	var companyIDStr *string
	err := r.pool.QueryRow(ctx, query, targetURL).Scan(
		&t.ID, &companyIDStr, &t.SourceName, &t.SourceType, &t.TargetURL, &t.CheckIntervalHours,
		&t.LastScrapedAt, &t.NextRunAt, &t.IsActive, &t.ConsecutiveFailures,
		&t.LastHTTPStatus, &t.LastErrorMsg, &t.HealthStatus, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	t.CompanyID = companyIDStr
	return &t, nil
}
type CrawlerErrorStats struct {
	TotalFailing        int `json:"total_failing"`
	HTTP429RateLimited  int `json:"http_429_rate_limited"`
	HTTP404DeadLinks    int `json:"http_404_dead_links"`
	HTTP500ServerErrors int `json:"http_500_server_errors"`
}

// GetCrawlerErrorStats fetches aggregate metrics on failing crawling targets.
func (r *CrawlerRepository) GetCrawlerErrorStats(ctx context.Context) (*CrawlerErrorStats, error) {
	if r.pool == nil {
		return &CrawlerErrorStats{}, nil
	}
	var stats CrawlerErrorStats
	query := `
		SELECT 
			COUNT(*) FILTER (WHERE last_http_status >= 400 OR health_status != 'HEALTHY'),
			COUNT(*) FILTER (WHERE last_http_status = 429),
			COUNT(*) FILTER (WHERE last_http_status IN (404, 410) OR health_status = 'DISABLED_DEAD_LINK'),
			COUNT(*) FILTER (WHERE last_http_status >= 500)
		FROM crawling_targets;
	`
	err := r.pool.QueryRow(ctx, query).Scan(
		&stats.TotalFailing,
		&stats.HTTP429RateLimited,
		&stats.HTTP404DeadLinks,
		&stats.HTTP500ServerErrors,
	)
	if err != nil {
		return &CrawlerErrorStats{}, nil
	}
	return &stats, nil
}

// ResetCrawlerErrors resets health status and failures for a target or all failing targets.
func (r *CrawlerRepository) ResetCrawlerErrors(ctx context.Context, targetID string) (int, error) {
	if r.pool == nil {
		return 0, fmt.Errorf("database pool is nil")
	}

	var query string
	var args []interface{}

	if targetID != "" && targetID != "ALL" {
		query = `
			UPDATE crawling_targets
			SET consecutive_failures = 0,
			    health_status = 'HEALTHY',
			    is_active = TRUE,
			    next_run_at = NOW(),
			    last_error_message = NULL,
			    updated_at = NOW()
			WHERE id = $1;
		`
		args = append(args, targetID)
	} else {
		query = `
			UPDATE crawling_targets
			SET consecutive_failures = 0,
			    health_status = 'HEALTHY',
			    is_active = TRUE,
			    next_run_at = NOW(),
			    last_error_message = NULL,
			    updated_at = NOW()
			WHERE last_http_status >= 400 OR health_status != 'HEALTHY';
		`
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to reset crawler errors: %w", err)
	}
	return int(tag.RowsAffected()), nil
}


