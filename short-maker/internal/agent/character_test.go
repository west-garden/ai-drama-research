package agent

import (
	"context"
	"testing"

	"github.com/west-garden/short-maker/internal/domain"
	"github.com/west-garden/short-maker/internal/llm"
	"github.com/west-garden/short-maker/internal/store"
)

const sampleCharacterResponse = `{
	"visual_prompt": "金色毛发的猴王战士，身穿华丽金甲红披风，头戴金箍，手持金箍棒，体格矫健，眼神锐利而调皮",
	"appearance": {
		"face": "棱角分明，金色锐利双眸，尖耳，调皮的笑容",
		"body": "精瘦有力，矫健身姿，约170cm",
		"clothing": "金色战甲，红色披风，虎皮裙",
		"distinctive_features": ["金箍", "金箍棒", "筋斗云靴"]
	}
}`

func TestCharacterAgent_Run(t *testing.T) {
	mockLLM := llm.NewMockClient()
	mockLLM.SetDefaultResponse(sampleCharacterResponse)

	tmpDir := t.TempDir()
	testStore, err := store.NewSQLiteStore(tmpDir + "/test.db")
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	defer testStore.Close()

	agent := NewCharacterAgent(mockLLM, "test-model", testStore)

	project := domain.NewProject("西游记测试", domain.StyleManga, 2)
	bp := domain.NewStoryBlueprint(project.ID)
	bp.AddCharacter("孙悟空", "齐天大圣", []string{"好斗", "忠诚"})
	bp.AddCharacter("唐僧", "取经人", []string{"慈悲", "坚定"})

	state := NewPipelineState(project, "script")
	state.Blueprint = bp

	result, err := agent.Run(context.Background(), state)
	if err != nil {
		t.Fatalf("CharacterAgent.Run: %v", err)
	}

	// Should have created 2 character assets
	if len(result.Assets) != 2 {
		t.Fatalf("expected 2 assets, got %d", len(result.Assets))
	}

	// Verify first asset
	a := result.Assets[0]
	if a.Type != domain.AssetTypeCharacter {
		t.Errorf("expected type character, got %v", a.Type)
	}
	if a.Scope != domain.AssetScopeProject {
		t.Errorf("expected scope project, got %v", a.Scope)
	}
	if a.ProjectID != project.ID {
		t.Errorf("expected projectID '%s', got '%s'", project.ID, a.ProjectID)
	}
	if a.Metadata["character_id"] != bp.Characters[0].ID {
		t.Errorf("expected metadata character_id '%s', got '%s'", bp.Characters[0].ID, a.Metadata["character_id"])
	}
	if a.Metadata["visual_prompt"] == "" {
		t.Error("expected non-empty visual_prompt in metadata")
	}

	// Verify LLM was called once per character
	if len(mockLLM.Calls()) != 2 {
		t.Errorf("expected 2 LLM calls, got %d", len(mockLLM.Calls()))
	}

	// Verify asset was persisted to store
	stored, err := testStore.ListAssets(context.Background(), domain.AssetScopeProject, project.ID, domain.AssetTypeCharacter)
	if err != nil {
		t.Fatalf("ListAssets: %v", err)
	}
	if len(stored) != 2 {
		t.Errorf("expected 2 stored assets, got %d", len(stored))
	}
}

func TestCharacterAgent_Phase(t *testing.T) {
	agent := NewCharacterAgent(llm.NewMockClient(), "model", nil)
	if agent.Phase() != PhaseCharacterAsset {
		t.Errorf("expected phase character_asset, got %s", agent.Phase())
	}
}

func TestCharacterAgent_NilBlueprint(t *testing.T) {
	mockLLM := llm.NewMockClient()
	agent := NewCharacterAgent(mockLLM, "model", nil)
	project := domain.NewProject("test", domain.StyleManga, 1)
	state := NewPipelineState(project, "script")
	// Blueprint is nil

	_, err := agent.Run(context.Background(), state)
	if err == nil {
		t.Error("expected error for nil blueprint, got nil")
	}
}
