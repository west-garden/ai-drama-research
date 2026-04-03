# Plan 2: Story Understanding Agent + Character Management Agent

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the first two mock agents with real LLM-powered agents — Story Understanding and Character Management — so that a script input produces a structured StoryBlueprint and character asset records with visual descriptions.

**Architecture:** Each agent receives dependencies (llm.Client, store.Store) via constructor. Agent core logic: build prompt → call LLM → parse structured JSON response → write to PipelineState. An OpenAI-compatible HTTP adapter implements the llm.Client interface, supporting any OpenAI-API-compatible provider (OpenAI, DeepSeek, Together, etc.).

**Tech Stack:** Go 1.22+, existing internal packages from Plan 1, `net/http` for OpenAI adapter, `httptest` for testing

**Spec reference:** `short-maker/docs/specs/2026-04-03-core-pipeline-design.md` — Sections 三 (阶段 1, 2), 四 (资产库)

**Depends on:** Plan 1 (complete) — domain types, Agent interface, Orchestrator, Store, LLM client interface, CLI

---

## File Structure

```
short-maker/
├── internal/
│   ├── agent/
│   │   ├── story.go               # StoryAgent — script → StoryBlueprint via LLM
│   │   ├── story_test.go
│   │   ├── character.go            # CharacterAgent — blueprint → character assets via LLM
│   │   ├── character_test.go
│   │   └── integration_test.go     # (modify) add real-agent integration test
│   └── llm/
│       ├── openai.go              # OpenAI-compatible HTTP adapter
│       ├── openai_test.go
│       ├── parse.go               # JSON extraction from LLM responses
│       └── parse_test.go
├── cmd/
│   └── shortmaker/
│       └── main.go                # (modify) wire real agents, add LLM config flags
└── testdata/
    └── sample-script.txt          # (exists from Plan 1)
```

---

### Task 1: JSON Extraction Utility

**Files:**
- Create: `short-maker/internal/llm/parse.go`
- Create: `short-maker/internal/llm/parse_test.go`

- [ ] **Step 1: Write tests for ExtractJSON**

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./internal/llm/ -v -run TestExtractJSON`
Expected: compilation error — ExtractJSON not defined

- [ ] **Step 3: Implement ExtractJSON**

```go
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./internal/llm/ -v -run TestExtractJSON`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add short-maker/internal/llm/parse.go short-maker/internal/llm/parse_test.go
git commit -m "feat(short-maker): JSON extraction utility for LLM responses"
```

---

### Task 2: OpenAI-Compatible LLM Adapter

**Files:**
- Create: `short-maker/internal/llm/openai.go`
- Create: `short-maker/internal/llm/openai_test.go`

- [ ] **Step 1: Write tests for OpenAIClient**

```go
// internal/llm/openai_test.go
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
		// Verify method and path
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("expected /chat/completions, got %s", r.URL.Path)
		}

		// Verify auth header
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-api-key" {
			t.Errorf("expected 'Bearer test-api-key', got '%s'", auth)
		}

		// Verify request body
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		json.Unmarshal(body, &req)
		if req["model"] != "test-model" {
			t.Errorf("expected model 'test-model', got %v", req["model"])
		}

		// Return canned response
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

	client := NewOpenAIClient("test-api-key", server.URL)
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

	client := NewOpenAIClient("test-key", server.URL)
	_, err := client.Chat(context.Background(), Request{
		Model:    "test-model",
		Messages: []Message{{Role: "user", Content: "hello"}},
	})
	if err == nil {
		t.Error("expected error for 429 response, got nil")
	}
}

func TestOpenAIClient_DefaultBaseURL(t *testing.T) {
	client := NewOpenAIClient("key", "")
	if client.baseURL != "https://api.openai.com/v1" {
		t.Errorf("expected default base URL, got '%s'", client.baseURL)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./internal/llm/ -v -run TestOpenAI`
Expected: compilation error — NewOpenAIClient not defined

- [ ] **Step 3: Implement OpenAIClient**

```go
// internal/llm/openai.go
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
func NewOpenAIClient(apiKey, baseURL string) *OpenAIClient {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &OpenAIClient{
		httpClient: &http.Client{Timeout: 120 * time.Second},
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./internal/llm/ -v -run TestOpenAI`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add short-maker/internal/llm/openai.go short-maker/internal/llm/openai_test.go
git commit -m "feat(short-maker): OpenAI-compatible LLM adapter with HTTP client"
```

---

### Task 3: Story Understanding Agent

**Files:**
- Create: `short-maker/internal/agent/story.go`
- Create: `short-maker/internal/agent/story_test.go`

- [ ] **Step 1: Write tests for StoryAgent**

```go
// internal/agent/story_test.go
package agent

