package store

import (
	"context"
	"testing"

	"github.com/west-garden/short-maker/internal/domain"
)

func TestSavePipelineRun(t *testing.T) {
	s := setupTestDB(t)
	ctx := context.Background()

	run := &PipelineRunRecord{
		ProjectID:    "proj_test_1",
		Status:       "running",
		CurrentPhase: "story_understanding",
	}
	if err := s.SavePipelineRun(ctx, run); err != nil {
		t.Fatalf("SavePipelineRun: %v", err)
	}

	got, err := s.GetPipelineRun(ctx, "proj_test_1")
	if err != nil {
		t.Fatalf("GetPipelineRun: %v", err)
	}
	if got.Status != "running" {
		t.Errorf("expected status 'running', got '%s'", got.Status)
	}
	if got.CurrentPhase != "story_understanding" {
		t.Errorf("expected phase 'story_understanding', got '%s'", got.CurrentPhase)
	}
}

func TestUpdatePipelineRun(t *testing.T) {
	s := setupTestDB(t)
	ctx := context.Background()

	run := &PipelineRunRecord{
		ProjectID:    "proj_test_2",
		Status:       "running",
		CurrentPhase: "story_understanding",
	}
	s.SavePipelineRun(ctx, run)

	if err := s.UpdatePipelineRun(ctx, "proj_test_2", "completed", "video_generation", ""); err != nil {
		t.Fatalf("UpdatePipelineRun: %v", err)
	}

	got, _ := s.GetPipelineRun(ctx, "proj_test_2")
	if got.Status != "completed" {
		t.Errorf("expected status 'completed', got '%s'", got.Status)
	}
	if got.CurrentPhase != "video_generation" {
		t.Errorf("expected phase 'video_generation', got '%s'", got.CurrentPhase)
	}
}

func TestSaveAndGetPipelineResult(t *testing.T) {
	s := setupTestDB(t)
	ctx := context.Background()

	resultJSON := []byte(`{"project":{"id":"proj_test_3"},"storyboard":[]}`)
	if err := s.SavePipelineResult(ctx, "proj_test_3", resultJSON); err != nil {
		t.Fatalf("SavePipelineResult: %v", err)
	}

	got, err := s.GetPipelineResult(ctx, "proj_test_3")
	if err != nil {
		t.Fatalf("GetPipelineResult: %v", err)
	}
	if string(got) != string(resultJSON) {
		t.Errorf("expected '%s', got '%s'", resultJSON, got)
	}
}

func TestGetPipelineRun_NotFound(t *testing.T) {
	s := setupTestDB(t)
	ctx := context.Background()

	_, err := s.GetPipelineRun(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent pipeline run")
	}
}

func TestListProjects(t *testing.T) {
	s := setupTestDB(t)
	ctx := context.Background()

	p1 := domain.NewProject("项目A", domain.StyleManga, 5)
	p2 := domain.NewProject("项目B", domain.Style3D, 10)
	s.SaveProject(ctx, p1)
	s.SaveProject(ctx, p2)

	projects, err := s.ListProjects(ctx)
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(projects))
	}
}

func TestRecoverRunningPipelines(t *testing.T) {
	s := setupTestDB(t)
	ctx := context.Background()

	run := &PipelineRunRecord{
		ProjectID:    "proj_running",
		Status:       "running",
		CurrentPhase: "storyboard",
	}
	s.SavePipelineRun(ctx, run)

	if err := s.RecoverRunningPipelines(ctx); err != nil {
		t.Fatalf("RecoverRunningPipelines: %v", err)
	}

	got, _ := s.GetPipelineRun(ctx, "proj_running")
	if got.Status != "failed" {
		t.Errorf("expected status 'failed', got '%s'", got.Status)
	}
	if got.Error != "server restarted" {
		t.Errorf("expected error 'server restarted', got '%s'", got.Error)
	}
}
