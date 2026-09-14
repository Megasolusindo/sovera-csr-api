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
	TypeCompanyLinkedInBatchCheck  = "task:company_linkedin_batch_check"
	QueueCompanyLinkedInBatchCheck = "company-linkedin-batch-queue"
)

type CompanyLinkedInWorker struct {
	dbPool   *pgxpool.Pool
	verifier *urlverifier.LinkedInVerifier
}

func NewCompanyLinkedInWorker(dbPool *pgxpool.Pool) *CompanyLinkedInWorker {
	return &CompanyLinkedInWorker{
		dbPool:   dbPool,
		verifier: urlverifier.NewLinkedInVerifier(),
	}
}

func NewCompanyLinkedInBatchCheckTask() (*asynq.Task, error) {
	return asynq.NewTask(TypeCompanyLinkedInBatchCheck, []byte("{}"), asynq.Queue(QueueCompanyLinkedInBatchCheck), asynq.MaxRetry(1)), nil
}

type CompanyLinkedInTarget struct {
	ID          string
	Name        string
	LinkedInURL string
}

func (w *CompanyLinkedInWorker) HandleCompanyLinkedInBatchCheck(ctx context.Context, t *asynq.Task) error {
	if w.dbPool == nil {
		log.Println("[CompanyLinkedInWorker] No DB pool available, skipping batch check")
		return nil
	}

	log.Println("[CompanyLinkedInWorker] Starting batch company LinkedIn URL validation...")

	var totalChecked, totalValid, totalInvalid int

	for {
		// Fetch next batch of 50 companies with non-null linkedin_url that haven't been verified in the last 12 hours
		rows, err := w.dbPool.Query(ctx, `
			SELECT id::text, name, linkedin_url
			FROM companies
			WHERE linkedin_url IS NOT NULL 
			  AND linkedin_url <> ''
			  AND (linkedin_verified_at IS NULL OR linkedin_verified_at < NOW() - INTERVAL '12 hours')
			ORDER BY (CASE WHEN linkedin_status IS NULL OR linkedin_status = 'UNVERIFIED' THEN 0 ELSE 1 END) ASC, created_at ASC
			LIMIT 50;
		`)
		if err != nil {
			return fmt.Errorf("failed to query companies for LinkedIn check: %w", err)
		}

		var targets []CompanyLinkedInTarget
		for rows.Next() {
			var tgt CompanyLinkedInTarget
			if err := rows.Scan(&tgt.ID, &tgt.Name, &tgt.LinkedInURL); err == nil {
				targets = append(targets, tgt)
			}
		}
		rows.Close()

		if len(targets) == 0 {
			log.Printf("[CompanyLinkedInWorker] Batch sweep finished. Total verified: checked=%d valid=%d invalid=%d",
				totalChecked, totalValid, totalInvalid)
			break
		}

		log.Printf("[CompanyLinkedInWorker] Verifying batch of %d companies (Total checked so far: %d)...", len(targets), totalChecked)

		for _, tgt := range targets {

			select {
			case <-ctx.Done():
				log.Println("[CompanyLinkedInWorker] Context cancelled, stopping early")
				return nil
			case <-time.After(300 * time.Millisecond): // 300ms fast rate limit
			}

			res := w.verifier.ValidateLinkedInURL(ctx, tgt.LinkedInURL)
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
				SET linkedin_status = $1,
					linkedin_verified_at = NOW(),
					linkedin_last_error = $2,
					updated_at = NOW()
				WHERE id::text = $3
			`, status, lastErr, tgt.ID)

			if err != nil {
				log.Printf("[CompanyLinkedInWorker] Error updating status for %s (%s): %v", tgt.Name, tgt.ID, err)
			} else {
				if totalChecked%10 == 0 || !res.IsValid {
					log.Printf("[CompanyLinkedInWorker] [%d] %s (%s) => %s | Reason: %s",
						totalChecked, tgt.Name, tgt.LinkedInURL, status, res.Reason)
				}
			}
		}
	}

	return nil
}
