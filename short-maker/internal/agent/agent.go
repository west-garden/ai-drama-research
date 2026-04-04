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
	Images     []*GeneratedShot       `json:"images,omitempty"`
	Videos     []*GeneratedShot       `json:"videos,omitempty"`
	Errors     []string               `json:"errors,omitempty"`
}

// GeneratedShot tracks the output of image and video generation for one shot.
type GeneratedShot struct {
	ShotNumber int          `json:"shot_number"`
	EpisodeNum int          `json:"episode_number"`
	ImagePath  string       `json:"image_path"`
	VideoPath  string       `json:"video_path"`
	Grade      domain.Grade `json:"grade"`
	ImageScore int          `json:"image_score"`
	VideoScore int          `json:"video_score"`
}

func NewPipelineState(project *domain.Project, script string) *PipelineState {
	return &PipelineState{
		Project: project,
		Script:  script,
	}
}