import (
	"context"
	"testing"

	"github.com/west-garden/short-maker/internal/domain"
	"github.com/west-garden/short-maker/internal/llm"
)

const sampleStoryResponse = `{
	"world_view": "西游记神话世界，充满妖魔鬼怪和仙法奇术",
	"characters": [
		{
			"name": "孙悟空",
			"description": "齐天大圣，被压五行山五百年后被唐僧解救",
			"traits": ["好斗", "忠诚", "机智"]
		},
		{
			"name": "唐僧",
			"description": "取经人，慈悲为怀的高僧",
			"traits": ["慈悲", "坚定", "善良"]
		}
	],
	"episodes": [
		{
			"number": 1,
			"role": "hook",
			"emotion_arc": "从平静到惊奇",
			"synopsis": "唐僧途经五行山，解救孙悟空",
			"scenes": [
				{
					"narrative_beat": "开场引入",
					"emotion_arc": "平静→好奇",
					"setting": "五行山脚下",
					"pacing": "slow",
					"character_count": 1
				},
				{
					"narrative_beat": "关键相遇",
					"emotion_arc": "惊奇→激动",
					"setting": "五行山顶",
					"pacing": "fast",
					"character_count": 2
				}
			]
		},
		{
			"number": 2,
			"role": "hook",
			"emotion_arc": "从混乱到和解",
			"synopsis": "孙悟空获自由后大闹，唐僧用紧箍咒制服",
			"scenes": [
				{
					"narrative_beat": "冲突爆发",
					"emotion_arc": "混乱→愤怒",
					"setting": "山林间",
					"pacing": "fast",
					"character_count": 2
				}
			]
		}
	],
	"relationships": [
		{
			"character_a": "孙悟空",
			"character_b": "唐僧",
			"type": "师徒"
		}
	]
}`

func TestStoryAgent_Run(t *testing.T) {
	mockLLM := llm.NewMockClient()
	mockLLM.SetDefaultResponse(sampleStoryResponse)

	agent := NewStoryAgent(mockLLM, "test-model")
	project := domain.NewProject("西游记测试", domain.StyleManga, 2)
	state := NewPipelineState(project, "第一集：初遇\n孙悟空从五行山下被唐僧解救。\n第二集：收服\n唐僧用紧箍咒制服孙悟空。")

	result, err := agent.Run(context.Background(), state)
	if err != nil {
		t.Fatalf("StoryAgent.Run: %v", err)
	}

	bp := result.Blueprint
	if bp == nil {
		t.Fatal("expected Blueprint to be set")
	}
	if bp.ProjectID != project.ID {
		t.Errorf("expected projectID '%s', got '%s'", project.ID, bp.ProjectID)
	}
	if bp.WorldView != "西游记神话世界，充满妖魔鬼怪和仙法奇术" {
		t.Errorf("unexpected world_view: %s", bp.WorldView)
	}
	if len(bp.Characters) != 2 {
		t.Fatalf("expected 2 characters, got %d", len(bp.Characters))
	}
	if bp.Characters[0].Name != "孙悟空" {
		t.Errorf("expected first character '孙悟空', got '%s'", bp.Characters[0].Name)
	}
	if bp.Characters[0].ID == "" {
		t.Error("expected character to have an ID")
	}
	if len(bp.Episodes) != 2 {
		t.Fatalf("expected 2 episodes, got %d", len(bp.Episodes))
	}
	if bp.Episodes[0].Role != domain.EpisodeRoleHook {
		t.Errorf("expected episode 1 role 'hook', got '%s'", bp.Episodes[0].Role)
	}
	if len(bp.Episodes[0].Scenes) != 2 {
		t.Errorf("expected 2 scenes in episode 1, got %d", len(bp.Episodes[0].Scenes))
	}
	if len(bp.Relationships) != 1 {
		t.Errorf("expected 1 relationship, got %d", len(bp.Relationships))
	}
}

func TestStoryAgent_Phase(t *testing.T) {
	agent := NewStoryAgent(llm.NewMockClient(), "model")
	if agent.Phase() != PhaseStoryUnderstanding {
		t.Errorf("expected phase story_understanding, got %s", agent.Phase())
	}
}

