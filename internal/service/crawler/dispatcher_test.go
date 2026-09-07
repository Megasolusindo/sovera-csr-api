package crawler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"sovera-core-api/internal/config"
	"sovera-core-api/internal/model"
)

func TestDispatcher_DispatchTask_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status":"ACCEPTED"}`))
	}))
	defer ts.Close()

	cfg := &config.Config{
		ScraperServiceURL: ts.URL,
		WebhookURL:        "http://localhost:4000/api/v1/webhooks/crawler",
		WebhookSecretKey:  "secret_key_123",
	}

	dispatcher := NewDispatcher(cfg)
	target := model.CrawlingTarget{
		ID:         "target_12345678",
		SourceType: "PDF_DOCUMENT",
		TargetURL:  "https://example.com/report.pdf",
	}

	statusCode, err := dispatcher.DispatchTask(context.Background(), target, "task_123")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if statusCode != http.StatusAccepted {
		t.Errorf("expected status code 202, got %d", statusCode)
	}
}

func TestMapSourceType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"IDX_ANNOUNCEMENT", "NEWS_ARTICLE"},
		{"CORPORATE_NEWSROOM", "NEWS_ARTICLE"},
		{"PDF_REPORTS", "PDF_DOCUMENT"},
		{"PDF_DOCUMENT", "PDF_DOCUMENT"},
		{"CSR_OPPORTUNITY_SEARCH", "CSR_OPPORTUNITY_SEARCH"},
		{"COMPANY_ENRICHMENT", "COMPANY_ENRICHMENT"},
		{"SEARCH_DISCOVERY", "SEARCH_DISCOVERY"},
		{"UNKNOWN_TYPE", "NEWS_ARTICLE"},
	}

	for _, tt := range tests {
		got := mapSourceType(tt.input)
		if got != tt.expected {
			t.Errorf("mapSourceType(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

func TestDispatcher_V2Endpoints(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/discovery":
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"task_id":"disc_123","status":"ACCEPTED"}`))
		case "/api/v1/crawl-jobs":
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"task_id":"crawl_123","status":"ACCEPTED"}`))
		case "/api/v1/inspect-batch":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"checked_at":"2026-09-04T00:00:00Z","total_checked":1,"results":[{"reference_id":"ref_1","target_url":"https://abc.com","http_status_code":200,"is_modified":false,"current_hash":"hash1"}]}`))
		case "/api/v1/documents/inspect":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"file_url":"https://abc.com/doc.pdf","http_status_code":200,"is_modified":false,"etag":"\"etag1\"","content_length":1024}`))
		case "/api/v1/tasks/task_status_999":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"task_id":"task_status_999","status":"COMPLETED","http_status_code":200,"execution_time_ms":1500,"content_hash":"hash_xyz"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	cfg := &config.Config{
		ScraperServiceURL: ts.URL + "/api/v1/scrape-tasks",
		WebhookURL:        "http://localhost:4000/api/v1/webhooks/crawler",
		WebhookSecretKey:  "secret_key_123",
	}

	dispatcher := NewDispatcher(cfg)
	ctx := context.Background()

	// 1. DispatchDiscovery
	discStatus, err := dispatcher.DispatchDiscovery(ctx, model.DiscoveryTaskPayload{
		TaskID: "disc_123",
		Query:  "perusahaan CSR",
	})
	if err != nil || discStatus != http.StatusAccepted {
		t.Errorf("DispatchDiscovery failed: status=%d, err=%v", discStatus, err)
	}

	// 2. DispatchCrawlJob
	crawlStatus, err := dispatcher.DispatchCrawlJob(ctx, model.CrawlJobPayload{
		TaskID:  "crawl_123",
		BaseURL: "https://abc.com",
	})
	if err != nil || crawlStatus != http.StatusAccepted {
		t.Errorf("DispatchCrawlJob failed: status=%d, err=%v", crawlStatus, err)
	}

	// 3. InspectBatch
	inspectBatchRes, batchStatus, err := dispatcher.InspectBatch(ctx, model.InspectBatchPayload{
		Items: []model.InspectBatchItemRequest{
			{ReferenceID: "ref_1", TargetURL: "https://abc.com", KnownHash: "hash1"},
		},
	})
	if err != nil || batchStatus != http.StatusOK || inspectBatchRes.TotalChecked != 1 {
		t.Errorf("InspectBatch failed: res=%v, status=%d, err=%v", inspectBatchRes, batchStatus, err)
	}

	// 4. InspectDocument
	docInspectRes, docStatus, err := dispatcher.InspectDocument(ctx, model.DocumentInspectPayload{
		FileURL: "https://abc.com/doc.pdf",
	})
	if err != nil || docStatus != http.StatusOK || docInspectRes.ContentLength != 1024 {
		t.Errorf("InspectDocument failed: res=%v, status=%d, err=%v", docInspectRes, docStatus, err)
	}

	// 5. GetTaskStatus
	taskStatusRes, taskStatusHttp, err := dispatcher.GetTaskStatus(ctx, "task_status_999")
	if err != nil || taskStatusHttp != http.StatusOK || taskStatusRes.Status != "COMPLETED" {
		t.Errorf("GetTaskStatus failed: res=%v, status=%d, err=%v", taskStatusRes, taskStatusHttp, err)
	}
}
