package agent

import (
	"context"
	"fmt"
	"log"

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

	log.Printf("[story-agent] sending request to LLM (model=%s, maxTokens=65536)", a.model)
	resp, err := a.llmClient.Chat(ctx, llm.Request{
		Model: a.model,
		Messages: []llm.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.3,
		MaxTokens:   65536,
	})
	if err != nil {
		return nil, fmt.Errorf("story LLM call: %w", err)
	}
	log.Printf("[story-agent] LLM response received (len=%d, tokens=%d)", len(resp.Content), resp.TokensUsed)

	var parsed storyAnalysisResponse
	if err := llm.ParseJSON(resp.Content, &parsed); err != nil {
		return nil, fmt.Errorf("parse story response: %w", err)
	}

	bp := convertToBlueprint(state.Project.ID, &parsed)
	state.Blueprint = bp
	return state, nil
}

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
