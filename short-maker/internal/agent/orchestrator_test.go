// internal/agent/orchestrator_test.go
package agent

import (
	"context"
	"testing"

	"github.com/west-garden/short-maker/internal/domain"
)

func TestOrchestrator_RunsDefaultFlow(t *testing.T) {
	var executedPhases []Phase

	makeAgent := func(phase Phase) Agent {
		return NewMockAgent(phase, func(ctx context.Context, state *PipelineState) (*PipelineState, error) {
			executedPhases = append(executedPhases, phase)
			return state, nil
		})
	}

	agents := map[Phase]Agent{
		PhaseStoryUnderstanding: makeAgent(PhaseStoryUnderstanding),
		PhaseCharacterAsset:     makeAgent(PhaseCharacterAsset),
		PhaseStoryboard:         makeAgent(PhaseStoryboard),
		PhaseImageGeneration:    makeAgent(PhaseImageGeneration),
		PhaseVideoGeneration:    makeAgent(PhaseVideoGeneration),
	}

	orch := NewOrchestrator(agents, nil)
	project := domain.NewProject("测试剧", domain.StyleManga, 10)
	state := NewPipelineState(project, "这是一个测试剧本...")

	result, err := orch.Run(context.Background(), state)
	if err != nil {
		t.Fatalf("Orchestrator.Run: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Verify all 5 phases ran in order
	if len(executedPhases) != 5 {
		t.Fatalf("expected 5 phases, got %d", len(executedPhases))
	}
	for i, expected := range DefaultFlow {
		if executedPhases[i] != expected {
			t.Errorf("phase %d: expected %s, got %s", i, expected, executedPhases[i])
		}
	}
}

func TestOrchestrator_CannotSkipPhases(t *testing.T) {
	// Missing PhaseStoryUnderstanding — orchestrator should error
	agents := map[Phase]Agent{
		PhaseCharacterAsset:  NewMockAgent(PhaseCharacterAsset, func(ctx context.Context, s *PipelineState) (*PipelineState, error) { return s, nil }),
		PhaseStoryboard:      NewMockAgent(PhaseStoryboard, func(ctx context.Context, s *PipelineState) (*PipelineState, error) { return s, nil }),
		PhaseImageGeneration: NewMockAgent(PhaseImageGeneration, func(ctx context.Context, s *PipelineState) (*PipelineState, error) { return s, nil }),
		PhaseVideoGeneration: NewMockAgent(PhaseVideoGeneration, func(ctx context.Context, s *PipelineState) (*PipelineState, error) { return s, nil }),
	}

	orch := NewOrchestrator(agents, nil)
	project := domain.NewProject("测试剧", domain.StyleManga, 10)
	state := NewPipelineState(project, "剧本")

	_, err := orch.Run(context.Background(), state)
	if err == nil {
		t.Error("expected error for missing phase, got nil")
	}
}

func TestOrchestrator_CheckpointHook(t *testing.T) {
	var checkpoints []Phase

	agents := map[Phase]Agent{}
	for _, p := range DefaultFlow {
		phase := p
		agents[phase] = NewMockAgent(phase, func(ctx context.Context, s *PipelineState) (*PipelineState, error) { return s, nil })
	}

	hook := func(phase Phase, state *PipelineState) error {
		checkpoints = append(checkpoints, phase)
		return nil
	}

	orch := NewOrchestrator(agents, hook)
	project := domain.NewProject("测试剧", domain.StyleManga, 10)
	state := NewPipelineState(project, "剧本")

	orch.Run(context.Background(), state)

	if len(checkpoints) != 5 {
		t.Errorf("expected 5 checkpoint calls, got %d", len(checkpoints))
	}
}
