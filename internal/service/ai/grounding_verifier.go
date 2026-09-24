package ai

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	ErrNegationDetected = errors.New("grounding check failed: negation or cancellation detected in quote")
	ErrTemporalMismatch = errors.New("grounding check failed: claim year does not match quote text year")
	ErrEntityMismatch   = errors.New("grounding check failed: entity name in quote does not match claim company")
)

type GroundingVerificationResult struct {
	IsGrounded      bool    `json:"is_grounded"`
	TemporalMatch   bool    `json:"temporal_match"`
	NegationMatch   bool    `json:"negation_match"` // true if no negation found
	EntityMatch     bool    `json:"entity_match"`
	RejectionReason string  `json:"rejection_reason,omitempty"`
	Confidence      float64 `json:"confidence"`
}

// VerifyClaimQuoteAlignment performs 3 deterministic verification checks (§5.1)
func VerifyClaimQuoteAlignment(companyName string, claimYear int, quoteText string) *GroundingVerificationResult {
	result := &GroundingVerificationResult{
		IsGrounded:    true,
		TemporalMatch: true,
		NegationMatch: true,
		EntityMatch:   true,
		Confidence:    1.0,
	}

	cleanQuote := strings.ToLower(quoteText)

	// 1. Intent & Negation Check (§5.1.2)
	negationWords := []string{"batal", "batalkan", "menunda", "ditunda", "gagal", "dihentikan", "bukan", "tidak pernah", "hoax", "hoaks"}
	for _, neg := range negationWords {
		re := regexp.MustCompile(fmt.Sprintf(`\b%s\b`, neg))
		if re.MatchString(cleanQuote) {
			result.NegationMatch = false
			result.IsGrounded = false
			result.RejectionReason = fmt.Sprintf("Negation word '%s' detected in quote", neg)
			result.Confidence = 0.0
			return result
		}
	}

	// 2. Temporal Alignment (§5.1.1)
	if claimYear > 0 {
		reYear := regexp.MustCompile(`\b(20[2-9][0-9])\b`)
		matches := reYear.FindAllString(cleanQuote, -1)
		if len(matches) > 0 {
			yearFound := false
			for _, m := range matches {
				y, _ := strconv.Atoi(m)
				if y == claimYear {
					yearFound = true
					break
				}
			}
			if !yearFound {
				result.TemporalMatch = false
				result.IsGrounded = false
				result.RejectionReason = fmt.Sprintf("Claim year %d not found among years in quote (%v)", claimYear, matches)
				result.Confidence = 0.3
				return result
			}
		}
	}

	// 3. Entity Match (§5.1.3)
	cleanCompany := strings.ToLower(companyName)
	cleanCompany = strings.ReplaceAll(cleanCompany, "pt ", "")
	cleanCompany = strings.ReplaceAll(cleanCompany, "tbk", "")
	cleanCompany = strings.TrimSpace(cleanCompany)

	if cleanCompany != "" {
		// Check full name, significant individual words (>3 chars), or acronym
		entityMatched := false
		if strings.Contains(cleanQuote, cleanCompany) {
			entityMatched = true
		} else {
			// Extract acronym (e.g. "bank syariah indonesia" -> "bsi")
			words := strings.Fields(cleanCompany)
			if len(words) > 1 {
				acronym := ""
				for _, w := range words {
					if len(w) > 0 {
						acronym += string(w[0])
					}
				}
				if len(acronym) >= 2 {
					reAcronym := regexp.MustCompile(fmt.Sprintf(`\b%s\b`, acronym))
					if reAcronym.MatchString(cleanQuote) {
						entityMatched = true
					}
				}
			}

			// Check if any major keyword (e.g. "syariah", "indonesia") matches
			if !entityMatched {
				for _, w := range words {
					if len(w) >= 4 && strings.Contains(cleanQuote, w) {
						entityMatched = true
						break
					}
				}
			}
		}

		if !entityMatched {
			result.EntityMatch = false
			result.IsGrounded = false
			result.RejectionReason = fmt.Sprintf("Company name keyword '%s' not present in quote text", cleanCompany)
			result.Confidence = 0.4
			return result
		}
	}

	return result
}
