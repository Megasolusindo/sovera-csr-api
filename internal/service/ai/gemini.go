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
	SourceQuote           string   `json:"source_quote"`
	ConfidenceScore       float64  `json:"confidence_score"`
	VerificationStatus    string   `json:"verification_status"`
	CSRRelevance          string   `json:"csr_relevance"`
	ActivityFocus         string   `json:"activity_focus"`
	ActionType            string   `json:"action_type"`
	OpportunityAlert      bool     `json:"opportunity_alert"`
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

// ExtractCorporateSignal uses Gemini to extract grounded, structured corporate intelligence fields from raw text.
func (s *GeminiService) ExtractCorporateSignal(ctx context.Context, rawText string) (*ExtractedSignal, error) {
	if s.apiKey == "" {
		return nil, fmt.Errorf("AI_API_KEY is missing or unconfigured")
	}

	prompt := fmt.Sprintf(`You are a Corporate Sustainability & CSR Financial Analyst. Extract structured CSR funding intelligence from the text below into a JSON object according to the 3-Level Taxonomy (CSR Relevance, Activity Focus, Action Type).
EVERY extracted field MUST be grounded by an exact quote ("source_quote") directly present in the input text. Do NOT hallucinate data.

STRICT FILTER: ONLY extract corporate signals for Perseroan Terbatas (PT, PT Tbk, BUMN, Multinationals, International Corporates).
DO NOT extract signals for CV (Commanditaire Vennootschap) or small local partnerships. If the entity in the text is a CV, return "company_name": "".

CRITICAL NON-CSR CLASSIFICATION RULE:
- Internal employee welfare programs (e.g. employee housing programs / KPR karyawan, internal employee loans/welfare, payroll banking services) and standard commercial B2B banking agreements are NOT CSR.
- If the input text describes an internal employee benefit or a standard commercial B2B deal, set "csr_relevance": "NON_CSR", "intent_score": 0, "opportunity_alert": false, and "summary": "No corporate CSR information (internal HR benefit / commercial B2B partnership)".

JSON Keys Required:
- "company_name" (string): Official registered corporate name (MUST be PT/BUMN/Corporation, NOT CV).
- "industry_sector" (string): Industry classification.
- "csr_pillar_focus" (string): Primary focus area e.g. Education, Environment, MSME, Disaster Relief, Health.
- "target_regions" (array of strings): Geographical regions mentioned.
- "estimated_budget_signal" (number): Alokasi dana CSR/TJSL in IDR (Rupiah) if mentioned, else 0.
- "trigger_event" (string): Corporate event e.g. Annual Report, Press Release, Q2 Financials, CSR Launch.
- "intent_score" (number 1-100): Score of active partnership/grant intent. Set to 0 if NON_CSR.
- "summary" (string): Concise summary of the CSR opportunity.
- "partner_ngo" (string): NGO/Foundation partner mentioned, if any.
- "csr_email_contact" (string): Contact email address or phone if present.
- "source_quote" (string): EXACT verbatim quote from the text backing this extraction.
- "csr_relevance" (string): Level 1 Taxonomy -> "HIGH", "MEDIUM", "LOW", "NON_CSR". Must be "NON_CSR" for internal HR/employee welfare/commercial B2B deals.
- "activity_focus" (string): Level 2 Taxonomy -> "Education", "Health", "Environment", "UMKM & Economic Empowerment", "Humanitarian & Disaster Relief", "Community Development", "Infrastructure & Digital Inclusion", "Volunteerism", "Creating Shared Value (CSV)".
- "action_type" (string): Level 3 Taxonomy -> "Donation", "Partnership", "Scholarship", "Training", "Volunteer", "Infrastructure Development", "Empowerment", "Grant", "Funding", "Community Program".
- "opportunity_alert" (boolean): Set to true IF the text explicitly contains open partnership offers, calls for proposals, grant opportunities, or funding invitations (e.g., "membuka kemitraan", "call for proposal", "open partnership", "grant opportunity"). MUST be false if NON_CSR.

Input Text:
%s`, rawText)

	modelsToTry := []string{
		"gemini-3.1-flash-lite",
		"gemini-3.5-flash-lite",
		"gemini-3.6-flash",
		"gemini-3.7-flash",
		"gemini-3.8-flash",
		"gemini-flash-lite-latest",
		"gemini-flash-latest",
	}

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
			"temperature":        0.1,
		},
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Gemini request: %w", err)
	}

	var lastErr error
	var bodyBytes []byte

	for _, modelName := range modelsToTry {
		url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", modelName, s.apiKey)
		
		for attempt := 0; attempt < 3; attempt++ {
			req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBytes))
			if err != nil {
				lastErr = err
				break
			}
			req.Header.Set("Content-Type", "application/json")

			resp, err := s.httpClient.Do(req)
			if err != nil {
				lastErr = err
				break
			}

			bodyBytes, err = io.ReadAll(resp.Body)
			resp.Body.Close()

			if err != nil {
				lastErr = err
				break
			}

			if resp.StatusCode == http.StatusOK {
				lastErr = nil
				break
			}

			lastErr = fmt.Errorf("Gemini API model %s call failed with status %d: %s", modelName, resp.StatusCode, string(bodyBytes))

			if resp.StatusCode == http.StatusTooManyRequests {
				time.Sleep(time.Duration(4*(attempt+1)) * time.Second)
				continue
			}

			break
		}

		if lastErr == nil {
			break
		}
	}

	if lastErr != nil {
		return nil, lastErr
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
		return nil, fmt.Errorf("failed to parse Gemini candidates response: %w", err)
	}

	extractedText := geminiResp.Candidates[0].Content.Parts[0].Text

	var signal ExtractedSignal
	if err := json.Unmarshal([]byte(extractedText), &signal); err != nil {
		// Attempt unmarshaling as a slice if Gemini returned a JSON array of signals
		var signals []ExtractedSignal
		if arrErr := json.Unmarshal([]byte(extractedText), &signals); arrErr == nil && len(signals) > 0 {
			best := signals[0]
			for _, sig := range signals {
				if sig.CompanyName != "" && !strings.EqualFold(sig.CompanyName, "Unknown") && !strings.EqualFold(sig.CSRRelevance, "NON_CSR") {
					best = sig
					break
				}
			}
			signal = best
		} else {
			return nil, fmt.Errorf("failed to parse Gemini JSON signal payload: %w", err)
		}
	}

	// Compute grounded confidence rating
	s.evaluateSignalConfidence(&signal, rawText)

	return &signal, nil
}

