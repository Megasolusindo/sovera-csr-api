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
	TypeCompanyInstagramDiscoveryBatch  = "task:company_instagram_discovery_batch"
	QueueCompanyInstagramDiscoveryBatch = "company-instagram-discovery-queue"
)

var (
	instagramURLFinderRegex = regexp.MustCompile(`(?i)https?://(?:[a-zA-Z0-9-]+\.)?(?:instagram\.com|instagr\.am)/[a-zA-Z0-9_\.-]+`)
)

type CompanyInstagramDiscoveryWorker struct {
	dbPool       *pgxpool.Pool
	verifier     *urlverifier.InstagramVerifier
	serperAPIKey string
	httpClient   *http.Client
}

func NewCompanyInstagramDiscoveryWorker(dbPool *pgxpool.Pool, serperAPIKey string) *CompanyInstagramDiscoveryWorker {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &CompanyInstagramDiscoveryWorker{
		dbPool:       dbPool,
		verifier:     urlverifier.NewInstagramVerifier(),
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

func NewCompanyInstagramDiscoveryBatchTask() (*asynq.Task, error) {
	return asynq.NewTask(TypeCompanyInstagramDiscoveryBatch, []byte("{}"), asynq.Queue(QueueCompanyInstagramDiscoveryBatch), asynq.MaxRetry(1)), nil
}

type InstagramDiscoveryTarget struct {
	ID           string
	Name         string
	Website      string
	InstagramURL string
	Status       string
}

func (w *CompanyInstagramDiscoveryWorker) HandleCompanyInstagramDiscoveryBatch(ctx context.Context, t *asynq.Task) error {
	if w.dbPool == nil {
		log.Println("[CompanyInstagramDiscoveryWorker] No DB pool available, skipping discovery sweep")
		return nil
	}

	log.Println("[CompanyInstagramDiscoveryWorker] Starting empirical Instagram URL discovery sweep for companies with INVALID or UNVERIFIED status...")

	var totalChecked, totalDiscovered, totalFailed int

	for {
		// Fetch next batch of 50 companies with INVALID or missing/UNVERIFIED Instagram status
		// Prioritize instagram_status = 'INVALID' first, followed by UNVERIFIED/NULL
		rows, err := w.dbPool.Query(ctx, `
			SELECT id::text, name, COALESCE(website, ''), COALESCE(instagram_url, ''), COALESCE(instagram_status, 'UNVERIFIED')
			FROM companies
			WHERE instagram_status = 'INVALID' 
			   OR (instagram_url IS NULL OR instagram_url = '') 
			   OR instagram_status = 'UNVERIFIED'
			ORDER BY 
			  (CASE WHEN instagram_status = 'INVALID' THEN 0 ELSE 1 END) ASC,
			  (CASE WHEN instagram_verified_at IS NULL THEN 0 ELSE 1 END) ASC,
			  created_at ASC
			LIMIT 50;
		`)
		if err != nil {
			return fmt.Errorf("failed to query companies for Instagram discovery: %w", err)
		}

		var targets []InstagramDiscoveryTarget
		for rows.Next() {
			var tgt InstagramDiscoveryTarget
			if err := rows.Scan(&tgt.ID, &tgt.Name, &tgt.Website, &tgt.InstagramURL, &tgt.Status); err == nil {
				targets = append(targets, tgt)
			}
		}
		rows.Close()

		if len(targets) == 0 {
			log.Printf("[CompanyInstagramDiscoveryWorker] Discovery sweep finished. Total processed=%d, newly discovered=%d, failed/not found=%d",
				totalChecked, totalDiscovered, totalFailed)
			break
		}

		log.Printf("[CompanyInstagramDiscoveryWorker] Processing batch of %d companies for Instagram re-discovery (Processed so far: %d)...", len(targets), totalChecked)

		for _, tgt := range targets {
			select {
			case <-ctx.Done():
				log.Println("[CompanyInstagramDiscoveryWorker] Context cancelled, stopping early")
				return nil
			case <-time.After(350 * time.Millisecond): // Fast rate-limiting delay between companies
			}

			totalChecked++

			discoveredURL, methodUsed, err := w.DiscoverInstagramForCompany(ctx, tgt)
			if err == nil && discoveredURL != "" {
				// Validate candidate URL empirically
				res := w.verifier.ValidateInstagramURL(ctx, discoveredURL)
				if res.IsValid {
					totalDiscovered++
					_, updateErr := w.dbPool.Exec(ctx, `
						UPDATE companies
						SET instagram_url = $1,
							instagram_status = 'VALID',
							instagram_verified_at = NOW(),
							instagram_last_error = '',
							updated_at = NOW()
						WHERE id::text = $2
					`, res.CanonicalURL, tgt.ID)

					if updateErr != nil {
						log.Printf("[CompanyInstagramDiscoveryWorker] Error updating discovered Instagram for %s: %v", tgt.Name, updateErr)
					} else {
						log.Printf("[CompanyInstagramDiscoveryWorker] [SUCCESS] [%s] %s -> Found Instagram via %s: %s",
							tgt.ID, tgt.Name, methodUsed, res.CanonicalURL)
					}
					continue
				}
			}

			// If discovery failed or candidate wasn't valid, update timestamp and error note to prevent immediate retry loop
			totalFailed++
			lastErr := "Discovery sweep: no valid Instagram profile found"
			if err != nil {
				lastErr = fmt.Sprintf("Discovery sweep error: %v", err)
			}

			_, _ = w.dbPool.Exec(ctx, `
				UPDATE companies
				SET instagram_verified_at = NOW(),
					instagram_last_error = $1,
					updated_at = NOW()
				WHERE id::text = $2
			`, lastErr, tgt.ID)

			log.Printf("[CompanyInstagramDiscoveryWorker] [NOT FOUND] [%s] %s -> No valid Instagram URL discovered", tgt.ID, tgt.Name)
		}
	}

	return nil
}

func (w *CompanyInstagramDiscoveryWorker) DiscoverInstagramForCompany(ctx context.Context, target InstagramDiscoveryTarget) (string, string, error) {
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
					matches := instagramURLFinderRegex.FindAllString(string(bodyBytes), -1)

					for _, candidate := range matches {
						res := w.verifier.ValidateInstagramURL(ctx, candidate)
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
		query := fmt.Sprintf("site:instagram.com %q", target.Name)
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
							// Check item link and snippet
							matches := instagramURLFinderRegex.FindAllString(item.Link+" "+item.Snippet, -1)
							for _, candidate := range matches {
								res := w.verifier.ValidateInstagramURL(ctx, candidate)
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
