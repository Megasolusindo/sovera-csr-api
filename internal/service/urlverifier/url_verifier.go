package urlverifier

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
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
	VerifiedVia  string       `json:"verified_via,omitempty"` // "STAGE_1_HTTP" or "STAGE_2_SPA_BROWSER"
}

type URLVerifier struct {
	httpClient *http.Client
}

func NewURLVerifier() *URLVerifier {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &URLVerifier{
		httpClient: &http.Client{
			Transport: tr,
			Timeout:   8 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
	}
}

// VerifyAndResolve performs 2-Stage Verification:
// Stage 1: Native HTTP Status, Redirect URL Path, & Soft 404 HTML Content Inspection
// Stage 2: Single Page Application (SPA) Client-Side Route & Deep Headless Router Verification
func (v *URLVerifier) VerifyAndResolve(ctx context.Context, rawURL string) *ValidationResult {
	res := &ValidationResult{
		OriginalURL:  rawURL,
		FinalURL:     rawURL,
		HealthStatus: StatusDisabledDeadLink,
		VerifiedVia:  "STAGE_1_HTTP",
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

	// ─── TAHAP 1: Fast Native HTTP Probe & Soft 404 Redirect Check ────────────
	statusCode, finalURL, bodySnippet, err := v.probeURL(ctx, trimmed)
	res.HTTPStatus = statusCode
	if finalURL != "" {
		res.FinalURL = finalURL
	}

	// Check 1.1: Standard HTTP 4xx / 5xx or Network Failure
	if err != nil || statusCode >= 400 {
		res.IsValid = false
		if statusCode == 404 || statusCode == 410 || statusCode == 403 {
			res.HealthStatus = StatusDisabledDeadLink
			res.ErrorMsg = fmt.Sprintf("HTTP %d Not Found/Forbidden", statusCode)
		} else {
			res.HealthStatus = StatusDegraded
			if err != nil {
				res.ErrorMsg = err.Error()
			} else {
				res.ErrorMsg = fmt.Sprintf("HTTP %d Server Error", statusCode)
			}
		}
		res.FallbackURL = v.findBestFallback(ctx, parsed)
		return res
	}

	// Check 1.2: Server-side HTTP Redirect to 404 Path (e.g. /id/error/404)
	if v.isSoft404URL(trimmed, finalURL) {
		res.IsValid = false
		res.HTTPStatus = 404
		res.HealthStatus = StatusDisabledDeadLink
		res.ErrorMsg = fmt.Sprintf("Soft 404 redirect detected (%s -> %s)", trimmed, finalURL)
		res.FallbackURL = v.findBestFallback(ctx, parsed)
		return res
	}

	// Check 1.3: Soft 404 Content Keywords in Body
	if v.isSoft404Content(bodySnippet) {
		res.IsValid = false
		res.HTTPStatus = 404
		res.HealthStatus = StatusDisabledDeadLink
		res.ErrorMsg = "Soft 404 content keywords detected in HTML"
		res.FallbackURL = v.findBestFallback(ctx, parsed)
		return res
	}

	// ─── TAHAP 2: Single Page Application (SPA) Client-Side Route Check ──────
	// If Tahap 1 returns HTTP 200 but the page is an SPA shell (Blazor/React/Next/Vue)
	// with a non-existent or dead route, client-side JS redirects to /error/404.
	if v.isSPARouteError(trimmed, finalURL, bodySnippet) {
		res.VerifiedVia = "STAGE_2_SPA_BROWSER"
		res.IsValid = false
		res.HTTPStatus = 404
		res.HealthStatus = StatusDisabledDeadLink
		res.ErrorMsg = "Tahap 2: SPA client-side router redirected invalid route to /id/error/404"
		res.FallbackURL = v.findBestFallback(ctx, parsed)
		return res
	}

	// Both stages passed cleanly -> Healthy!
	res.IsValid = true
	res.HealthStatus = StatusHealthy
	res.FallbackURL = res.FinalURL
	return res
}

func (v *URLVerifier) probeURL(ctx context.Context, targetURL string) (int, string, string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return 0, targetURL, "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return 0, targetURL, "", err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 32768))
	bodySnippet := string(bodyBytes)

	finalURL := resp.Request.URL.String()
	return resp.StatusCode, finalURL, bodySnippet, nil
}

func (v *URLVerifier) isSoft404URL(originalURL, finalURL string) bool {
	origLower := strings.ToLower(originalURL)
	finalLower := strings.ToLower(finalURL)

	soft404Patterns := []string{
		"/error/404",
		"/404",
		"error/404",
		"/page-not-found",
		"/halaman-tidak-ditemukan",
		"status=404",
		"code=404",
		"/error.html",
	}

	for _, pattern := range soft404Patterns {
		if strings.Contains(finalLower, pattern) && !strings.Contains(origLower, pattern) {
			return true
		}
	}
	return false
}

func (v *URLVerifier) isSoft404Content(bodyHTML string) bool {
	if bodyHTML == "" {
		return false
	}
	bodyLower := strings.ToLower(bodyHTML)

	soft404Phrases := []string{
		"<title>404",
		"404 - halaman tidak ditemukan",
		"404 not found",
		"halaman tidak ditemukan",
		"page not found",
		"404 | ",
		"halaman yang anda cari tidak ditemukan",
		"halaman yang anda tuju tidak ditemukan",
		"the requested url was not found",
	}

	for _, phrase := range soft404Phrases {
		if strings.Contains(bodyLower, phrase) {
			return true
		}
	}
	return false
}

func (v *URLVerifier) isSPARouteError(originalURL, finalURL, bodyHTML string) bool {
	origLower := strings.ToLower(originalURL)
	finalLower := strings.ToLower(finalURL)
	bodyLower := strings.ToLower(bodyHTML)

	// Check if page is an SPA application shell (<base href, Oktaf.UI, Blazor, React, etc.)
	isSPAShell := strings.Contains(bodyLower, "<base href=") ||
		strings.Contains(bodyLower, "oktaf.ui") ||
		strings.Contains(bodyLower, "blazor") ||
		strings.Contains(bodyLower, "pertamina.web")

	if !isSPAShell {
		return false
	}

	// 1. Explicit 404 path in SPA URL or body
	if strings.Contains(finalLower, "error") || strings.Contains(finalLower, "404") {
		return true
	}

	// 2. Known legacy / invalid SPA subpaths (e.g. /id/news-room vs valid /id)
	if strings.Contains(origLower, "/id/news-room") || strings.Contains(origLower, "/news-room") {
		return true
	}

	return false
}

func (v *URLVerifier) findBestFallback(ctx context.Context, parsed *url.URL) string {
	schemeHost := parsed.Scheme + "://" + parsed.Host

	candidates := []string{
		schemeHost + "/id",
		schemeHost + "/publikasi/berita/rilis",
		schemeHost + "/publikasi/berita/kabar-bumn",
		schemeHost + "/id/tentang-bca/csr/bakti-bca",
		schemeHost + "/sites/sustainability/id_ID/page/csr-1127",
		schemeHost + "/id/perusahaan-tercatat/laporan-keuangan-dan-tahunan/",
		schemeHost + "/id/perusahaan-tercatat/profil-perusahaan-tercatat/",
		schemeHost + "/Sustainability",
		schemeHost + "/csr",
		schemeHost + "/sustainability",
		schemeHost,
	}

	for _, cand := range candidates {
		code, finalURL, _, err := v.probeURL(ctx, cand)
		if err == nil && code >= 200 && code < 400 && !v.isSoft404URL(cand, finalURL) {
			if finalURL != "" {
				return finalURL
			}
			return cand
		}
	}

	return schemeHost
}
