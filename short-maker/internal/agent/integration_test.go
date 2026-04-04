// internal/agent/integration_test.go
package agent

import (
	"context"
	"os"
	"testing"

	"github.com/west-garden/short-maker/internal/domain"
	"github.com/west-garden/short-maker/internal/llm"
	"github.com/west-garden/short-maker/internal/quality"
	"github.com/west-garden/short-maker/internal/router"
	"github.com/west-garden/short-maker/internal/store"
	"github.com/west-garden/short-maker/internal/strategy"
)

func TestIntegration_FullPipelineWithMockAgents(t *testing.T) {
	// This test verifies the full pipeline runs end-to-end with mock agents
	// that produce realistic-looking output at each stage.

	script := "第一集：初遇\n孙悟空从五行山下被唐僧解救。"
	project := domain.NewProject("西游记测试", domain.StyleManga, 2)
	state := NewPipelineState(project, script)

	// Story Understanding: produces a blueprint
	storyAgent := NewMockAgent(PhaseStoryUnderstanding, func(ctx context.Context, s *PipelineState) (*PipelineState, error) {
		bp := domain.NewStoryBlueprint(s.Project.ID)
		bp.WorldView = "西游记世界"
		bp.AddCharacter("孙悟空", "齐天大圣", []string{"好斗", "忠诚"})
		bp.AddCharacter("唐僧", "取经人", []string{"慈悲", "坚定"})
		bp.AddEpisodeBlueprintWithRole(1, domain.EpisodeRoleHook, "紧张→释放")
		bp.AddEpisodeBlueprintWithRole(2, domain.EpisodeRoleTransition, "冲突→和解")
		s.Blueprint = bp
		return s, nil
	})

	// Character Asset: produces assets
	charAgent := NewMockAgent(PhaseCharacterAsset, func(ctx context.Context, s *PipelineState) (*PipelineState, error) {
		for _, ch := range s.Blueprint.Characters {
			asset := domain.NewAsset(ch.Name+"_三视图", domain.AssetTypeCharacter, domain.AssetScopeProject, s.Project.ID)
			asset.FilePath = "output/" + ch.ID + "_ref.png"
			s.Assets = append(s.Assets, asset)
		}
		return s, nil
	})

	// Storyboard: produces shot specs
	storyboardAgent := NewMockAgent(PhaseStoryboard, func(ctx context.Context, s *PipelineState) (*PipelineState, error) {
		shot1 := domain.NewShotSpec(1, 1)
		shot1.FrameType = "wide"
		shot1.Emotion = "tense"
		shot1.RhythmPosition = domain.RhythmOpenHook
		shot1.ContentType = domain.ContentFirstAppear
		shot1.Prompt = "五行山全景，唐僧骑马走来"
		shot1.AddCharacterRef(s.Blueprint.Characters[1].ID)

		shot2 := domain.NewShotSpec(1, 2)
		shot2.FrameType = "close_up"
		shot2.Emotion = "excited"
		shot2.RhythmPosition = domain.RhythmEmotionPeak
		shot2.ContentType = domain.ContentFirstAppear
		shot2.Prompt = "孙悟空从石缝中探出头"
		shot2.AddCharacterRef(s.Blueprint.Characters[0].ID)

		s.Storyboard = []*domain.ShotSpec{shot1, shot2}
		return s, nil
	})

	// Image Generation: marks shots as having images
	imageAgent := NewMockAgent(PhaseImageGeneration, func(ctx context.Context, s *PipelineState) (*PipelineState, error) {
		for i := range s.Project.Episodes {
			for j := range s.Project.Episodes[i].Shots {
				s.Project.Episodes[i].Shots[j].ImagePath = "output/img_placeholder.png"
			}
		}
		return s, nil
	})

	// Video Generation: marks shots as having video
	videoAgent := NewMockAgent(PhaseVideoGeneration, func(ctx context.Context, s *PipelineState) (*PipelineState, error) {
		return s, nil
	})

	agents := map[Phase]Agent{
		PhaseStoryUnderstanding: storyAgent,
		PhaseCharacterAsset:     charAgent,
		PhaseStoryboard:         storyboardAgent,
		PhaseImageGeneration:    imageAgent,
		PhaseVideoGeneration:    videoAgent,
	}

	orch := NewOrchestrator(agents, nil)
	result, err := orch.Run(context.Background(), state)
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}

	// Verify pipeline produced expected outputs
	if result.Blueprint == nil {
		t.Fatal("expected blueprint to be set")
	}
	if len(result.Blueprint.Characters) != 2 {
		t.Errorf("expected 2 characters, got %d", len(result.Blueprint.Characters))
	}
	if len(result.Assets) != 2 {
		t.Errorf("expected 2 assets, got %d", len(result.Assets))
	}
	if len(result.Storyboard) != 2 {
		t.Errorf("expected 2 shot specs, got %d", len(result.Storyboard))
	}

	// Verify importance scoring works on the shot specs
	shot1Score := domain.NewImportanceScore(
		domain.EpisodeRoleHook,
		result.Storyboard[0].RhythmPosition,
		result.Storyboard[0].ContentType,
	)
	if shot1Score.Grade() != domain.GradeS {
		t.Errorf("expected shot 1 (episode 1 open hook, first appear) to be grade S, got %v (score: %v)",
			shot1Score.Grade(), shot1Score.Score())
	}
}

