package ai

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	ErrSSRFForbidden      = errors.New("SSRF protection: access to private/internal IP ranges forbidden")
	ErrInvalidURL         = errors.New("invalid or unparseable target URL")
	ErrEgressQueryBlocked = errors.New("egress security: query string contains potential sensitive parameters")
)

// SecureFetcher wraps HTTP fetching with SSRF protection and query string sanitization for Research Agent.
type SecureFetcher struct {
	httpClient *http.Client
}

func NewSecureFetcher(timeout time.Duration) *SecureFetcher {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &SecureFetcher{
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// SanitizeAndValidateURL checks URL against SSRF rules and removes potentially sensitive query parameters
func SanitizeAndValidateURL(rawURL string) (*url.URL, error) {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil, ErrInvalidURL
	}

	// Must be HTTP or HTTPS
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("scheme '%s' not allowed; only http and https allowed", u.Scheme)
	}

	hostname := strings.ToLower(u.Hostname())

	// SSRF Check: Block localhost, 127.0.0.1, 10.x.x.x, 172.16.x.x, 192.168.x.x, metadata IPs
	if isPrivateHost(hostname) {
		return nil, ErrSSRFForbidden
	}

	// Egress Sanitization: Strip sensitive query parameters (e.g. token, secret, auth, tenant_id, key)
	query := u.Query()
	modified := false
	sensitiveKeys := []string{"token", "secret", "auth", "api_key", "apikey", "tenant_id", "jwt", "pass", "password"}

	for key := range query {
		lowerKey := strings.ToLower(key)
		for _, sens := range sensitiveKeys {
			if strings.Contains(lowerKey, sens) {
				query.Del(key)
				modified = true
				break
			}
		}
	}

	if modified {
		u.RawQuery = query.Encode()
	}

	return u, nil
}

func isPrivateHost(host string) bool {
	if host == "localhost" || host == "127.0.0.1" || host == "::1" || host == "0.0.0.0" {
		return true
	}
	if strings.HasPrefix(host, "10.") || strings.HasPrefix(host, "192.168.") || strings.HasPrefix(host, "169.254.") {
		return true
	}
	if strings.HasPrefix(host, "172.16.") || strings.HasPrefix(host, "172.17.") || strings.HasPrefix(host, "172.18.") || strings.HasPrefix(host, "172.19.") || strings.HasPrefix(host, "172.20.") || strings.HasPrefix(host, "172.30.") || strings.HasPrefix(host, "172.31.") {
		return true
	}
	if strings.HasSuffix(host, ".internal") || strings.HasSuffix(host, ".local") {
		return true
	}
	return false
}

// FetchUntrustedContent safely fetches web content for the GLOBAL Research Agent sandbox
func (f *SecureFetcher) FetchUntrustedContent(ctx context.Context, targetURL string) (string, error) {
	sanitizedURL, err := SanitizeAndValidateURL(targetURL)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sanitizedURL.String(), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) CSRmatics-ResearchBot/1.2")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return "", fmt.Errorf("http error status code: %d", resp.StatusCode)
	}

	// Limit read size to 2MB to prevent zip bombs / memory exhaustion
	limitReader := io.LimitReader(resp.Body, 2*1024*1024)
	bodyBytes, err := io.ReadAll(limitReader)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(bodyBytes), nil
}
