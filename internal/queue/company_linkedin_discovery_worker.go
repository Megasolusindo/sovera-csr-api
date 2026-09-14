package queue

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	"sovera-core-api/internal/service/urlverifier"
)

const (
	TypeCompanyLinkedInDiscoveryBatch  = "task:company_linkedin_discovery_batch"
	QueueCompanyLinkedInDiscoveryBatch = "company-linkedin-discovery-queue"
)

var (
	linkedinURLFinderRegex = regexp.MustCompile(`(?i)https?://(?:[a-zA-Z0-9-]+\.)?linkedin\.com/(?:company|school)/[a-zA-Z0-9_-]+`)
)

type CompanyLinkedInDiscoveryWorker struct {
	dbPool       *pgxpool.Pool
	verifier     *urlverifier.LinkedInVerifier
	serperAPIKey string
	httpClient   *http.Client
}

func NewCompanyLinkedInDiscoveryWorker(dbPool *pgxpool.Pool, serperAPIKey string) *CompanyLinkedInDiscoveryWorker {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &CompanyLinkedInDiscoveryWorker{
		dbPool:       dbPool,
		verifier:     urlverifier.NewLinkedInVerifier(),
		serperAPIKey: serperAPIKey,
		httpClient: &http.Client{
			Transport: tr,
			Timeout:   7 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
	}
}

func NewCompanyLinkedInDiscoveryBatchTask() (*asynq.Task, error) {
	return asynq.NewTask(TypeCompanyLinkedInDiscoveryBatch, []byte("{}"), asynq.Queue(QueueCompanyLinkedInDiscoveryBatch), asynq.MaxRetry(1)), nil
}

type DiscoveryTarget struct {
	ID          string
	Name        string
	Website     string
	LinkedInURL string
	Status      string
}

type SerperResponse struct {
	Organic []struct {
		Title   string `json:"title"`
		Link    string `json:"link"`
		Snippet string `json:"snippet"`
	} `json:"organic"`
}

func (w *CompanyLinkedInDiscoveryWorker) HandleCompanyLinkedInDiscoveryBatch(ctx context.Context, t *asynq.Task) error {
	if w.dbPool == nil {
		log.Println("[CompanyLinkedInDiscoveryWorker] No DB pool available, skipping discovery sweep")
		return nil
	}

	log.Println("[CompanyLinkedInDiscoveryWorker] Starting empirical LinkedIn URL discovery sweep for companies with INVALID or UNVERIFIED status...")

	var totalChecked, totalDiscovered, totalFailed int

	for {
		// Fetch next batch of 50 companies with INVALID or missing/UNVERIFIED LinkedIn status
		// Prioritize linkedin_status = 'INVALID' first, followed by UNVERIFIED/NULL
		rows, err := w.dbPool.Query(ctx, `
			SELECT id::text, name, COALESCE(website, ''), COALESCE(linkedin_url, ''), COALESCE(linkedin_status, 'UNVERIFIED')
			FROM companies
			WHERE linkedin_status = 'INVALID' 
			   OR (linkedin_url IS NULL OR linkedin_url = '') 
			   OR linkedin_status = 'UNVERIFIED'
			ORDER BY 
			  (CASE WHEN linkedin_status = 'INVALID' THEN 0 ELSE 1 END) ASC,
			  (CASE WHEN linkedin_verified_at IS NULL THEN 0 ELSE 1 END) ASC,
			  created_at ASC
			LIMIT 50;
		`)
		if err != nil {
			return fmt.Errorf("failed to query companies for LinkedIn discovery: %w", err)
		}

		var targets []DiscoveryTarget
		for rows.Next() {
			var tgt DiscoveryTarget
			if err := rows.Scan(&tgt.ID, &tgt.Name, &tgt.Website, &tgt.LinkedInURL, &tgt.Status); err == nil {
				targets = append(targets, tgt)
			}
		}
		rows.Close()

		if len(targets) == 0 {
			log.Printf("[CompanyLinkedInDiscoveryWorker] Discovery sweep finished. Total processed=%d, newly discovered=%d, failed/not found=%d",
				totalChecked, totalDiscovered, totalFailed)
			break
		}

		log.Printf("[CompanyLinkedInDiscoveryWorker] Processing batch of %d companies for LinkedIn re-discovery (Processed so far: %d)...", len(targets), totalChecked)

		for _, tgt := range targets {
			select {
			case <-ctx.Done():
				log.Println("[CompanyLinkedInDiscoveryWorker] Context cancelled, stopping early")
				return nil
			case <-time.After(350 * time.Millisecond): // Fast rate-limiting delay between companies
			}

			totalChecked++

			discoveredURL, methodUsed, err := w.DiscoverLinkedInForCompany(ctx, tgt)
			if err == nil && discoveredURL != "" {
				// Validate candidate URL empirically
				res := w.verifier.ValidateLinkedInURL(ctx, discoveredURL)
				if res.IsValid {
					totalDiscovered++
					_, updateErr := w.dbPool.Exec(ctx, `
						UPDATE companies
						SET linkedin_url = $1,
							linkedin_status = 'VALID',
							linkedin_verified_at = NOW(),
							linkedin_last_error = '',
							updated_at = NOW()
						WHERE id::text = $2
					`, res.CanonicalURL, tgt.ID)

					if updateErr != nil {
						log.Printf("[CompanyLinkedInDiscoveryWorker] Error updating discovered LinkedIn for %s: %v", tgt.Name, updateErr)
					} else {
						log.Printf("[CompanyLinkedInDiscoveryWorker] [SUCCESS] [%s] %s -> Found LinkedIn via %s: %s",
							tgt.ID, tgt.Name, methodUsed, res.CanonicalURL)
					}
					continue
				}
			}

			// If discovery failed or candidate wasn't valid, update timestamp and error note to prevent immediate retry loop
			totalFailed++
			lastErr := "Discovery sweep: no valid LinkedIn profile found"
			if err != nil {
				lastErr = fmt.Sprintf("Discovery sweep error: %v", err)
			}

			_, _ = w.dbPool.Exec(ctx, `
				UPDATE companies
				SET linkedin_verified_at = NOW(),
					linkedin_last_error = $1,
					updated_at = NOW()
				WHERE id::text = $2
			`, lastErr, tgt.ID)

			log.Printf("[CompanyLinkedInDiscoveryWorker] [NOT FOUND] [%s] %s -> No valid LinkedIn URL discovered", tgt.ID, tgt.Name)
		}
	}

	return nil
}

func (w *CompanyLinkedInDiscoveryWorker) DiscoverLinkedInForCompany(ctx context.Context, target DiscoveryTarget) (string, string, error) {
	// Method 1: Empirical Website HTML Scraping
	if target.Website != "" {
		webURL := target.Website
		if !strings.HasPrefix(webURL, "http://") && !strings.HasPrefix(webURL, "https://") {
			webURL = "https://" + webURL
		}

		req, err := http.NewRequestWithContext(ctx, "GET", webURL, nil)
		if err == nil {
			req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
			req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

			resp, err := w.httpClient.Do(req)
			if err == nil {
				defer resp.Body.Close()
				if resp.StatusCode >= 200 && resp.StatusCode < 400 {
					bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 512*1024)) // 512KB max
					matches := linkedinURLFinderRegex.FindAllString(string(bodyBytes), -1)

					for _, candidate := range matches {
						res := w.verifier.ValidateLinkedInURL(ctx, candidate)
						if res.IsValid {
							return res.CanonicalURL, "OFFICIAL_WEBSITE_HTML", nil
						}
					}
				}
			}
		}
	}

	// Method 2: Empirical Serper API Search
	if w.serperAPIKey != "" {
		query := fmt.Sprintf("site:linkedin.com/company %q", target.Name)
		serperURL := "https://google.serper.dev/search"

		reqBody, _ := json.Marshal(map[string]interface{}{
			"q":   query,
			"num": 5,
		})

		req, err := http.NewRequestWithContext(ctx, "POST", serperURL, bytes.NewBuffer(reqBody))
		if err == nil {
			req.Header.Set("X-API-KEY", w.serperAPIKey)
			req.Header.Set("Content-Type", "application/json")

			resp, err := w.httpClient.Do(req)
			if err == nil {
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					var serperRes SerperResponse
					if err := json.NewDecoder(resp.Body).Decode(&serperRes); err == nil {
						for _, item := range serperRes.Organic {
							// Check item link
							matches := linkedinURLFinderRegex.FindAllString(item.Link+" "+item.Snippet, -1)
							for _, candidate := range matches {
								res := w.verifier.ValidateLinkedInURL(ctx, candidate)
								if res.IsValid {
									return res.CanonicalURL, "SERPER_GOOGLE_SEARCH", nil
								}
							}
						}
					}
				}
			}
		}
	}

	return "", "", nil
}
