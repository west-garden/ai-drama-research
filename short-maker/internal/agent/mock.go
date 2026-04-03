package agent

import "context"

// MockAgent returns a canned PipelineState for any phase.
// Used in orchestrator tests.
type MockAgent struct {
	phase   Phase
	runFunc func(ctx context.Context, input *PipelineState) (*PipelineState, error)
}

func NewMockAgent(phase Phase, fn func(ctx context.Context, input *PipelineState) (*PipelineState, error)) *MockAgent {
	return &MockAgent{phase: phase, runFunc: fn}
}

func (m *MockAgent) Phase() Phase { return m.phase }

func (m *MockAgent) Run(ctx context.Context, input *PipelineState) (*PipelineState, error) {
	return m.runFunc(ctx, input)
}
