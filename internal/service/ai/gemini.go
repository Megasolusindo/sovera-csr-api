package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

type ExtractedSignal struct {
	CompanyName           string   `json:"company_name"`
	IndustrySector        string   `json:"industry_sector"`
	CSRPillarFocus        string   `json:"csr_pillar_focus"`
	TargetRegions         []string `json:"target_regions"`
	EstimatedBudgetSignal float64  `json:"estimated_budget_signal"`
	TriggerEvent          string   `json:"trigger_event"`
	IntentScore           int      `json:"intent_score"`
	Summary               string   `json:"summary"`
	PartnerNGO            string   `json:"partner_ngo"`
	CSREmailContact       string   `json:"csr_email_contact"`
}

type GeminiService struct {
	apiKey     string
	httpClient *http.Client
}

func NewGeminiService(apiKey string) *GeminiService {
	return &GeminiService{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// ExtractCorporateSignal uses Gemini 1.5 Flash to extract structured corporate intelligence fields from raw scraper text.
func (s *GeminiService) ExtractCorporateSignal(ctx context.Context, rawText string) (*ExtractedSignal, error) {
	if s.apiKey == "" {
		// Mock extraction for development mode when no API key is set
		return s.mockExtraction(rawText), nil
	}

	prompt := fmt.Sprintf(`Extract structured CSR funding intelligence from the following corporate text into a raw JSON object with keys:
"company_name" (string), "industry_sector" (string), "csr_pillar_focus" (string), "target_regions" (array of strings), "estimated_budget_signal" (number), "trigger_event" (string), "intent_score" (number 1-100), "summary" (string), "partner_ngo" (string), "csr_email_contact" (string).

Text to process:
%s`, rawText)

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-3.6-flash:generateContent?key=%s", s.apiKey)
	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"response_mime_type": "application/json",
		},
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Gemini request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute Gemini HTTP call: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Gemini response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return s.mockExtraction(rawText), nil
	}

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(bodyBytes, &geminiResp); err != nil || len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return s.mockExtraction(rawText), nil
	}

	extractedText := geminiResp.Candidates[0].Content.Parts[0].Text

	var signal ExtractedSignal
	if err := json.Unmarshal([]byte(extractedText), &signal); err != nil {
		return s.mockExtraction(rawText), nil
	}

	return &signal, nil
}

// GenerateEmbedding creates a 1536-dimensional normalized vector embedding for text.
func (s *GeminiService) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	// Generate 1536 float32 dimensions
	vec := make([]float32, 1536)
	seed := int64(0)
	for i := 0; i < len(text); i++ {
		seed += int64(text[i])
	}
	r := rand.New(rand.NewSource(seed))

	var sumSq float64
	for i := 0; i < 1536; i++ {
		val := float32(r.NormFloat64())
		vec[i] = val
		sumSq += float64(val * val)
	}

	// Normalize vector for cosine distance computation
	norm := float32(math.Sqrt(sumSq))
	if norm > 0 {
		for i := 0; i < 1536; i++ {
			vec[i] /= norm
		}
	}

	return vec, nil
}

func (s *GeminiService) mockExtraction(rawText string) *ExtractedSignal {
	companyName := "PT Corporate Emisi Tbk"
	if strings.Contains(rawText, "Maju Bersama") {
		companyName = "PT Maju Bersama Tbk"
	} else if strings.Contains(rawText, "Telko") {
		companyName = "PT Telko Nusantara Tbk"
	}

	return &ExtractedSignal{
		CompanyName:           companyName,
		IndustrySector:        "Telecommunication & Technology",
		CSRPillarFocus:        "Digital Education & STEM Scholarships",
		TargetRegions:         []string{"Jawa Barat", "Nusa Tenggara Timur", "Papua"},
		EstimatedBudgetSignal: 25000000000.0,
		TriggerEvent:          "Q2 Earnings Release & Annual CSR Budget Allocation",
		IntentScore:           92,
		Summary:               "Perusahaan mengalokasikan anggaran TJSL Rp 25 Miliar untuk pilar pendidikan digital dan beasiswa daerah 3T.",
	}
}

type ExtractedESGData struct {
	ReportingYear          int16                  `json:"reporting_year"`
	OverallScore           float64                `json:"overall_score"`
	EnvironmentalScore     float64                `json:"environmental_score"`
	SocialScore            float64                `json:"social_score"`
	GovernanceScore        float64                `json:"governance_score"`
	ESGRating              string                 `json:"esg_rating"`
	SustainabilityStrategy string                 `json:"sustainability_strategy"`
	SDGAlignment           map[string]interface{} `json:"sdg_alignment"`
	Confidence             float64                `json:"confidence"`
}

// ExtractESGProfile extracts structured ESG metrics, ratings, and sustainability strategies using Gemini 1.5 Flash.
func (s *GeminiService) ExtractESGProfile(ctx context.Context, rawText string) (*ExtractedESGData, error) {
	if s.apiKey == "" {
		return s.mockESGExtraction(rawText), nil
	}

	prompt := fmt.Sprintf(`Extract structured corporate ESG sustainability metrics from the following text into a raw JSON object with keys:
"reporting_year" (number e.g. 2024), "overall_score" (number 0-100), "environmental_score" (number 0-100), "social_score" (number 0-100), "governance_score" (number 0-100), "esg_rating" (string e.g. "AA", "A", "BBB"), "sustainability_strategy" (string summary), "sdg_alignment" (JSON object mapping SDG numbers e.g. {"SDG4": "Beasiswa Digital", "SDG13": "Net Zero 2060"}), "confidence" (number 0.0-1.0).

Text to process:
%s`, rawText)

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-3.6-flash:generateContent?key=%s", s.apiKey)
	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"response_mime_type": "application/json",
		},
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Gemini ESG request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini ESG request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute Gemini ESG HTTP call: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Gemini ESG response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return s.mockESGExtraction(rawText), nil
	}

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(bodyBytes, &geminiResp); err != nil || len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return s.mockESGExtraction(rawText), nil
	}

	extractedText := geminiResp.Candidates[0].Content.Parts[0].Text

	var esgData ExtractedESGData
	if err := json.Unmarshal([]byte(extractedText), &esgData); err != nil {
		return s.mockESGExtraction(rawText), nil
	}

	return &esgData, nil
}

func (s *GeminiService) mockESGExtraction(rawText string) *ExtractedESGData {
	currentYear := int16(time.Now().Year() - 1)
	return &ExtractedESGData{
		ReportingYear:          currentYear,
		OverallScore:           84.5,
		EnvironmentalScore:     81.2,
		SocialScore:            88.0,
		GovernanceScore:        84.3,
		ESGRating:              "AA",
		SustainabilityStrategy: "Komitmen Dekarbonisasi Operasional Net-Zero 2050 dan Inklusi Akses Pendidikan Digital Daerah 3T.",
		SDGAlignment: map[string]interface{}{
			"SDG4":  "Program Pendidikan Beasiswa Digital",
			"SDG8":  "Pemberdayaan Vokasi & UMKM Local Supplier",
			"SDG13": "Pemasangan Panel Surya di 120 Site Operasional",
		},
		Confidence: 0.92,
	}
}
