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

type LinkedInEntityType string

const (
	EntityCompany  LinkedInEntityType = "COMPANY"
	EntityProfile  LinkedInEntityType = "PROFILE"
	EntitySchool   LinkedInEntityType = "SCHOOL"
	EntityShowcase LinkedInEntityType = "SHOWCASE"
	EntityJob      LinkedInEntityType = "JOB"
	EntityPost     LinkedInEntityType = "POST"
	EntityPulse    LinkedInEntityType = "PULSE"
	EntityUnknown  LinkedInEntityType = "UNKNOWN"
)

type LinkedInValidationResult struct {
	IsValid       bool               `json:"is_valid"`
	OriginalURL   string             `json:"original_url"`
	CanonicalURL  string             `json:"canonical_url"`
	CanonicalSlug string             `json:"canonical_slug,omitempty"`
	EntityType    LinkedInEntityType `json:"entity_type"`
	HTTPStatus    int                `json:"http_status"`
	HealthStatus  HealthStatus       `json:"health_status"`
	Reason        string             `json:"reason"`
	VerifiedVia   string             `json:"verified_via"`
}

type LinkedInVerifier struct {
	httpClient *http.Client
}

func NewLinkedInVerifier() *LinkedInVerifier {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &LinkedInVerifier{
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

// Regex patterns for LinkedIn URL paths
var (
	companyRegex  = regexp.MustCompile(`(?i)^/company/([^/\?#]+)`)
	schoolRegex   = regexp.MustCompile(`(?i)^/school/([^/\?#]+)`)
	profileRegex  = regexp.MustCompile(`(?i)^/(?:in|pub)/([^/\?#]+)`)
	showcaseRegex = regexp.MustCompile(`(?i)^/showcase/([^/\?#]+)`)
	jobRegex      = regexp.MustCompile(`(?i)^/jobs/view/([^/\?#]+)`)
	postRegex     = regexp.MustCompile(`(?i)^/posts/([^/\?#]+)`)
	pulseRegex    = regexp.MustCompile(`(?i)^/pulse/([^/\?#]+)`)
)

func (v *LinkedInVerifier) ValidateLinkedInURL(ctx context.Context, rawURL string) *LinkedInValidationResult {
	res := &LinkedInValidationResult{
		OriginalURL:  rawURL,
		CanonicalURL: rawURL,
		EntityType:   EntityUnknown,
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
	if !strings.HasSuffix(hostLower, "linkedin.com") && hostLower != "linkedin.com" {
		res.Reason = fmt.Sprintf("Domain '%s' is not a valid LinkedIn domain (*.linkedin.com)", parsed.Host)
		return res
	}

	path := parsed.Path
	if path == "" {
		path = "/"
	}

	// Parse Entity Type and Slug
	entityType, slug := v.parseEntityAndSlug(path)
	res.EntityType = entityType
	res.CanonicalSlug = slug

	if entityType == EntityUnknown || slug == "" {
		// General LinkedIn path check (could be a valid landing page if no slug but valid subpath)
		if path == "/" || path == "/feed" || path == "/feed/" {
			res.Reason = "Generic LinkedIn homepage or feed link (lacks specific entity slug)"
			res.CanonicalURL = "https://www.linkedin.com" + path
			res.HealthStatus = StatusDegraded
			return res
		}
		res.Reason = fmt.Sprintf("Unrecognized LinkedIn path format: '%s'. Expected /company/{slug}, /in/{username}, /school/{slug}, etc.", path)
		return res
	}

	// Form canonical clean LinkedIn URL
	canonicalPath := v.formatCanonicalPath(entityType, slug)
	res.CanonicalURL = "https://www.linkedin.com" + canonicalPath

	// ─── TAHAP 2: Native HTTP Probe ───────────────────────────────────────────
	res.VerifiedVia = "STAGE_2_HTTP_PROBE"
	statusCode, finalURL, bodySnippet, err := v.probeLinkedIn(ctx, res.CanonicalURL)
	res.HTTPStatus = statusCode

	if err != nil {
		res.Reason = fmt.Sprintf("Network connection error probing LinkedIn: %v", err)
		res.HealthStatus = StatusDegraded
		return res
	}

	// Analyze response for LinkedIn anti-bot / authwall / 404
	if statusCode == 404 {
		res.IsValid = false
		res.HealthStatus = StatusDisabledDeadLink
		res.Reason = fmt.Sprintf("LinkedIn 404 Page Not Found (Target entity '%s' does not exist)", slug)
		return res
	}

	if statusCode == 410 {
		res.IsValid = false
		res.HealthStatus = StatusDisabledDeadLink
		res.Reason = fmt.Sprintf("LinkedIn 410 Gone (Entity '%s' removed/deleted)", slug)
		return res
	}

	// Check for redirect to authwall or sign-in (Indicates target EXISTS on LinkedIn, but protected behind authwall)
	if strings.Contains(finalURL, "/authwall") || strings.Contains(finalURL, "/login") || strings.Contains(finalURL, "signup") {
		res.IsValid = true
		res.HealthStatus = StatusHealthy
		res.Reason = fmt.Sprintf("Valid LinkedIn %s URL (Confirmed active via LinkedIn Authwall redirect: %s)", entityType, slug)
		return res
	}

	// Check HTTP 999 (LinkedIn specific request denied - indicates active target under anti-bot block)
	if statusCode == 999 {
		res.IsValid = true
		res.HealthStatus = StatusHealthy
		res.Reason = fmt.Sprintf("Valid LinkedIn %s URL syntax & active route (HTTP 999 anti-bot response for %s)", entityType, slug)
		return res
	}

	// Check soft 404 indicators in body or final URL redirect path
	if v.isLinkedInSoft404(finalURL, bodySnippet) {
		res.IsValid = false
		res.HealthStatus = StatusDisabledDeadLink
		res.Reason = fmt.Sprintf("LinkedIn soft 404 detected ('This page doesn't exist' for entity '%s')", slug)
		return res
	}

	if statusCode >= 200 && statusCode < 400 {
		res.IsValid = true
		res.HealthStatus = StatusHealthy
		res.Reason = fmt.Sprintf("Valid LinkedIn %s URL (HTTP %d OK)", entityType, statusCode)
		return res
	}

	// Fallback for other status codes
	res.IsValid = false
	res.HealthStatus = StatusDegraded
	res.Reason = fmt.Sprintf("LinkedIn returned unexpected HTTP status %d", statusCode)
	return res
}

func (v *LinkedInVerifier) parseEntityAndSlug(path string) (LinkedInEntityType, string) {
	if m := companyRegex.FindStringSubmatch(path); len(m) > 1 {
		return EntityCompany, m[1]
	}
	if m := profileRegex.FindStringSubmatch(path); len(m) > 1 {
		return EntityProfile, m[1]
	}
	if m := schoolRegex.FindStringSubmatch(path); len(m) > 1 {
		return EntitySchool, m[1]
	}
	if m := showcaseRegex.FindStringSubmatch(path); len(m) > 1 {
		return EntityShowcase, m[1]
	}
	if m := jobRegex.FindStringSubmatch(path); len(m) > 1 {
		return EntityJob, m[1]
	}
	if m := postRegex.FindStringSubmatch(path); len(m) > 1 {
		return EntityPost, m[1]
	}
	if m := pulseRegex.FindStringSubmatch(path); len(m) > 1 {
		return EntityPulse, m[1]
	}
	return EntityUnknown, ""
}

func (v *LinkedInVerifier) formatCanonicalPath(entityType LinkedInEntityType, slug string) string {
	switch entityType {
	case EntityCompany:
		return "/company/" + slug + "/"
	case EntityProfile:
		return "/in/" + slug + "/"
	case EntitySchool:
		return "/school/" + slug + "/"
	case EntityShowcase:
		return "/showcase/" + slug + "/"
	case EntityJob:
		return "/jobs/view/" + slug + "/"
	case EntityPost:
		return "/posts/" + slug + "/"
	case EntityPulse:
		return "/pulse/" + slug + "/"
	default:
		return "/" + slug
	}
}

func (v *LinkedInVerifier) probeLinkedIn(ctx context.Context, targetURL string) (int, string, string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return 0, targetURL, "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,id;q=0.8")

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return 0, targetURL, "", err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 16384))
	bodySnippet := string(bodyBytes)

	finalURL := resp.Request.URL.String()
	return resp.StatusCode, finalURL, bodySnippet, nil
}

func (v *LinkedInVerifier) isLinkedInSoft404(finalURL, bodySnippet string) bool {
	finalLower := strings.ToLower(finalURL)
	if strings.Contains(finalLower, "/404") || strings.Contains(finalLower, "page-not-found") {
		return true
	}

	bodyLower := strings.ToLower(bodySnippet)
	soft404Phrases := []string{
		"this page doesn't exist",
		"page not found",
		"this profile is not available",
		"an error has occurred",
		"this content is unavailable",
		"halaman tidak ditemukan",
	}

	for _, phrase := range soft404Phrases {
		if strings.Contains(bodyLower, phrase) {
			return true
		}
	}
	return false
}
