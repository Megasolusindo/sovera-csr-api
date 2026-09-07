package companyenricher

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"sovera-core-api/internal/model"
	"sovera-core-api/internal/repository"
	"sovera-core-api/internal/service/crawler"
)

type EnricherService struct {
	repo       *repository.CompanyRepository
	dispatcher *crawler.Dispatcher
	httpClient *http.Client
}

func NewEnricherService(repo *repository.CompanyRepository, dispatcher *crawler.Dispatcher) *EnricherService {
	return &EnricherService{
		repo:       repo,
		dispatcher: dispatcher,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return fmt.Errorf("stopped after 5 redirects")
				}
				return nil
			},
		},
	}
}

type IDXCompanyProfile struct {
	KodeEmiten string `json:"KodeEmiten"`
	NamaEmiten string `json:"NamaEmiten"`
	Website    string `json:"Website"`
}

type IDXResponse struct {
	Profiles []IDXCompanyProfile `json:"Profiles"`
}

// EnrichMissingWebsites runs an enrichment sweep over companies where website IS NULL.
func (s *EnricherService) EnrichMissingWebsites(ctx context.Context, limit int) (int, error) {
	if s.repo == nil {
		return 0, fmt.Errorf("repository is nil")
	}

	companies, err := s.repo.GetCompaniesMissingWebsite(ctx, limit)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch companies with missing website: %w", err)
	}

	if len(companies) == 0 {
		log.Println("No companies with missing website found.")
		return 0, nil
	}

	log.Printf("Found %d companies with missing website. Starting enrichment sweep...", len(companies))

	// 1. Fetch IDX Profiles for Tbk companies
	idxProfilesMap := s.fetchIDXCompanyProfiles(ctx)

	enrichedCount := 0
	for _, c := range companies {
		var candidateURL string

		// Check if ticker matches IDX emiten profile
		if c.Ticker != nil && *c.Ticker != "" {
			tUpper := strings.ToUpper(*c.Ticker)
			if idxWeb, exists := idxProfilesMap[tUpper]; exists && idxWeb != "" {
				candidateURL = idxWeb
			}
		}

		if candidateURL != "" {
			normalized := s.normalizeURL(candidateURL)
			if s.verifyWebsite(ctx, normalized) {
				log.Printf("Verified official website for [%s] (%s): %s", c.Name, c.ID, normalized)
				if updateErr := s.repo.UpdateCompanyWebsite(ctx, c.ID, normalized); updateErr == nil {
					enrichedCount++
					continue
				}
			}
		}

		// If no direct website found and dispatcher is available, dispatch WebScraper v2 COMPANY_ENRICHMENT discovery task
		if s.dispatcher != nil {
			taskID := fmt.Sprintf("enrich-comp-%s-%d", c.ID, time.Now().Unix())
			payload := model.DiscoveryTaskPayload{
				TaskID:       taskID,
				ClientOrigin: "sovera_b2b_engine",
				Query:        fmt.Sprintf("%s official website", c.Name),
				Engine:       "google",
				Limit:        5,
			}
			if status, err := s.dispatcher.DispatchDiscovery(ctx, payload); err == nil {
				log.Printf("Dispatched WebScraper v2 COMPANY_ENRICHMENT discovery for [%s] (Status: %d)", c.Name, status)
			}
		}
	}

	log.Printf("Company website enrichment sweep finished. Total enriched: %d", enrichedCount)
	return enrichedCount, nil
}

func (s *EnricherService) fetchIDXCompanyProfiles(ctx context.Context) map[string]string {
	result := make(map[string]string)

	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.idx.co.id/primary/ListedCompany/GetCompanyProfiles", nil)
	if err != nil {
		return result
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)")
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return result
	}
	defer resp.Body.Close()

	var idxData IDXResponse
	if err := json.NewDecoder(resp.Body).Decode(&idxData); err == nil {
		for _, p := range idxData.Profiles {
			if p.KodeEmiten != "" && p.Website != "" {
				result[strings.ToUpper(p.KodeEmiten)] = p.Website
			}
		}
	}
	return result
}

func (s *EnricherService) normalizeURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		raw = "https://" + raw
	}
	return strings.TrimSuffix(raw, "/")
}

func (s *EnricherService) verifyWebsite(ctx context.Context, targetURL string) bool {
	req, err := http.NewRequestWithContext(ctx, "HEAD", targetURL, nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", "SoveraBot/2.0 (+https://sovera.id)")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		// Fallback GET
		req.Method = "GET"
		resp, err = s.httpClient.Do(req)
		if err != nil {
			return false
		}
	}
	defer resp.Body.Close()

	// Check HTTP Status
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return true
	}
	return false
}
