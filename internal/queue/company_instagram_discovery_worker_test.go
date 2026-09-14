package queue

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCompanyInstagramDiscoveryWorker_HTMLScraping(t *testing.T) {
	// Setup a mock HTTP server representing a corporate website with an Instagram link
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`
			<!DOCTYPE html>
			<html>
			<head><title>PT Pertamina Official Site</title></head>
			<body>
				<h1>Welcome to Pertamina</h1>
				<footer>
					<a href="https://www.instagram.com/pertamina">Follow us on Instagram</a>
				</footer>
			</body>
			</html>
		`))
	}))
	defer server.Close()

	worker := NewCompanyInstagramDiscoveryWorker(nil, "")
	ctx := context.Background()

	target := InstagramDiscoveryTarget{
		ID:      "comp-1",
		Name:    "PT Pertamina (Persero)",
		Website: server.URL,
	}

	discoveredURL, methodUsed, err := worker.DiscoverInstagramForCompany(ctx, target)
	if err != nil {
		t.Fatalf("unexpected error during Instagram discovery: %v", err)
	}

	if methodUsed != "OFFICIAL_WEBSITE_HTML" {
		t.Errorf("expected method OFFICIAL_WEBSITE_HTML, got %s", methodUsed)
	}

	expectedURL := "https://www.instagram.com/pertamina/"
	if discoveredURL != expectedURL {
		t.Errorf("expected discovered URL %s, got %s", expectedURL, discoveredURL)
	}
}
