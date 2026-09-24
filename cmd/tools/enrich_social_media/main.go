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
	// Strip PT, Tbk, Persero prefixes/suffixes for clean slug matching
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
	log.Println("=== SOVERA MULTI-PLATFORM SOCIAL MEDIA ENRICHMENT AGENT ===")
	cfg := config.LoadConfig()

	ctx := context.Background()
	dbPool, err := repository.InitDBPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Fatal: Could not connect to database: %v", err)
	}
	defer dbPool.Close()

	// Query companies needing social media enrichment
	rows, err := dbPool.Query(ctx, `
		SELECT id, name, slug, website
		FROM company.companies
		WHERE instagram_url IS NULL OR facebook_url IS NULL OR youtube_url IS NULL
		ORDER BY (CASE WHEN website IS NOT NULL AND website <> '' THEN 0 ELSE 1 END) ASC, created_at ASC;
	`)
	if err != nil {
		log.Fatalf("Failed to query companies for social media enrichment: %v", err)
	}
	defer rows.Close()

	type companyItem struct {
		ID      string
		Name    string
		Slug    string
		Website *string
	}

	var companies []companyItem
	for rows.Next() {
		var item companyItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Slug, &item.Website); err != nil {
			log.Printf("Error scanning company row: %v", err)
			continue
		}
		companies = append(companies, item)
	}

	log.Printf("Found %d companies eligible for Multi-Platform Social Media Enrichment.", len(companies))

	var enrichedCount, targetsAdded int
	chunkSize := 500

	for i := 0; i < len(companies); i += chunkSize {
		end := i + chunkSize
		if end > len(companies) {
			end = len(companies)
		}
		chunk := companies[i:end]

			// 1. Register empirical Google News RSS Feed Targets for Instagram, Facebook, and YouTube
			igRssQuery := url.QueryEscape(fmt.Sprintf("site:instagram.com %s TJSL OR CSR OR Beasiswa OR UMKM", c.Name))
			igRssURL := fmt.Sprintf("https://news.google.com/rss/search?q=%s&hl=id&gl=ID&ceid=ID:id", igRssQuery)
			igRssName := fmt.Sprintf("Google News RSS (Instagram Posts) - %s", c.Name)
			batch.Queue(`
				INSERT INTO crawling_targets (company_id, source_name, source_type, target_url, check_interval_hours, is_active, health_status, next_run_at, created_at, updated_at)
				VALUES ($1, $2, 'NEWS_RSS', $3, 12, true, 'HEALTHY', NOW(), NOW(), NOW())
				ON CONFLICT DO NOTHING;
			`, c.ID, igRssName, igRssURL)

			fbRssQuery := url.QueryEscape(fmt.Sprintf("site:facebook.com %s TJSL OR CSR OR Beasiswa OR UMKM", c.Name))
			fbRssURL := fmt.Sprintf("https://news.google.com/rss/search?q=%s&hl=id&gl=ID&ceid=ID:id", fbRssQuery)
			fbRssName := fmt.Sprintf("Google News RSS (Facebook Posts) - %s", c.Name)
			batch.Queue(`
				INSERT INTO crawling_targets (company_id, source_name, source_type, target_url, check_interval_hours, is_active, health_status, next_run_at, created_at, updated_at)
				VALUES ($1, $2, 'NEWS_RSS', $3, 12, true, 'HEALTHY', NOW(), NOW(), NOW())
				ON CONFLICT DO NOTHING;
			`, c.ID, fbRssName, fbRssURL)

			// 4. Queue YouTube Targets
			ytOfficial := fmt.Sprintf("YouTube Official - %s", c.Name)
			batch.Queue(`
				INSERT INTO crawling_targets (company_id, source_name, source_type, target_url, check_interval_hours, is_active, health_status, next_run_at, created_at, updated_at)
				VALUES ($1, $2, 'COMPANY_WEBSITE', $3, 24, true, 'HEALTHY', NOW(), NOW(), NOW())
				ON CONFLICT DO NOTHING;
			`, c.ID, ytOfficial, youtubeURL)

			ytRssQuery := url.QueryEscape(fmt.Sprintf("site:youtube.com %s TJSL OR CSR OR Beasiswa OR UMKM", c.Name))
			ytRssURL := fmt.Sprintf("https://news.google.com/rss/search?q=%s&hl=id&gl=ID&ceid=ID:id", ytRssQuery)
			ytRssName := fmt.Sprintf("Google News RSS (YouTube Videos) - %s", c.Name)
			batch.Queue(`
				INSERT INTO crawling_targets (company_id, source_name, source_type, target_url, check_interval_hours, is_active, health_status, next_run_at, created_at, updated_at)
				VALUES ($1, $2, 'NEWS_RSS', $3, 12, true, 'HEALTHY', NOW(), NOW(), NOW())
				ON CONFLICT DO NOTHING;
			`, c.ID, ytRssName, ytRssURL)
		}

		br := dbPool.SendBatch(ctx, batch)
		for j := 0; j < batch.Len(); j++ {
			ct, err := br.Exec()
			if err != nil {
				log.Printf("Batch exec error at batch item %d: %v", j, err)
			} else {
				if j%7 == 0 {
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

	log.Printf("=== COMPLETED MULTI-PLATFORM SOCIAL MEDIA ENRICHMENT AGENT RUN ===")
	log.Printf("Total Companies Enriched: %d", enrichedCount)
	log.Printf("Total Crawling Targets Created in crawling_targets: %d", targetsAdded)
}
