package ai

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

type PressReleaseOrigin struct {
	WireHash         string   `json:"wire_hash"`
	PrimaryOrigin    string   `json:"primary_origin"`
	SyndicatedURLs   []string `json:"syndicated_urls"`
	IsSyndicatedCopy bool     `json:"is_syndicated_copy"`
}

// ComputePressReleaseHash generates an invariant content hash for press releases (§15 Origin Tracking)
// It normalizes text by removing boilerplate headers, timestamps, media outlets, and whitespaces.
func ComputePressReleaseHash(rawArticleText string) string {
	clean := strings.ToLower(rawArticleText)

	// Remove common wire boilerplate & timestamp markers
	reDate := regexp.MustCompile(`\b(jakarta|bandung|surabaya|semarang)\s*,\s*\d{1,2}\s+[a-z]+\s+\d{4}\b`)
	clean = reDate.ReplaceAllString(clean, "")

	reMedia := regexp.MustCompile(`\b(antara|detik|kompas|tribun|liputan6|bisnis\.com|kontan)\b`)
	clean = reMedia.ReplaceAllString(clean, "")

	// Remove non-alphanumeric characters
	reNonAlpha := regexp.MustCompile(`[^a-z0-9]`)
	clean = reNonAlpha.ReplaceAllString(clean, "")

	// Take first 500 normalized characters for canonical wire signature
	if len(clean) > 500 {
		clean = clean[:500]
	}

	hash := sha256.Sum256([]byte(clean))
	return hex.EncodeToString(hash[:])
}

// TrackSyndication checks if an article is a syndicated copy of an existing wire hash
func TrackSyndication(existingHashes map[string]string, articleText, sourceURL string) *PressReleaseOrigin {
	wireHash := ComputePressReleaseHash(articleText)

	origin, exists := existingHashes[wireHash]
	if exists {
		return &PressReleaseOrigin{
			WireHash:         wireHash,
			PrimaryOrigin:    origin,
			SyndicatedURLs:   []string{sourceURL},
			IsSyndicatedCopy: true,
		}
	}

	return &PressReleaseOrigin{
		WireHash:         wireHash,
		PrimaryOrigin:    sourceURL,
		SyndicatedURLs:   []string{sourceURL},
		IsSyndicatedCopy: false,
	}
}