func TestStoryAgent_LLMCalledWithCorrectPrompt(t *testing.T) {
	mockLLM := llm.NewMockClient()
	mockLLM.SetDefaultResponse(sampleStoryResponse)

	agent := NewStoryAgent(mockLLM, "gpt-4o")
	project := domain.NewProject("测试剧", domain.StyleManga, 5)
	state := NewPipelineState(project, "测试剧本内容")

	agent.Run(context.Background(), state)

	calls := mockLLM.Calls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 LLM call, got %d", len(calls))
	}
	if calls[0].Model != "gpt-4o" {
		t.Errorf("expected model 'gpt-4o', got '%s'", calls[0].Model)
	}
	// System prompt should mention JSON output
	if len(calls[0].Messages) < 2 {
		t.Fatal("expected at least 2 messages (system + user)")
	}
	if calls[0].Messages[0].Role != "system" {
		t.Errorf("expected first message role 'system', got '%s'", calls[0].Messages[0].Role)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./internal/agent/ -v -run TestStoryAgent`
Expected: compilation error — NewStoryAgent not defined

- [ ] **Step 3: Implement StoryAgent**

```go
// internal/agent/story.go
package agent

import (
	"context"
	"fmt"

	"github.com/west-garden/short-maker/internal/domain"
	"github.com/west-garden/short-maker/internal/llm"
)

// StoryAgent analyzes a script via LLM and produces a StoryBlueprint.
// Implements the Agent interface for PhaseStoryUnderstanding.
type StoryAgent struct {
	llmClient llm.Client
	model     string
}

func NewStoryAgent(llmClient llm.Client, model string) *StoryAgent {
	return &StoryAgent{llmClient: llmClient, model: model}
}

func (a *StoryAgent) Phase() Phase { return PhaseStoryUnderstanding }

func (a *StoryAgent) Run(ctx context.Context, state *PipelineState) (*PipelineState, error) {
	systemPrompt := buildStorySystemPrompt()
	userPrompt := buildStoryUserPrompt(state.Script, string(state.Project.Style), state.Project.EpisodeCount)

	resp, err := a.llmClient.Chat(ctx, llm.Request{
		Model: a.model,
		Messages: []llm.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.3,
		MaxTokens:   4096,
	})
	if err != nil {
		return nil, fmt.Errorf("story LLM call: %w", err)
	}

	var parsed storyAnalysisResponse
	if err := llm.ParseJSON(resp.Content, &parsed); err != nil {
		return nil, fmt.Errorf("parse story response: %w", err)
	}

	bp := convertToBlueprint(state.Project.ID, &parsed)
	state.Blueprint = bp
	return state, nil
}

// --- Prompt builders ---

func buildStorySystemPrompt() string {
	return `You are a script analysis agent for an AI short drama production system.
Analyze the provided script and output a structured JSON representation.

Output ONLY valid JSON with this exact schema (no other text):
{
  "world_view": "brief description of the world/setting",
  "characters": [
    {
      "name": "character name",
      "description": "role and background description",
      "traits": ["personality_trait_1", "personality_trait_2"]
    }
  ],
  "episodes": [
    {
      "number": 1,
      "role": "hook|paywall|climax|transition",
      "emotion_arc": "emotional progression description",
      "synopsis": "episode synopsis",
      "scenes": [
        {
          "narrative_beat": "beat description",
          "emotion_arc": "scene emotional arc",
          "setting": "location/setting",
          "pacing": "fast|medium|slow",
          "character_count": 2
        }
      ]
    }
  ],
  "relationships": [
    {
      "character_a": "character name",
      "character_b": "character name",
      "type": "relationship type"
    }
  ]
}

Episode role rules:
- "hook": first 3-5 episodes that must grab viewer attention
- "paywall": episodes 8-12 where free-to-paid conversion happens
- "climax": major turning points and finale
- "transition": everything else (setup, daily life, subplots)

Requirements:
- Every episode must have at least one scene
- Traits should be personality traits, not physical descriptions
- Scene pacing: "fast" for action/conflict, "slow" for setup/emotion, "medium" for dialogue`
}

func buildStoryUserPrompt(script, style string, episodeCount int) string {
	return fmt.Sprintf(`Analyze this script for a %s short drama with %d episodes:

%s`, style, episodeCount, script)
}

// --- Response parsing types ---

type storyAnalysisResponse struct {
	WorldView     string                 `json:"world_view"`
	Characters    []characterResponse    `json:"characters"`
	Episodes      []episodeResponse      `json:"episodes"`
	Relationships []relationshipResponse `json:"relationships"`
}

type characterResponse struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Traits      []string `json:"traits"`
}

type episodeResponse struct {
	Number     int             `json:"number"`
	Role       string          `json:"role"`
	EmotionArc string          `json:"emotion_arc"`
	Synopsis   string          `json:"synopsis"`
	Scenes     []sceneResponse `json:"scenes"`
}

