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

func TestVideoGenAgent_Phase(t *testing.T) {
	agent := NewVideoGenAgent(nil, nil, "")
	if agent.Phase() != PhaseVideoGeneration {
		t.Errorf("expected phase video_generation, got %s", agent.Phase())
	}
}

func TestVideoGenAgent_NilImages(t *testing.T) {
	agent := NewVideoGenAgent(
		router.NewModelRouter(router.NewMockVideoAdapter()),
		quality.NewMockChecker(),
		t.TempDir(),
	)
	project := domain.NewProject("test", domain.StyleManga, 1)
	state := NewPipelineState(project, "script")
	_, err := agent.Run(context.Background(), state)
	if err == nil {
		t.Error("expected error for nil images")
	}
}

func TestVideoGenAgent_Run(t *testing.T) {
	outputDir := t.TempDir()
	r := router.NewModelRouter(router.NewMockVideoAdapter())
	checker := quality.NewMockChecker()
	agent := NewVideoGenAgent(r, checker, outputDir)

	project := domain.NewProject("测试项目", domain.StyleManga, 1)
	bp := domain.NewStoryBlueprint(project.ID)
	bp.AddEpisodeBlueprintWithRole(1, domain.EpisodeRoleHook, "紧张")

	shot1 := domain.NewShotSpec(1, 1)
	shot1.CameraMove = "pan"
	shot1.RhythmPosition = domain.RhythmOpenHook
	shot1.ContentType = domain.ContentFirstAppear
	shot1.Prompt = "test prompt 1"

	shot2 := domain.NewShotSpec(1, 2)
	shot2.CameraMove = "zoom_in"
	shot2.RhythmPosition = domain.RhythmMidNarration
	shot2.ContentType = domain.ContentDialogue
	shot2.Prompt = "test prompt 2"

	// Pre-create image files (simulating ImageGenAgent output)
	img1Path := filepath.Join(outputDir, project.ID, "ep01", "shot001.png")
	img2Path := filepath.Join(outputDir, project.ID, "ep01", "shot002.png")
	os.MkdirAll(filepath.Dir(img1Path), 0755)
	os.WriteFile(img1Path, []byte("fake-png"), 0644)
	os.WriteFile(img2Path, []byte("fake-png"), 0644)

	state := NewPipelineState(project, "script")
	state.Blueprint = bp
	state.Storyboard = []*domain.ShotSpec{shot1, shot2}
	state.Images = []*GeneratedShot{
		{ShotNumber: 1, EpisodeNum: 1, ImagePath: img1Path, Grade: domain.GradeS, ImageScore: 90},
		{ShotNumber: 2, EpisodeNum: 1, ImagePath: img2Path, Grade: domain.GradeB, ImageScore: 85},
	}

	result, err := agent.Run(context.Background(), state)
	if err != nil {
		t.Fatalf("VideoGenAgent.Run: %v", err)
	}

	if len(result.Videos) != 2 {
		t.Fatalf("expected 2 videos, got %d", len(result.Videos))
	}

	vid1 := result.Videos[0]
	if vid1.ShotNumber != 1 {
		t.Errorf("expected shot number 1, got %d", vid1.ShotNumber)
	}
	if vid1.VideoPath == "" {
		t.Error("expected non-empty video path")
	}
	// Grade is recomputed: hook × open_hook × first_appear = 1.5 × 1.4 × 1.3 = 2.73 → S
	if vid1.Grade != domain.GradeS {
		t.Errorf("expected grade S, got %s", vid1.Grade)
	}
	if vid1.ImagePath != img1Path {
		t.Errorf("expected ImagePath carried from input, got %s", vid1.ImagePath)
	}

	expectedVideoURL := fmt.Sprintf("/output/%s/ep01/shot001.mp4", project.ID)
	if vid1.VideoPath != expectedVideoURL {
		t.Errorf("expected video URL path %q, got %q", expectedVideoURL, vid1.VideoPath)
	}

	// Verify the actual file exists on disk at the filesystem path
	fsPath := filepath.Join(outputDir, project.ID, "ep01", "shot001.mp4")
	if _, err := os.Stat(fsPath); err != nil {
		t.Errorf("video file not found at %s: %v", fsPath, err)
	}
}
