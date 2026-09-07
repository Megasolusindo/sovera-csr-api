package urlverifier

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type HealthStatus string

const (
	StatusHealthy          HealthStatus = "HEALTHY"
	StatusDegraded         HealthStatus = "DEGRADED"
	StatusDisabledDeadLink HealthStatus = "DISABLED_DEAD_LINK"
)

type ValidationResult struct {
	IsValid      bool         `json:"is_valid"`
	OriginalURL  string       `json:"original_url"`
	FinalURL     string       `json:"final_url"`
	FallbackURL  string       `json:"fallback_url"`
	HTTPStatus   int          `json:"http_status"`
	HealthStatus HealthStatus `json:"health_status"`
	ErrorMsg     string       `json:"error_msg,omitempty"`
}

type URLVerifier struct {
	httpClient *http.Client
}

func NewURLVerifier() *URLVerifier {
	// Custom HTTP client with 6s timeout & TLS skip for verification robustness
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &URLVerifier{
		httpClient: &http.Client{
			Transport: tr,
			Timeout:   6 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
	}
}

// VerifyAndResolve checks HTTP accessibility of a URL, follows redirects,
// and produces a smart fallback URL if the primary URL returns 404 or fails.
func (v *URLVerifier) VerifyAndResolve(ctx context.Context, rawURL string) *ValidationResult {
	res := &ValidationResult{
		OriginalURL:  rawURL,
		FinalURL:     rawURL,
		HealthStatus: StatusDisabledDeadLink,
	}

	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" || (!strings.HasPrefix(trimmed, "http://") && !strings.HasPrefix(trimmed, "https://")) {
		res.ErrorMsg = "Invalid URL protocol or empty URL"
		return res
	}

	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Host == "" {
		res.ErrorMsg = "Failed to parse URL host"
		return res
	}

	// Step 1: Probe Primary URL via HEAD, fallback to GET
	statusCode, finalURL, err := v.probeURL(ctx, trimmed)
	res.HTTPStatus = statusCode
	if finalURL != "" {
		res.FinalURL = finalURL
	}

	if err == nil && statusCode >= 200 && statusCode < 400 {
		res.IsValid = true
		res.HealthStatus = StatusHealthy
		res.FallbackURL = res.FinalURL
		return res
	}

	// Step 2: Primary URL failed (404/500/DNS error). Generate Smart Fallback URL
	res.IsValid = false
	if statusCode == 404 || statusCode == 410 {
		res.HealthStatus = StatusDisabledDeadLink
		if err != nil {
			res.ErrorMsg = err.Error()
		} else {
			res.ErrorMsg = "HTTP 404 Not Found"
		}
	} else {
		res.HealthStatus = StatusDegraded
		if err != nil {
			res.ErrorMsg = err.Error()
		} else {
			res.ErrorMsg = "Server or network error"
		}
	}

	// Try Resolving Base Domain / Canonical Corporate Fallback
	fallbackCandidate := v.findBestFallback(ctx, parsed)
	res.FallbackURL = fallbackCandidate

	return res
}

func (v *URLVerifier) probeURL(ctx context.Context, targetURL string) (int, string, error) {
	req, err := http.NewRequestWithContext(ctx, "HEAD", targetURL, nil)
	if err != nil {
		return 0, targetURL, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := v.httpClient.Do(req)
	if err != nil || resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode >= 400 {
		// Fallback to GET request if HEAD is rejected or returns error
		getReq, getErr := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
		if getErr != nil {
			return 0, targetURL, getErr
		}
		getReq.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
		
		getResp, err2 := v.httpClient.Do(getReq)
		if err2 != nil {
			return 0, targetURL, err2
		}
		defer getResp.Body.Close()
		return getResp.StatusCode, getResp.Request.URL.String(), nil
	}
	defer resp.Body.Close()

	return resp.StatusCode, resp.Request.URL.String(), nil
}

func (v *URLVerifier) findBestFallback(ctx context.Context, parsed *url.URL) string {
	schemeHost := parsed.Scheme + "://" + parsed.Host

	// Common corporate CSR/sustainability path candidates
	candidates := []string{
		schemeHost + "/id/tentang-bca/csr/bakti-bca",
		schemeHost + "/sites/sustainability/id_ID/page/csr-1127",
		schemeHost + "/id/perusahaan-tercatat/laporan-keuangan-dan-tahunan/",
		schemeHost + "/id/perusahaan-tercatat/profil-perusahaan-tercatat/",
		schemeHost + "/aktivitas/",
		schemeHost + "/Sustainability",
		schemeHost + "/csr",
		schemeHost + "/sustainability",
		schemeHost + "/id",
		schemeHost,
	}

	for _, cand := range candidates {
		code, finalURL, err := v.probeURL(ctx, cand)
		if err == nil && code >= 200 && code < 400 {
			if finalURL != "" {
				return finalURL
			}
			return cand
		}
	}

	return schemeHost
}
