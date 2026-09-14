package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
	"sovera-core-api/internal/config"
	"sovera-core-api/internal/repository"
)

var nonAlphaNumRegex = regexp.MustCompile(`[^a-z0-9]+`)

func generateCompanySlug(name string) string {
	cleaned := strings.ToLower(strings.TrimSpace(name))
	// Strip PT, Tbk, Persero prefixes/suffixes for clean LinkedIn slug matching
	cleaned = strings.ReplaceAll(cleaned, "pt.", "")
	cleaned = strings.ReplaceAll(cleaned, "pt ", "")
	cleaned = strings.ReplaceAll(cleaned, "(persero)", "")
	cleaned = strings.ReplaceAll(cleaned, "tbk", "")
	cleaned = strings.TrimSpace(cleaned)

	slug := nonAlphaNumRegex.ReplaceAllString(cleaned, "-")
	slug = strings.Trim(slug, "-")
	return slug
}

func main() {
	log.Println("=== SOVERA COMPANY LINKEDIN ENRICHMENT & TARGET DISCOVERY AGENT ===")
	cfg := config.LoadConfig()

	ctx := context.Background()
	dbPool, err := repository.InitDBPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Fatal: Could not connect to database: %v", err)
	}
	defer dbPool.Close()

	// Query companies that need LinkedIn enrichment
	rows, err := dbPool.Query(ctx, `
		SELECT id, name, slug, website, ticker
		FROM company.companies
		WHERE linkedin_url IS NULL OR linkedin_url = ''
		ORDER BY (CASE WHEN website IS NOT NULL AND website <> '' THEN 0 ELSE 1 END) ASC, created_at ASC;
	`)
	if err != nil {
		log.Fatalf("Failed to query companies for LinkedIn enrichment: %v", err)
	}
	defer rows.Close()

	type companyItem struct {
		ID      string
		Name    string
		Slug    string
		Website *string
		Ticker  *string
	}

	var companies []companyItem
	for rows.Next() {
		var item companyItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Slug, &item.Website, &item.Ticker); err != nil {
			log.Printf("Error scanning company row: %v", err)
			continue
		}
		companies = append(companies, item)
	}

	log.Printf("Found %d companies eligible for LinkedIn enrichment.", len(companies))

	var enrichedCount, targetsAdded int
	chunkSize := 500

	for i := 0; i < len(companies); i += chunkSize {
		end := i + chunkSize
		if end > len(companies) {
			end = len(companies)
		}
		chunk := companies[i:end]

		batch := &pgx.Batch{}
		for _, c := range chunk {
			linkedinSlug := generateCompanySlug(c.Name)
			if linkedinSlug == "" {
				linkedinSlug = c.Slug
			}

			linkedinURL := fmt.Sprintf("https://www.linkedin.com/company/%s", linkedinSlug)

			// 1. Queue Update company.companies with linkedin_url
			batch.Queue(`
				UPDATE company.companies
				SET linkedin_url = $1, updated_at = NOW()
				WHERE id = $2;
			`, linkedinURL, c.ID)

			// 2. Queue Target 1: Official LinkedIn Company Page
			sourceNameOfficial := fmt.Sprintf("LinkedIn Official - %s", c.Name)
			batch.Queue(`
				INSERT INTO crawling_targets (company_id, source_name, source_type, target_url, check_interval_hours, is_active, health_status, next_run_at, created_at, updated_at)
				VALUES ($1, $2, 'COMPANY_WEBSITE', $3, 24, true, 'HEALTHY', NOW(), NOW(), NOW())
				ON CONFLICT DO NOTHING;
			`, c.ID, sourceNameOfficial, linkedinURL)

			// 3. Queue Target 2: Google News RSS LinkedIn Signal Indexing
			encodedQuery := url.QueryEscape(fmt.Sprintf("site:linkedin.com %s TJSL OR CSR OR Beasiswa OR UMKM", c.Name))
			rssURL := fmt.Sprintf("https://news.google.com/rss/search?q=%s&hl=id&gl=ID&ceid=ID:id", encodedQuery)
			sourceNameRSS := fmt.Sprintf("Google News RSS (LinkedIn Posts) - %s", c.Name)

			batch.Queue(`
				INSERT INTO crawling_targets (company_id, source_name, source_type, target_url, check_interval_hours, is_active, health_status, next_run_at, created_at, updated_at)
				VALUES ($1, $2, 'NEWS_RSS', $3, 12, true, 'HEALTHY', NOW(), NOW(), NOW())
				ON CONFLICT DO NOTHING;
			`, c.ID, sourceNameRSS, rssURL)
		}

		br := dbPool.SendBatch(ctx, batch)
		for j := 0; j < batch.Len(); j++ {
			ct, err := br.Exec()
			if err != nil {
				log.Printf("Batch exec error at batch item %d: %v", j, err)
			} else {
				if j%3 == 0 {
					enrichedCount += int(ct.RowsAffected())
				} else {
					targetsAdded += int(ct.RowsAffected())
				}
			}
		}
		if err := br.Close(); err != nil {
			log.Printf("Error closing batch result: %v", err)
		}

		log.Printf("Progress: [%d/%d] companies processed. Enriched: %d, Targets Added: %d", end, len(companies), enrichedCount, targetsAdded)
	}

	log.Printf("=== COMPLETED LINKEDIN ENRICHMENT AGENT RUN ===")
	log.Printf("Total Companies Enriched: %d", enrichedCount)
	log.Printf("Total LinkedIn Targets Created in crawling_targets: %d", targetsAdded)
}
