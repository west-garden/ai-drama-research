package agent

import (
	"context"
	"fmt"
	"log"

	"github.com/west-garden/short-maker/internal/domain"
	"github.com/west-garden/short-maker/internal/llm"
	"github.com/west-garden/short-maker/internal/strategy"
)

// StoryboardAgent generates structured shot specifications for each scene
// in the blueprint. Uses the strategy engine for candidate selection and
// LLM for final shot details. Implements Agent for PhaseStoryboard.
type StoryboardAgent struct {
	llmClient    llm.Client
	model        string
	strategyRepo *strategy.Repository
}

func NewStoryboardAgent(llmClient llm.Client, model string, repo *strategy.Repository) *StoryboardAgent {
	return &StoryboardAgent{llmClient: llmClient, model: model, strategyRepo: repo}
}

func (a *StoryboardAgent) Phase() Phase { return PhaseStoryboard }

func (a *StoryboardAgent) Run(ctx context.Context, state *PipelineState) (*PipelineState, error) {
	if state.Blueprint == nil {
		return nil, fmt.Errorf("storyboard agent requires a Blueprint")
	}

	// Build character name -> ID lookup from blueprint
	nameToID := make(map[string]string, len(state.Blueprint.Characters))
	for _, ch := range state.Blueprint.Characters {
		nameToID[ch.Name] = ch.ID
	}

	shotCounter := 0
	for _, ep := range state.Blueprint.Episodes {
		log.Printf("[storyboard-agent] episode %d (%s): %d scenes", ep.Number, ep.Role, len(ep.Scenes))

		var prevShots []string // brief context of previous shots for continuity

		for _, scene := range ep.Scenes {
			// Phase 1: Coarse filter -- get candidate strategies
			var candidates []strategy.ScoredStrategy
			if a.strategyRepo != nil {
				candidates = strategy.MatchScene(a.strategyRepo, scene, 6)
			}

			// Phase 2: LLM -- select strategy + generate shot details
			shots, err := a.generateShotsForScene(ctx, ep, scene, candidates, prevShots, state)
			if err != nil {
				return nil, fmt.Errorf("generate shots for ep%d scene '%s': %w", ep.Number, scene.NarrativeBeat, err)
			}

			for _, shot := range shots {
				shotCounter++
				spec := domain.NewShotSpec(ep.Number, shotCounter)
				spec.FrameType = shot.FrameType
				spec.Composition = shot.Composition
				spec.CameraMove = shot.CameraMove
				spec.Emotion = shot.Emotion
				spec.Prompt = shot.Prompt
				spec.SceneRef = shot.SceneRef
				spec.StrategyID = shot.StrategyID
				spec.RhythmPosition = domain.RhythmPosition(shot.RhythmPosition)
				spec.ContentType = domain.ContentType(shot.ContentType)

				// Resolve character names to IDs
				for _, name := range shot.CharacterNames {
					if id, ok := nameToID[name]; ok {
						spec.AddCharacterRef(id)
					}
				}

				state.Storyboard = append(state.Storyboard, spec)
				prevShots = append(prevShots, fmt.Sprintf("shot%d: %s %s", shotCounter, spec.FrameType, spec.Emotion))
			}
		}
	}

	log.Printf("[storyboard-agent] generated %d total shots", shotCounter)
	return state, nil
}