type sceneResponse struct {
	NarrativeBeat  string `json:"narrative_beat"`
	EmotionArc     string `json:"emotion_arc"`
	Setting        string `json:"setting"`
	Pacing         string `json:"pacing"`
	CharacterCount int    `json:"character_count"`
}

type relationshipResponse struct {
	CharacterA string `json:"character_a"`
	CharacterB string `json:"character_b"`
	Type       string `json:"type"`
}

// --- Converter ---

func convertToBlueprint(projectID string, resp *storyAnalysisResponse) *domain.StoryBlueprint {
	bp := domain.NewStoryBlueprint(projectID)
	bp.WorldView = resp.WorldView

	for _, ch := range resp.Characters {
		bp.AddCharacter(ch.Name, ch.Description, ch.Traits)
	}

	for _, ep := range resp.Episodes {
		role := domain.EpisodeRole(ep.Role)
		epBP := bp.AddEpisodeBlueprintWithRole(ep.Number, role, ep.EmotionArc)
		epBP.Synopsis = ep.Synopsis
		for _, sc := range ep.Scenes {
			epBP.Scenes = append(epBP.Scenes, domain.SceneTag{
				NarrativeBeat:  sc.NarrativeBeat,
				EmotionArc:     sc.EmotionArc,
				Setting:        sc.Setting,
				Pacing:         sc.Pacing,
				CharacterCount: sc.CharacterCount,
			})
		}
	}

	for _, rel := range resp.Relationships {
		bp.Relationships = append(bp.Relationships, domain.Relationship{
			CharacterA: rel.CharacterA,
			CharacterB: rel.CharacterB,
			Type:       rel.Type,
		})
	}

	return bp
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./internal/agent/ -v -run TestStoryAgent`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add short-maker/internal/agent/story.go short-maker/internal/agent/story_test.go
git commit -m "feat(short-maker): Story Understanding Agent — script analysis via LLM"
```

---

### Task 4: Character Management Agent

**Files:**
- Create: `short-maker/internal/agent/character.go`
- Create: `short-maker/internal/agent/character_test.go`

- [ ] **Step 1: Write tests for CharacterAgent**

```go
// internal/agent/character_test.go
package agent

import (
	"context"
	"testing"

	"github.com/west-garden/short-maker/internal/domain"
	"github.com/west-garden/short-maker/internal/llm"
	"github.com/west-garden/short-maker/internal/store"
)

const sampleCharacterResponse = `{
	"visual_prompt": "金色毛发的猴王战士，身穿华丽金甲红披风，头戴金箍，手持金箍棒，体格矫健，眼神锐利而调皮",
	"appearance": {
		"face": "棱角分明，金色锐利双眸，尖耳，调皮的笑容",
		"body": "精瘦有力，矫健身姿，约170cm",
		"clothing": "金色战甲，红色披风，虎皮裙",
		"distinctive_features": ["金箍", "金箍棒", "筋斗云靴"]
	}
}`

func TestCharacterAgent_Run(t *testing.T) {
	mockLLM := llm.NewMockClient()
	mockLLM.SetDefaultResponse(sampleCharacterResponse)

	tmpDir := t.TempDir()
	testStore, err := store.NewSQLiteStore(tmpDir + "/test.db")
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	defer testStore.Close()

	agent := NewCharacterAgent(mockLLM, "test-model", testStore)

	project := domain.NewProject("西游记测试", domain.StyleManga, 2)
	bp := domain.NewStoryBlueprint(project.ID)
	bp.AddCharacter("孙悟空", "齐天大圣", []string{"好斗", "忠诚"})
	bp.AddCharacter("唐僧", "取经人", []string{"慈悲", "坚定"})

	state := NewPipelineState(project, "script")
	state.Blueprint = bp

	result, err := agent.Run(context.Background(), state)
	if err != nil {
		t.Fatalf("CharacterAgent.Run: %v", err)
	}

	// Should have created 2 character assets
	if len(result.Assets) != 2 {
		t.Fatalf("expected 2 assets, got %d", len(result.Assets))
	}

	// Verify first asset
	a := result.Assets[0]
	if a.Type != domain.AssetTypeCharacter {
		t.Errorf("expected type character, got %v", a.Type)
	}
	if a.Scope != domain.AssetScopeProject {
		t.Errorf("expected scope project, got %v", a.Scope)
	}
	if a.ProjectID != project.ID {
		t.Errorf("expected projectID '%s', got '%s'", project.ID, a.ProjectID)
	}
	if a.Metadata["character_id"] != bp.Characters[0].ID {
		t.Errorf("expected metadata character_id '%s', got '%s'", bp.Characters[0].ID, a.Metadata["character_id"])
	}
	if a.Metadata["visual_prompt"] == "" {
		t.Error("expected non-empty visual_prompt in metadata")
	}

	// Verify LLM was called once per character
	if len(mockLLM.Calls()) != 2 {
		t.Errorf("expected 2 LLM calls, got %d", len(mockLLM.Calls()))
	}

	// Verify asset was persisted to store
	stored, err := testStore.ListAssets(context.Background(), domain.AssetScopeProject, project.ID, domain.AssetTypeCharacter)
	if err != nil {
		t.Fatalf("ListAssets: %v", err)
	}
	if len(stored) != 2 {
		t.Errorf("expected 2 stored assets, got %d", len(stored))
	}
}

func TestCharacterAgent_Phase(t *testing.T) {
	agent := NewCharacterAgent(llm.NewMockClient(), "model", nil)
	if agent.Phase() != PhaseCharacterAsset {
		t.Errorf("expected phase character_asset, got %s", agent.Phase())
	}
}

func TestCharacterAgent_NilBlueprint(t *testing.T) {
	mockLLM := llm.NewMockClient()
	agent := NewCharacterAgent(mockLLM, "model", nil)
	project := domain.NewProject("test", domain.StyleManga, 1)
	state := NewPipelineState(project, "script")
	// Blueprint is nil

	_, err := agent.Run(context.Background(), state)
	if err == nil {
		t.Error("expected error for nil blueprint, got nil")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./internal/agent/ -v -run TestCharacterAgent`
Expected: compilation error — NewCharacterAgent not defined

- [ ] **Step 3: Implement CharacterAgent**

```go
// internal/agent/character.go
package agent

import (
	"context"
	"fmt"
	"log"

	"github.com/west-garden/short-maker/internal/domain"
	"github.com/west-garden/short-maker/internal/llm"
	"github.com/west-garden/short-maker/internal/store"
)

// CharacterAgent generates visual descriptions for each character in the blueprint
// and creates character asset records. Implements Agent for PhaseCharacterAsset.
type CharacterAgent struct {
	llmClient llm.Client
	model     string
	store     store.Store
}

func NewCharacterAgent(llmClient llm.Client, model string, store store.Store) *CharacterAgent {
	return &CharacterAgent{llmClient: llmClient, model: model, store: store}
}

func (a *CharacterAgent) Phase() Phase { return PhaseCharacterAsset }

func (a *CharacterAgent) Run(ctx context.Context, state *PipelineState) (*PipelineState, error) {
	if state.Blueprint == nil {
		return nil, fmt.Errorf("character agent requires a Blueprint (run story understanding first)")
	}

	style := string(state.Project.Style)

	for _, ch := range state.Blueprint.Characters {
		log.Printf("[character-agent] generating visual description for: %s", ch.Name)

		visual, err := a.generateVisualDescription(ctx, ch, style)
		if err != nil {
			return nil, fmt.Errorf("generate visual for %s: %w", ch.Name, err)
		}

		asset := domain.NewAsset(
			ch.Name+"_参考图",
			domain.AssetTypeCharacter,
			domain.AssetScopeProject,
			state.Project.ID,
		)
		asset.Metadata["character_id"] = ch.ID
		asset.Metadata["visual_prompt"] = visual.VisualPrompt
		asset.Metadata["face"] = visual.Appearance.Face
		asset.Metadata["body"] = visual.Appearance.Body
		asset.Metadata["clothing"] = visual.Appearance.Clothing
		asset.Tags = ch.Traits

		if a.store != nil {
			if err := a.store.SaveAsset(ctx, asset); err != nil {
				return nil, fmt.Errorf("save asset for %s: %w", ch.Name, err)
			}
		}

		state.Assets = append(state.Assets, asset)
	}

	return state, nil
}

func (a *CharacterAgent) generateVisualDescription(ctx context.Context, ch *domain.CharacterProfile, style string) (*characterVisualResponse, error) {
	systemPrompt := buildCharacterSystemPrompt()
	userPrompt := buildCharacterUserPrompt(ch.Name, ch.Description, ch.Traits, style)

	resp, err := a.llmClient.Chat(ctx, llm.Request{
		Model: a.model,
		Messages: []llm.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.5,
		MaxTokens:   1024,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM call: %w", err)
	}

	var visual characterVisualResponse
	if err := llm.ParseJSON(resp.Content, &visual); err != nil {
		return nil, fmt.Errorf("parse visual response: %w", err)
	}
	return &visual, nil
}

// --- Prompt builders ---

func buildCharacterSystemPrompt() string {
	return `You are a character design agent for an AI short drama production system.
Generate a detailed visual description suitable for AI image generation.

Output ONLY valid JSON with this exact schema (no other text):
{
  "visual_prompt": "complete visual description for image generation, including art style, clothing, pose, expression, and key visual features",
  "appearance": {
    "face": "detailed facial features description",
    "body": "body type, build, and posture",
    "clothing": "default clothing/outfit description",
    "distinctive_features": ["feature1", "feature2"]
  }
}

Requirements:
- visual_prompt should be a single paragraph suitable as an image generation prompt
- Include the art style (manga, 3D, live-action) in the visual_prompt
- Focus on visually distinctive features that maintain character consistency
- Descriptions should be specific enough for AI image generation`
}

func buildCharacterUserPrompt(name, description string, traits []string, style string) string {
	return fmt.Sprintf(`Generate a visual description for this character in %s style:

Name: %s
Role: %s
Personality: %v`, style, name, description, traits)
}

// --- Response types ---

type characterVisualResponse struct {
	VisualPrompt string              `json:"visual_prompt"`
	Appearance   characterAppearance `json:"appearance"`
}

type characterAppearance struct {
	Face                string   `json:"face"`
	Body                string   `json:"body"`
	Clothing            string   `json:"clothing"`
	DistinctiveFeatures []string `json:"distinctive_features"`
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./internal/agent/ -v -run TestCharacterAgent`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add short-maker/internal/agent/character.go short-maker/internal/agent/character_test.go
git commit -m "feat(short-maker): Character Management Agent — visual descriptions via LLM"
```

---

### Task 5: Wire Real Agents into CLI

**Files:**
- Modify: `short-maker/cmd/shortmaker/main.go`

- [ ] **Step 1: Read the current main.go**

Read: `short-maker/cmd/shortmaker/main.go`

- [ ] **Step 2: Replace main.go with real agent wiring**

```go
// cmd/shortmaker/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/west-garden/short-maker/internal/agent"
	"github.com/west-garden/short-maker/internal/domain"
	"github.com/west-garden/short-maker/internal/llm"
	"github.com/west-garden/short-maker/internal/store"
)

