package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAIClient_Chat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("expected /chat/completions, got %s", r.URL.Path)
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-api-key" {
			t.Errorf("expected 'Bearer test-api-key', got '%s'", auth)
		}
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		json.Unmarshal(body, &req)
		if req["model"] != "test-model" {
			t.Errorf("expected model 'test-model', got %v", req["model"])
		}
		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]string{
						"role":    "assistant",
						"content": "Hello from mock!",
					},
				},
			},
			"usage": map[string]int{
				"total_tokens": 42,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOpenAIClient("test-api-key", server.URL, "")
	resp, err := client.Chat(context.Background(), Request{
		Model: "test-model",
		Messages: []Message{
			{Role: "user", Content: "hello"},
		},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.Content != "Hello from mock!" {
		t.Errorf("expected 'Hello from mock!', got '%s'", resp.Content)
	}
	if resp.TokensUsed != 42 {
		t.Errorf("expected 42 tokens, got %d", resp.TokensUsed)
	}
	if resp.Model != "test-model" {
		t.Errorf("expected model 'test-model', got '%s'", resp.Model)
	}
}

func TestOpenAIClient_ChatError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error": {"message": "rate limited"}}`))
	}))
	defer server.Close()

	client := NewOpenAIClient("test-key", server.URL, "")
	_, err := client.Chat(context.Background(), Request{
		Model:    "test-model",
		Messages: []Message{{Role: "user", Content: "hello"}},
	})
	if err == nil {
		t.Error("expected error for 429 response, got nil")
	}
}

func TestOpenAIClient_DefaultBaseURL(t *testing.T) {
	client := NewOpenAIClient("key", "", "")
	if client.baseURL != "https://api.openai.com/v1" {
		t.Errorf("expected default base URL, got '%s'", client.baseURL)
	}
}
