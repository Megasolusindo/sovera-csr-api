package queue

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	"sovera-core-api/internal/service/urlverifier"
)

const (
	TypeCompanyInstagramBatchCheck  = "task:company_instagram_batch_check"
	QueueCompanyInstagramBatchCheck = "company-instagram-batch-queue"
)

type CompanyInstagramWorker struct {
	dbPool   *pgxpool.Pool
	verifier *urlverifier.InstagramVerifier
}

func NewCompanyInstagramWorker(dbPool *pgxpool.Pool) *CompanyInstagramWorker {
	return &CompanyInstagramWorker{
		dbPool:   dbPool,
		verifier: urlverifier.NewInstagramVerifier(),
	}
}

func NewCompanyInstagramBatchCheckTask() (*asynq.Task, error) {
	return asynq.NewTask(TypeCompanyInstagramBatchCheck, []byte("{}"), asynq.Queue(QueueCompanyInstagramBatchCheck), asynq.MaxRetry(1)), nil
}

type CompanyInstagramTarget struct {
	ID           string
	Name         string
	InstagramURL string
}

func (w *CompanyInstagramWorker) HandleCompanyInstagramBatchCheck(ctx context.Context, t *asynq.Task) error {
	if w.dbPool == nil {
		log.Println("[CompanyInstagramWorker] No DB pool available, skipping batch check")
		return nil
	}

	log.Println("[CompanyInstagramWorker] Starting batch company Instagram URL validation...")

	var totalChecked, totalValid, totalInvalid int

	for {
		// Fetch next batch of 50 companies with non-null instagram_url that haven't been verified in the last 12 hours
		rows, err := w.dbPool.Query(ctx, `
			SELECT id::text, name, instagram_url
			FROM companies
			WHERE instagram_url IS NOT NULL 
			  AND instagram_url <> ''
			  AND (instagram_verified_at IS NULL OR instagram_verified_at < NOW() - INTERVAL '12 hours')
			ORDER BY (CASE WHEN instagram_status IS NULL OR instagram_status = 'UNVERIFIED' THEN 0 ELSE 1 END) ASC, created_at ASC
			LIMIT 50;
		`)
		if err != nil {
			return fmt.Errorf("failed to query companies for Instagram check: %w", err)
		}

		var targets []CompanyInstagramTarget
		for rows.Next() {
			var tgt CompanyInstagramTarget
			if err := rows.Scan(&tgt.ID, &tgt.Name, &tgt.InstagramURL); err == nil {
				targets = append(targets, tgt)
			}
		}
		rows.Close()

		if len(targets) == 0 {
			log.Printf("[CompanyInstagramWorker] Batch sweep finished. Total verified: checked=%d valid=%d invalid=%d",
				totalChecked, totalValid, totalInvalid)
			break
		}

		log.Printf("[CompanyInstagramWorker] Verifying batch of %d companies (Total checked so far: %d)...", len(targets), totalChecked)

		for _, tgt := range targets {

			select {
			case <-ctx.Done():
				log.Println("[CompanyInstagramWorker] Context cancelled, stopping early")
				return nil
			case <-time.After(300 * time.Millisecond): // 300ms rate limit delay
			}

			res := w.verifier.ValidateInstagramURL(ctx, tgt.InstagramURL)
			totalChecked++

			status := "INVALID"
			lastErr := ""
			if res.IsValid {
				status = "VALID"
				totalValid++
			} else {
				status = "INVALID"
				lastErr = res.Reason
				totalInvalid++
			}

			_, err := w.dbPool.Exec(ctx, `
				UPDATE companies
				SET instagram_status = $1,
					instagram_verified_at = NOW(),
					instagram_last_error = $2,
					updated_at = NOW()
				WHERE id::text = $3
			`, status, lastErr, tgt.ID)

			if err != nil {
				log.Printf("[CompanyInstagramWorker] Error updating status for %s (%s): %v", tgt.Name, tgt.ID, err)
			} else {
				if totalChecked%10 == 0 || !res.IsValid {
					log.Printf("[CompanyInstagramWorker] [%d] %s (%s) => %s | Reason: %s",
						totalChecked, tgt.Name, tgt.InstagramURL, status, res.Reason)
				}
			}
		}
	}

	return nil
}
