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

type CompanyYoutubeTarget struct {
	ID         string
	Name       string
	YoutubeURL string
}

type CompanyYoutubeWorker struct {
	dbPool   *pgxpool.Pool
	verifier *urlverifier.YoutubeVerifier
}

func NewCompanyYoutubeWorker(dbPool *pgxpool.Pool) *CompanyYoutubeWorker {
	return &CompanyYoutubeWorker{
		dbPool:   dbPool,
		verifier: urlverifier.NewYoutubeVerifier(),
	}
}

func (w *CompanyYoutubeWorker) HandleCompanyYoutubeBatchCheck(ctx context.Context, t *asynq.Task) error {
	if w.dbPool == nil {
		log.Println("[CompanyYoutubeWorker] No DB pool available, skipping batch check")
		return nil
	}

	log.Println("[CompanyYoutubeWorker] Starting batch company YouTube URL validation...")

	var totalChecked, totalValid, totalInvalid int

	for {
		rows, err := w.dbPool.Query(ctx, `
			SELECT id::text, name, youtube_url
			FROM company.companies
			WHERE youtube_url IS NOT NULL 
			  AND youtube_url <> ''
			  AND (youtube_verified_at IS NULL OR youtube_verified_at < NOW() - INTERVAL '12 hours')
			ORDER BY (CASE WHEN youtube_status IS NULL OR youtube_status = 'UNVERIFIED' THEN 0 ELSE 1 END) ASC, created_at ASC
			LIMIT 50;
		`)
		if err != nil {
			return fmt.Errorf("failed to query companies for YouTube check: %w", err)
		}

		var targets []CompanyYoutubeTarget
		for rows.Next() {
			var tgt CompanyYoutubeTarget
			if err := rows.Scan(&tgt.ID, &tgt.Name, &tgt.YoutubeURL); err == nil {
				targets = append(targets, tgt)
			}
		}
		rows.Close()

		if len(targets) == 0 {
			log.Printf("[CompanyYoutubeWorker] Batch sweep finished. Total verified: checked=%d valid=%d invalid=%d",
				totalChecked, totalValid, totalInvalid)
			break
		}

		log.Printf("[CompanyYoutubeWorker] Verifying batch of %d companies (Total checked so far: %d)...", len(targets), totalChecked)

		for _, tgt := range targets {
			select {
			case <-ctx.Done():
				log.Println("[CompanyYoutubeWorker] Context cancelled, stopping early")
				return nil
			case <-time.After(300 * time.Millisecond):
			}

			res := w.verifier.ValidateYoutubeURL(ctx, tgt.YoutubeURL)
			totalChecked++

			var status string
			var lastErr string

			if res.IsValid {
				status = "VALID"
				totalValid++
				_, err := w.dbPool.Exec(ctx, `
					UPDATE company.companies
					SET youtube_status = 'VALID',
						youtube_verified_at = NOW(),
						youtube_last_error = NULL,
						updated_at = NOW()
					WHERE id::text = $1
				`, tgt.ID)
				if err != nil {
					log.Printf("[CompanyYoutubeWorker] Error updating status for %s (%s): %v", tgt.Name, tgt.ID, err)
				}
			} else {
				status = "INVALID"
				lastErr = res.Reason
				totalInvalid++
				_, err := w.dbPool.Exec(ctx, `
					UPDATE company.companies
					SET youtube_url = NULL,
						youtube_status = NULL,
						youtube_verified_at = NOW(),
						youtube_last_error = $1,
						updated_at = NOW()
					WHERE id::text = $2
				`, lastErr, tgt.ID)
				if err != nil {
					log.Printf("[CompanyYoutubeWorker] Error clearing invalid URL for %s (%s): %v", tgt.Name, tgt.ID, err)
				}
			}

			if totalChecked%10 == 0 || !res.IsValid {
				log.Printf("[CompanyYoutubeWorker] [%d] %s (%s) => %s | Reason: %s",
					totalChecked, tgt.Name, tgt.ID, status, res.Reason)
			}
		}
	}

	return nil
}
