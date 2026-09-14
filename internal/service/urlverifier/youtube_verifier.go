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

type YoutubeEntityType string

const (
	YoutubeEntityChannel YoutubeEntityType = "CHANNEL"
	YoutubeEntityUser    YoutubeEntityType = "USER"
	YoutubeEntityVideo   YoutubeEntityType = "VIDEO"
	YoutubeEntityUnknown YoutubeEntityType = "UNKNOWN"
)

type YoutubeValidationResult struct {
	IsValid         bool              `json:"is_valid"`
	OriginalURL     string            `json:"original_url"`
	CanonicalURL    string            `json:"canonical_url"`
	CanonicalHandle string            `json:"canonical_handle,omitempty"`
	EntityType      YoutubeEntityType `json:"entity_type"`
	HTTPStatus      int               `json:"http_status"`
	HealthStatus    HealthStatus      `json:"health_status"`
	Reason          string            `json:"reason"`
	VerifiedVia     string            `json:"verified_via"`
}

type YoutubeVerifier struct {
	httpClient *http.Client
}

func NewYoutubeVerifier() *YoutubeVerifier {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &YoutubeVerifier{
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
	ytHandleRegex  = regexp.MustCompile(`(?i)^/@([a-zA-Z0-9_\.-]{1,60})/?$`)
	ytChannelRegex = regexp.MustCompile(`(?i)^/channel/([a-zA-Z0-9_-]+)`)
	ytCustomRegex  = regexp.MustCompile(`(?i)^/c/([a-zA-Z0-9_-]+)`)
	ytUserRegex    = regexp.MustCompile(`(?i)^/user/([a-zA-Z0-9_-]+)`)
	ytWatchRegex   = regexp.MustCompile(`(?i)^/watch\?v=([a-zA-Z0-9_-]+)`)
	youtuBeRegex   = regexp.MustCompile(`(?i)^/([a-zA-Z0-9_-]+)`)
)

func (v *YoutubeVerifier) ValidateYoutubeURL(ctx context.Context, rawURL string) *YoutubeValidationResult {
	res := &YoutubeValidationResult{
		OriginalURL:  rawURL,
		CanonicalURL: rawURL,
		EntityType:   YoutubeEntityUnknown,
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
	isYoutubeDomain := hostLower == "youtube.com" || strings.HasSuffix(hostLower, ".youtube.com") ||
		hostLower == "youtu.be" || strings.HasSuffix(hostLower, ".youtu.be")

	if !isYoutubeDomain {
		res.Reason = fmt.Sprintf("Domain '%s' is not a valid YouTube domain (*.youtube.com, *.youtu.be)", parsed.Host)
		return res
	}

	path := parsed.Path
	if path == "" || path == "/" {
		res.Reason = "Generic YouTube homepage link (lacks specific channel or video handle)"
		res.CanonicalURL = "https://www.youtube.com"
		res.HealthStatus = StatusDegraded
		return res
	}

	entityType, handle := v.parseEntityAndHandle(hostLower, path, parsed.RawQuery)
	res.EntityType = entityType
	res.CanonicalHandle = handle

	if entityType == YoutubeEntityUnknown || handle == "" {
		res.Reason = fmt.Sprintf("Unrecognized YouTube path format: '%s'", path)
		return res
	}

	res.CanonicalURL = v.formatCanonicalURL(entityType, handle)

	// Stage 2: Native HTTP Probe
	res.VerifiedVia = "STAGE_2_HTTP_PROBE"
	statusCode, _, bodySnippet, err := v.probeYoutube(ctx, res.CanonicalURL)
	res.HTTPStatus = statusCode

	if err != nil {
		res.Reason = fmt.Sprintf("Network connection error probing YouTube: %v", err)
		res.HealthStatus = StatusDegraded
		return res
	}

	if statusCode == 404 || statusCode == 410 {
		res.IsValid = false
		res.HealthStatus = StatusDisabledDeadLink
		res.Reason = fmt.Sprintf("YouTube 404 Page Not Found (Target '%s' does not exist)", handle)
		return res
	}

	if statusCode >= 200 && statusCode < 400 {
		lowerBody := strings.ToLower(bodySnippet)
		if strings.Contains(lowerBody, "this page isn't available") ||
			strings.Contains(lowerBody, "this account has been terminated") ||
			strings.Contains(lowerBody, "channel does not exist") {
			res.IsValid = false
			res.HealthStatus = StatusDisabledDeadLink
			res.Reason = fmt.Sprintf("YouTube content unavailable detected for handle '%s'", handle)
			return res
		}

		res.IsValid = true
		res.HealthStatus = StatusHealthy
		res.Reason = fmt.Sprintf("Valid & Reachable YouTube %s URL (HTTP %d)", entityType, statusCode)
		return res
	}

	res.Reason = fmt.Sprintf("YouTube returned HTTP status %d", statusCode)
	res.HealthStatus = StatusDegraded
	return res
}

func (v *YoutubeVerifier) parseEntityAndHandle(host, path, query string) (YoutubeEntityType, string) {
	if strings.Contains(host, "youtu.be") {
		cleanPath := strings.Trim(path, "/")
		if cleanPath != "" {
			return YoutubeEntityVideo, cleanPath
		}
	}

	fullPathQuery := path
	if query != "" {
		fullPathQuery += "?" + query
	}

	if matches := ytWatchRegex.FindStringSubmatch(fullPathQuery); len(matches) > 1 {
		return YoutubeEntityVideo, matches[1]
	}
	if matches := ytHandleRegex.FindStringSubmatch(path); len(matches) > 1 {
		return YoutubeEntityChannel, matches[1]
	}
	if matches := ytChannelRegex.FindStringSubmatch(path); len(matches) > 1 {
		return YoutubeEntityChannel, matches[1]
	}
	if matches := ytCustomRegex.FindStringSubmatch(path); len(matches) > 1 {
		return YoutubeEntityChannel, matches[1]
	}
	if matches := ytUserRegex.FindStringSubmatch(path); len(matches) > 1 {
		return YoutubeEntityUser, matches[1]
	}

	cleanPath := strings.Trim(path, "/")
	if cleanPath != "" && !strings.Contains(cleanPath, "/") {
		return YoutubeEntityChannel, cleanPath
	}

	return YoutubeEntityUnknown, ""
}

func (v *YoutubeVerifier) formatCanonicalURL(entityType YoutubeEntityType, handle string) string {
	switch entityType {
	case YoutubeEntityChannel:
		if strings.HasPrefix(handle, "@") {
			return fmt.Sprintf("https://www.youtube.com/%s", handle)
		}
		if len(handle) == 24 && strings.HasPrefix(handle, "UC") {
			return fmt.Sprintf("https://www.youtube.com/channel/%s", handle)
		}
		return fmt.Sprintf("https://www.youtube.com/@%s", handle)
	case YoutubeEntityVideo:
		return fmt.Sprintf("https://www.youtube.com/watch?v=%s", handle)
	case YoutubeEntityUser:
		return fmt.Sprintf("https://www.youtube.com/user/%s", handle)
	default:
		return fmt.Sprintf("https://www.youtube.com/@%s", handle)
	}
}

func (v *YoutubeVerifier) probeYoutube(ctx context.Context, targetURL string) (int, string, string, error) {
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
