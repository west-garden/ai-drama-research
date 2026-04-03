// internal/store/store.go
package store

import (
	"context"

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

// Store combines all storage interfaces. SQLite implements all of them.
type Store interface {
	ProjectStore
	AssetStore
	BlueprintStore
	Close() error
}
