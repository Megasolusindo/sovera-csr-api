package queue

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"


	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	"sovera-core-api/internal/pkg/phoneverifier"
)

var (
	emailFinderRegex   = regexp.MustCompile(`(?i)[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	phoneFinderRegex   = regexp.MustCompile(`(?:\+?62|0)8[1-9][0-9]{7,10}|(?:\+?62|0)[2-9][0-9]{1,3}[- ]?[0-9]{5,8}`)
	addressFinderRegex = regexp.MustCompile(`(?i)(?:Alamat|Address|Kantor\s+Pusat|Head\s+Office|Gedung|Headquarters)\s*[:\-]\s*([^\r\n<]{15,200})|(?:Jl\.|Jalan)\s+[A-Za-z0-9\s.,\-\/]+(?:No\.\s*\d+|Kav\.\s*\d+)[^\r\n<]{5,100}`)
	ahuFinderRegex     = regexp.MustCompile(`(?i)\bAHU[-.\s]?\d+[-.\s]?AH[-.\s]?\d+[-.\s]?\d+[-.\s]?(?:Tahun\s*\d{4}|\d{4})\b`)
	nibFinderRegex     = regexp.MustCompile(`(?i)\bNIB\s*[:\.]?\s*(\d{13})\b`)
	kbliFinderRegex    = regexp.MustCompile(`(?i)\bKBLI\s*[:\.]?\s*(\d{5})\b`)

	ignoredEmailDomains = []string{
		"example.com", "domain.com", "email.com", "sentry.io", "wixpress.com",
		"github.com", "schema.org", "w3.org", "google.com", "facebook.com",
	}
)

type CompanyContactDiscoveryWorker struct {
	dbPool       *pgxpool.Pool
	serperAPIKey string
	httpClient   *http.Client
}

func NewCompanyContactDiscoveryWorker(dbPool *pgxpool.Pool, serperAPIKey string) *CompanyContactDiscoveryWorker {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &CompanyContactDiscoveryWorker{
		dbPool:       dbPool,
		serperAPIKey: serperAPIKey,
		httpClient: &http.Client{
			Transport: tr,
			Timeout:   8 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
	}
}

type ContactDiscoveryTarget struct {
	ID           string
	Name         string
	Ticker       string
	Website      string
	Phone        string
	Email        string
	Headquarters string
}

func (w *CompanyContactDiscoveryWorker) HandleCompanyContactDiscoveryBatch(ctx context.Context, t *asynq.Task) error {
	if w.dbPool == nil {
		log.Println("[CompanyContactDiscoveryWorker] No DB pool available, skipping contact discovery sweep")
		return nil
	}

	log.Println("[CompanyContactDiscoveryWorker] Starting empirical Phone, Email & Address contact discovery sweep...")

	rows, err := w.dbPool.Query(ctx, `
		SELECT id::text, name, COALESCE(ticker, ''), COALESCE(website, ''), COALESCE(phone, ''), COALESCE(email, ''), COALESCE(headquarters, '')
		FROM company.companies
		WHERE phone IS NULL OR phone = '' OR email IS NULL OR email = '' OR headquarters IS NULL OR headquarters = ''
		ORDER BY (CASE WHEN ticker IS NOT NULL AND ticker <> '' THEN 0 ELSE 1 END) ASC, created_at ASC
		LIMIT 50;
	`)
	if err != nil {
		return fmt.Errorf("failed to query companies for contact discovery: %w", err)
	}
	defer rows.Close()

	var targets []ContactDiscoveryTarget
	for rows.Next() {
		var target ContactDiscoveryTarget
		if err := rows.Scan(&target.ID, &target.Name, &target.Ticker, &target.Website, &target.Phone, &target.Email, &target.Headquarters); err != nil {
			log.Printf("[CompanyContactDiscoveryWorker] Error scanning target row: %v", err)
			continue
		}
		targets = append(targets, target)
	}

	if len(targets) == 0 {
		log.Println("[CompanyContactDiscoveryWorker] No companies with missing phone, email, or headquarters found.")
		return nil
	}

	log.Printf("[CompanyContactDiscoveryWorker] Found %d target companies missing phone, email, or headquarters info.", len(targets))

	var enrichedCount int

	for _, target := range targets {
		var newPhone, newEmail, newHQ, newAHU, newNIB, newKBLI string

		// Tier 1: Inspect direct website contact pages if website is present
		if target.Website != "" && target.Website != "https://-" {
			p, e, hq, ahu, nib, kbli := w.scrapeWebsiteContactPages(ctx, target.Website)
			if p != "" && target.Phone == "" {
				newPhone = p
			}
			if e != "" && target.Email == "" {
				newEmail = e
			}
			if hq != "" && target.Headquarters == "" {
				newHQ = hq
			}
			newAHU = ahu
			newNIB = nib
			newKBLI = kbli
		}

		// Tier 2: Serper API / Google Organic Search discovery fallback
		if (newPhone == "" && target.Phone == "") || (newEmail == "" && target.Email == "") {
			pSerper, eSerper := w.discoverContactViaSerper(ctx, target.Name, target.Ticker)
			if newPhone == "" && target.Phone == "" && pSerper != "" {
				newPhone = pSerper
			}
			if newEmail == "" && target.Email == "" && eSerper != "" {
				newEmail = eSerper
			}
		}

		// Tier 3: Validation & Database Persistence
		if newPhone != "" || newEmail != "" || newHQ != "" || newAHU != "" || newNIB != "" || newKBLI != "" {
			var validPhone *string
			var validEmail *string
			var validHQ *string
			var validAHU *string
			var validNIB *string
			var validKBLI *string

			if newPhone != "" {
				ok, normalized, err := phoneverifier.DefaultVerifier.Verify(newPhone)
				if ok && err == nil {
					validPhone = &normalized
				} else {
					log.Printf("[CompanyContactDiscoveryWorker] Rejected candidate phone '%s' for '%s': %v", newPhone, target.Name, err)
				}
			}

			if newEmail != "" && w.isValidEmail(newEmail) {
				cleanEmail := strings.ToLower(strings.TrimSpace(newEmail))
				validEmail = &cleanEmail
			}

			if newHQ != "" {
				cleanHQ := strings.TrimSpace(newHQ)
				if len(cleanHQ) >= 10 && len(cleanHQ) <= 250 {
					validHQ = &cleanHQ
				}
			}

			if newAHU != "" {
				validAHU = &newAHU
			}
			if newNIB != "" {
				validNIB = &newNIB
			}
			if newKBLI != "" {
				validKBLI = &newKBLI
			}

			if validPhone != nil || validEmail != nil || validHQ != nil || validAHU != nil || validNIB != nil || validKBLI != nil {
				updateQuery := `
					UPDATE company.companies
					SET phone = COALESCE($1, phone),
					    email = COALESCE($2, email),
					    headquarters = COALESCE($3, headquarters),
					    ahu_number = COALESCE($4, ahu_number),
					    nib = COALESCE($5, nib),
					    kbli_code = COALESCE($6, kbli_code),
					    updated_at = NOW()
					WHERE id = $7;
				`
				_, updateErr := w.dbPool.Exec(ctx, updateQuery, validPhone, validEmail, validHQ, validAHU, validNIB, validKBLI, target.ID)
				if updateErr == nil {
					enrichedCount++
					pStr, eStr, hqStr := "<none>", "<none>", "<none>"
					if validPhone != nil {
						pStr = *validPhone
					}
					if validEmail != nil {
						eStr = *validEmail
					}
					if validHQ != nil {
						hqStr = *validHQ
					}
					log.Printf("[CompanyContactDiscoveryWorker] ✅ Enriched contact/legal info for '%s' (ID: %s) -> Phone: %s | Email: %s | HQ: %s | AHU: %v | NIB: %v | KBLI: %v", target.Name, target.ID, pStr, eStr, hqStr, validAHU, validNIB, validKBLI)
				} else {
					log.Printf("[CompanyContactDiscoveryWorker] Failed to update company contact/legal info for '%s': %v", target.Name, updateErr)
				}
			}
		}
	}

	log.Printf("[CompanyContactDiscoveryWorker] Completed Contact Discovery sweep. Total companies enriched: %d", enrichedCount)
	return nil
}

func (w *CompanyContactDiscoveryWorker) scrapeWebsiteContactPages(ctx context.Context, baseWebsite string) (string, string, string, string, string, string) {
	if !strings.HasPrefix(baseWebsite, "http://") && !strings.HasPrefix(baseWebsite, "https://") {
		baseWebsite = "https://" + baseWebsite
	}
	baseWebsite = strings.TrimSuffix(baseWebsite, "/")

	subPaths := []string{"", "/contact", "/kontak", "/about", "/tentang-kami"}
	var foundPhone, foundEmail, foundHQ, foundAHU, foundNIB, foundKBLI string

	for _, path := range subPaths {
		targetURL := baseWebsite + path
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko)")

		resp, err := w.httpClient.Do(req)
		if err != nil || resp.StatusCode < 200 || resp.StatusCode >= 400 {
			if resp != nil {
				_ = resp.Body.Close()
			}
			continue
		}

		bodyBytes, readErr := io.ReadAll(io.LimitReader(resp.Body, 500*1024))
		_ = resp.Body.Close()
		if readErr != nil {
			continue
		}

		bodyText := string(bodyBytes)

		if foundEmail == "" {
			emails := emailFinderRegex.FindAllString(bodyText, -1)
			for _, em := range emails {
				if w.isValidEmail(em) {
					foundEmail = em
					break
				}
			}
		}

		if foundPhone == "" {
			phones := phoneFinderRegex.FindAllString(bodyText, -1)
			for _, ph := range phones {
				ok, normalized, _ := phoneverifier.DefaultVerifier.Verify(ph)
				if ok {
					foundPhone = normalized
					break
				}
			}
		}

		if foundHQ == "" {
			foundHQ = extractAddressFromHTML(bodyText)
		}

		if foundAHU == "" {
			ahu := ahuFinderRegex.FindString(bodyText)
			if ahu != "" {
				foundAHU = strings.TrimSpace(ahu)
			}
		}

		if foundNIB == "" {
			nibMatch := nibFinderRegex.FindStringSubmatch(bodyText)
			if len(nibMatch) > 1 {
				foundNIB = strings.TrimSpace(nibMatch[1])
			}
		}

		if foundKBLI == "" {
			kbliMatch := kbliFinderRegex.FindStringSubmatch(bodyText)
			if len(kbliMatch) > 1 {
				foundKBLI = strings.TrimSpace(kbliMatch[1])
			}
		}

		if foundPhone != "" && foundEmail != "" && foundHQ != "" && foundAHU != "" && foundNIB != "" {
			break
		}
	}

	return foundPhone, foundEmail, foundHQ, foundAHU, foundNIB, foundKBLI
}

func extractAddressFromHTML(bodyText string) string {
	// Method 1: Schema.org JSON-LD streetAddress extraction
	if strings.Contains(bodyText, "streetAddress") {
		re := regexp.MustCompile(`"streetAddress"\s*:\s*"([^"]+)"`)
		match := re.FindStringSubmatch(bodyText)
		if len(match) > 1 {
			clean := cleanAddressString(match[1])
			if len(clean) >= 10 && len(clean) <= 250 {
				return clean
			}
		}
	}

	// Method 2: Keyword pattern match (Alamat:, Address:, Head Office:, Gedung ...)
	matches := addressFinderRegex.FindStringSubmatch(bodyText)
	if len(matches) > 1 && matches[1] != "" {
		clean := cleanAddressString(matches[1])
		if len(clean) >= 10 && len(clean) <= 250 {
			return clean
		}
	}

	// Method 3: Regex match for Jalan / Jl.
	fullMatch := addressFinderRegex.FindString(bodyText)
	if fullMatch != "" {
		clean := cleanAddressString(fullMatch)
		if len(clean) >= 10 && len(clean) <= 250 {
			return clean
		}
	}

	return ""
}

func cleanAddressString(raw string) string {
	reHTML := regexp.MustCompile(`<[^>]*>`)
	clean := reHTML.ReplaceAllString(raw, " ")
	clean = strings.Join(strings.Fields(clean), " ")
	clean = strings.Trim(clean, " \t\r\n:-,.")
	if strings.Contains(clean, "{") || strings.Contains(clean, "function") || strings.Contains(clean, "var ") || strings.Contains(clean, "class=") {
		return ""
	}
	return clean
}

func (w *CompanyContactDiscoveryWorker) discoverContactViaSerper(ctx context.Context, name, ticker string) (string, string) {
	// Serper API integration moved 100% to web-scraper discovery engine
	return "", ""
}

func (w *CompanyContactDiscoveryWorker) isValidEmail(email string) bool {

	email = strings.TrimSpace(strings.ToLower(email))
	if len(email) < 6 || !strings.Contains(email, "@") {
		return false
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return false
	}

	domain := parts[1]
	for _, ignored := range ignoredEmailDomains {
		if domain == ignored || strings.HasSuffix(domain, "."+ignored) {
			return false
		}
	}

	return true
}