func TestIntegration_StoryAndCharacterAgentsWithMockLLM(t *testing.T) {
	storyJSON := `{
		"world_view": "古代仙侠世界",
		"characters": [
			{
				"name": "李逍遥",
				"description": "天资聪颖的少年侠客",
				"traits": ["正义", "热血", "重情义"]
			},
			{
				"name": "赵灵儿",
				"description": "苗族圣女，温柔善良",
				"traits": ["温柔", "善良", "坚强"]
			}
		],
		"episodes": [
			{
				"number": 1,
				"role": "hook",
				"emotion_arc": "好奇→震撼",
				"synopsis": "李逍遥在仙灵岛邂逅赵灵儿",
				"scenes": [
					{
						"narrative_beat": "开场",
						"emotion_arc": "平静→好奇",
						"setting": "仙灵岛",
						"pacing": "medium",
						"character_count": 2
					}
				]
			}
		],
		"relationships": [
			{
				"character_a": "李逍遥",
				"character_b": "赵灵儿",
				"type": "恋人"
			}
		]
	}`

	characterJSON := `{
		"visual_prompt": "一位英俊的少年侠客，身穿蓝色长袍，手持长剑，眼神坚定",
		"appearance": {
			"face": "剑眉星目，英俊潇洒",
			"body": "身材修长，姿态飘逸",
			"clothing": "蓝色仙侠长袍，腰佩长剑",
			"distinctive_features": ["蓝色长袍", "配剑"]
		}
	}`

	// sequentialMockClient returns different responses for each call.
	customMock := &sequentialMockClient{
		responses: []string{storyJSON, characterJSON, characterJSON},
	}

	tmpDir := t.TempDir()
	testStore, err := store.NewSQLiteStore(tmpDir + "/test.db")
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	defer testStore.Close()

	storyAgent := NewStoryAgent(customMock, "test-model")
	charAgent := NewCharacterAgent(customMock, "test-model", testStore)

	agents := map[Phase]Agent{
		PhaseStoryUnderstanding: storyAgent,
		PhaseCharacterAsset:     charAgent,
		PhaseStoryboard: NewMockAgent(PhaseStoryboard, func(ctx context.Context, s *PipelineState) (*PipelineState, error) {
			return s, nil
		}),
		PhaseImageGeneration: NewMockAgent(PhaseImageGeneration, func(ctx context.Context, s *PipelineState) (*PipelineState, error) {
			return s, nil
		}),
		PhaseVideoGeneration: NewMockAgent(PhaseVideoGeneration, func(ctx context.Context, s *PipelineState) (*PipelineState, error) {
			return s, nil
		}),
	}

	orch := NewOrchestrator(agents, nil)
	project := domain.NewProject("仙剑奇侠传", domain.StyleManga, 1)
	state := NewPipelineState(project, "第一集：仙灵岛\n李逍遥在仙灵岛邂逅赵灵儿。")

	result, err := orch.Run(context.Background(), state)
	if err != nil {
		t.Fatalf("pipeline: %v", err)
	}

	// Verify StoryAgent output
	if result.Blueprint == nil {
		t.Fatal("expected Blueprint")
	}
	if result.Blueprint.WorldView != "古代仙侠世界" {
		t.Errorf("unexpected world_view: %s", result.Blueprint.WorldView)
	}
	if len(result.Blueprint.Characters) != 2 {
		t.Fatalf("expected 2 characters, got %d", len(result.Blueprint.Characters))
	}
	if result.Blueprint.Characters[0].Name != "李逍遥" {
		t.Errorf("expected first character '李逍遥', got '%s'", result.Blueprint.Characters[0].Name)
	}

	// Verify CharacterAgent output
	if len(result.Assets) != 2 {
		t.Fatalf("expected 2 assets, got %d", len(result.Assets))
	}
	for _, a := range result.Assets {
		if a.Type != domain.AssetTypeCharacter {
			t.Errorf("expected asset type character, got %v", a.Type)
		}
		if a.Metadata["visual_prompt"] == "" {
			t.Errorf("expected non-empty visual_prompt for asset %s", a.Name)
		}
		if a.Metadata["character_id"] == "" {
			t.Errorf("expected non-empty character_id for asset %s", a.Name)
		}
	}

	// Verify persistence
	storedAssets, _ := testStore.ListAssets(context.Background(), domain.AssetScopeProject, project.ID, domain.AssetTypeCharacter)
	if len(storedAssets) != 2 {
		t.Errorf("expected 2 persisted assets, got %d", len(storedAssets))
	}
}

