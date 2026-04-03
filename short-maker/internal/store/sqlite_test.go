// internal/store/sqlite_test.go
package store

import (
	"context"
	"os"
	"testing"

	"github.com/west-garden/short-maker/internal/domain"
)

func setupTestDB(t *testing.T) *SQLiteStore {
	t.Helper()
	tmpFile := t.TempDir() + "/test.db"
	s, err := NewSQLiteStore(tmpFile)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() {
		s.Close()
		os.Remove(tmpFile)
	})
	return s
}

func TestSQLiteStore_SaveAndGetProject(t *testing.T) {
	s := setupTestDB(t)
	ctx := context.Background()

	p := domain.NewProject("测试剧", domain.StyleManga, 10)
	if err := s.SaveProject(ctx, p); err != nil {
		t.Fatalf("SaveProject: %v", err)
	}

	got, err := s.GetProject(ctx, p.ID)
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if got.Name != "测试剧" {
		t.Errorf("expected name '测试剧', got '%s'", got.Name)
	}
	if got.Style != domain.StyleManga {
		t.Errorf("expected style manga, got %v", got.Style)
	}
}

func TestSQLiteStore_UpdateProjectStatus(t *testing.T) {
	s := setupTestDB(t)
	ctx := context.Background()

	p := domain.NewProject("测试剧", domain.StyleManga, 10)
	s.SaveProject(ctx, p)

	if err := s.UpdateProjectStatus(ctx, p.ID, domain.StatusProcessing); err != nil {
		t.Fatalf("UpdateProjectStatus: %v", err)
	}

	got, _ := s.GetProject(ctx, p.ID)
	if got.Status != domain.StatusProcessing {
		t.Errorf("expected status processing, got %v", got.Status)
	}
}

func TestSQLiteStore_SaveAndListAssets(t *testing.T) {
	s := setupTestDB(t)
	ctx := context.Background()

	a1 := domain.NewAsset("孙悟空", domain.AssetTypeCharacter, domain.AssetScopeProject, "proj_1")
	a2 := domain.NewAsset("宫殿", domain.AssetTypeScene, domain.AssetScopeProject, "proj_1")
	a3 := domain.NewAsset("通用森林", domain.AssetTypeScene, domain.AssetScopeGlobal, "")
	s.SaveAsset(ctx, a1)
	s.SaveAsset(ctx, a2)
	s.SaveAsset(ctx, a3)

	// List project characters
	chars, err := s.ListAssets(ctx, domain.AssetScopeProject, "proj_1", domain.AssetTypeCharacter)
	if err != nil {
		t.Fatalf("ListAssets: %v", err)
	}
	if len(chars) != 1 {
		t.Errorf("expected 1 character, got %d", len(chars))
	}

	// List global scenes
	scenes, err := s.ListAssets(ctx, domain.AssetScopeGlobal, "", domain.AssetTypeScene)
	if err != nil {
		t.Fatalf("ListAssets: %v", err)
	}
	if len(scenes) != 1 {
		t.Errorf("expected 1 global scene, got %d", len(scenes))
	}
}