var rootCmd = &cobra.Command{
	Use:   "shortmaker",
	Short: "AI short drama production pipeline",
}

var runCmd = &cobra.Command{
	Use:   "run [script-file]",
	Short: "Run the production pipeline on a script file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		scriptPath := args[0]
		script, err := os.ReadFile(scriptPath)
		if err != nil {
			return fmt.Errorf("read script: %w", err)
		}

		styleName, _ := cmd.Flags().GetString("style")
		episodes, _ := cmd.Flags().GetInt("episodes")
		useMock, _ := cmd.Flags().GetBool("mock")
		llmModel, _ := cmd.Flags().GetString("model")
		dbPath, _ := cmd.Flags().GetString("db")

		style := domain.Style(styleName)
		project := domain.NewProject(scriptPath, style, episodes)
		state := agent.NewPipelineState(project, string(script))

		agents, cleanup, err := buildAgents(useMock, llmModel, dbPath)
		if err != nil {
			return err
		}
		defer cleanup()

		checkpoint := func(phase agent.Phase, s *agent.PipelineState) error {
			log.Printf("  [checkpoint] phase %s completed", phase)
			return nil
		}

		orch := agent.NewOrchestrator(agents, checkpoint)
		result, err := orch.Run(context.Background(), state)
		if err != nil {
			return fmt.Errorf("pipeline failed: %w", err)
		}

		printSummary(result)
		return nil
	},
}

