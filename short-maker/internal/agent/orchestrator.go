// internal/agent/orchestrator.go
package agent

import (
	"context"
	"fmt"
	"log"
)

// CheckpointHook is called after each phase completes.
// Return a non-nil error to halt the pipeline.
type CheckpointHook func(phase Phase, state *PipelineState) error

type Orchestrator struct {
	agents     map[Phase]Agent
	checkpoint CheckpointHook
}

func NewOrchestrator(agents map[Phase]Agent, checkpoint CheckpointHook) *Orchestrator {
	return &Orchestrator{
		agents:     agents,
		checkpoint: checkpoint,
	}
}

func (o *Orchestrator) Run(ctx context.Context, state *PipelineState) (*PipelineState, error) {
	for _, phase := range DefaultFlow {
		agent, ok := o.agents[phase]
		if !ok {
			return nil, fmt.Errorf("missing required agent for phase: %s", phase)
		}

		log.Printf("[orchestrator] starting phase: %s", phase)

		result, err := agent.Run(ctx, state)
		if err != nil {
			return nil, fmt.Errorf("phase %s failed: %w", phase, err)
		}
		state = result

		if o.checkpoint != nil {
			if err := o.checkpoint(phase, state); err != nil {
				return nil, fmt.Errorf("checkpoint halted at phase %s: %w", phase, err)
			}
		}

		log.Printf("[orchestrator] completed phase: %s", phase)
	}

	return state, nil
}
