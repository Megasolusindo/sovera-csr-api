package crawler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"sovera-core-api/internal/config"
	"sovera-core-api/internal/model"
)

// isBlockedHost checks if a hostname should be blocked to prevent SSRF attacks.
// Blocks localhost, loopback, link-local, and cloud metadata endpoints.
// If DNS lookup fails, the URL is allowed (fallback to permissive mode).
func isBlockedHost(hostname string) bool {
	if hostname == "" {
		return true
	}
	if hostname == "localhost" {
		return true
	}
	lower := strings.ToLower(hostname)
	for _, blocked := range []string{"metadata.google.internal", "metadata", "kubernetes", "kubernetes.default", "kubernetes.default.svc", "consul", "etcd", "vault"} {
		if lower == blocked || strings.HasPrefix(lower, blocked+".") {
			return true
		}
	}
	// Allow if DNS lookup fails (fallback to permissive mode)
	ips, err := net.LookupIP(hostname)
	if err != nil {
		return false
	}
	for _, ip := range ips {
		if isPrivateOrLoopbackIP(ip) {
			return true
		}
	}
	return false
}

func isPrivateOrLoopbackIP(ip net.IP) bool {
	if ip.IsLoopback() {
		return true
	}
	if ip.IsLinkLocalUnicast() {
		return true
	}
	if ip.IsPrivate() {
		return true
	}
	if v4 := ip.To4(); v4 != nil {
		if v4[0] == 0 {
			return true
		}
	}
	if ip.IsUnspecified() {
		return true
	}
	return false
}

type Dispatcher struct {
	cfg        *config.Config
	httpClient *http.Client
}

func NewDispatcher(cfg *config.Config) *Dispatcher {
	return &Dispatcher{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (d *Dispatcher) getEndpointURL(endpointPath string) string {
	rawURL := d.cfg.ScraperServiceURL
	if rawURL == "" {
		return ""
	}
	if strings.HasSuffix(rawURL, "/api/v1/scrape-tasks") {
		return strings.TrimSuffix(rawURL, "/api/v1/scrape-tasks") + endpointPath
	}
	if strings.HasSuffix(rawURL, "/scrape-tasks") {
		return strings.TrimSuffix(rawURL, "/scrape-tasks") + endpointPath
	}
	return strings.TrimRight(rawURL, "/") + endpointPath
}

func (d *Dispatcher) sendRequest(ctx context.Context, method, targetURL string, payload interface{}) (*http.Response, error) {
	// Validate URL to prevent SSRF (Server-Side Request Forgery)
	if isBlockedHost(targetURL) {
		return nil, fmt.Errorf("blocked: URL contains disallowed host")
	}

	parsed, err := url.Parse(targetURL)
	if err != nil || parsed.Host == "" {
		return nil, fmt.Errorf("invalid URL syntax: %w", err)
	}

	var bodyReader *bytes.Buffer
	if payload != nil {
		bodyBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
		bodyReader = bytes.NewBuffer(bodyBytes)
	} else {
		bodyReader = bytes.NewBuffer(nil)
	}

	req, err := http.NewRequestWithContext(ctx, method, targetURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	apiKey := d.cfg.ScraperAPIKey
	if apiKey == "" {
		apiKey = d.cfg.WebhookSecretKey
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute http request: %w", err)
	}

	return resp, nil
}

// DispatchTask sends a scrape task request to Scraper Service (POST /api/v1/scrape-tasks)
func (d *Dispatcher) DispatchTask(ctx context.Context, target model.CrawlingTarget, taskID string) (int, error) {
	if d.cfg.ScraperServiceURL == "" {
		return 0, fmt.Errorf("SCRAPER_SERVICE_URL is not configured")
	}

	companyID := ""
	if target.CompanyID != nil {
		companyID = *target.CompanyID
	}

	payload := model.ScrapeTaskPayload{
		TaskID:       taskID,
		TargetID:     target.ID,
		CompanyID:    companyID,
		ClientOrigin: "sovera_b2b_engine",
		SourceType:   mapSourceType(target.SourceType),
		TargetURL:    target.TargetURL,
		CallbackURL:  d.cfg.WebhookURL,
		Config: &model.ScrapeTaskConfig{
			RenderJS:      false,
			BypassAntiBot: true,
			MaxPages:      50,
		},
	}

	resp, err := d.sendRequest(ctx, http.MethodPost, d.cfg.ScraperServiceURL, payload)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return resp.StatusCode, fmt.Errorf("scraper service returned non-2xx status: %d", resp.StatusCode)
	}

	return resp.StatusCode, nil
}

// DispatchDiscovery sends a search & discovery task request (POST /api/v1/discovery)
func (d *Dispatcher) DispatchDiscovery(ctx context.Context, payload model.DiscoveryTaskPayload) (int, error) {
	endpoint := d.getEndpointURL("/api/v1/discovery")
	if endpoint == "" {
		return 0, fmt.Errorf("SCRAPER_SERVICE_URL is not configured")
	}

	if payload.ClientOrigin == "" {
		payload.ClientOrigin = "sovera_b2b_engine"
	}
	if payload.CallbackURL == "" {
		cb := d.cfg.WebhookURL
		if idx := strings.Index(cb, "?"); idx != -1 {
			cb = cb[:idx]
		}
		if cb == "" {
			cb = "http://localhost:4000/api/v1/webhooks/crawler"
		}
		payload.CallbackURL = cb
	}

	resp, err := d.sendRequest(ctx, http.MethodPost, endpoint, payload)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return resp.StatusCode, fmt.Errorf("scraper service discovery returned status %d", resp.StatusCode)
	}

	return resp.StatusCode, nil
}

// DispatchCrawlJob sends a recursive crawl job request (POST /api/v1/crawl-jobs)
func (d *Dispatcher) DispatchCrawlJob(ctx context.Context, payload model.CrawlJobPayload) (int, error) {
	endpoint := d.getEndpointURL("/api/v1/crawl-jobs")
	if endpoint == "" {
		return 0, fmt.Errorf("SCRAPER_SERVICE_URL is not configured")
	}

	if payload.ClientOrigin == "" {
		payload.ClientOrigin = "sovera_b2b_engine"
	}
	if payload.CallbackURL == "" {
		payload.CallbackURL = d.cfg.WebhookURL
	}

	resp, err := d.sendRequest(ctx, http.MethodPost, endpoint, payload)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return resp.StatusCode, fmt.Errorf("scraper service crawl-job returned status %d", resp.StatusCode)
	}

	return resp.StatusCode, nil
}

