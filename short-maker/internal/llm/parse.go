// internal/llm/parse.go
package llm

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
)

// stripThinkingTags removes <think>...</think> or <thinking>...</thinking> blocks
// that some models (e.g., Gemini 2.5 Flash) may include in their responses.
var thinkingTagRe = regexp.MustCompile(`(?s)<think(?:ing)?>.*?</think(?:ing)?>`)

// ExtractJSON extracts a JSON object from an LLM response that may contain
// surrounding text, markdown code blocks, thinking tags, or other non-JSON content.
func ExtractJSON(text string) (string, error) {
	text = strings.TrimSpace(text)

	// Strip thinking tags first
	text = thinkingTagRe.ReplaceAllString(text, "")
	text = strings.TrimSpace(text)

	// Strip markdown code fences (```json ... ``` or ``` ... ```)
	if idx := strings.Index(text, "```"); idx != -1 {
		// Find the content between the first ``` and the last ```
		afterFirst := text[idx+3:]
		// Skip language tag (e.g., "json\n")
		if nlIdx := strings.Index(afterFirst, "\n"); nlIdx != -1 {
			afterFirst = afterFirst[nlIdx+1:]
		}
		if lastIdx := strings.LastIndex(afterFirst, "```"); lastIdx != -1 {
			text = strings.TrimSpace(afterFirst[:lastIdx])
		}
	}

	// Find JSON by matching braces
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start == -1 || end == -1 || end <= start {
		return "", fmt.Errorf("no JSON object found in response (len=%d)", len(text))
	}

	candidate := text[start : end+1]
	if json.Valid([]byte(candidate)) {
		return candidate, nil
	}

	// Try to repair common LLM JSON issues (unescaped control chars in strings)
	repaired := repairJSON(candidate)
	if json.Valid([]byte(repaired)) {
		log.Printf("[parse] repaired invalid JSON (len=%d -> %d)", len(candidate), len(repaired))
		return repaired, nil
	}

	// Log a preview of the invalid content for debugging
	preview := candidate
	if len(preview) > 500 {
		preview = preview[:250] + "\n...[truncated]...\n" + preview[len(preview)-250:]
	}
	log.Printf("[parse] invalid JSON extracted (len=%d):\n%s", len(candidate), preview)
	return "", fmt.Errorf("extracted text is not valid JSON (len=%d)", len(candidate))
}

// repairJSON attempts to fix common JSON issues from LLM output:
// - Literal newlines/tabs inside string values (not escaped)
// - Control characters inside strings
func repairJSON(text string) string {
	var result strings.Builder
	result.Grow(len(text))

	inString := false
	escaped := false

	for i := 0; i < len(text); i++ {
		ch := text[i]

		if escaped {
			result.WriteByte(ch)
			escaped = false
			continue
		}

		if ch == '\\' && inString {
			result.WriteByte(ch)
			escaped = true
			continue
		}

		if ch == '"' {
			inString = !inString
			result.WriteByte(ch)
			continue
		}

		if inString {
			// Replace literal control characters inside strings
			switch ch {
			case '\n':
				result.WriteString(`\n`)
			case '\r':
				result.WriteString(`\r`)
			case '\t':
				result.WriteString(`\t`)
			default:
				if ch < 0x20 {
					// Skip other control characters
					continue
				}
				result.WriteByte(ch)
			}
		} else {
			result.WriteByte(ch)
		}
	}

	return result.String()
}

// ParseJSON extracts JSON from an LLM response and unmarshals it into dst.
func ParseJSON(text string, dst any) error {
	jsonStr, err := ExtractJSON(text)
	if err != nil {
		// Log response preview for debugging
		preview := text
		if len(preview) > 300 {
			preview = preview[:150] + "\n...[truncated]...\n" + preview[len(preview)-150:]
		}
		log.Printf("[parse] failed to extract JSON from LLM response (len=%d):\n%s", len(text), preview)
		return fmt.Errorf("extract JSON: %w", err)
	}
	if err := json.Unmarshal([]byte(jsonStr), dst); err != nil {
		return fmt.Errorf("unmarshal JSON: %w", err)
	}
	return nil
}
