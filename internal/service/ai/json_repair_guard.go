package ai

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	ErrExtractionFailed = errors.New("EXTRACTION_FAILED: secondary LLM output violated JSON schema and could not be safely repaired")
)

// CleanAndRepairJSON removes markdown fences, unescaped whitespace, and trailing commas from raw LLM string output
func CleanAndRepairJSON(rawOutput string) string {
	cleaned := strings.TrimSpace(rawOutput)

	// Remove markdown fences ```json ... ``` or ``` ... ```
	reFence := regexp.MustCompile("(?s)^\\s*```(?:json)?\\s*(.*?)\\s*```\\s*$")
	if matches := reFence.FindStringSubmatch(cleaned); len(matches) > 1 {
		cleaned = strings.TrimSpace(matches[1])
	}

	// Remove trailing commas before closing braces/brackets (e.g. , } -> })
	reTrailingComma := regexp.MustCompile(`,[ \t\r\n]*([\}\]])`)
	cleaned = reTrailingComma.ReplaceAllString(cleaned, "$1")

	return cleaned
}

// ParseAndValidateJSON parses raw LLM JSON output into target struct with explicit failure guardrail (§9.3)
func ParseAndValidateJSON[T any](rawOutput string) (*T, error) {
	repairedJSON := CleanAndRepairJSON(rawOutput)

	var target T
	if err := json.Unmarshal([]byte(repairedJSON), &target); err != nil {
		// Strict No Dummy Fallback Directive: Explicitly return EXTRACTION_FAILED error
		return nil, fmt.Errorf("%w: %v (raw JSON snippet: %.100s)", ErrExtractionFailed, err, repairedJSON)
	}

	return &target, nil
}
