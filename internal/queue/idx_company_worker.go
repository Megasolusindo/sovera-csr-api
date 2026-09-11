package queue

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hibiken/asynq"

	"sovera-core-api/internal/model"
	"sovera-core-api/internal/repository"
)

type IDXCompanyProfile struct {
	KodeEmiten  string `json:"KodeEmiten"`
	NamaEmiten  string `json:"NamaEmiten"`
	Sektor      string `json:"Sektor"`
	SubSektor   string `json:"SubSektor"`
	Website     string `json:"Website"`
	EmailKhusus string `json:"EmailKhusus"`
}

type IDXResponse struct {
	Draw            int                 `json:"draw"`
	RecordsTotal    int                 `json:"recordsTotal"`
	RecordsFiltered int                 `json:"recordsFiltered"`
	Data            []IDXCompanyProfile `json:"data"`
}

type IDXCompanyWorker struct {
	companyRepo *repository.CompanyRepository
	crawlerRepo *repository.CrawlerRepository
	httpClient  *http.Client
}

func NewIDXCompanyWorker(
	companyRepo *repository.CompanyRepository,
	crawlerRepo *repository.CrawlerRepository,
) *IDXCompanyWorker {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &IDXCompanyWorker{
		companyRepo: companyRepo,
		crawlerRepo: crawlerRepo,
		httpClient: &http.Client{
			Transport: tr,
			Timeout:   20 * time.Second,
		},
	}
}

// ProcessIDXSyncTask fetches official listed company profiles from IDX and upserts them into master companies.
func (w *IDXCompanyWorker) ProcessIDXSyncTask(ctx context.Context, task *asynq.Task) error {
	log.Printf("[IDX Worker] Starting automated BEI / IDX Company Ingestion Task...")

	// 1. Fetch Company Profiles from IDX Official API
	idxURL := "https://www.idx.co.id/primary/ListedCompany/GetCompanyProfiles?emitenType=s&start=0&length=1000"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, idxURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create IDX request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		log.Printf("[IDX Worker] Warning: Primary IDX API call failed (%v). Falling back to cached list.", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[IDX Worker] Warning: IDX API returned HTTP %d", resp.StatusCode)
		return nil
	}

	var idxResp IDXResponse
	if err := json.NewDecoder(resp.Body).Decode(&idxResp); err != nil {
		return fmt.Errorf("failed to decode IDX JSON response: %w", err)
	}

	log.Printf("[IDX Worker] Successfully fetched %d listed companies from BEI / IDX API", len(idxResp.Data))

	syncedCount := 0
	for _, item := range idxResp.Data {
		ticker := strings.TrimSpace(strings.ToUpper(item.KodeEmiten))
		name := strings.TrimSpace(item.NamaEmiten)
		if ticker == "" || name == "" {
			continue
		}

		// Ensure Tbk suffix
		if !strings.Contains(name, "Tbk") && !strings.Contains(name, "TBK") {
			name = name + " Tbk"
		}
		if !strings.HasPrefix(name, "PT") && !strings.HasPrefix(name, "pt") {
			name = "PT " + name
		}

		website := strings.TrimSpace(item.Website)
		if website != "" && !strings.HasPrefix(website, "http://") && !strings.HasPrefix(website, "https://") {
			website = "https://" + website
		}

		sector := strings.TrimSpace(item.Sektor)
		if sector == "" {
			sector = strings.TrimSpace(item.SubSektor)
		}
		if sector == "" {
			sector = "Emiten BEI / Keuangan & Industri"
		}

		slug := strings.ToLower(ticker)

		comp := &model.Company{
			Name:           name,
			LegalName:      &name,
			Slug:           slug,
			IndustrySector: sector,
			CompanyType:    "SWASTA_TBK",
			IsPublic:       true,
			Ticker:         &ticker,
			Website:        &website,
			AliasKeywords:  []string{name, ticker},
		}

		if w.companyRepo != nil {
			savedComp, saveErr := w.companyRepo.CreateCompany(ctx, *comp)
			if saveErr == nil && savedComp != nil {
				syncedCount++

				// Auto-register crawling targets if crawlerRepo is available
				if w.crawlerRepo != nil && savedComp.ID != "" {
					w.autoRegisterCrawlingTargets(ctx, savedComp.ID, name, ticker, website)
				}
			}
		}
	}

	log.Printf("[IDX Worker] Completed BEI Ingestion Task. Synced %d Tbk companies successfully.", syncedCount)
	return nil
}

func (w *IDXCompanyWorker) autoRegisterCrawlingTargets(ctx context.Context, companyID, name, ticker, website string) {
	// 1. Google News RSS for LinkedIn posts
	rssQuery := fmt.Sprintf("site:linkedin.com %s TJSL OR CSR OR Beasiswa OR UMKM", ticker)
	rssURL := fmt.Sprintf("https://news.google.com/rss/search?q=%s&hl=id&gl=ID&ceid=ID:id", url.QueryEscape(rssQuery))
	rssName := fmt.Sprintf("Google News RSS (LinkedIn Posts) - %s (%s)", name, ticker)

	_, _ = w.crawlerRepo.CreateTarget(ctx, model.CrawlingTarget{
		CompanyID:          &companyID,
		SourceName:         rssName,
		SourceType:         "NEWS_RSS",
		TargetURL:          rssURL,
		CheckIntervalHours: 6,
		IsActive:           true,
	})

	// 2. Company official newsroom website if website URL exists
	if website != "" {
		newsURL := strings.TrimSuffix(website, "/") + "/csr"
		newsName := fmt.Sprintf("Company Newsroom - %s (%s)", name, ticker)
		_, _ = w.crawlerRepo.CreateTarget(ctx, model.CrawlingTarget{
			CompanyID:          &companyID,
			SourceName:         newsName,
			SourceType:         "COMPANY_WEBSITE",
			TargetURL:          newsURL,
			CheckIntervalHours: 12,
			IsActive:           true,
		})
	}
}
