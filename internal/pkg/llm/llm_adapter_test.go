package llm

import (
	"context"
	"fmt"
	"testing"
)

type MockSuccessProvider struct {
	name ProviderType
}

func (m *MockSuccessProvider) Name() ProviderType { return m.name }
func (m *MockSuccessProvider) ModelName() string  { return "mock-model" }
func (m *MockSuccessProvider) GenerateText(ctx context.Context, prompt string) (string, error) {
	return "mock response: " + prompt, nil
}
func (m *MockSuccessProvider) GenerateStructuredJSON(ctx context.Context, prompt string, target interface{}) error {
	return nil
}
func (m *MockSuccessProvider) EstimateCost(promptTokens, completionTokens int) float64 {
	return 0.001
}

type MockFailProvider struct {
	name ProviderType
}

func (m *MockFailProvider) Name() ProviderType { return m.name }
func (m *MockFailProvider) ModelName() string  { return "mock-fail" }
func (m *MockFailProvider) GenerateText(ctx context.Context, prompt string) (string, error) {
	return "", fmt.Errorf("simulated provider error 500")
}
func (m *MockFailProvider) GenerateStructuredJSON(ctx context.Context, prompt string, target interface{}) error {
	return fmt.Errorf("simulated provider error")
}
func (m *MockFailProvider) EstimateCost(promptTokens, completionTokens int) float64 {
	return 0.0
}

func TestMultiProviderClient_FallbackChain(t *testing.T) {
	failP := &MockFailProvider{name: ProviderGemini}
	successP := &MockSuccessProvider{name: ProviderOpenAI}

	client := NewMultiProviderClient(failP, successP)

	res, usedProvider, err := client.GenerateTextWithFallback(context.Background(), "hello world")
	if err != nil {
		t.Fatalf("expected fallback to succeed, got error: %v", err)
	}

	if usedProvider.Name() != ProviderOpenAI {
		t.Errorf("expected fallback provider OpenAI, got %s", usedProvider.Name())
	}

	if res != "mock response: hello world" {
		t.Errorf("unexpected response text: %s", res)
	}
}
