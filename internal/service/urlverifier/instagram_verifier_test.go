package urlverifier

import (
	"context"
	"testing"
)

func TestInstagramVerifier_ValidateInstagramURL(t *testing.T) {
	verifier := NewInstagramVerifier()

	tests := []struct {
		name             string
		rawURL           string
		wantValid        bool
		wantEntityType   InstagramEntityType
		wantHandle       string
		wantCanonical    string
		wantHealthStatus HealthStatus
	}{
		{
			name:             "Valid official profile URL",
			rawURL:           "https://www.instagram.com/pertamina?utm_medium=copy_link",
			wantValid:        true,
			wantEntityType:   InstagramEntityProfile,
			wantHandle:       "pertamina",
			wantCanonical:    "https://www.instagram.com/pertamina/",
			wantHealthStatus: StatusHealthy,
		},
		{
			name:             "Subdomain handle URL",
			rawURL:           "https://instagr.am/bankcentralasia/",
			wantValid:        true,
			wantEntityType:   InstagramEntityProfile,
			wantHandle:       "bankcentralasia",
			wantCanonical:    "https://www.instagram.com/bankcentralasia/",
			wantHealthStatus: StatusHealthy,
		},
		{
			name:             "Instagram post URL",
			rawURL:           "https://www.instagram.com/p/C31xyz12345/",
			wantValid:        true,
			wantEntityType:   InstagramEntityPost,
			wantHandle:       "C31xyz12345",
			wantCanonical:    "https://www.instagram.com/p/C31xyz12345/",
			wantHealthStatus: StatusHealthy,
		},
		{
			name:             "Instagram reel URL",
			rawURL:           "https://www.instagram.com/reel/C45abc67890/",
			wantValid:        true,
			wantEntityType:   InstagramEntityReel,
			wantHandle:       "C45abc67890",
			wantCanonical:    "https://www.instagram.com/reel/C45abc67890/",
			wantHealthStatus: StatusHealthy,
		},
		{
			name:             "Invalid domain",
			rawURL:           "https://www.fake-instagram.com/pertamina",
			wantValid:        false,
			wantEntityType:   InstagramEntityUnknown,
			wantHandle:       "",
			wantCanonical:    "https://www.fake-instagram.com/pertamina",
			wantHealthStatus: StatusDisabledDeadLink,
		},
	}

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := verifier.ValidateInstagramURL(ctx, tt.rawURL)

			if res.EntityType != tt.wantEntityType {
				t.Errorf("ValidateInstagramURL() EntityType = %v, want %v", res.EntityType, tt.wantEntityType)
			}
			if res.CanonicalHandle != tt.wantHandle {
				t.Errorf("ValidateInstagramURL() CanonicalHandle = %v, want %v", res.CanonicalHandle, tt.wantHandle)
			}
			if res.CanonicalURL != tt.wantCanonical {
				t.Errorf("ValidateInstagramURL() CanonicalURL = %v, want %v", res.CanonicalURL, tt.wantCanonical)
			}
		})
	}
}