// sequentialMockClient returns different responses for each call.
type sequentialMockClient struct {
	responses []string
	index     int
}

func (m *sequentialMockClient) Chat(ctx context.Context, req llm.Request) (*llm.Response, error) {
	if m.index >= len(m.responses) {
		return &llm.Response{Content: m.responses[len(m.responses)-1], Model: req.Model}, nil
	}
	resp := m.responses[m.index]
	m.index++
	return &llm.Response{Content: resp, TokensUsed: len(resp), Model: req.Model}, nil
}

func TestIntegration_FullPipelineWithStoryboardAgent(t *testing.T) {
	storyJSON := `{
		"world_view": "古代仙侠世界",
		"characters": [
			{
				"name": "李逍遥",
				"description": "天资聪颖的少年侠客",
				"traits": ["正义", "热血"]
			},
			{
				"name": "赵灵儿",
				"description": "苗族圣女",
				"traits": ["温柔", "善良"]
			}
		],
		"episodes": [
			{
				"number": 1,
				"role": "hook",
				"emotion_arc": "好奇→震撼",
				"synopsis": "邂逅于仙灵岛",
				"scenes": [
					{
						"narrative_beat": "开场",
						"emotion_arc": "平静→好奇",
						"setting": "仙灵岛",
						"pacing": "medium",
						"character_count": 2
					}
				]
			}
		],
		"relationships": [{"character_a": "李逍遥", "character_b": "赵灵儿", "type": "恋人"}]
	}`

	characterJSON := `{
		"visual_prompt": "少年侠客形象",
		"appearance": {
			"face": "剑眉星目",
			"body": "身材修长",
			"clothing": "蓝色长袍",
			"distinctive_features": ["配剑"]
		}
	}`

	storyboardJSON := `{
		"shots": [
			{
				"strategy_id": "strat_002",
				"frame_type": "extreme_wide",
				"composition": "center",
				"camera_move": "pan",
				"emotion": "壮阔",
				"prompt": "manga style, extreme wide shot of mystical island",
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
				"emotion": "惊喜",
				"prompt": "manga style, close-up of young swordsman meeting maiden",
				"character_names": ["李逍遥", "赵灵儿"],
				"scene_ref": "仙灵岛",
				"rhythm_position": "emotion_peak",
				"content_type": "first_appear"
			}
		]
	}`

	// Story call → Character call 1 → Character call 2 → Storyboard call
	customMock := &sequentialMockClient{
		responses: []string{storyJSON, characterJSON, characterJSON, storyboardJSON},
	}

	tmpDir := t.TempDir()
	testStore, err := store.NewSQLiteStore(tmpDir + "/test.db")
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	defer testStore.Close()

	strategyJSON := `[
		{"id":"strat_001","name":"悬念特写","tags":{"narrative_beat":["冲突"],"emotion_arc":["紧张"],"pacing":["fast"],"character_count":[1,2]},"shot_formula":{"frame_type":"close_up","composition":"center","camera_move":"zoom_in","duration":"short"},"examples":[],"weight":1.0},
		{"id":"strat_002","name":"全景建立","tags":{"narrative_beat":["开场"],"emotion_arc":["平静"],"pacing":["slow","medium"],"character_count":[0,1,2]},"shot_formula":{"frame_type":"extreme_wide","composition":"center","camera_move":"pan","duration":"long"},"examples":[],"weight":1.0}
	]`
	repo, _ := strategy.LoadFromJSON([]byte(strategyJSON))

	agents := map[Phase]Agent{
		PhaseStoryUnderstanding: NewStoryAgent(customMock, "test-model"),
		PhaseCharacterAsset:     NewCharacterAgent(customMock, "test-model", testStore),
		PhaseStoryboard:         NewStoryboardAgent(customMock, "test-model", repo),
		PhaseImageGeneration: NewMockAgent(PhaseImageGeneration, func(ctx context.Context, s *PipelineState) (*PipelineState, error) {
			return s, nil
		}),
		PhaseVideoGeneration: NewMockAgent(PhaseVideoGeneration, func(ctx context.Context, s *PipelineState) (*PipelineState, error) {
			return s, nil
		}),
	}

	orch := NewOrchestrator(agents, nil)
	project := domain.NewProject("仙剑奇侠传", domain.StyleManga, 1)
	state := NewPipelineState(project, "第一集：仙灵岛")

	result, err := orch.Run(context.Background(), state)
	if err != nil {
		t.Fatalf("pipeline: %v", err)
	}

	// Verify all three real agents produced output
	if result.Blueprint == nil {
		t.Fatal("expected Blueprint from StoryAgent")
	}
	if len(result.Assets) != 2 {
		t.Fatalf("expected 2 assets from CharacterAgent, got %d", len(result.Assets))
	}
	if len(result.Storyboard) != 2 {
		t.Fatalf("expected 2 shots from StoryboardAgent, got %d", len(result.Storyboard))
	}

	// Verify storyboard shot annotations
	shot1 := result.Storyboard[0]
	if shot1.RhythmPosition != domain.RhythmOpenHook {
		t.Errorf("expected rhythm open_hook, got %s", shot1.RhythmPosition)
	}
	if shot1.ContentType != domain.ContentEmpty {
		t.Errorf("expected content empty, got %s", shot1.ContentType)
	}

	shot2 := result.Storyboard[1]
	if shot2.ContentType != domain.ContentFirstAppear {
		t.Errorf("expected content first_appear, got %s", shot2.ContentType)
	}
	// Character refs should be resolved to IDs (not names)
	if len(shot2.CharacterRefs) != 2 {
		t.Fatalf("expected 2 character refs, got %d", len(shot2.CharacterRefs))
	}
	// Refs should be IDs (start with "char_"), not names
	for _, ref := range shot2.CharacterRefs {
		if len(ref) < 5 {
			t.Errorf("character ref '%s' looks like a name, not an ID", ref)
		}
	}

	// Verify importance scoring works on the annotations
	importance := domain.NewImportanceScore(
		result.Blueprint.Episodes[0].Role, // hook
		shot1.RhythmPosition,              // open_hook
		shot1.ContentType,                 // empty
	)
	// hook(1.5) × open_hook(1.4) × empty(0.8) = 1.68 → Grade A
	if importance.Grade() != domain.GradeA {
		t.Errorf("expected grade A for hook/open_hook/empty, got %s (score: %.2f)", importance.Grade(), importance.Score())
	}
}

