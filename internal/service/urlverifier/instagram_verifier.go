package urlverifier

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type InstagramEntityType string

const (
	InstagramEntityProfile InstagramEntityType = "PROFILE"
	InstagramEntityPost    InstagramEntityType = "POST"
	InstagramEntityReel    InstagramEntityType = "REEL"
	InstagramEntityStories InstagramEntityType = "STORIES"
	InstagramEntityUnknown InstagramEntityType = "UNKNOWN"
)

type InstagramValidationResult struct {
	IsValid       bool                `json:"is_valid"`
	OriginalURL   string              `json:"original_url"`
	CanonicalURL  string              `json:"canonical_url"`
	CanonicalHandle string            `json:"canonical_handle,omitempty"`
	EntityType    InstagramEntityType `json:"entity_type"`
	HTTPStatus    int                 `json:"http_status"`
	HealthStatus  HealthStatus        `json:"health_status"`
	Reason        string              `json:"reason"`
	VerifiedVia   string              `json:"verified_via"`
}

type InstagramVerifier struct {
	httpClient *http.Client
}

func NewInstagramVerifier() *InstagramVerifier {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &InstagramVerifier{
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

// Regex patterns for Instagram URL paths
var (
	igProfileRegex = regexp.MustCompile(`(?i)^/([a-zA-Z0-9_\.]{1,30})/?$`)
	igPostRegex    = regexp.MustCompile(`(?i)^/p/([^/\?#]+)`)
	igReelRegex    = regexp.MustCompile(`(?i)^/reel/([^/\?#]+)`)
	igStoriesRegex = regexp.MustCompile(`(?i)^/stories/([^/\?#]+)`)
)

func (v *InstagramVerifier) ValidateInstagramURL(ctx context.Context, rawURL string) *InstagramValidationResult {
	res := &InstagramValidationResult{
		OriginalURL:  rawURL,
		CanonicalURL: rawURL,
		EntityType:   InstagramEntityUnknown,
		HealthStatus: StatusDisabledDeadLink,
		VerifiedVia:  "STAGE_1_SYNTAX_PARSER",
	}

	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		res.Reason = "Empty URL provided"
		return res
	}

	if !strings.HasPrefix(trimmed, "http://") && !strings.HasPrefix(trimmed, "https://") {
		trimmed = "https://" + trimmed
	}

	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Host == "" {
		res.Reason = "Invalid URL structure or unparseable host"
		return res
	}

	hostLower := strings.ToLower(parsed.Host)
	isInstagramDomain := hostLower == "instagram.com" || strings.HasSuffix(hostLower, ".instagram.com") ||
		hostLower == "instagr.am" || strings.HasSuffix(hostLower, ".instagr.am")

	if !isInstagramDomain {
		res.Reason = fmt.Sprintf("Domain '%s' is not a valid Instagram domain (*.instagram.com, *.instagr.am)", parsed.Host)
		return res
	}

	path := parsed.Path
	if path == "" {
		path = "/"
	}

	entityType, handle := v.parseEntityAndHandle(path)
	res.EntityType = entityType
	res.CanonicalHandle = handle

	if entityType == InstagramEntityUnknown || handle == "" {
		if path == "/" || path == "/explore" || path == "/explore/" {
			res.Reason = "Generic Instagram homepage or explore link (lacks specific profile handle)"
			res.CanonicalURL = "https://www.instagram.com" + path
			res.HealthStatus = StatusDegraded
			return res
		}
		res.Reason = fmt.Sprintf("Unrecognized Instagram path format: '%s'. Expected /{handle}, /p/{code}, or /reel/{code}.", path)
		return res
	}

	// Form clean canonical Instagram URL
	res.CanonicalURL = v.formatCanonicalURL(entityType, handle)

	// ─── STAGE 2: Native HTTP Probe ───────────────────────────────────────────
	res.VerifiedVia = "STAGE_2_HTTP_PROBE"
	statusCode, finalURL, bodySnippet, err := v.probeInstagram(ctx, res.CanonicalURL)
	res.HTTPStatus = statusCode

	if err != nil {
		res.Reason = fmt.Sprintf("Network connection error probing Instagram: %v", err)
		res.HealthStatus = StatusDegraded
		return res
	}

	if statusCode == 404 {
		res.IsValid = false
		res.HealthStatus = StatusDisabledDeadLink
		res.Reason = fmt.Sprintf("Instagram 404 Page Not Found (Target handle '%s' does not exist)", handle)
		return res
	}

	if statusCode == 410 {
		res.IsValid = false
		res.HealthStatus = StatusDisabledDeadLink
		res.Reason = fmt.Sprintf("Instagram 410 Gone (Target handle '%s' removed/deleted)", handle)
		return res
	}

	// Instagram redirects non-authenticated users or valid profiles to /accounts/login/ or shows login modal
	if strings.Contains(finalURL, "/accounts/login") || strings.Contains(finalURL, "/login") {
		res.IsValid = true
		res.HealthStatus = StatusHealthy
		res.Reason = fmt.Sprintf("Valid Instagram %s URL (Confirmed active via Instagram login redirect: %s)", entityType, handle)
		return res
	}

	if statusCode == 429 {
		res.IsValid = false
		res.HealthStatus = StatusDegraded
		res.Reason = "Instagram returned unexpected HTTP status 429 (Rate-limited temporary)"
		return res
	}

	if statusCode >= 200 && statusCode < 350 {
		if strings.Contains(strings.ToLower(bodySnippet), "sorry, this page isn't available") ||
			strings.Contains(strings.ToLower(bodySnippet), "the link you followed may be broken") {
			res.IsValid = false
			res.HealthStatus = StatusDisabledDeadLink
			res.Reason = fmt.Sprintf("Instagram 404 Page Not Available content detected for handle '%s'", handle)
			return res
		}

		res.IsValid = true
		res.HealthStatus = StatusHealthy
		res.Reason = fmt.Sprintf("Valid & Reachable Instagram %s URL (HTTP %d)", entityType, statusCode)
		return res
	}

	res.Reason = fmt.Sprintf("Instagram returned HTTP status %d", statusCode)
	res.HealthStatus = StatusDegraded
	return res
}

func (v *InstagramVerifier) parseEntityAndHandle(path string) (InstagramEntityType, string) {
	cleanPath := strings.TrimRight(path, "/")
	if cleanPath == "" {
		return InstagramEntityUnknown, ""
	}

	// Reserved non-handle subpaths
	reserved := map[string]bool{
		"explore": true, "direct": true, "reels": true, "stories": true,
		"accounts": true, "developer": true, "about": true, "legal": true,
		"press": true, "api": true, "graphql": true, "static": true,
	}

	if matches := igPostRegex.FindStringSubmatch(path); len(matches) > 1 {
		return InstagramEntityPost, matches[1]
	}
	if matches := igReelRegex.FindStringSubmatch(path); len(matches) > 1 {
		return InstagramEntityReel, matches[1]
	}
	if matches := igStoriesRegex.FindStringSubmatch(path); len(matches) > 1 {
		return InstagramEntityStories, matches[1]
	}

	if matches := igProfileRegex.FindStringSubmatch(cleanPath); len(matches) > 1 {
		handle := strings.ToLower(matches[1])
		if !reserved[handle] {
			return InstagramEntityProfile, handle
		}
	}

	return InstagramEntityUnknown, ""
}

func (v *InstagramVerifier) formatCanonicalURL(entityType InstagramEntityType, handle string) string {
	switch entityType {
	case InstagramEntityProfile:
		return fmt.Sprintf("https://www.instagram.com/%s/", handle)
	case InstagramEntityPost:
		return fmt.Sprintf("https://www.instagram.com/p/%s/", handle)
	case InstagramEntityReel:
		return fmt.Sprintf("https://www.instagram.com/reel/%s/", handle)
	case InstagramEntityStories:
		return fmt.Sprintf("https://www.instagram.com/stories/%s/", handle)
	default:
		return fmt.Sprintf("https://www.instagram.com/%s/", handle)
	}
}

func (v *InstagramVerifier) probeInstagram(ctx context.Context, targetURL string) (int, string, string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return 0, "", "", err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return 0, "", "", err
	}
	defer resp.Body.Close()

	finalURL := targetURL
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}

	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024*32))
	bodySnippet := string(bodyBytes)

	return resp.StatusCode, finalURL, bodySnippet, nil
}
