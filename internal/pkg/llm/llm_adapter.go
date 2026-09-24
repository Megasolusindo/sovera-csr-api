package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"sovera-core-api/internal/service/ai"
)

// ProviderType identifies the LLM vendor
type ProviderType string

const (
	ProviderGemini    ProviderType = "GEMINI"
	ProviderOpenAI    ProviderType = "OPENAI"
	ProviderAnthropic ProviderType = "ANTHROPIC"
	ProviderOllama    ProviderType = "OLLAMA"
)

// LLMProvider interface abstracts multi-vendor LLM capabilities
type LLMProvider interface {
	Name() ProviderType
	ModelName() string
	GenerateText(ctx context.Context, prompt string) (string, error)
	GenerateStructuredJSON(ctx context.Context, prompt string, target interface{}) error
	EstimateCost(promptTokens, completionTokens int) float64
}

// ─── 1. Gemini Adapter ────────────────────────────────────────────────────────
type GeminiAdapter struct {
	apiKey    string
	modelName string
}

func NewGeminiAdapter(apiKey, modelName string) *GeminiAdapter {
	if modelName == "" {
		modelName = "gemini-1.5-flash"
	}
	return &GeminiAdapter{apiKey: apiKey, modelName: modelName}
}

func (g *GeminiAdapter) Name() ProviderType { return ProviderGemini }
func (g *GeminiAdapter) ModelName() string { return g.modelName }

func (g *GeminiAdapter) GenerateText(ctx context.Context, prompt string) (string, error) {
	if g.apiKey == "" {
		return "", fmt.Errorf("gemini api key is empty")
	}
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", g.modelName, g.apiKey)

	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{"text": prompt},
				},
			},
		},
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gemini api returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	var res struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(respBytes, &res); err != nil {
		return "", err
	}

	if len(res.Candidates) > 0 && len(res.Candidates[0].Content.Parts) > 0 {
		return res.Candidates[0].Content.Parts[0].Text, nil
	}

	return "", fmt.Errorf("empty response from gemini")
}

func (g *GeminiAdapter) GenerateStructuredJSON(ctx context.Context, prompt string, target interface{}) error {
	rawText, err := g.GenerateText(ctx, prompt)
	if err != nil {
		return err
	}
	repairedJSON := ai.CleanAndRepairJSON(rawText)
	if err := json.Unmarshal([]byte(repairedJSON), target); err != nil {
		return fmt.Errorf("%w: %v", ai.ErrExtractionFailed, err)
	}
	return nil
}

func (g *GeminiAdapter) EstimateCost(promptTokens, completionTokens int) float64 {
	// $0.075 per 1M prompt tokens, $0.30 per 1M completion tokens
	return (float64(promptTokens)*0.000000075) + (float64(completionTokens)*0.00000030)
}

// ─── 2. OpenAI Adapter ───────────────────────────────────────────────────────
type OpenAIAdapter struct {
	apiKey    string
	modelName string
}

func NewOpenAIAdapter(apiKey, modelName string) *OpenAIAdapter {
	if modelName == "" {
		modelName = "gpt-4o-mini"
	}
	return &OpenAIAdapter{apiKey: apiKey, modelName: modelName}
}

func (o *OpenAIAdapter) Name() ProviderType { return ProviderOpenAI }
func (o *OpenAIAdapter) ModelName() string { return o.modelName }

func (o *OpenAIAdapter) GenerateText(ctx context.Context, prompt string) (string, error) {
	if o.apiKey == "" {
		return "", fmt.Errorf("openai api key is empty")
	}
	reqBody := map[string]interface{}{
		"model": o.modelName,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("openai api status %d: %s", resp.StatusCode, string(respBytes))
	}

	var res struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(respBytes, &res); err != nil {
		return "", err
	}
	if len(res.Choices) > 0 {
		return res.Choices[0].Message.Content, nil
	}
	return "", fmt.Errorf("empty choices from openai")
}

func (o *OpenAIAdapter) GenerateStructuredJSON(ctx context.Context, prompt string, target interface{}) error {
	raw, err := o.GenerateText(ctx, prompt)
	if err != nil {
		return err
	}
	repaired := ai.CleanAndRepairJSON(raw)
	return json.Unmarshal([]byte(repaired), target)
}

func (o *OpenAIAdapter) EstimateCost(promptTokens, completionTokens int) float64 {
	return (float64(promptTokens)*0.00000015) + (float64(completionTokens)*0.00000060)
}