func TestIntegration_FullPipelineWithGeneration(t *testing.T) {
	storyJSON := `{
		"world_view": "古代仙侠世界",
		"characters": [
			{"name": "李逍遥", "description": "少年侠客", "traits": ["正义"]},
			{"name": "赵灵儿", "description": "苗族圣女", "traits": ["温柔"]}
		],
		"episodes": [
			{
				"number": 1,
				"role": "hook",
				"emotion_arc": "好奇→震撼",
				"synopsis": "仙灵岛邂逅",
				"scenes": [
					{"narrative_beat": "开场", "emotion_arc": "平静→好奇", "setting": "仙灵岛", "pacing": "medium", "character_count": 2}
				]
			}
		],
		"relationships": [{"character_a": "李逍遥", "character_b": "赵灵儿", "type": "恋人"}]
	}`

	characterJSON := `{
		"visual_prompt": "少年侠客",
		"appearance": {"face": "剑眉星目", "body": "修长", "clothing": "蓝色长袍", "distinctive_features": ["配剑"]}
	}`

	storyboardJSON := `{
		"shots": [
			{
				"strategy_id": "strat_002",
				"frame_type": "extreme_wide",
				"composition": "center",
				"camera_move": "pan",
				"emotion": "壮阔",
				"prompt": "manga style, mystical island wide shot",
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
				"emotion": "惊喜",
				"prompt": "manga style, close-up meeting scene",
				"character_names": ["李逍遥", "赵灵儿"],
				"scene_ref": "仙灵岛",
				"rhythm_position": "emotion_peak",
				"content_type": "first_appear"
			}
		]
	}`

	customMock := &sequentialMockClient{
		responses: []string{storyJSON, characterJSON, characterJSON, storyboardJSON},
	}

	tmpDir := t.TempDir()
	testStore, err := store.NewSQLiteStore(tmpDir + "/test.db")
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	defer testStore.Close()

	stratJSON := `[
		{"id":"strat_001","name":"悬念特写","tags":{"narrative_beat":["冲突"],"emotion_arc":["紧张"],"pacing":["fast"],"character_count":[1,2]},"shot_formula":{"frame_type":"close_up","composition":"center","camera_move":"zoom_in","duration":"short"},"examples":[],"weight":1.0},
		{"id":"strat_002","name":"全景建立","tags":{"narrative_beat":["开场"],"emotion_arc":["平静"],"pacing":["slow","medium"],"character_count":[0,1,2]},"shot_formula":{"frame_type":"extreme_wide","composition":"center","camera_move":"pan","duration":"long"},"examples":[],"weight":1.0}
	]`
	repo, _ := strategy.LoadFromJSON([]byte(stratJSON))

	outputDir := tmpDir + "/output"
	modelRouter := router.NewModelRouter(router.NewMockImageAdapter(), router.NewMockVideoAdapter())
	checker := quality.NewMockChecker()

	agents := map[Phase]Agent{
		PhaseStoryUnderstanding: NewStoryAgent(customMock, "test-model"),
		PhaseCharacterAsset:     NewCharacterAgent(customMock, "test-model", testStore),
		PhaseStoryboard:         NewStoryboardAgent(customMock, "test-model", repo),
		PhaseImageGeneration:    NewImageGenAgent(modelRouter, checker, outputDir),
		PhaseVideoGeneration:    NewVideoGenAgent(modelRouter, checker, outputDir),
	}

	orch := NewOrchestrator(agents, nil)
	project := domain.NewProject("仙剑奇侠传", domain.StyleManga, 1)
	state := NewPipelineState(project, "第一集：仙灵岛")

	result, err := orch.Run(context.Background(), state)
	if err != nil {
		t.Fatalf("pipeline: %v", err)
	}

	// Verify all 5 stages produced output
	if result.Blueprint == nil {
		t.Fatal("expected Blueprint from StoryAgent")
	}
	if len(result.Assets) != 2 {
		t.Fatalf("expected 2 assets, got %d", len(result.Assets))
	}
	if len(result.Storyboard) != 2 {
		t.Fatalf("expected 2 shots, got %d", len(result.Storyboard))
	}
	if len(result.Images) != 2 {
		t.Fatalf("expected 2 images, got %d", len(result.Images))
	}
	if len(result.Videos) != 2 {
		t.Fatalf("expected 2 videos, got %d", len(result.Videos))
	}

	// Verify image files exist
	for _, img := range result.Images {
		if _, err := os.Stat(img.ImagePath); err != nil {
			t.Errorf("image file not found for shot %d: %v", img.ShotNumber, err)
		}
	}

	// Verify video files exist
	for _, vid := range result.Videos {
		if _, err := os.Stat(vid.VideoPath); err != nil {
			t.Errorf("video file not found for shot %d: %v", vid.ShotNumber, err)
		}
	}

	// Verify importance grades
	// Shot 1: hook × open_hook × empty = 1.5 × 1.4 × 0.8 = 1.68 → A
	if result.Images[0].Grade != domain.GradeA {
		t.Errorf("expected shot 1 grade A, got %s", result.Images[0].Grade)
	}
	// Shot 2: hook × emotion_peak × first_appear = 1.5 × 1.2 × 1.3 = 2.34 → S
	if result.Images[1].Grade != domain.GradeS {
		t.Errorf("expected shot 2 grade S, got %s", result.Images[1].Grade)
	}

	// Verify quality scores (MockChecker returns 90)
	for _, img := range result.Images {
		if img.ImageScore != 90 {
			t.Errorf("expected image score 90, got %d for shot %d", img.ImageScore, img.ShotNumber)
		}
	}
	for _, vid := range result.Videos {
		if vid.VideoScore != 90 {
			t.Errorf("expected video score 90, got %d for shot %d", vid.VideoScore, vid.ShotNumber)
		}
	}
}
