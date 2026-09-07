package urlverifier

import (
	"context"
	"testing"
)

func TestURLVerifier_VerifyAndResolve(t *testing.T) {
	verifier := NewURLVerifier()
	ctx := context.Background()

	tests := []struct {
		name         string
		url          string
		wantValid    bool
		wantStatus   HealthStatus
	}{
		{
			name:       "Valid BCA Bakti CSR",
			url:        "https://www.bca.co.id/id/tentang-bca/csr/bakti-bca",
			wantValid:  true,
			wantStatus: StatusHealthy,
		},
		{
			name:       "Dead BCA CSR link (404) with Smart Fallback",
			url:        "https://www.bca.co.id/csr-news",
			wantValid:  false,
			wantStatus: StatusDisabledDeadLink,
		},
		{
			name:       "Valid Telkom Sustainability",
			url:        "https://www.telkom.co.id/sites/sustainability/id_ID/page/csr-1127",
			wantValid:  true,
			wantStatus: StatusHealthy,
		},
		{
			name:       "Unreachable DNS Subdomain",
			url:        "https://yayasan.djarumfoundation.org/call-for-proposals-2026",
			wantValid:  false,
			wantStatus: StatusDisabledDeadLink,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := verifier.VerifyAndResolve(ctx, tt.url)
			if res.IsValid != tt.wantValid {
				t.Errorf("VerifyAndResolve(%s) IsValid = %v, want %v (Final: %s, Fallback: %s)", tt.url, res.IsValid, tt.wantValid, res.FinalURL, res.FallbackURL)
			}
			t.Logf("Result for %s => IsValid: %v, Status: %d, FinalURL: %s, FallbackURL: %s, Err: %s",
				tt.name, res.IsValid, res.HTTPStatus, res.FinalURL, res.FallbackURL, res.ErrorMsg)
		})
	}
}
