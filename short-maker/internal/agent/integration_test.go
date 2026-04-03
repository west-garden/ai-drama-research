// internal/agent/integration_test.go
package agent

import (
	"context"
	"testing"

	"github.com/west-garden/short-maker/internal/domain"
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
