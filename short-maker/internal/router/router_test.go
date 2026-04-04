// internal/router/router_test.go
package router

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/west-garden/short-maker/internal/domain"
)

func TestModelRouter_RouteImage(t *testing.T) {
	r := NewModelRouter(NewMockImageAdapter(), NewMockVideoAdapter())
	adapter, err := r.Route(domain.GradeB, "manga", ModelTypeImage)
	if err != nil {
		t.Fatalf("Route: %v", err)
	}
	if adapter.Name() != "mock-image" {
		t.Errorf("expected mock-image, got %s", adapter.Name())
	}
}

func TestModelRouter_RouteVideo(t *testing.T) {
	r := NewModelRouter(NewMockImageAdapter(), NewMockVideoAdapter())
	adapter, err := r.Route(domain.GradeB, "manga", ModelTypeVideo)
	if err != nil {
		t.Fatalf("Route: %v", err)
	}
	if adapter.Name() != "mock-video" {
		t.Errorf("expected mock-video, got %s", adapter.Name())
	}
}

func TestModelRouter_NoMatch(t *testing.T) {
	r := NewModelRouter(NewMockImageAdapter())
	_, err := r.Route(domain.GradeB, "manga", ModelTypeVideo)
	if err == nil {
		t.Error("expected error for no matching adapter")
	}
}

func TestModelRouter_StyleFilter(t *testing.T) {
	r := NewModelRouter(NewMockImageAdapter())
	_, err := r.Route(domain.GradeB, "manga", ModelTypeImage)
	if err != nil {
		t.Fatalf("Route with manga style: %v", err)
	}
	_, err = r.Route(domain.GradeB, "3d", ModelTypeImage)
	if err != nil {
		t.Fatalf("Route with 3d style: %v", err)
	}
}

func TestModelRouter_Generate(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "test.png")

	r := NewModelRouter(NewMockImageAdapter(), NewMockVideoAdapter())
	resp, err := r.Generate(context.Background(), domain.GradeA, "manga", ModelTypeImage, GenerateRequest{
		Prompt:     "test",
		Style:      "manga",
		OutputPath: outPath,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if resp.FilePath != outPath {
		t.Errorf("expected path '%s', got '%s'", outPath, resp.FilePath)
	}
}
