package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/west-garden/short-maker/internal/domain"
	"github.com/west-garden/short-maker/internal/quality"
	"github.com/west-garden/short-maker/internal/router"
)

func TestImageGenAgent_Phase(t *testing.T) {
	agent := NewImageGenAgent(nil, nil, "")
	if agent.Phase() != PhaseImageGeneration {
		t.Errorf("expected phase image_generation, got %s", agent.Phase())
	}
}

func TestImageGenAgent_NilStoryboard(t *testing.T) {
	agent := NewImageGenAgent(
		router.NewModelRouter(router.NewMockImageAdapter()),
		quality.NewMockChecker(),
		t.TempDir(),
	)
	project := domain.NewProject("test", domain.StyleManga, 1)
	state := NewPipelineState(project, "script")
	_, err := agent.Run(context.Background(), state)
	if err == nil {
		t.Error("expected error for nil storyboard")
	}
}

func TestImageGenAgent_Run(t *testing.T) {
	outputDir := t.TempDir()
	r := router.NewModelRouter(router.NewMockImageAdapter())
	checker := quality.NewMockChecker()
	agent := NewImageGenAgent(r, checker, outputDir)

	project := domain.NewProject("测试项目", domain.StyleManga, 1)
	bp := domain.NewStoryBlueprint(project.ID)
	ch := bp.AddCharacter("孙悟空", "齐天大圣", []string{"好斗"})
	bp.AddEpisodeBlueprintWithRole(1, domain.EpisodeRoleHook, "紧张→释放")

	shot1 := domain.NewShotSpec(1, 1)
	shot1.FrameType = "wide"
	shot1.RhythmPosition = domain.RhythmOpenHook
	shot1.ContentType = domain.ContentFirstAppear
	shot1.Prompt = "五行山全景"
	shot1.AddCharacterRef(ch.ID)

	shot2 := domain.NewShotSpec(1, 2)
	shot2.FrameType = "close_up"
	shot2.RhythmPosition = domain.RhythmMidNarration
	shot2.ContentType = domain.ContentDialogue
	shot2.Prompt = "孙悟空特写"
	shot2.AddCharacterRef(ch.ID)

	charAsset := domain.NewAsset("孙悟空_ref", domain.AssetTypeCharacter, domain.AssetScopeProject, project.ID)
	charAsset.Metadata["character_id"] = ch.ID

	state := NewPipelineState(project, "script")
	state.Blueprint = bp
	state.Storyboard = []*domain.ShotSpec{shot1, shot2}
	state.Assets = []*domain.Asset{charAsset}

	result, err := agent.Run(context.Background(), state)
	if err != nil {
		t.Fatalf("ImageGenAgent.Run: %v", err)
	}

	if len(result.Images) != 2 {
		t.Fatalf("expected 2 images, got %d", len(result.Images))
	}

	img1 := result.Images[0]
	if img1.ShotNumber != 1 {
		t.Errorf("expected shot number 1, got %d", img1.ShotNumber)
	}
	if img1.EpisodeNum != 1 {
		t.Errorf("expected episode 1, got %d", img1.EpisodeNum)
	}
	expectedURL := fmt.Sprintf("/output/%s/ep01/shot001.png", project.ID)
	if img1.ImagePath != expectedURL {
		t.Errorf("expected image URL path %q, got %q", expectedURL, img1.ImagePath)
	}
	if img1.ImageScore != 90 {
		t.Errorf("expected image score 90 (from mock checker), got %d", img1.ImageScore)
	}

	// Verify the actual file exists on disk at the filesystem path
	fsPath := filepath.Join(outputDir, project.ID, "ep01", "shot001.png")
	if _, err := os.Stat(fsPath); err != nil {
		t.Errorf("image file not found at %s: %v", fsPath, err)
	}
}

func TestImageGenAgent_ImportanceGrade(t *testing.T) {
	outputDir := t.TempDir()
	r := router.NewModelRouter(router.NewMockImageAdapter())
	checker := quality.NewMockChecker()
	agent := NewImageGenAgent(r, checker, outputDir)

	project := domain.NewProject("test", domain.StyleManga, 1)
	bp := domain.NewStoryBlueprint(project.ID)
	bp.AddEpisodeBlueprintWithRole(1, domain.EpisodeRoleHook, "紧张")

	// S-grade shot: hook × open_hook × first_appear = 1.5 × 1.4 × 1.3 = 2.73
	shotS := domain.NewShotSpec(1, 1)
	shotS.RhythmPosition = domain.RhythmOpenHook
	shotS.ContentType = domain.ContentFirstAppear
	shotS.Prompt = "s-grade test"

	// C-grade shot: transition × mid_narration × empty = 1.0 × 1.0 × 0.8 = 0.8 → C
	bp.AddEpisodeBlueprintWithRole(2, domain.EpisodeRoleTransition, "平静")
	shotC := domain.NewShotSpec(2, 2)
	shotC.RhythmPosition = domain.RhythmMidNarration
	shotC.ContentType = domain.ContentEmpty
	shotC.Prompt = "c-grade test"

	state := NewPipelineState(project, "script")
	state.Blueprint = bp
	state.Storyboard = []*domain.ShotSpec{shotS, shotC}

	result, err := agent.Run(context.Background(), state)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if result.Images[0].Grade != domain.GradeS {
		t.Errorf("expected shot 1 grade S, got %s", result.Images[0].Grade)
	}
	if result.Images[1].Grade != domain.GradeC {
		t.Errorf("expected shot 2 grade C, got %s", result.Images[1].Grade)
	}
}

func TestImageGenAgent_RetryOnFailure(t *testing.T) {
	outputDir := t.TempDir()
	r := router.NewModelRouter(router.NewMockImageAdapter())

	failOnce := &failOnceChecker{failCount: 1}
	agent := NewImageGenAgent(r, failOnce, outputDir)

	project := domain.NewProject("test", domain.StyleManga, 1)
	bp := domain.NewStoryBlueprint(project.ID)
	bp.AddEpisodeBlueprintWithRole(1, domain.EpisodeRoleHook, "紧张")

	// Grade A shot: hook × emotion_peak × dialogue = 1.5 × 1.2 × 1.0 = 1.8 → A (maxRetries=2)
	shot := domain.NewShotSpec(1, 1)
	shot.RhythmPosition = domain.RhythmEmotionPeak
	shot.ContentType = domain.ContentDialogue
	shot.Prompt = "retry test"

	state := NewPipelineState(project, "script")
	state.Blueprint = bp
	state.Storyboard = []*domain.ShotSpec{shot}

	result, err := agent.Run(context.Background(), state)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(result.Images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(result.Images))
	}
	if result.Images[0].ImageScore <= 0 {
		t.Error("expected positive image score after retry")
	}
	if failOnce.callCount != 2 {
		t.Errorf("expected 2 quality check calls (1 fail + 1 pass), got %d", failOnce.callCount)
	}
}

// failOnceChecker fails the first N checks, then passes.
type failOnceChecker struct {
	failCount int
	callCount int
}

func (c *failOnceChecker) Check(_ context.Context, _ string, shotSpec *domain.ShotSpec, _ []*domain.Asset) (*quality.QualityReport, error) {
	c.callCount++
	if c.callCount <= c.failCount {
		return quality.NewReport(shotSpec.ShotNumber, quality.DefaultDimensions(50), domain.GradeS), nil
	}
	return quality.NewReport(shotSpec.ShotNumber, quality.DefaultDimensions(90), domain.GradeC), nil
}