func buildAgents(useMock bool, llmModel, dbPath string) (map[agent.Phase]agent.Agent, func(), error) {
	agents := map[agent.Phase]agent.Agent{}
	cleanup := func() {}

	if useMock {
		// All mock agents
		for _, phase := range agent.DefaultFlow {
			p := phase
			agents[p] = agent.NewMockAgent(p, func(ctx context.Context, s *agent.PipelineState) (*agent.PipelineState, error) {
				log.Printf("  [mock-%s] processing...", p)
				return s, nil
			})
		}
		return agents, cleanup, nil
	}

	// Real agents for story understanding and character management
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, nil, fmt.Errorf("OPENAI_API_KEY environment variable is required when --mock=false")
	}
	baseURL := os.Getenv("OPENAI_BASE_URL")

	llmClient := llm.NewOpenAIClient(apiKey, baseURL)

	var st store.Store
	if dbPath != "" {
		sqliteStore, err := store.NewSQLiteStore(dbPath)
		if err != nil {
			return nil, nil, fmt.Errorf("open database: %w", err)
		}
		st = sqliteStore
		cleanup = func() { sqliteStore.Close() }
	}

	agents[agent.PhaseStoryUnderstanding] = agent.NewStoryAgent(llmClient, llmModel)
	agents[agent.PhaseCharacterAsset] = agent.NewCharacterAgent(llmClient, llmModel, st)

	// Remaining phases still use mocks — real implementations come in Plan 3-4
	for _, phase := range agent.DefaultFlow {
		if _, ok := agents[phase]; !ok {
			p := phase
			agents[p] = agent.NewMockAgent(p, func(ctx context.Context, s *agent.PipelineState) (*agent.PipelineState, error) {
				log.Printf("  [mock-%s] processing...", p)
				return s, nil
			})
		}
	}

	return agents, cleanup, nil
}

