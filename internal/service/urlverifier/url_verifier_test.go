package urlverifier

import (
	"context"
	"testing"
)

func TestURLVerifier_VerifyAndResolve(t *testing.T) {
	verifier := NewURLVerifier()
	ctx := context.Background()

	tests := []struct {
		name       string
		url        string
		wantValid  bool
		wantStatus HealthStatus
	}{
		{
			name:       "Japanese Embassy GGP 404 URL",
			url:        "https://www.id.emb-japan.go.jp/ggp.html",
			wantValid:  false,
			wantStatus: StatusDisabledDeadLink,
		},
		{
			name:       "Australian Embassy DAP 404 URL",
			url:        "https://indonesia.embassy.gov.au/jkti/dap.html",
			wantValid:  false,
			wantStatus: StatusDisabledDeadLink,
		},
		{
			name:       "Pertamina Newsroom Soft 404 Redirect",
			url:        "https://www.pertamina.com/id/news-room",
			wantValid:  false,
			wantStatus: StatusDisabledDeadLink,
		},
		{
			name:       "Dead BCA CSR link (404) with Smart Fallback",
			url:        "https://www.bca.co.id/csr-news",
			wantValid:  false,
			wantStatus: StatusDisabledDeadLink,
		},
		{
			name:       "Unreachable DNS Subdomain",
			url:        "https://yayasan.djarumfoundation.org/call-for-proposals-2026",
			wantValid:  false,
			wantStatus: StatusDisabledDeadLink,
		},
		{
			name:       "BUMN Press Release 404 URL",
			url:        "https://bumn.go.id/media/press-release",
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
			t.Logf("Result for %s => IsValid: %v, HealthStatus: %s, FinalURL: %s, FallbackURL: %s, Err: %s",
				tt.name, res.IsValid, res.HealthStatus, res.FinalURL, res.FallbackURL, res.ErrorMsg)
		})
	}
}
