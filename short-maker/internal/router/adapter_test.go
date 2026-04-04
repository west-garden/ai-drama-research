package router

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestMockImageAdapter_Name(t *testing.T) {
	a := NewMockImageAdapter()
	if a.Name() != "mock-image" {
		t.Errorf("expected name 'mock-image', got '%s'", a.Name())
	}
}

func TestMockImageAdapter_Capabilities(t *testing.T) {
	a := NewMockImageAdapter()
	caps := a.Capabilities()
	if caps.Type != ModelTypeImage {
		t.Errorf("expected type image, got %s", caps.Type)
	}
	if len(caps.Styles) != 3 {
		t.Errorf("expected 3 styles, got %d", len(caps.Styles))
	}
}

func TestMockImageAdapter_Generate(t *testing.T) {
	a := NewMockImageAdapter()
	dir := t.TempDir()
	outPath := filepath.Join(dir, "test_shot.png")

	resp, err := a.Generate(context.Background(), GenerateRequest{
		Prompt:     "test prompt",
		Style:      "manga",
		OutputPath: outPath,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if resp.FilePath != outPath {
		t.Errorf("expected file path '%s', got '%s'", outPath, resp.FilePath)
	}
	if resp.ModelUsed != "mock-image" {
		t.Errorf("expected model 'mock-image', got '%s'", resp.ModelUsed)
	}

	// Verify file was created and has content (valid PNG)
	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	if info.Size() == 0 {
		t.Error("expected non-empty PNG file")
	}
}

func TestMockImageAdapter_HealthCheck(t *testing.T) {
	a := NewMockImageAdapter()
	if err := a.HealthCheck(context.Background()); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}