func printSummary(result *agent.PipelineState) {
	log.Printf("=== Pipeline Complete ===")
	log.Printf("Project: %s", result.Project.Name)
	if result.Blueprint != nil {
		log.Printf("World: %s", result.Blueprint.WorldView)
		log.Printf("Characters: %d", len(result.Blueprint.Characters))
		for _, ch := range result.Blueprint.Characters {
			log.Printf("  - %s: %s", ch.Name, ch.Description)
		}
		log.Printf("Episodes: %d", len(result.Blueprint.Episodes))
	}
	log.Printf("Assets: %d", len(result.Assets))
	for _, a := range result.Assets {
		log.Printf("  - [%s] %s", a.Type, a.Name)
	}
	log.Printf("Storyboard shots: %d", len(result.Storyboard))
	log.Printf("Errors: %d", len(result.Errors))
}

func init() {
	runCmd.Flags().String("style", "manga", "Content style: manga, 3d, live_action")
	runCmd.Flags().Int("episodes", 10, "Number of episodes")
	runCmd.Flags().Bool("mock", true, "Use mock agents (set false for real LLM calls)")
	runCmd.Flags().String("model", "gpt-4o-mini", "LLM model name")
	runCmd.Flags().String("db", "", "SQLite database path (optional, enables persistence)")
	rootCmd.AddCommand(runCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
```

- [ ] **Step 3: Verify mock mode still works**

Run: `go run -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./cmd/shortmaker run testdata/sample-script.txt --style manga --episodes 2`
Expected: same output as before (all mock agents)

- [ ] **Step 4: Verify real mode rejects missing API key**

Run: `go run -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./cmd/shortmaker run testdata/sample-script.txt --mock=false`
Expected: error message about OPENAI_API_KEY

- [ ] **Step 5: Commit**

```bash
git add short-maker/cmd/shortmaker/main.go
git commit -m "feat(short-maker): wire real Story + Character agents into CLI with LLM config"
```

---

### Task 6: End-to-End Integration Test

**Files:**
- Modify: `short-maker/internal/agent/integration_test.go`

- [ ] **Step 1: Read the current integration test**

Read: `short-maker/internal/agent/integration_test.go`

- [ ] **Step 2: Add a new integration test using real agents with MockClient**

Add this test to the existing `integration_test.go` file. Do NOT remove the existing `TestIntegration_FullPipelineWithMockAgents` test.

```go
func TestIntegration_StoryAndCharacterAgentsWithMockLLM(t *testing.T) {
	// This test verifies the real StoryAgent and CharacterAgent work together,
	// producing a Blueprint and character Assets from a script.

	storyJSON := `{
		"world_view": "古代仙侠世界",
		"characters": [
			{
				"name": "李逍遥",
				"description": "天资聪颖的少年侠客",
				"traits": ["正义", "热血", "重情义"]
			},
			{
				"name": "赵灵儿",
				"description": "苗族圣女，温柔善良",
				"traits": ["温柔", "善良", "坚强"]
			}
		],
		"episodes": [
			{
				"number": 1,
				"role": "hook",
				"emotion_arc": "好奇→震撼",
				"synopsis": "李逍遥在仙灵岛邂逅赵灵儿",
				"scenes": [
					{
						"narrative_beat": "开场",
						"emotion_arc": "平静→好奇",
						"setting": "仙灵岛",
						"pacing": "medium",
						"character_count": 2
					}
				]
			}
		],
		"relationships": [
			{
				"character_a": "李逍遥",
				"character_b": "赵灵儿",
				"type": "恋人"
			}
		]
	}`

	characterJSON := `{
		"visual_prompt": "一位英俊的少年侠客，身穿蓝色长袍，手持长剑，眼神坚定",
		"appearance": {
			"face": "剑眉星目，英俊潇洒",
			"body": "身材修长，姿态飘逸",
			"clothing": "蓝色仙侠长袍，腰佩长剑",
			"distinctive_features": ["蓝色长袍", "配剑"]
		}
	}`

	mockLLM := llm.NewMockClient()
	// First call returns story analysis, subsequent calls return character visuals
	callCount := 0
	customMock := &sequentialMockClient{
		responses: []string{storyJSON, characterJSON, characterJSON},
	}

	tmpDir := t.TempDir()
	testStore, err := store.NewSQLiteStore(tmpDir + "/test.db")
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	defer testStore.Close()

	_ = mockLLM
	_ = callCount

	storyAgent := NewStoryAgent(customMock, "test-model")
	charAgent := NewCharacterAgent(customMock, "test-model", testStore)

	// Build agent map with real story + character, mock for the rest
	agents := map[Phase]Agent{
		PhaseStoryUnderstanding: storyAgent,
		PhaseCharacterAsset:     charAgent,
		PhaseStoryboard: NewMockAgent(PhaseStoryboard, func(ctx context.Context, s *PipelineState) (*PipelineState, error) {
			return s, nil
		}),
		PhaseImageGeneration: NewMockAgent(PhaseImageGeneration, func(ctx context.Context, s *PipelineState) (*PipelineState, error) {
			return s, nil
		}),
		PhaseVideoGeneration: NewMockAgent(PhaseVideoGeneration, func(ctx context.Context, s *PipelineState) (*PipelineState, error) {
			return s, nil
		}),
	}

	orch := NewOrchestrator(agents, nil)
	project := domain.NewProject("仙剑奇侠传", domain.StyleManga, 1)
	state := NewPipelineState(project, "第一集：仙灵岛\n李逍遥在仙灵岛邂逅赵灵儿。")

	result, err := orch.Run(context.Background(), state)
	if err != nil {
		t.Fatalf("pipeline: %v", err)
	}

	// Verify StoryAgent output
	if result.Blueprint == nil {
		t.Fatal("expected Blueprint")
	}
	if result.Blueprint.WorldView != "古代仙侠世界" {
		t.Errorf("unexpected world_view: %s", result.Blueprint.WorldView)
	}
	if len(result.Blueprint.Characters) != 2 {
		t.Fatalf("expected 2 characters, got %d", len(result.Blueprint.Characters))
	}
	if result.Blueprint.Characters[0].Name != "李逍遥" {
		t.Errorf("expected first character '李逍遥', got '%s'", result.Blueprint.Characters[0].Name)
	}

	// Verify CharacterAgent output
	if len(result.Assets) != 2 {
		t.Fatalf("expected 2 assets, got %d", len(result.Assets))
	}
	for _, a := range result.Assets {
		if a.Type != domain.AssetTypeCharacter {
			t.Errorf("expected asset type character, got %v", a.Type)
		}
		if a.Metadata["visual_prompt"] == "" {
			t.Errorf("expected non-empty visual_prompt for asset %s", a.Name)
		}
		if a.Metadata["character_id"] == "" {
			t.Errorf("expected non-empty character_id for asset %s", a.Name)
		}
	}

	// Verify persistence
	storedAssets, _ := testStore.ListAssets(context.Background(), domain.AssetScopeProject, project.ID, domain.AssetTypeCharacter)
	if len(storedAssets) != 2 {
		t.Errorf("expected 2 persisted assets, got %d", len(storedAssets))
	}
}

// sequentialMockClient returns different responses for each call.
type sequentialMockClient struct {
	responses []string
	index     int
}

func (m *sequentialMockClient) Chat(ctx context.Context, req llm.Request) (*llm.Response, error) {
	if m.index >= len(m.responses) {
		return &llm.Response{Content: m.responses[len(m.responses)-1], Model: req.Model}, nil
	}
	resp := m.responses[m.index]
	m.index++
	return &llm.Response{Content: resp, TokensUsed: len(resp), Model: req.Model}, nil
}
```

- [ ] **Step 3: Run the new integration test**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./internal/agent/ -v -run TestIntegration_StoryAndCharacter`
Expected: PASS

- [ ] **Step 4: Run ALL tests to verify no regressions**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./... -count=1`
Expected: all PASS across domain, agent, llm, store packages

- [ ] **Step 5: Commit**

```bash
git add short-maker/internal/agent/integration_test.go
git commit -m "feat(short-maker): integration test — Story + Character agents with mock LLM"
```

- [ ] **Step 6: Run go mod tidy and commit**

```bash
GOWORK=off go mod tidy -C /Users/rain/code/west-garden/ai-drama-research/short-maker
git add short-maker/go.mod short-maker/go.sum
git commit -m "chore(short-maker): go mod tidy"
```
