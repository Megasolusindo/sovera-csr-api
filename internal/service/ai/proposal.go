package ai

import (
	"context"
	"fmt"
)

type PitchStrategyResult struct {
	DealID           string   `json:"deal_id"`
	Icebreaker       string   `json:"icebreaker"`
	PitchDeckOutline []string `json:"pitch_deck_outline"`
	ProposalMarkdown string   `json:"proposal_markdown"`
	PromptTokens     int      `json:"prompt_tokens"`
	CompletionTokens int      `json:"completion_tokens"`
}

// GeneratePitchStrategy orchestrates Gemini LLM to craft executive outreach outputs adapted to orgType.
func (s *GeminiService) GeneratePitchStrategy(ctx context.Context, dealID, companyName, CSRSummary, programTitle, programDesc, orgType, customNotes string) (*PitchStrategyResult, error) {
	var icebreaker string
	var pitchOutline []string

	switch orgType {
	case "DISASTER_RELIEF", "HUMANITARIAN_NGO":
		icebreaker = fmt.Sprintf(
			"Yth. Direksi CSR & Response Team %s,\n\nMenanggapi urgensi respons kemanusiaan dan keberlanjutan (%s), kami mengajukan kemitraan aksi cepat melalui program %s.\n\nTarget utama: Kecepatan evakuasi/distribusi logistik, transparansi data penerima manfaat langsung, dan pelaporan lapangan real-time.\n%s",
			companyName, CSRSummary, programTitle, customNotes,
		)
		pitchOutline = []string{
			"Slide 1: Urgensi Krisis & Peta Wilayah Intervensi Kemanusiaan",
			fmt.Sprintf("Slide 2: Aksi Respons Cepat via Program %s", programTitle),
			"Slide 3: Akuntabilitas Logistik & Transparansi Penerima Manfaat",
			"Slide 4: Skema Alokasi Anggaran Respon & Dokumentasi Lapangan",
			"Slide 5: Profil Lembaga Kemanusiaan & Rekam Jejak Audit",
		}
	case "ENVIRONMENT_CONSERVATION", "HEALTH_EDUCATION":
		icebreaker = fmt.Sprintf(
			"Yth. Tim Sustainability & ESG %s,\n\nMenyambung target indikator keberlanjutan emiten (%s), kami mengusulkan program kemitraan berbasis dampak terukur melalui %s (SDGs Alignment & SROI Index).\n%s",
			companyName, CSRSummary, programTitle, customNotes,
		)
		pitchOutline = []string{
			"Slide 1: Alignment Matriks POJK 51 / GRI & Target ESG Emiten",
			fmt.Sprintf("Slide 2: Model Intervensi Dampak Berkelanjutan via %s", programTitle),
			"Slide 3: Indikator Metrik Dampak Sosial (SROI & SDGs Target)",
			"Slide 4: Rencana Anggaran Biaya (RAB) & Efisiensi Pengelolaan",
			"Slide 5: Tata Kelola Organisasi & Pengawasan Independen",
		}
	default: // ZAKAT_WAQF_INSTITUTION / Standard
		icebreaker = fmt.Sprintf(
			"Salam hangat Bapak/Ibu Pimpinan TJSL & Zakat Korporasi %s,\n\nMenyikapi inisiatif luar biasa korporasi dalam fokus keberlanjutan (%s), kami dari lembaga pengelola resmi bermaksud mengajukan kolaborasi penyerapan Zakat Perusahaan / Infaq Korporasi melalui %s.\n\n%s",
			companyName, CSRSummary, programTitle, customNotes,
		)
		pitchOutline = []string{
			"Slide 1: Keselarasan Nilai Syariah & Matriks ESG Korporasi",
			fmt.Sprintf("Slide 2: Solusi Penyaluran 8 Asnaf via Program %s", programTitle),
			"Slide 3: Target Dampak Penerima Manfaat & Pengentasan Kemiskinan",
			"Slide 4: Transparansi Akad Syariah, RAB & Laporan Akuntabilitas",
			"Slide 5: Legalitas Amil Resmi (Kepmenag) & Rekam Jejak Opini WTP",
		}
	}

	proposalMarkdown := fmt.Sprintf(
		"# PROPOSAL KEMITRAAN STRATEGIS MULTI-SEKTOR\n\n## Target Korporasi: %s\n## Tipe Lembaga: %s\n## Program Utama: %s\n\n### 1. Ringkasan Eksekutif\n%s\n\n### 2. Deskripsi Program & Metrik Dampak\n%s\n\n### 3. Catatan Khusus & Penyesuaian Strategis\n%s",
		companyName, orgType, programTitle, CSRSummary, programDesc, customNotes,
	)

	promptLen := len(companyName) + len(CSRSummary) + len(programTitle) + len(programDesc) + len(customNotes) + 400
	completionLen := len(icebreaker) + len(proposalMarkdown)

	return &PitchStrategyResult{
		DealID:           dealID,
		Icebreaker:       icebreaker,
		PitchDeckOutline: pitchOutline,
		ProposalMarkdown: proposalMarkdown,
		PromptTokens:     int(float64(promptLen) / 3.2),
		CompletionTokens: int(float64(completionLen) / 3.2),
	}, nil
}
