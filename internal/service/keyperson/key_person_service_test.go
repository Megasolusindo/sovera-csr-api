package keyperson

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"sovera-core-api/internal/model"
)

func TestKeyPersonService_IngestSocialSignal_Actionable(t *testing.T) {
	svc := NewKeyPersonService(nil) // Testing standalone validation & keyword evaluation

	signal := &model.KeyPersonSocialSignal{
		CompanyID:       "comp-123",
		Platform:        model.PlatformLinkedIn,
		PostText:        "Program beasiswa CSR 2026",
		MatchedKeywords: []string{"beasiswa", "pendidikan"},
	}

	// Should fail on nil repo
	_, err := svc.IngestSocialSignal(context.Background(), signal)
	assert.Error(t, err)
	assert.Equal(t, true, signal.IsActionable)
}

func TestKeyPersonService_IngestSocialSignal_Validation(t *testing.T) {
	svc := NewKeyPersonService(nil)

	// Missing CompanyID
	_, err := svc.IngestSocialSignal(context.Background(), &model.KeyPersonSocialSignal{
		PostText: "Sample post",
	})
	assert.ErrorContains(t, err, "company_id is required")

	// Missing PostText
	_, err = svc.IngestSocialSignal(context.Background(), &model.KeyPersonSocialSignal{
		CompanyID: "comp-123",
	})
	assert.ErrorContains(t, err, "post_text is required")
}
