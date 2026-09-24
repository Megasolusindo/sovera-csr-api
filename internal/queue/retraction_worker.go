package queue

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const BatchSize = 100

type RetractionTask struct {
	ClaimID   string `json:"claim_id"`
	Reason    string `json:"reason"`
	Retracted string `json:"retracted_by"`
}

type RetractionWorker struct {
	dbPool *pgxpool.Pool
}

func NewRetractionWorker(dbPool *pgxpool.Pool) *RetractionWorker {
	return &RetractionWorker{dbPool: dbPool}
}

// ProcessRetractionCascade executes graph traversal retraction in 100-record batches (§9.1)
func (w *RetractionWorker) ProcessRetractionCascade(ctx context.Context, claimID, reason string) (int, error) {
	if w.dbPool == nil {
		return 0, fmt.Errorf("db pool is nil")
	}

	log.Printf("[RetractionWorker] Starting asynchronous lineage retraction for claim %s (Reason: %s)...", claimID, reason)

	totalProcessed := 0

	for {
		// 1. Mark batch of signals associated with retracted claim as RETRACTED
		tx, err := w.dbPool.Begin(ctx)
		if err != nil {
			return totalProcessed, fmt.Errorf("failed to begin transaction: %w", err)
		}

		query := `
			WITH target_signals AS (
				SELECT id FROM intelligence.company_signals
				WHERE (content_hash = $1 OR summary ILIKE $2)
				  AND status != 'RETRACTED'
				LIMIT $3
				FOR UPDATE SKIP LOCKED
			)
			UPDATE intelligence.company_signals s
			SET status = 'RETRACTED',
				csr_relevance = 'LOW',
				opportunity_alert = false
			FROM target_signals t
			WHERE s.id = t.id
			RETURNING s.id;
		`

		rows, err := tx.Query(ctx, query, "claim_"+claimID, "%"+claimID+"%", BatchSize)
		if err != nil {
			_ = tx.Rollback(ctx)
			return totalProcessed, fmt.Errorf("retraction update query failed: %w", err)
		}

		batchCount := 0
		for rows.Next() {
			batchCount++
		}
		rows.Close()

		if err := tx.Commit(ctx); err != nil {
			return totalProcessed, fmt.Errorf("failed to commit batch retraction: %w", err)
		}

		totalProcessed += batchCount

		if batchCount < BatchSize {
			// No more records in this batch loop
			break
		}

		// Micro-sleep between batches to prevent DB lock contention
		time.Sleep(10 * time.Millisecond)
	}

	log.Printf("[RetractionWorker] Completed retraction cascade for claim %s. Total signals retracted: %d", claimID, totalProcessed)
	return totalProcessed, nil
}