// InspectBatch checks page hashes synchronously via POST /api/v1/inspect-batch
func (d *Dispatcher) InspectBatch(ctx context.Context, payload model.InspectBatchPayload) (*model.InspectBatchResponse, int, error) {
	endpoint := d.getEndpointURL("/api/v1/inspect-batch")
	if endpoint == "" {
		return nil, 0, fmt.Errorf("SCRAPER_SERVICE_URL is not configured")
	}

	resp, err := d.sendRequest(ctx, http.MethodPost, endpoint, payload)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, fmt.Errorf("scraper service inspect-batch returned status %d", resp.StatusCode)
	}

	var result model.InspectBatchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to decode inspect-batch response: %w", err)
	}

	return &result, resp.StatusCode, nil
}

// InspectDocument checks document metadata & ETag synchronously via POST /api/v1/documents/inspect
func (d *Dispatcher) InspectDocument(ctx context.Context, payload model.DocumentInspectPayload) (*model.DocumentInspectResponse, int, error) {
	endpoint := d.getEndpointURL("/api/v1/documents/inspect")
	if endpoint == "" {
		return nil, 0, fmt.Errorf("SCRAPER_SERVICE_URL is not configured")
	}

	resp, err := d.sendRequest(ctx, http.MethodPost, endpoint, payload)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, fmt.Errorf("scraper service documents/inspect returned status %d", resp.StatusCode)
	}

	var result model.DocumentInspectResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to decode documents/inspect response: %w", err)
	}

	return &result, resp.StatusCode, nil
}

// GetTaskStatus queries task status from Scraper Service (GET /api/v1/tasks/{task_id})
func (d *Dispatcher) GetTaskStatus(ctx context.Context, taskID string) (*model.TaskStatusResponse, int, error) {
	endpoint := d.getEndpointURL(fmt.Sprintf("/api/v1/tasks/%s", taskID))
	if endpoint == "" {
		return nil, 0, fmt.Errorf("SCRAPER_SERVICE_URL is not configured")
	}

	resp, err := d.sendRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, fmt.Errorf("scraper service get task status returned status %d", resp.StatusCode)
	}

	var result model.TaskStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to decode task status response: %w", err)
	}

	return &result, resp.StatusCode, nil
}

func mapSourceType(st string) string {
	switch st {
	case "IDX_ANNOUNCEMENT", "CORPORATE_NEWSROOM":
		return "NEWS_ARTICLE"
	case "PDF_REPORTS":
		return "PDF_DOCUMENT"
	case "PDF_DOCUMENT", "NEWS_ARTICLE", "NEWS_RSS", "BUMN_PORTAL", "GRANTS_PORTAL", "RAW_WEB", "SOCIAL_POST", "CSR_OPPORTUNITY_SEARCH", "COMPANY_ENRICHMENT", "SEARCH_DISCOVERY":
		return st
	default:
		return "NEWS_ARTICLE"
	}
}
