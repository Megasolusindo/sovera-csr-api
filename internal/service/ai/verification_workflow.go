package ai

import (
	"fmt"
	"strings"
)

type SourceTier string

const (
	TierA SourceTier = "TIER_A" // Official Website, BEI Annual Report, Ministry Press Release
	TierB SourceTier = "TIER_B" // Major Media (Kompas, Detik, Bisnis.com, Antara)
	TierC SourceTier = "TIER_C" // User Blog, Social Media Post, Forum
)

type SignalStatus string

const (
	StatusVerified          SignalStatus = "VERIFIED"
	StatusPartiallyVerified SignalStatus = "PARTIALLY_VERIFIED"
	StatusUnverified        SignalStatus = "UNVERIFIED"
	StatusRejected          SignalStatus = "REJECTED"
)

type VerificationInput struct {
	ClaimType       string       `json:"claim_type"`
	CompanyName     string       `json:"company_name"`
	ClaimYear       int          `json:"claim_year"`
	QuoteText       string       `json:"quote_text"`
	SourceTiers     []SourceTier `json:"source_tiers"`
	HasDirectQuote  bool         `json:"has_direct_quote"`
	HasLocationData bool         `json:"has_location_data"`
}

type VerificationOutput struct {
	Status           SignalStatus                 `json:"status"`
	GroundingResult  *GroundingVerificationResult `json:"grounding_result"`
	TierSummary      string                       `json:"tier_summary"`
	CanPublishAlert  bool                         `json:"can_publish_alert"`
}

// EvaluateVerificationMatrix applies the Source Tiering Matrix rules per §5.2
func EvaluateVerificationMatrix(input VerificationInput) *VerificationOutput {
	// Step 1: Perform Grounding Verification
	grounding := VerifyClaimQuoteAlignment(input.CompanyName, input.ClaimYear, input.QuoteText)
	if !grounding.IsGrounded {
		return &VerificationOutput{
			Status:          StatusRejected,
			GroundingResult: grounding,
			TierSummary:     fmt.Sprintf("Rejected during grounding check: %s", grounding.RejectionReason),
			CanPublishAlert: false,
		}
	}

	tierACount := 0
	tierBCount := 0
	for _, t := range input.SourceTiers {
		if t == TierA {
			tierACount++
		} else if t == TierB {
			tierBCount++
		}
	}

	tierSummary := fmt.Sprintf("Tier A sources: %d, Tier B sources: %d", tierACount, tierBCount)
	claimType := strings.ToUpper(input.ClaimType)

	switch claimType {
	case "FUNDING_SIGNAL":
		// Rule: 1x Tier A OR 2x Tier B -> VERIFIED. Otherwise -> PARTIALLY_VERIFIED
		if tierACount >= 1 || tierBCount >= 2 {
			return &VerificationOutput{
				Status:          StatusVerified,
				GroundingResult: grounding,
				TierSummary:     tierSummary,
				CanPublishAlert: true,
			}
		}
		return &VerificationOutput{
			Status:          StatusPartiallyVerified,
			GroundingResult: grounding,
			TierSummary:     tierSummary + " (Insufficient sources for public alert)",
			CanPublishAlert: false,
		}

	case "PROGRAM_LAUNCH":
		// Rule: 1x Tier A OR (1x Tier B with direct quote & location) -> VERIFIED. Otherwise -> UNVERIFIED
		if tierACount >= 1 || (tierBCount >= 1 && input.HasDirectQuote && input.HasLocationData) {
			return &VerificationOutput{
				Status:          StatusVerified,
				GroundingResult: grounding,
				TierSummary:     tierSummary,
				CanPublishAlert: true,
			}
		}
		return &VerificationOutput{
			Status:          StatusUnverified,
			GroundingResult: grounding,
			TierSummary:     tierSummary + " (Missing direct quote or location data)",
			CanPublishAlert: false,
		}

	case "POLICY_CHANGE":
		// Rule: Wajib 1x Tier A (Official Press/Ministry). Otherwise -> REJECTED
		if tierACount >= 1 {
			return &VerificationOutput{
				Status:          StatusVerified,
				GroundingResult: grounding,
				TierSummary:     tierSummary,
				CanPublishAlert: true,
			}
		}
		return &VerificationOutput{
			Status:          StatusRejected,
			GroundingResult: grounding,
			TierSummary:     tierSummary + " (POLICY_CHANGE requires mandatory Tier A official source)",
			CanPublishAlert: false,
		}

	default:
		if tierACount >= 1 || tierBCount >= 1 {
			return &VerificationOutput{
				Status:          StatusVerified,
				GroundingResult: grounding,
				TierSummary:     tierSummary,
				CanPublishAlert: true,
			}
		}
		return &VerificationOutput{
			Status:          StatusUnverified,
			GroundingResult: grounding,
			TierSummary:     tierSummary,
			CanPublishAlert: false,
		}
	}
}