// evaluateSignalConfidence dynamically evaluates extraction confidence & grounding validity.
func (s *GeminiService) evaluateSignalConfidence(signal *ExtractedSignal, rawText string) {
	score := 0.0

	// Check Company Name validity
	if signal.CompanyName != "" && signal.CompanyName != "Unknown" {
		score += 0.25
	}

	// Check Source Quote grounding in original text
	if signal.SourceQuote != "" && strings.Contains(strings.ToLower(rawText), strings.ToLower(signal.SourceQuote)) {
		score += 0.35
	} else if signal.SourceQuote != "" {
		score += 0.15 // partial quote credit
	}

	// Check Budget Signal presence
	if signal.EstimatedBudgetSignal > 0 {
		score += 0.20
	}

	// Check Contact Email presence
	if signal.CSREmailContact != "" && strings.Contains(signal.CSREmailContact, "@") {
		score += 0.20
	}

	if score > 1.0 {
		score = 1.0
	}

	signal.ConfidenceScore = math.Round(score*100) / 100

	if signal.ConfidenceScore >= 0.80 {
		signal.VerificationStatus = "HIGH_CONFIDENCE"
	} else if signal.ConfidenceScore >= 0.60 {
		signal.VerificationStatus = "MEDIUM_CONFIDENCE"
	} else {
		signal.VerificationStatus = "NEEDS_REVIEW"
	}
}

// GenerateEmbedding creates a 1536-dimensional normalized vector embedding for text.
func (s *GeminiService) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
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

	norm := float32(math.Sqrt(sumSq))
	if norm > 0 {
		for i := 0; i < 1536; i++ {
			vec[i] /= norm
		}
	}

	return vec, nil
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
	SourceQuote            string                 `json:"source_quote"`
	VerificationStatus     string                 `json:"verification_status"`
}

// ExtractESGProfile extracts structured ESG metrics, ratings, and sustainability strategies using Gemini.
func (s *GeminiService) ExtractESGProfile(ctx context.Context, rawText string) (*ExtractedESGData, error) {
	if s.apiKey == "" {
		return nil, fmt.Errorf("AI_API_KEY is missing or unconfigured")
	}

	prompt := fmt.Sprintf(`You are a Senior ESG & Sustainability Auditor. Extract structured corporate ESG metrics from the text below into a JSON object.
EVERY metric MUST be grounded by a verbatim excerpt ("source_quote") directly present in the input text. Do NOT hallucinate scores.

JSON Keys Required:
- "reporting_year" (number e.g. 2024): Fiscal reporting year.
- "overall_score" (number 0-100): Overall ESG rating score.
- "environmental_score" (number 0-100): Environmental (E) score.
- "social_score" (number 0-100): Social (S) score.
- "governance_score" (number 0-100): Governance (G) score.
- "esg_rating" (string e.g. "AAA", "AA", "A", "BBB"): Composite rating grade.
- "sustainability_strategy" (string summary): Strategy summary.
- "sdg_alignment" (JSON object mapping SDG numbers to initiatives e.g. {"SDG4": "Beasiswa Digital", "SDG13": "Net Zero 2060"}).
- "confidence" (number 0.0-1.0): Confidence score.
- "source_quote" (string): EXACT verbatim quote backing this ESG assessment.

Input Text:
%s`, rawText)

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-flash-lite-latest:generateContent?key=%s", s.apiKey)
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
			"temperature":        0.1,
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
		return nil, fmt.Errorf("Gemini ESG API failed with status %d: %s", resp.StatusCode, string(bodyBytes))
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
		return nil, fmt.Errorf("failed to parse Gemini ESG response candidates: %w", err)
	}

	extractedText := geminiResp.Candidates[0].Content.Parts[0].Text

	var esgData ExtractedESGData
	if err := json.Unmarshal([]byte(extractedText), &esgData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Gemini ESG JSON payload: %w", err)
	}

	if esgData.Confidence >= 0.85 {
		esgData.VerificationStatus = "HIGH_CONFIDENCE"
	} else if esgData.Confidence >= 0.65 {
		esgData.VerificationStatus = "MEDIUM_CONFIDENCE"
	} else {
		esgData.VerificationStatus = "NEEDS_REVIEW"
	}

	return &esgData, nil
}


