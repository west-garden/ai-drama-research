// internal/llm/parse.go
package llm

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ExtractJSON extracts a JSON object from an LLM response that may contain
// surrounding text, markdown code blocks, or other non-JSON content.
// It finds the first '{' and last '}' and validates the result.
func ExtractJSON(text string) (string, error) {
	text = strings.TrimSpace(text)

	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start == -1 || end == -1 || end <= start {
		return "", fmt.Errorf("no JSON object found in response")
	}

	candidate := text[start : end+1]
	if !json.Valid([]byte(candidate)) {
		return "", fmt.Errorf("extracted text is not valid JSON")
	}
	return candidate, nil
}

// ParseJSON extracts JSON from an LLM response and unmarshals it into dst.
func ParseJSON(text string, dst any) error {
	jsonStr, err := ExtractJSON(text)
	if err != nil {
		return fmt.Errorf("extract JSON: %w", err)
	}
	if err := json.Unmarshal([]byte(jsonStr), dst); err != nil {
		return fmt.Errorf("unmarshal JSON: %w", err)
	}
	return nil
}
