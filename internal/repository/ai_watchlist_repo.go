package repository

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"sovera-core-api/internal/model"
)

type AIWatchlistRepository struct {
	dbPool *pgxpool.Pool
}

func NewAIWatchlistRepository(dbPool *pgxpool.Pool) *AIWatchlistRepository {
	return &AIWatchlistRepository{dbPool: dbPool}
}

// AddOrUpdateWatchlist adds or updates a company in the OpenClaw watchlist
func (r *AIWatchlistRepository) AddOrUpdateWatchlist(ctx context.Context, item model.AICompanyWatchlist) (*model.AICompanyWatchlist, error) {
	if r.dbPool == nil {
		return nil, fmt.Errorf("database pool is not initialized")
	}

	keywords := item.MonitoringKeywords
	if len(keywords) == 0 {
		keywords = []string{"CSR", "TJSL", "Keberlanjutan", "ESG"}
	}

	intervalHours := item.CheckIntervalHours
	if intervalHours <= 0 {
		intervalHours = 12
	}

	query := `
		INSERT INTO ai_company_watchlist (
			company_id, company_name, monitoring_keywords, check_interval_hours, is_active, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, NOW(), NOW()
		)
		ON CONFLICT (company_id) DO UPDATE
		SET company_name = EXCLUDED.company_name,
		    monitoring_keywords = EXCLUDED.monitoring_keywords,
		    check_interval_hours = EXCLUDED.check_interval_hours,
		    is_active = EXCLUDED.is_active,
		    updated_at = NOW()
		RETURNING id, company_id, company_name, monitoring_keywords, check_interval_hours, last_monitored_at, is_active, created_at, updated_at;
	`

	var w model.AICompanyWatchlist
	err := r.dbPool.QueryRow(ctx, query,
		item.CompanyID,
		item.CompanyName,
		keywords,
		intervalHours,
		item.IsActive,
	).Scan(
		&w.ID,
		&w.CompanyID,
		&w.CompanyName,
		&w.MonitoringKeywords,
		&w.CheckIntervalHours,
		&w.LastMonitoredAt,
		&w.IsActive,
		&w.CreatedAt,
		&w.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to save watchlist item: %w", err)
	}

	return &w, nil
}

// ListActiveWatchlist retrieves all active monitored companies for OpenClaw
func (r *AIWatchlistRepository) ListActiveWatchlist(ctx context.Context) ([]model.AICompanyWatchlist, error) {
	if r.dbPool == nil {
		return []model.AICompanyWatchlist{}, nil
	}

	query := `
		SELECT id, company_id, company_name, monitoring_keywords, check_interval_hours, last_monitored_at, is_active, created_at, updated_at
		FROM ai_company_watchlist
		WHERE is_active = true
		ORDER BY company_name ASC;
	`

	rows, err := r.dbPool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch active watchlist: %w", err)
	}
	defer rows.Close()

	var list []model.AICompanyWatchlist
	for rows.Next() {
		var w model.AICompanyWatchlist
		err := rows.Scan(
			&w.ID,
			&w.CompanyID,
			&w.CompanyName,
			&w.MonitoringKeywords,
			&w.CheckIntervalHours,
			&w.LastMonitoredAt,
			&w.IsActive,
			&w.CreatedAt,
			&w.UpdatedAt,
		)
		if err != nil {
			log.Printf("Scan error in ListActiveWatchlist: %v", err)
			continue
		}
		list = append(list, w)
	}

	return list, nil
}

// ListAllWatchlist retrieves watchlist items with pagination for Admin Console
func (r *AIWatchlistRepository) ListAllWatchlist(ctx context.Context, limit, offset int) ([]model.AICompanyWatchlist, int, error) {
	if r.dbPool == nil {
		return []model.AICompanyWatchlist{}, 0, nil
	}

	if limit <= 0 {
		limit = 20
	}

	var total int
	countQuery := `SELECT COUNT(*) FROM ai_company_watchlist;`
	err := r.dbPool.QueryRow(ctx, countQuery).Scan(&total)
	if err != nil {
		return []model.AICompanyWatchlist{}, 0, err
	}

	query := `
		SELECT id, company_id, company_name, monitoring_keywords, check_interval_hours, last_monitored_at, is_active, created_at, updated_at
		FROM ai_company_watchlist
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2;
	`

	rows, err := r.dbPool.Query(ctx, query, limit, offset)
	if err != nil {
		return []model.AICompanyWatchlist{}, 0, err
	}
	defer rows.Close()

	list := make([]model.AICompanyWatchlist, 0)
	for rows.Next() {
		var w model.AICompanyWatchlist
		err := rows.Scan(
			&w.ID,
			&w.CompanyID,
			&w.CompanyName,
			&w.MonitoringKeywords,
			&w.CheckIntervalHours,
			&w.LastMonitoredAt,
			&w.IsActive,
			&w.CreatedAt,
			&w.UpdatedAt,
		)
		if err != nil {
			log.Printf("Scan error in ListAllWatchlist: %v", err)
			continue
		}
		list = append(list, w)
	}

	return list, total, nil
}

// UpdateLastMonitored updates last_monitored_at timestamp for a company
func (r *AIWatchlistRepository) UpdateLastMonitored(ctx context.Context, companyID uuid.UUID) error {
	if r.dbPool == nil {
		return nil
	}

	query := `UPDATE ai_company_watchlist SET last_monitored_at = NOW(), updated_at = NOW() WHERE company_id = $1;`
	_, err := r.dbPool.Exec(ctx, query, companyID)
	return err
}

// RemoveFromWatchlist removes a company from the monitoring watchlist
func (r *AIWatchlistRepository) RemoveFromWatchlist(ctx context.Context, companyID uuid.UUID) error {
	if r.dbPool == nil {
		return nil
	}

	query := `DELETE FROM ai_company_watchlist WHERE company_id = $1 OR id = $1;`
	_, err := r.dbPool.Exec(ctx, query, companyID)
	return err
}
