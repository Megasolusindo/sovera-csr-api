package urlverifier

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// URLVerifier checks if a website URL is valid, reachable, and returns an HTTP 2xx/3xx status code.
type URLVerifier struct {
	httpClient *http.Client
}

// DefaultVerifier is a singleton instance with 5-second timeout.
var DefaultVerifier = NewURLVerifier(5 * time.Second)

func NewURLVerifier(timeout time.Duration) *URLVerifier {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &URLVerifier{
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, // Allow self-signed or internal SSL certs for enterprise company websites
				},
				DialContext: (&net.Dialer{
					Timeout:   3 * time.Second,
					KeepAlive: 10 * time.Second,
				}).DialContext,
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return fmt.Errorf("stopped after 5 redirects")
				}
				return nil
			},
		},
	}
}

// NormalizeURL cleans up raw URL input, ensuring scheme http/https and trimming whitespace/slashes
func NormalizeURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		raw = "https://" + raw
	}
	return strings.TrimSuffix(raw, "/")
}

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
	ips, err := net.LookupIP(hostname)
	if err != nil {
		return true
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
	// Block 0.0.0.0/8 and ::1
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

// VerifyWebsite checks whether targetURL is formatted correctly and reachable over HTTP/HTTPS (2xx/3xx response).
func (v *URLVerifier) VerifyWebsite(ctx context.Context, targetURL string) (bool, string, error) {
	normalized := NormalizeURL(targetURL)
	if normalized == "" {
		return false, "", fmt.Errorf("empty URL")
	}

	parsed, err := url.Parse(normalized)
	if err != nil || parsed.Host == "" {
		return false, normalized, fmt.Errorf("invalid URL syntax: %w", err)
	}

	// Reject known non-routable / local loopbacks and cloud metadata URLs
	if isBlockedHost(parsed.Hostname()) {
		return false, normalized, fmt.Errorf("invalid corporate website host: %s", parsed.Host)
	}

	// Try HEAD request first for efficiency
	req, err := http.NewRequestWithContext(ctx, "HEAD", normalized, nil)
	if err != nil {
		return false, normalized, fmt.Errorf("failed to create HEAD request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) SoveraBot/2.0 (+https://sovera.id)")

	resp, err := v.httpClient.Do(req)
	if err != nil || resp.StatusCode >= 400 {
		// Fallback to GET request in case server blocks HEAD method (e.g. 405 Method Not Allowed or 403 Forbidden on HEAD)
		reqGET, errGET := http.NewRequestWithContext(ctx, "GET", normalized, nil)
		if errGET != nil {
			return false, normalized, fmt.Errorf("failed to create GET request: %w", errGET)
		}
		reqGET.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) SoveraBot/2.0 (+https://sovera.id)")

		respGET, errGET := v.httpClient.Do(reqGET)
		if errGET != nil {
			return false, normalized, fmt.Errorf("website cannot be reached (connection refused / timeout / DNS failure): %w", errGET)
		}
		defer respGET.Body.Close()

		if respGET.StatusCode >= 200 && respGET.StatusCode < 400 {
			return true, normalized, nil
		}
		return false, normalized, fmt.Errorf("website returned HTTP status %d", respGET.StatusCode)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return true, normalized, nil
	}

	return false, normalized, fmt.Errorf("website returned HTTP status %d", resp.StatusCode)
}