// ─── 3. Anthropic Adapter ────────────────────────────────────────────────────
type AnthropicAdapter struct {
	apiKey    string
	modelName string
}

func NewAnthropicAdapter(apiKey, modelName string) *AnthropicAdapter {
	if modelName == "" {
		modelName = "claude-3-5-sonnet-20241022"
	}
	return &AnthropicAdapter{apiKey: apiKey, modelName: modelName}
}

func (a *AnthropicAdapter) Name() ProviderType { return ProviderAnthropic }
func (a *AnthropicAdapter) ModelName() string { return a.modelName }

func (a *AnthropicAdapter) GenerateText(ctx context.Context, prompt string) (string, error) {
	if a.apiKey == "" {
		return "", fmt.Errorf("anthropic api key is empty")
	}
	reqBody := map[string]interface{}{
		"model":      a.modelName,
		"max_tokens": 2048,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("anthropic api status %d: %s", resp.StatusCode, string(respBytes))
	}

	var res struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.Unmarshal(respBytes, &res); err != nil {
		return "", err
	}
	if len(res.Content) > 0 {
		return res.Content[0].Text, nil
	}
	return "", fmt.Errorf("empty response from anthropic")
}

func (a *AnthropicAdapter) GenerateStructuredJSON(ctx context.Context, prompt string, target interface{}) error {
	raw, err := a.GenerateText(ctx, prompt)
	if err != nil {
		return err
	}
	repaired := ai.CleanAndRepairJSON(raw)
	return json.Unmarshal([]byte(repaired), target)
}

func (a *AnthropicAdapter) EstimateCost(promptTokens, completionTokens int) float64 {
	return (float64(promptTokens)*0.00000300) + (float64(completionTokens)*0.00001500)
}

// ─── 4. Ollama Fallback Adapter ──────────────────────────────────────────────
type OllamaAdapter struct {
	baseURL   string
	modelName string
}

func NewOllamaAdapter(baseURL, modelName string) *OllamaAdapter {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if modelName == "" {
		modelName = "llama3.3"
	}
	return &OllamaAdapter{baseURL: strings.TrimSuffix(baseURL, "/"), modelName: modelName}
}

func (o *OllamaAdapter) Name() ProviderType { return ProviderOllama }
func (o *OllamaAdapter) ModelName() string { return o.modelName }

func (o *OllamaAdapter) GenerateText(ctx context.Context, prompt string) (string, error) {
	reqBody := map[string]interface{}{
		"model":  o.modelName,
		"prompt": prompt,
		"stream": false,
	}
	bodyBytes, _ := json.Marshal(reqBody)
	url := fmt.Sprintf("%s/api/generate", o.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama status %d: %s", resp.StatusCode, string(respBytes))
	}

	var res struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(respBytes, &res); err != nil {
		return "", err
	}
	return res.Response, nil
}

func (o *OllamaAdapter) GenerateStructuredJSON(ctx context.Context, prompt string, target interface{}) error {
	raw, err := o.GenerateText(ctx, prompt)
	if err != nil {
		return err
	}
	repaired := ai.CleanAndRepairJSON(raw)
	if err := json.Unmarshal([]byte(repaired), target); err != nil {
		return fmt.Errorf("%w: %v", ai.ErrExtractionFailed, err)
	}
	return nil
}

func (o *OllamaAdapter) EstimateCost(promptTokens, completionTokens int) float64 {
	return 0.0 // Local compute is $0 USD API cost
}

// ─── 5. Multi-Provider Fallback Client ───────────────────────────────────────
type MultiProviderClient struct {
	providers []LLMProvider
}

func NewMultiProviderClient(providers ...LLMProvider) *MultiProviderClient {
	return &MultiProviderClient{providers: providers}
}

func (m *MultiProviderClient) GenerateTextWithFallback(ctx context.Context, prompt string) (string, LLMProvider, error) {
	if len(m.providers) == 0 {
		return "", nil, fmt.Errorf("no LLM providers configured in fallback chain")
	}

	var lastErr error
	for _, p := range m.providers {
		res, err := p.GenerateText(ctx, prompt)
		if err == nil {
			return res, p, nil
		}
		lastErr = err
	}

	return "", nil, fmt.Errorf("all LLM providers in fallback chain failed. Last error: %w", lastErr)
}
