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

type CompanyFacebookTarget struct {
	ID          string
	Name        string
	FacebookURL string
}

type CompanyFacebookWorker struct {
	dbPool   *pgxpool.Pool
	verifier *urlverifier.FacebookVerifier
}

func NewCompanyFacebookWorker(dbPool *pgxpool.Pool) *CompanyFacebookWorker {
	return &CompanyFacebookWorker{
		dbPool:   dbPool,
		verifier: urlverifier.NewFacebookVerifier(),
	}
}

func (w *CompanyFacebookWorker) HandleCompanyFacebookBatchCheck(ctx context.Context, t *asynq.Task) error {
	if w.dbPool == nil {
		log.Println("[CompanyFacebookWorker] No DB pool available, skipping batch check")
		return nil
	}

	log.Println("[CompanyFacebookWorker] Starting batch company Facebook URL validation...")

	var totalChecked, totalValid, totalInvalid int

	for {
		rows, err := w.dbPool.Query(ctx, `
			SELECT id::text, name, facebook_url
			FROM company.companies
			WHERE facebook_url IS NOT NULL 
			  AND facebook_url <> ''
			  AND (facebook_verified_at IS NULL OR facebook_verified_at < NOW() - INTERVAL '12 hours')
			ORDER BY (CASE WHEN facebook_status IS NULL OR facebook_status = 'UNVERIFIED' THEN 0 ELSE 1 END) ASC, created_at ASC
			LIMIT 50;
		`)
		if err != nil {
			return fmt.Errorf("failed to query companies for Facebook check: %w", err)
		}

		var targets []CompanyFacebookTarget
		for rows.Next() {
			var tgt CompanyFacebookTarget
			if err := rows.Scan(&tgt.ID, &tgt.Name, &tgt.FacebookURL); err == nil {
				targets = append(targets, tgt)
			}
		}
		rows.Close()

		if len(targets) == 0 {
			log.Printf("[CompanyFacebookWorker] Batch sweep finished. Total verified: checked=%d valid=%d invalid=%d",
				totalChecked, totalValid, totalInvalid)
			break
		}

		log.Printf("[CompanyFacebookWorker] Verifying batch of %d companies (Total checked so far: %d)...", len(targets), totalChecked)

		for _, tgt := range targets {
			select {
			case <-ctx.Done():
				log.Println("[CompanyFacebookWorker] Context cancelled, stopping early")
				return nil
			case <-time.After(300 * time.Millisecond):
			}

			res := w.verifier.ValidateFacebookURL(ctx, tgt.FacebookURL)
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
				UPDATE company.companies
				SET facebook_status = $1,
					facebook_verified_at = NOW(),
					facebook_last_error = $2,
					updated_at = NOW()
				WHERE id::text = $3
			`, status, lastErr, tgt.ID)

			if err != nil {
				log.Printf("[CompanyFacebookWorker] Error updating status for %s (%s): %v", tgt.Name, tgt.ID, err)
			} else if totalChecked%10 == 0 || !res.IsValid {
				log.Printf("[CompanyFacebookWorker] [%d] %s (%s) => %s | Reason: %s",
					totalChecked, tgt.Name, tgt.ID, status, res.Reason)
			}
		}
	}

	return nil
}
