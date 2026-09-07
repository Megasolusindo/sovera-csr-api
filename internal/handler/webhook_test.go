package handler

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"sovera-core-api/internal/service/normalizer"
)

func TestHandleCrawlerWebhook_SearchDiscovery(t *testing.T) {
	app := fiber.New()
	norm := normalizer.NewNormalizer()
	handler := NewWebhookHandler(nil, nil, norm)

	app.Post("/webhooks/crawler", handler.HandleCrawlerWebhook)

	payload := CrawlerPayload{
		TaskID:          "disc_job_881920",
		Status:          "COMPLETED",
		HTTPStatusCode:  200,
		SourceType:      "SEARCH_DISCOVERY",
		Query:           "perusahaan CSR Jawa Barat",
		DiscoveredItems: []DiscoveredItem{
			{
				Title:   "Program TJSL PT ABC",
				URL:     "https://abc.co.id/tjsl",
				Snippet: "Program keberlanjutan 2026...",
				Rank:    1,
			},
		},
	}

	bodyBytes, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/webhooks/crawler", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to execute request: %v", err)
	}

	if resp.StatusCode != fiber.StatusAccepted {
		t.Errorf("expected HTTP 202 Accepted, got %d", resp.StatusCode)
	}
}

func TestHandleCrawlerWebhook_CompanyEnrichment(t *testing.T) {
	app := fiber.New()
	norm := normalizer.NewNormalizer()
	handler := NewWebhookHandler(nil, nil, norm)

	app.Post("/webhooks/crawler", handler.HandleCrawlerWebhook)

	payload := CrawlerPayload{
		TaskID:          "job_enrich_telkom_3391",
		Status:          "COMPLETED",
		HTTPStatusCode:  200,
		SourceType:      "COMPANY_ENRICHMENT",
		SourceURL:       "https://www.telkom.co.id/sites/sustainability/id_ID/page/csr-1127",
		RawText:         "Kontak TJSL Telkom: tjsl@telkom.co.id",
		MarkdownContent: "## Profil TJSL Telkom",
	}

	bodyBytes, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/webhooks/crawler", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to execute request: %v", err)
	}

	if resp.StatusCode != fiber.StatusAccepted {
		t.Errorf("expected HTTP 202 Accepted, got %d", resp.StatusCode)
	}
}
