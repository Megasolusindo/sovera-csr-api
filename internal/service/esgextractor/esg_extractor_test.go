package esgextractor

import (
	"context"
	"testing"

	"sovera-core-api/internal/service/ai"
	"sovera-core-api/internal/service/entityresolver"
	"sovera-core-api/internal/service/normalizer"
)

func TestProcessESGExtraction_Unconfigured(t *testing.T) {
	gemini := ai.NewGeminiService("") // Missing API key
	norm := normalizer.NewNormalizer()
	resolver := entityresolver.NewEntityResolver(nil)

	extractor := NewESGExtractor(gemini, nil, norm, resolver)

	compID := "comp_12345678"
	rawText := "PT Telkom Indonesia Laporan Keberlanjutan 2024. Skor ESG 84.5 dengan rating AA."

	_, err := extractor.ProcessESGExtraction(
		context.Background(),
		rawText, "",
		"PT Telkom Indonesia Tbk", "Telecommunication",
		&compID,
	)

	if err == nil {
		t.Fatalf("expected error due to missing API key, got nil")
	}
}
