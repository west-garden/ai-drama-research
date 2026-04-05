package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// OpenAIClient implements the Client interface using the OpenAI-compatible
// chat completions API. Works with OpenAI, DeepSeek, Together, and any
// provider that implements the same API format.
type OpenAIClient struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
}

// NewOpenAIClient creates a client for an OpenAI-compatible API.
// If baseURL is empty, defaults to "https://api.openai.com/v1".
// If proxyURL is non-empty, HTTP requests are routed through that proxy.
func NewOpenAIClient(apiKey, baseURL, proxyURL string) *OpenAIClient {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if proxyURL != "" {
		if u, err := url.Parse(proxyURL); err == nil {
			transport.Proxy = http.ProxyURL(u)
		}
	}
	return &OpenAIClient{
		httpClient: &http.Client{Timeout: 120 * time.Second, Transport: transport},
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
	}
}

type openaiRequest struct {
	Model       string          `json:"model"`
	Messages    []openaiMessage `json:"messages"`
	Temperature float64         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
}

type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openaiResponse struct {
	Choices []struct {
		Message openaiMessage `json:"message"`
	} `json:"choices"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

func (c *OpenAIClient) Chat(ctx context.Context, req Request) (*Response, error) {
	messages := make([]openaiMessage, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = openaiMessage{Role: m.Role, Content: m.Content}
	}

	oaiReq := openaiRequest{
		Model:       req.Model,
		Messages:    messages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	body, err := json.Marshal(oaiReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", httpResp.StatusCode, string(respBody))
	}

	var oaiResp openaiResponse
	if err := json.Unmarshal(respBody, &oaiResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if len(oaiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &Response{
		Content:    oaiResp.Choices[0].Message.Content,
		TokensUsed: oaiResp.Usage.TotalTokens,
		Model:      req.Model,
	}, nil
}
