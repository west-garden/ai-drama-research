package agent

import (
	"context"
	"fmt"
	"time"

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

// NodeStatusEntry tracks the execution status of a single workflow node.
type NodeStatusEntry struct {
	Status    domain.NodeStatus `json:"status"`
	Error     string            `json:"error,omitempty"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// PipelineState is the structured data passed between agents.
// Each agent reads what it needs and writes its outputs.
type PipelineState struct {
	Project      *domain.Project                `json:"project"`
	Script       string                         `json:"script"`
	Blueprint    *domain.StoryBlueprint         `json:"blueprint,omitempty"`
	Assets       []*domain.Asset                `json:"assets,omitempty"`
	Storyboard   []*domain.ShotSpec             `json:"storyboard,omitempty"`
	Images       []*GeneratedShot               `json:"images,omitempty"`
	Videos       []*GeneratedShot               `json:"videos,omitempty"`
	Errors       []string                       `json:"errors,omitempty"`
	NodeStatuses map[string]NodeStatusEntry      `json:"node_statuses,omitempty"`
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
		Project:      project,
		Script:       script,
		NodeStatuses: make(map[string]NodeStatusEntry),
	}
}

// NodeKey builds the key for a workflow node.
// Project-level phases (episode=0): "story_understanding", "character_asset"
// Episode-level phases: "storyboard:ep1", "image_generation:ep3"
func NodeKey(phase Phase, episode int) string {
	if episode <= 0 {
		return string(phase)
	}
	return fmt.Sprintf("%s:ep%d", phase, episode)
}

// SetNodeStatus updates the status of a workflow node.
func (s *PipelineState) SetNodeStatus(nodeKey string, status domain.NodeStatus, errMsg string) {
	if s.NodeStatuses == nil {
		s.NodeStatuses = make(map[string]NodeStatusEntry)
	}
	s.NodeStatuses[nodeKey] = NodeStatusEntry{
		Status:    status,
		Error:     errMsg,
		UpdatedAt: time.Now(),
	}
}

// GetNodeStatus returns the status of a workflow node.
func (s *PipelineState) GetNodeStatus(nodeKey string) domain.NodeStatus {
	if s.NodeStatuses == nil {
		return domain.NodeStatusPending
	}
	entry, ok := s.NodeStatuses[nodeKey]
	if !ok {
		return domain.NodeStatusPending
	}
	return entry.Status
}
