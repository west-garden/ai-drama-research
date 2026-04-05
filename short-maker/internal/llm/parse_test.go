// internal/llm/parse_test.go
package llm

import "testing"

func TestExtractJSON_PureJSON(t *testing.T) {
	input := `{"name": "test", "value": 42}`
	got, err := ExtractJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != input {
		t.Errorf("expected %q, got %q", input, got)
	}
}

func TestExtractJSON_MarkdownCodeBlock(t *testing.T) {
	input := "Here is the result:\n```json\n{\"name\": \"test\"}\n```\nDone."
	got, err := ExtractJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := `{"name": "test"}`
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestExtractJSON_TextAroundJSON(t *testing.T) {
	input := "Analysis complete. {\"world_view\": \"fantasy\"} End of response."
	got, err := ExtractJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := `{"world_view": "fantasy"}`
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestExtractJSON_NestedBraces(t *testing.T) {
	input := `{"outer": {"inner": "value"}}`
	got, err := ExtractJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != input {
		t.Errorf("expected %q, got %q", input, got)
	}
}

func TestExtractJSON_ThinkingTags(t *testing.T) {
	input := "<think>\nLet me analyze this script. The world has {complex} elements.\n</think>\n{\"world_view\": \"fantasy\"}"
	got, err := ExtractJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := `{"world_view": "fantasy"}`
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestExtractJSON_UnescapedNewlines(t *testing.T) {
	// LLMs sometimes output literal newlines inside JSON string values
	input := "{\"prompt\": \"A girl stands\nin the rain\"}"
	got, err := ExtractJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := `{"prompt": "A girl stands\nin the rain"}`
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestExtractJSON_NoJSON(t *testing.T) {
	input := "This response has no JSON at all."
	_, err := ExtractJSON(input)
	if err == nil {
		t.Error("expected error for input with no JSON, got nil")
	}
}

func TestExtractJSON_InvalidJSON(t *testing.T) {
	input := `{"broken": "json`
	_, err := ExtractJSON(input)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}
