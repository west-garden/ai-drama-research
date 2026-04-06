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

		visual, err := a.generateVisualDescription(ctx, ch, style, state.Project.PromptLanguage)
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

func (a *CharacterAgent) generateVisualDescription(ctx context.Context, ch *domain.CharacterProfile, style string, promptLanguage string) (*characterVisualResponse, error) {
	systemPrompt := buildCharacterSystemPrompt(promptLanguage)
	userPrompt := buildCharacterUserPrompt(ch.Name, ch.Description, ch.Traits, style)

	resp, err := a.llmClient.Chat(ctx, llm.Request{
		Model: a.model,
		Messages: []llm.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.5,
		MaxTokens:   4096,
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

func buildCharacterSystemPrompt(promptLanguage string) string {
	base := `You are a character design agent for an AI short drama production system.
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

	if promptLanguage == "zh" {
		base += `

IMPORTANT language rules:
- "visual_prompt" MUST be in English (for AI image generation)
- "face", "body", "clothing" fields MUST be in Chinese (中文)
- "distinctive_features" array values MUST be in Chinese (中文)`
	}

	return base
}

func buildCharacterUserPrompt(name, description string, traits []string, style string) string {
	return fmt.Sprintf(`Generate a visual description for this character in %s style:

Name: %s
Role: %s
Personality: %v`, style, name, description, traits)
}

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
