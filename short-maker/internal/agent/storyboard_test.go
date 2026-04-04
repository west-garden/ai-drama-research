package agent

import (
	"context"
	"testing"

	"github.com/west-garden/short-maker/internal/domain"
	"github.com/west-garden/short-maker/internal/llm"
	"github.com/west-garden/short-maker/internal/strategy"
)

const testStrategiesJSON = `[
	{
		"id": "strat_001",
		"name": "悬念特写",
		"tags": {
			"narrative_beat": ["冲突", "反转"],
			"emotion_arc": ["紧张", "震惊"],
			"pacing": ["fast"],
			"character_count": [1, 2]
		},
		"shot_formula": {
			"frame_type": "close_up",
			"composition": "center",
			"camera_move": "zoom_in",
			"duration": "short"
		},
		"examples": ["角色发现真相的震惊特写"],
		"weight": 1.0
	},
	{
		"id": "strat_002",
		"name": "全景建立",
		"tags": {
			"narrative_beat": ["开场", "转场"],
			"emotion_arc": ["平静", "期待"],
			"pacing": ["slow", "medium"],
			"character_count": [0, 1, 2, 3]
		},
		"shot_formula": {
			"frame_type": "extreme_wide",
			"composition": "center",
			"camera_move": "pan",
			"duration": "long"
		},
		"examples": ["远景展示宏大场景"],
		"weight": 1.0
	}
]`

const sampleStoryboardResponse = `{
	"shots": [
		{
			"strategy_id": "strat_002",
			"frame_type": "extreme_wide",
			"composition": "center",
			"camera_move": "pan",
			"emotion": "壮阔而神秘",
			"prompt": "manga style, extreme wide shot of a mystical island floating in clouds, lush green vegetation, ancient stone structures, ethereal mist, golden sunlight filtering through clouds",
			"character_names": [],
			"scene_ref": "仙灵岛",
			"rhythm_position": "open_hook",
			"content_type": "empty"
		},
		{
			"strategy_id": "strat_001",
			"frame_type": "close_up",
			"composition": "center",
			"camera_move": "zoom_in",
			"emotion": "好奇与惊喜",
			"prompt": "manga style, close-up of a young swordsman discovering a beautiful maiden by a spring, surprised expression, cherry blossoms falling, soft lighting",
			"character_names": ["李逍遥", "赵灵儿"],
			"scene_ref": "仙灵岛",
			"rhythm_position": "emotion_peak",
			"content_type": "first_appear"
		}
	]
}`

func TestStoryboardAgent_Run(t *testing.T) {
	mockLLM := llm.NewMockClient()
	mockLLM.SetDefaultResponse(sampleStoryboardResponse)

	repo, err := strategy.LoadFromJSON([]byte(testStrategiesJSON))
	if err != nil {
		t.Fatalf("load strategies: %v", err)
	}

	agent := NewStoryboardAgent(mockLLM, "test-model", repo)

	project := domain.NewProject("仙剑测试", domain.StyleManga, 1)
	bp := domain.NewStoryBlueprint(project.ID)
	ch1 := bp.AddCharacter("李逍遥", "少年侠客", []string{"正义"})
	ch2 := bp.AddCharacter("赵灵儿", "苗族圣女", []string{"温柔"})

	ep := bp.AddEpisodeBlueprintWithRole(1, domain.EpisodeRoleHook, "好奇→震撼")
	ep.Synopsis = "李逍遥在仙灵岛邂逅赵灵儿"
	ep.Scenes = []domain.SceneTag{
		{
			NarrativeBeat:  "开场",
			EmotionArc:     "平静→好奇",
			Setting:        "仙灵岛",
			Pacing:         "medium",
			CharacterCount: 2,
		},
	}

	state := NewPipelineState(project, "script")
	state.Blueprint = bp
	state.Assets = []*domain.Asset{
		{ID: "asset_1", Metadata: map[string]string{"character_id": ch1.ID}},
		{ID: "asset_2", Metadata: map[string]string{"character_id": ch2.ID}},
	}

	result, err := agent.Run(context.Background(), state)
	if err != nil {
		t.Fatalf("StoryboardAgent.Run: %v", err)
	}

	// Should have 2 shots from the mock response
	if len(result.Storyboard) != 2 {
		t.Fatalf("expected 2 shots, got %d", len(result.Storyboard))
	}

	// Verify first shot
	shot1 := result.Storyboard[0]
	if shot1.EpisodeNumber != 1 {
		t.Errorf("expected episode 1, got %d", shot1.EpisodeNumber)
	}
	if shot1.ShotNumber != 1 {
		t.Errorf("expected shot 1, got %d", shot1.ShotNumber)
	}
	if shot1.FrameType != "extreme_wide" {
		t.Errorf("expected frame_type extreme_wide, got %s", shot1.FrameType)
	}
	if shot1.StrategyID != "strat_002" {
		t.Errorf("expected strategy strat_002, got %s", shot1.StrategyID)
	}
	if shot1.RhythmPosition != domain.RhythmOpenHook {
		t.Errorf("expected rhythm open_hook, got %s", shot1.RhythmPosition)
	}
	if shot1.ContentType != domain.ContentEmpty {
		t.Errorf("expected content empty, got %s", shot1.ContentType)
	}
	if shot1.Prompt == "" {
		t.Error("expected non-empty prompt")
	}

	// Verify second shot has character refs resolved to IDs
	shot2 := result.Storyboard[1]
	if len(shot2.CharacterRefs) != 2 {
		t.Fatalf("expected 2 character refs, got %d", len(shot2.CharacterRefs))
	}
	if shot2.CharacterRefs[0] != ch1.ID {
		t.Errorf("expected character ref '%s', got '%s'", ch1.ID, shot2.CharacterRefs[0])
	}
	if shot2.ContentType != domain.ContentFirstAppear {
		t.Errorf("expected content first_appear, got %s", shot2.ContentType)
	}
}

func TestStoryboardAgent_Phase(t *testing.T) {
	agent := NewStoryboardAgent(llm.NewMockClient(), "model", nil)
	if agent.Phase() != PhaseStoryboard {
		t.Errorf("expected phase storyboard, got %s", agent.Phase())
	}
}

func TestStoryboardAgent_NilBlueprint(t *testing.T) {
	agent := NewStoryboardAgent(llm.NewMockClient(), "model", nil)
	project := domain.NewProject("test", domain.StyleManga, 1)
	state := NewPipelineState(project, "script")

	_, err := agent.Run(context.Background(), state)
	if err == nil {
		t.Error("expected error for nil blueprint")
	}
}
