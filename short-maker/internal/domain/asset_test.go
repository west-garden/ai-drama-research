// internal/domain/asset_test.go
package domain

import "testing"

func TestNewAsset(t *testing.T) {
	a := NewAsset("孙悟空正面照", AssetTypeCharacter, AssetScopeProject, "proj_1")
	if a.Name != "孙悟空正面照" {
		t.Errorf("expected name '孙悟空正面照', got '%s'", a.Name)
	}
	if a.Type != AssetTypeCharacter {
		t.Errorf("expected type Character, got %v", a.Type)
	}
	if a.Scope != AssetScopeProject {
		t.Errorf("expected scope Project, got %v", a.Scope)
	}
}

func TestAssetPromoteToGlobal(t *testing.T) {
	a := NewAsset("通用宫殿背景", AssetTypeScene, AssetScopeProject, "proj_1")
	a.PromoteToGlobal()
	if a.Scope != AssetScopeGlobal {
		t.Errorf("expected scope Global after promotion, got %v", a.Scope)
	}
	if a.ProjectID != "" {
		t.Error("expected empty projectID after promotion")
	}
}
