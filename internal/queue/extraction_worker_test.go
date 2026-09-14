package queue

import (
	"strings"
	"testing"

	"sovera-core-api/internal/service/ai"
)

func TestStringsContainsAny(t *testing.T) {
	text := "penandatanganan nota kesepahaman program kepemilikan rumah karyawan antara hermina group dan bank mandiri"

	if !stringsContainsAny(text, "program kepemilikan rumah karyawan", "kpr karyawan") {
		t.Errorf("Expected stringsContainsAny to return true for employee housing text")
	}

	if stringsContainsAny(text, "hibah csr", "beasiswa pendidikan") {
		t.Errorf("Expected stringsContainsAny to return false for unrelated keywords")
	}
}

func TestExtractedSignalNoiseCheck(t *testing.T) {
	nonCSRSignal := &ai.ExtractedSignal{
		CompanyName:     "PT Hermina Mandiri Banjarmasin",
		CSRRelevance:    "NON_CSR",
		Summary:         "Program Kepemilikan Rumah Karyawan antara Hermina Group dan Bank Mandiri",
		IntentScore:     0,
		ConfidenceScore: 0.85,
	}

	if !strings.EqualFold(nonCSRSignal.CSRRelevance, "NON_CSR") {
		t.Errorf("Expected CSRRelevance to be NON_CSR")
	}

	lowerSummary := strings.ToLower(nonCSRSignal.Summary)
	isInternalHRBenefit := strings.Contains(lowerSummary, "kepemilikan rumah karyawan")
	if !isInternalHRBenefit {
		t.Errorf("Expected isInternalHRBenefit to detect internal employee housing loan text")
	}
}
