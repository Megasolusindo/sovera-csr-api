package urlverifier

import (
	"context"
	"testing"
)

func TestLinkedInVerifier_ValidateLinkedInURL(t *testing.T) {
	verifier := NewLinkedInVerifier()
	ctx := context.Background()

	tests := []struct {
		name             string
		rawURL           string
		wantValid        bool
		wantEntityType   LinkedInEntityType
		wantCanonical    string
		wantSlug         string
		wantHealthStatus HealthStatus
	}{
		{
			name:             "Valid Company URL",
			rawURL:           "https://www.linkedin.com/company/pertamina?trk=public_profile_topcard-current-company",
			wantValid:        true,
			wantEntityType:   EntityCompany,
			wantCanonical:    "https://www.linkedin.com/company/pertamina/",
			wantSlug:         "pertamina",
			wantHealthStatus: StatusHealthy,
		},
		{
			name:             "Valid Company URL with ID Subdomain",
			rawURL:           "https://id.linkedin.com/company/bank-central-asia/",
			wantValid:        true,
			wantEntityType:   EntityCompany,
			wantCanonical:    "https://www.linkedin.com/company/bank-central-asia/",
			wantSlug:         "bank-central-asia",
			wantHealthStatus: StatusHealthy,
		},
		{
			name:             "Valid Personal Profile URL",
			rawURL:           "https://www.linkedin.com/in/satya-nadella",
			wantValid:        true,
			wantEntityType:   EntityProfile,
			wantCanonical:    "https://www.linkedin.com/in/satya-nadella/",
			wantSlug:         "satya-nadella",
			wantHealthStatus: StatusHealthy,
		},
		{
			name:             "Valid School URL",
			rawURL:           "https://www.linkedin.com/school/universitas-indonesia/",
			wantValid:        true,
			wantEntityType:   EntitySchool,
			wantCanonical:    "https://www.linkedin.com/school/universitas-indonesia/",
			wantSlug:         "universitas-indonesia",
			wantHealthStatus: StatusHealthy,
		},
		{
			name:             "Invalid Domain - Twitter URL",
			rawURL:           "https://twitter.com/company/pertamina",
			wantValid:        false,
			wantEntityType:   EntityUnknown,
			wantCanonical:    "https://twitter.com/company/pertamina",
			wantSlug:         "",
			wantHealthStatus: StatusDisabledDeadLink,
		},
		{
			name:             "Invalid Path Structure",
			rawURL:           "https://www.linkedin.com/invalidpath/random-slug-123",
			wantValid:        false,
			wantEntityType:   EntityUnknown,
			wantCanonical:    "https://www.linkedin.com/invalidpath/random-slug-123",
			wantSlug:         "",
			wantHealthStatus: StatusDisabledDeadLink,
		},
		{
			name:             "Empty URL",
			rawURL:           "",
			wantValid:        false,
			wantEntityType:   EntityUnknown,
			wantCanonical:    "",
			wantSlug:         "",
			wantHealthStatus: StatusDisabledDeadLink,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := verifier.ValidateLinkedInURL(ctx, tt.rawURL)

			if res.IsValid != tt.wantValid {
				t.Errorf("ValidateLinkedInURL() IsValid = %v, want %v (Reason: %s)", res.IsValid, tt.wantValid, res.Reason)
			}
			if res.EntityType != tt.wantEntityType {
				t.Errorf("ValidateLinkedInURL() EntityType = %v, want %v", res.EntityType, tt.wantEntityType)
			}
			if res.CanonicalSlug != tt.wantSlug {
				t.Errorf("ValidateLinkedInURL() CanonicalSlug = %v, want %v", res.CanonicalSlug, tt.wantSlug)
			}
			if tt.wantCanonical != "" && res.CanonicalURL != tt.wantCanonical {
				t.Errorf("ValidateLinkedInURL() CanonicalURL = %v, want %v", res.CanonicalURL, tt.wantCanonical)
			}
			if res.HealthStatus != tt.wantHealthStatus {
				t.Errorf("ValidateLinkedInURL() HealthStatus = %v, want %v", res.HealthStatus, tt.wantHealthStatus)
			}
		})
	}
}
