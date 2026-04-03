package agent

import (
	"context"

	"github.com/west-garden/short-maker/internal/domain"
)

type Phase string

const (
	PhaseStoryUnderstanding Phase = "story_understanding"
	PhaseCharacterAsset     Phase = "character_asset"
	PhaseStoryboard         Phase = "storyboard"
	PhaseImageGeneration    Phase = "image_generation"
	PhaseVideoGeneration    Phase = "video_generation"
	PhaseQualityCheck       Phase = "quality_check"
)

// DefaultFlow is the fixed pipeline order. Orchestrator cannot skip these.
var DefaultFlow = []Phase{
	PhaseStoryUnderstanding,
	PhaseCharacterAsset,
	PhaseStoryboard,
	PhaseImageGeneration,
	PhaseVideoGeneration,
}

type Agent interface {
	Phase() Phase
	Run(ctx context.Context, input *PipelineState) (*PipelineState, error)
}

// PipelineState is the structured data passed between agents.
// Each agent reads what it needs and writes its outputs.
type PipelineState struct {
	Project    *domain.Project        `json:"project"`
	Script     string                 `json:"script"`
	Blueprint  *domain.StoryBlueprint `json:"blueprint,omitempty"`
	Assets     []*domain.Asset        `json:"assets,omitempty"`
	Storyboard []*domain.ShotSpec     `json:"storyboard,omitempty"`
	Errors     []string               `json:"errors,omitempty"`
}

func NewPipelineState(project *domain.Project, script string) *PipelineState {
	return &PipelineState{
		Project: project,
		Script:  script,
	}
}
