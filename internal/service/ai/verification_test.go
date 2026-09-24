package ai

import (
	"testing"
)

func TestVerifyClaimQuoteAlignment_Success(t *testing.T) {
	company := "PT Bank Syariah Indonesia Tbk"
	year := 2026
	quote := "BSI menyalurkan anggaran CSR sebesar Rp 153 Miliar pada tahun 2026 untuk program zakat dan wakaf."

	res := VerifyClaimQuoteAlignment(company, year, quote)
	if !res.IsGrounded {
		t.Fatalf("expected claim to be grounded, got rejected: %s", res.RejectionReason)
	}
}

func TestVerifyClaimQuoteAlignment_NegationRejection(t *testing.T) {
	company := "Bank Syariah Indonesia"
	year := 2026
	quote := "BSI menunda penyaluran dana CSR untuk tahun 2026 sampai batas waktu yang belum ditentukan."

	res := VerifyClaimQuoteAlignment(company, year, quote)
	if res.IsGrounded {
		t.Fatalf("expected claim to be rejected due to negation word 'menunda'")
	}
	if res.NegationMatch {
		t.Errorf("expected NegationMatch to be false")
	}
}

func TestEvaluateVerificationMatrix_FundingSignal(t *testing.T) {
	// Case 1: 1x Tier B source -> PARTIALLY_VERIFIED (No alert)
	input1 := VerificationInput{
		ClaimType:   "FUNDING_SIGNAL",
		CompanyName: "Bank Syariah Indonesia",
		ClaimYear:   2026,
		QuoteText:   "BSI menganggarkan Rp 153 Miliar dana CSR 2026.",
		SourceTiers: []SourceTier{TierB},
	}

	out1 := EvaluateVerificationMatrix(input1)
	if out1.Status != StatusPartiallyVerified {
		t.Errorf("expected StatusPartiallyVerified, got %s", out1.Status)
	}
	if out1.CanPublishAlert {
		t.Errorf("partially verified signal should not publish alert")
	}

	// Case 2: 1x Tier A source -> VERIFIED (Alert allowed)
	input2 := VerificationInput{
		ClaimType:   "FUNDING_SIGNAL",
		CompanyName: "Bank Syariah Indonesia",
		ClaimYear:   2026,
		QuoteText:   "BSI menganggarkan Rp 153 Miliar dana CSR 2026.",
		SourceTiers: []SourceTier{TierA},
	}

	out2 := EvaluateVerificationMatrix(input2)
	if out2.Status != StatusVerified {
		t.Errorf("expected StatusVerified, got %s", out2.Status)
	}
	if !out2.CanPublishAlert {
		t.Errorf("verified signal should be allowed to publish alert")
	}
}

func TestEvaluateVerificationMatrix_PolicyChangeMandatoryTierA(t *testing.T) {
	inputMediaOnly := VerificationInput{
		ClaimType:   "POLICY_CHANGE",
		CompanyName: "Bank Syariah Indonesia",
		ClaimYear:   2026,
		QuoteText:   "BSI mengubah kebijakan kemitraan CSR 2026.",
		SourceTiers: []SourceTier{TierB, TierB},
	}

	out := EvaluateVerificationMatrix(inputMediaOnly)
	if out.Status != StatusRejected {
		t.Errorf("POLICY_CHANGE without Tier A must be REJECTED per §5.2, got %s", out.Status)
	}
}
