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

type FacebookEntityType string

const (
	FacebookEntityProfile FacebookEntityType = "PROFILE"
	FacebookEntityPage    FacebookEntityType = "PAGE"
	FacebookEntityGroup   FacebookEntityType = "GROUP"
	FacebookEntityUnknown FacebookEntityType = "UNKNOWN"
)

type FacebookValidationResult struct {
	IsValid         bool               `json:"is_valid"`
	OriginalURL     string             `json:"original_url"`
	CanonicalURL    string             `json:"canonical_url"`
	CanonicalHandle string             `json:"canonical_handle,omitempty"`
	EntityType      FacebookEntityType `json:"entity_type"`
	HTTPStatus      int                `json:"http_status"`
	HealthStatus    HealthStatus       `json:"health_status"`
	Reason          string             `json:"reason"`
	VerifiedVia     string             `json:"verified_via"`
}

type FacebookVerifier struct {
	httpClient *http.Client
}

func NewFacebookVerifier() *FacebookVerifier {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &FacebookVerifier{
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

var (
	fbProfileRegex = regexp.MustCompile(`(?i)^/([a-zA-Z0-9_\.-]{1,60})/?$`)
	fbPageGroupRegex = regexp.MustCompile(`(?i)^/(pages|groups|people)/([^/\?#]+)`)
)

func (v *FacebookVerifier) ValidateFacebookURL(ctx context.Context, rawURL string) *FacebookValidationResult {
	res := &FacebookValidationResult{
		OriginalURL:  rawURL,
		CanonicalURL: rawURL,
		EntityType:   FacebookEntityUnknown,
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
	isFacebookDomain := hostLower == "facebook.com" || strings.HasSuffix(hostLower, ".facebook.com") ||
		hostLower == "fb.com" || strings.HasSuffix(hostLower, ".fb.com") ||
		hostLower == "fb.me" || strings.HasSuffix(hostLower, ".fb.me")

	if !isFacebookDomain {
		res.Reason = fmt.Sprintf("Domain '%s' is not a valid Facebook domain (*.facebook.com, *.fb.me)", parsed.Host)
		return res
	}

	path := parsed.Path
	if path == "" || path == "/" {
		res.Reason = "Generic Facebook homepage link (lacks specific profile/page handle)"
		res.CanonicalURL = "https://www.facebook.com"
		res.HealthStatus = StatusDegraded
		return res
	}

	entityType, handle := v.parseEntityAndHandle(path)
	res.EntityType = entityType
	res.CanonicalHandle = handle

	if entityType == FacebookEntityUnknown || handle == "" {
		res.Reason = fmt.Sprintf("Unrecognized Facebook path format: '%s'", path)
		return res
	}

	res.CanonicalURL = fmt.Sprintf("https://www.facebook.com/%s/", handle)

	// Stage 2: Native HTTP Probe
	res.VerifiedVia = "STAGE_2_HTTP_PROBE"
	statusCode, _, bodySnippet, err := v.probeFacebook(ctx, res.CanonicalURL)
	res.HTTPStatus = statusCode

	if err != nil {
		res.Reason = fmt.Sprintf("Network connection error probing Facebook: %v", err)
		res.HealthStatus = StatusDegraded
		return res
	}

	if statusCode == 404 || statusCode == 410 {
		res.IsValid = false
		res.HealthStatus = StatusDisabledDeadLink
		res.Reason = fmt.Sprintf("Facebook 404 Page Not Found (Target '%s' does not exist)", handle)
		return res
	}

	if statusCode >= 200 && statusCode < 400 {
		lowerBody := strings.ToLower(bodySnippet)
		if strings.Contains(lowerBody, "this content isn't available right now") ||
			strings.Contains(lowerBody, "page not found") ||
			strings.Contains(lowerBody, "link you followed may be broken") {
			res.IsValid = false
			res.HealthStatus = StatusDisabledDeadLink
			res.Reason = fmt.Sprintf("Facebook content unavailable detected for '%s'", handle)
			return res
		}

		res.IsValid = true
		res.HealthStatus = StatusHealthy
		res.Reason = fmt.Sprintf("Valid & Reachable Facebook %s URL (HTTP %d)", entityType, statusCode)
		return res
	}

	res.Reason = fmt.Sprintf("Facebook returned HTTP status %d", statusCode)
	res.HealthStatus = StatusDegraded
	return res
}

func (v *FacebookVerifier) parseEntityAndHandle(path string) (FacebookEntityType, string) {
	cleanPath := strings.TrimRight(path, "/")
	if cleanPath == "" {
		return FacebookEntityUnknown, ""
	}

	reserved := map[string]bool{
		"events": true, "groups": true, "marketplace": true, "gaming": true,
		"watch": true, "stories": true, "login": true, "recover": true,
		"help": true, "privacy": true, "terms": true, "ads": true,
	}

	if matches := fbPageGroupRegex.FindStringSubmatch(path); len(matches) > 2 {
		kind := strings.ToLower(matches[1])
		if kind == "groups" {
			return FacebookEntityGroup, matches[2]
		}
		return FacebookEntityPage, matches[2]
	}

	if matches := fbProfileRegex.FindStringSubmatch(cleanPath); len(matches) > 1 {
		handle := strings.ToLower(matches[1])
		if !reserved[handle] {
			return FacebookEntityPage, handle
		}
	}

	return FacebookEntityUnknown, ""
}

func (v *FacebookVerifier) probeFacebook(ctx context.Context, targetURL string) (int, string, string, error) {
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
	return resp.StatusCode, finalURL, string(bodyBytes), nil
}
