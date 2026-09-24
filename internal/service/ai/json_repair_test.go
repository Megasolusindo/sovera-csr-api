package ai

import (
	"errors"
	"testing"
)

type SampleClaimSchema struct {
	CompanyName string  `json:"company_name"`
	Budget      float64 `json:"budget"`
	Pillar      string  `json:"pillar"`
}

func TestCleanAndRepairJSON_MarkdownFence(t *testing.T) {
	raw := "```json\n{\n  \"company_name\": \"PT BSI\",\n  \"budget\": 150000000.0,\n  \"pillar\": \"Ekonomi\",\n}\n```"

	parsed, err := ParseAndValidateJSON[SampleClaimSchema](raw)
	if err != nil {
		t.Fatalf("failed to repair valid JSON with markdown fence and trailing comma: %v", err)
	}

	if parsed.CompanyName != "PT BSI" || parsed.Budget != 150000000.0 || parsed.Pillar != "Ekonomi" {
		t.Errorf("parsed fields mismatch: %+v", parsed)
	}
}

func TestParseAndValidateJSON_ExplicitFailureOnInvalid(t *testing.T) {
	invalidRaw := "This is an unparseable response from LLM that is not JSON."

	_, err := ParseAndValidateJSON[SampleClaimSchema](invalidRaw)
	if !errors.Is(err, ErrExtractionFailed) {
		t.Fatalf("expected ErrExtractionFailed on broken JSON, got: %v", err)
	}
}