func (a *StoryboardAgent) generateShotsForScene(
	ctx context.Context,
	ep *domain.EpisodeBlueprint,
	scene domain.SceneTag,
	candidates []strategy.ScoredStrategy,
	prevShots []string,
	state *PipelineState,
) ([]storyboardShotResponse, error) {
	systemPrompt := buildStoryboardSystemPrompt()
	userPrompt := buildStoryboardUserPrompt(ep, scene, candidates, prevShots, state)

	resp, err := a.llmClient.Chat(ctx, llm.Request{
		Model: a.model,
		Messages: []llm.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.4,
		MaxTokens:   2048,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM call: %w", err)
	}

	var parsed storyboardResponse
	if err := llm.ParseJSON(resp.Content, &parsed); err != nil {
		return nil, fmt.Errorf("parse storyboard response: %w", err)
	}
	return parsed.Shots, nil
}

// --- Prompt builders ---

func buildStoryboardSystemPrompt() string {
	return `You are a storyboard generation agent for an AI short drama production system.
Given a scene description and candidate shot strategies, generate structured shot specifications.

For each shot, select the most appropriate strategy and customize it for the scene context.
Annotate each shot with rhythm_position and content_type for importance scoring.

Output ONLY valid JSON with this schema:
{
  "shots": [
    {
      "strategy_id": "id of the selected strategy (or empty if no candidates)",
      "frame_type": "close_up|medium|wide|extreme_wide",
      "composition": "description of shot composition",
      "camera_move": "static|pan|zoom_in|zoom_out|tracking",
      "emotion": "emotional tone of the shot in Chinese",
      "prompt": "detailed visual description for image generation — include art style, specific visual elements, lighting, atmosphere",
      "character_names": ["character names visible in this shot"],
      "scene_ref": "scene or setting name",
      "rhythm_position": "open_hook|emotion_peak|tail_hook|mid_narration",
      "content_type": "first_appear|fight|dialogue|empty"
    }
  ]
}

rhythm_position rules:
- "open_hook": first shot of the episode (must grab attention)
- "emotion_peak": shots at emotional turning points (conflict, revelation, reunion)
- "tail_hook": last shot of the episode (create suspense for next episode)
- "mid_narration": everything else (setup, transition, daily scenes)

content_type rules:
- "first_appear": a character's first appearance in the series (visual anchor point)
- "fight": action, combat, special effects, large-scale scenes
- "dialogue": conversation, close-up on expressions
- "empty": scenery, environment, flashback, no characters

Generate 1-3 shots per scene. Each shot should have a distinct purpose in the narrative.`
}

func buildStoryboardUserPrompt(
	ep *domain.EpisodeBlueprint,
	scene domain.SceneTag,
	candidates []strategy.ScoredStrategy,
	prevShots []string,
	state *PipelineState,
) string {
	prompt := fmt.Sprintf("Episode %d (role: %s), emotion arc: %s\n", ep.Number, ep.Role, ep.EmotionArc)
	prompt += fmt.Sprintf("Synopsis: %s\n\n", ep.Synopsis)
	prompt += fmt.Sprintf("Scene: %s\n", scene.NarrativeBeat)
	prompt += fmt.Sprintf("Scene emotion: %s\n", scene.EmotionArc)
	prompt += fmt.Sprintf("Setting: %s\n", scene.Setting)
	prompt += fmt.Sprintf("Pacing: %s\n", scene.Pacing)
	prompt += fmt.Sprintf("Character count: %d\n", scene.CharacterCount)
	prompt += fmt.Sprintf("Style: %s\n\n", state.Project.Style)

	// Characters available
	if state.Blueprint != nil && len(state.Blueprint.Characters) > 0 {
		prompt += "Characters:\n"
		for _, ch := range state.Blueprint.Characters {
			prompt += fmt.Sprintf("- %s: %s\n", ch.Name, ch.Description)
		}
		prompt += "\n"
	}

	// Candidate strategies
	if len(candidates) > 0 {
		prompt += "Candidate strategies (pick the most appropriate for each shot):\n"
		for i, c := range candidates {
			prompt += fmt.Sprintf("%d. [%s] %s — %s (score: %.1f)\n",
				i+1, c.Strategy.ID, c.Strategy.Name, c.Strategy.ShotFormula.String(), c.Score)
			if len(c.Strategy.Examples) > 0 {
				prompt += fmt.Sprintf("   Example: %s\n", c.Strategy.Examples[0])
			}
		}
		prompt += "\n"
	}

	// Previous shots for continuity
	if len(prevShots) > 0 {
		prompt += "Previous shots in this episode (for continuity):\n"
		limit := 3
		if len(prevShots) < limit {
			limit = len(prevShots)
		}
		for _, ps := range prevShots[len(prevShots)-limit:] {
			prompt += fmt.Sprintf("- %s\n", ps)
		}
	}

	return prompt
}

// --- Response types ---

type storyboardResponse struct {
	Shots []storyboardShotResponse `json:"shots"`
}

type storyboardShotResponse struct {
	StrategyID     string   `json:"strategy_id"`
	FrameType      string   `json:"frame_type"`
	Composition    string   `json:"composition"`
	CameraMove     string   `json:"camera_move"`
	Emotion        string   `json:"emotion"`
	Prompt         string   `json:"prompt"`
	CharacterNames []string `json:"character_names"`
	SceneRef       string   `json:"scene_ref"`
	RhythmPosition string   `json:"rhythm_position"`
	ContentType    string   `json:"content_type"`
}
