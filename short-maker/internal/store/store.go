// internal/store/store.go
package store

import (
	"context"
	"time"

	"github.com/west-garden/short-maker/internal/domain"
)

type ProjectStore interface {
	SaveProject(ctx context.Context, project *domain.Project) error
	GetProject(ctx context.Context, id string) (*domain.Project, error)
	UpdateProjectStatus(ctx context.Context, id string, status domain.Status) error
}

type AssetStore interface {
	SaveAsset(ctx context.Context, asset *domain.Asset) error
	GetAsset(ctx context.Context, id string) (*domain.Asset, error)
	ListAssets(ctx context.Context, scope domain.AssetScope, projectID string, assetType domain.AssetType) ([]*domain.Asset, error)
	SearchAssets(ctx context.Context, scope domain.AssetScope, tags []string) ([]*domain.Asset, error)
}

type BlueprintStore interface {
	SaveBlueprint(ctx context.Context, blueprint *domain.StoryBlueprint) error
	GetBlueprint(ctx context.Context, projectID string) (*domain.StoryBlueprint, error)
}

type PipelineRunRecord struct {
	ProjectID    string    `json:"project_id"`
	Status       string    `json:"status"`
	CurrentPhase string    `json:"current_phase"`
	Error        string    `json:"error"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type PipelineRunStore interface {
	SavePipelineRun(ctx context.Context, run *PipelineRunRecord) error
	GetPipelineRun(ctx context.Context, projectID string) (*PipelineRunRecord, error)
	UpdatePipelineRun(ctx context.Context, projectID string, status string, phase string, errMsg string) error
	SavePipelineResult(ctx context.Context, projectID string, resultJSON []byte) error
	GetPipelineResult(ctx context.Context, projectID string) ([]byte, error)
	ListProjects(ctx context.Context) ([]*domain.Project, error)
	RecoverRunningPipelines(ctx context.Context) error
}

// Store combines all storage interfaces. SQLite implements all of them.
type Store interface {
	ProjectStore
	AssetStore
	BlueprintStore
	PipelineRunStore
	Close() error
}
