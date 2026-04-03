// internal/domain/asset.go
package domain

import "time"

type AssetType string

const (
	AssetTypeCharacter AssetType = "character"
	AssetTypeScene     AssetType = "scene"
	AssetTypeProp      AssetType = "prop"
	AssetTypeCostume   AssetType = "costume"
	AssetTypeStyle     AssetType = "style"
	AssetTypeAudio     AssetType = "audio"
)

type AssetScope string

const (
	AssetScopeGlobal  AssetScope = "global"
	AssetScopeProject AssetScope = "project"
)

type Asset struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Type      AssetType         `json:"type"`
	Scope     AssetScope        `json:"scope"`
	ProjectID string            `json:"project_id,omitempty"`
	FilePath  string            `json:"file_path"`
	Tags      []string          `json:"tags"`
	Metadata  map[string]string `json:"metadata"`
	CreatedAt time.Time         `json:"created_at"`
}

func NewAsset(name string, assetType AssetType, scope AssetScope, projectID string) *Asset {
	return &Asset{
		ID:        generateID("asset"),
		Name:      name,
		Type:      assetType,
		Scope:     scope,
		ProjectID: projectID,
		Tags:      []string{},
		Metadata:  map[string]string{},
		CreatedAt: time.Now(),
	}
}

func (a *Asset) PromoteToGlobal() {
	a.Scope = AssetScopeGlobal
	a.ProjectID = ""
}
