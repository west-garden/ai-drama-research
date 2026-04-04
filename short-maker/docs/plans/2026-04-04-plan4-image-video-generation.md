# Plan 4: Image/Video Generation + Model Router + Quality Check

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the Image Generation Agent (Phase 4), Video Generation Agent (Phase 5), Model Router, and Quality Checker — all with mock backends — to complete the full 5-stage pipeline with a generate→evaluate→retry closed loop.

**Architecture:** Two new infrastructure packages (`internal/router` for model routing, `internal/quality` for quality checking) provide interfaces that agents depend on. ImageGenAgent and VideoGenAgent iterate over shots, compute ImportanceScore grades, route to the appropriate mock adapter, run quality checks, and retry on failure. All generation uses mock adapters that create placeholder files on disk.

**Tech Stack:** Go 1.22+, existing internal packages, `image/png` for minimal PNG creation

**Spec reference:** `short-maker/docs/specs/2026-04-04-plan4-image-video-generation.md`

**Depends on:** Plan 1 (domain types, Agent interface, Orchestrator, ImportanceScore), Plan 2 (StoryAgent, CharacterAgent), Plan 3 (StoryboardAgent, Strategy Engine)

---

## File Structure

```
short-maker/
├── internal/
│   ├── router/
│   │   ├── adapter.go           # ModelAdapter interface, ModelType, Capabilities, GenerateRequest/Response
│   │   ├── adapter_test.go      # Tests for mock adapters
│   │   ├── router.go            # ModelRouter — grade+style+type → adapter selection
│   │   ├── router_test.go       # Tests for routing logic
│   │   ├── mock_image.go        # MockImageAdapter — creates 1x1 PNG placeholder
│   │   └── mock_video.go        # MockVideoAdapter — creates empty MP4 placeholder
│   ├── quality/
│   │   ├── checker.go           # Checker interface, Dimension, QualityReport, NewReport helper
│   │   ├── checker_test.go      # Tests for mock checker and report scoring
│   │   └── mock_checker.go      # MockChecker — always passes with score 90
│   └── agent/
│       ├── agent.go             # (modify) add GeneratedShot type + Images/Videos fields to PipelineState
│       ├── imagegen.go          # ImageGenAgent — shot → router → quality check → retry loop
│       ├── imagegen_test.go     # Tests for image generation + retry behavior
│       ├── videogen.go          # VideoGenAgent — image+camera → router → quality check → retry loop
│       ├── videogen_test.go     # Tests for video generation
│       └── integration_test.go  # (modify) add full 5-stage pipeline test with generation
├── cmd/shortmaker/
│   └── main.go                  # (modify) wire new agents + --output flag
```

---

### Task 1: ModelAdapter Interface + Mock Image Adapter

**Files:**
- Create: `short-maker/internal/router/adapter.go`
- Create: `short-maker/internal/router/mock_image.go`
- Create: `short-maker/internal/router/adapter_test.go`

- [ ] **Step 1: Write tests for MockImageAdapter**

```go
// internal/router/adapter_test.go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./internal/router/ -v -run TestMockImageAdapter`
Expected: compilation error — package does not exist

- [ ] **Step 3: Implement adapter types**

```go
// internal/router/adapter.go
package router

import "context"

// ModelType distinguishes image vs video generation models.
type ModelType string

const (
	ModelTypeImage ModelType = "image"
	ModelTypeVideo ModelType = "video"
)

// Capabilities declares what a model adapter can do.
type Capabilities struct {
	Type           ModelType
	Styles         []string // manga, 3d, live_action
	MaxResolution  string   // e.g. "1024x1024"
	SupportsFusion bool     // supports character reference injection
}

// GenerateRequest is the unified input for all generation adapters.
type GenerateRequest struct {
	Prompt        string            // generation prompt
	Style         string            // target style
	CharacterRefs []string          // character reference image paths (for fusion)
	CameraMove    string            // camera movement directive (video only)
	SourceImage   string            // source image path (video only)
	OutputPath    string            // where to write the output file
	Metadata      map[string]string // extension parameters
}

// GenerateResponse is the unified output from generation adapters.
type GenerateResponse struct {
	FilePath   string  // actual output file path
	ModelUsed  string  // which model was used
	Cost       float64 // cost in USD
	DurationMs int64   // wall-clock milliseconds
}

// ModelAdapter is the interface every generation model must implement.
// Adding a new model = implementing this interface.
type ModelAdapter interface {
	Name() string
	Capabilities() Capabilities
	Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error)
	HealthCheck(ctx context.Context) error
}
```

- [ ] **Step 4: Implement MockImageAdapter**

```go
// internal/router/mock_image.go
package router

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"time"
)

// MockImageAdapter creates a 1x1 PNG placeholder for each generation request.
type MockImageAdapter struct{}

func NewMockImageAdapter() *MockImageAdapter {
	return &MockImageAdapter{}
}

func (a *MockImageAdapter) Name() string { return "mock-image" }

func (a *MockImageAdapter) Capabilities() Capabilities {
	return Capabilities{
		Type:           ModelTypeImage,
		Styles:         []string{"manga", "3d", "live_action"},
		MaxResolution:  "1024x1024",
		SupportsFusion: false,
	}
}

func (a *MockImageAdapter) Generate(_ context.Context, req GenerateRequest) (*GenerateResponse, error) {
	start := time.Now()

	f, err := os.Create(req.OutputPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.White)
	if err := png.Encode(f, img); err != nil {
		return nil, err
	}

	return &GenerateResponse{
		FilePath:   req.OutputPath,
		ModelUsed:  "mock-image",
		Cost:       0.0,
		DurationMs: time.Since(start).Milliseconds(),
	}, nil
}

func (a *MockImageAdapter) HealthCheck(_ context.Context) error {
	return nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./internal/router/ -v -run TestMockImageAdapter`
Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git -C /Users/rain/code/west-garden/ai-drama-research add \
  short-maker/internal/router/adapter.go \
  short-maker/internal/router/mock_image.go \
  short-maker/internal/router/adapter_test.go
git -C /Users/rain/code/west-garden/ai-drama-research commit -m "feat(short-maker): ModelAdapter interface + MockImageAdapter"
```

---

### Task 2: Mock Video Adapter

**Files:**
- Create: `short-maker/internal/router/mock_video.go`
- Modify: `short-maker/internal/router/adapter_test.go`

- [ ] **Step 1: Write tests for MockVideoAdapter**

Append to `internal/router/adapter_test.go`:

```go
func TestMockVideoAdapter_Name(t *testing.T) {
	a := NewMockVideoAdapter()
	if a.Name() != "mock-video" {
		t.Errorf("expected name 'mock-video', got '%s'", a.Name())
	}
}

func TestMockVideoAdapter_Capabilities(t *testing.T) {
	a := NewMockVideoAdapter()
	caps := a.Capabilities()
	if caps.Type != ModelTypeVideo {
		t.Errorf("expected type video, got %s", caps.Type)
	}
}

func TestMockVideoAdapter_Generate(t *testing.T) {
	a := NewMockVideoAdapter()
	dir := t.TempDir()
	outPath := filepath.Join(dir, "test_shot.mp4")

	resp, err := a.Generate(context.Background(), GenerateRequest{
		Prompt:      "test prompt",
		Style:       "manga",
		CameraMove:  "pan",
		SourceImage: "/tmp/source.png",
		OutputPath:  outPath,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if resp.FilePath != outPath {
		t.Errorf("expected file path '%s', got '%s'", outPath, resp.FilePath)
	}
	if resp.ModelUsed != "mock-video" {
		t.Errorf("expected model 'mock-video', got '%s'", resp.ModelUsed)
	}

	// Verify file was created
	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	if info.Size() == 0 {
		t.Error("expected non-empty placeholder file")
	}
}

func TestMockVideoAdapter_HealthCheck(t *testing.T) {
	a := NewMockVideoAdapter()
	if err := a.HealthCheck(context.Background()); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./internal/router/ -v -run TestMockVideoAdapter`
Expected: compilation error — NewMockVideoAdapter not defined

- [ ] **Step 3: Implement MockVideoAdapter**

```go
// internal/router/mock_video.go
package router

import (
	"context"
	"os"
	"time"
)

// MockVideoAdapter creates a placeholder MP4 file for each generation request.
type MockVideoAdapter struct{}

func NewMockVideoAdapter() *MockVideoAdapter {
	return &MockVideoAdapter{}
}

func (a *MockVideoAdapter) Name() string { return "mock-video" }

func (a *MockVideoAdapter) Capabilities() Capabilities {
	return Capabilities{
		Type:           ModelTypeVideo,
		Styles:         []string{"manga", "3d", "live_action"},
		MaxResolution:  "1024x1024",
		SupportsFusion: false,
	}
}

func (a *MockVideoAdapter) Generate(_ context.Context, req GenerateRequest) (*GenerateResponse, error) {
	start := time.Now()

	// Write a placeholder file (not a valid MP4, just a marker)
	if err := os.WriteFile(req.OutputPath, []byte("mock-video-placeholder"), 0644); err != nil {
		return nil, err
	}

	return &GenerateResponse{
		FilePath:   req.OutputPath,
		ModelUsed:  "mock-video",
		Cost:       0.0,
		DurationMs: time.Since(start).Milliseconds(),
	}, nil
}

func (a *MockVideoAdapter) HealthCheck(_ context.Context) error {
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./internal/router/ -v`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git -C /Users/rain/code/west-garden/ai-drama-research add \
  short-maker/internal/router/mock_video.go \
  short-maker/internal/router/adapter_test.go
git -C /Users/rain/code/west-garden/ai-drama-research commit -m "feat(short-maker): MockVideoAdapter — placeholder MP4 generation"
```

---

### Task 3: ModelRouter

**Files:**
- Create: `short-maker/internal/router/router.go`
- Create: `short-maker/internal/router/router_test.go`

- [ ] **Step 1: Write tests for ModelRouter**

```go
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
	// Router with only image adapter — video route should fail
	r := NewModelRouter(NewMockImageAdapter())
	_, err := r.Route(domain.GradeB, "manga", ModelTypeVideo)
	if err == nil {
		t.Error("expected error for no matching adapter")
	}
}

func TestModelRouter_StyleFilter(t *testing.T) {
	// Mock adapters support all styles, so this should work
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./internal/router/ -v -run TestModelRouter`
Expected: compilation error — NewModelRouter not defined

- [ ] **Step 3: Implement ModelRouter**

```go
// internal/router/router.go
package router

import (
	"context"
	"fmt"

	"github.com/west-garden/short-maker/internal/domain"
)

// ModelRouter selects the appropriate model adapter based on importance
// grade, target style, and model type. MVP returns the first match;
// future versions will rank by grade-weighted quality history.
type ModelRouter struct {
	adapters []ModelAdapter
}

func NewModelRouter(adapters ...ModelAdapter) *ModelRouter {
	return &ModelRouter{adapters: adapters}
}

// Route selects an adapter:
// 1. Filter by ModelType
// 2. Filter by Style (adapter must list the style in Capabilities)
// 3. Return first match (MVP — no grade-based ranking yet)
func (r *ModelRouter) Route(grade domain.Grade, style string, modelType ModelType) (ModelAdapter, error) {
	for _, a := range r.adapters {
		caps := a.Capabilities()
		if caps.Type != modelType {
			continue
		}
		if !hasStyle(caps.Styles, style) {
			continue
		}
		return a, nil
	}
	return nil, fmt.Errorf("no adapter found for type=%s style=%s grade=%s", modelType, style, grade)
}

// Generate combines Route + adapter.Generate in one call.
func (r *ModelRouter) Generate(ctx context.Context, grade domain.Grade, style string, modelType ModelType, req GenerateRequest) (*GenerateResponse, error) {
	adapter, err := r.Route(grade, style, modelType)
	if err != nil {
		return nil, err
	}
	return adapter.Generate(ctx, req)
}

func hasStyle(styles []string, target string) bool {
	for _, s := range styles {
		if s == target {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./internal/router/ -v`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git -C /Users/rain/code/west-garden/ai-drama-research add \
  short-maker/internal/router/router.go \
  short-maker/internal/router/router_test.go
git -C /Users/rain/code/west-garden/ai-drama-research commit -m "feat(short-maker): ModelRouter — grade+style+type adapter selection"
```

---

### Task 4: Quality Checker Interface + Mock Checker

**Files:**
- Create: `short-maker/internal/quality/checker.go`
- Create: `short-maker/internal/quality/mock_checker.go`
- Create: `short-maker/internal/quality/checker_test.go`

- [ ] **Step 1: Write tests**

```go
// internal/quality/checker_test.go
package quality

import (
	"context"
	"math"
	"testing"

	"github.com/west-garden/short-maker/internal/domain"
)

func TestDefaultDimensions_WeightsSumToOne(t *testing.T) {
	dims := DefaultDimensions(90)
	var total float64
	for _, d := range dims {
		total += d.Weight
	}
	if math.Abs(total-1.0) > 0.001 {
		t.Errorf("expected weights to sum to 1.0, got %f", total)
	}
}

func TestDefaultDimensions_Count(t *testing.T) {
	dims := DefaultDimensions(90)
	if len(dims) != 5 {
		t.Errorf("expected 5 dimensions, got %d", len(dims))
	}
}

func TestNewReport_WeightedScore(t *testing.T) {
	// All dimensions score 80, weighted sum should be 80
	report := NewReport(1, DefaultDimensions(80), domain.GradeB)
	if report.TotalScore != 80 {
		t.Errorf("expected total score 80, got %d", report.TotalScore)
	}
	// Grade B threshold is 65, so 80 should pass
	if !report.Passed {
		t.Error("expected Passed=true for score 80 with grade B")
	}
}

func TestNewReport_FailsBelowThreshold(t *testing.T) {
	// Grade S threshold is 85, score of 60 should fail
	report := NewReport(1, DefaultDimensions(60), domain.GradeS)
	if report.TotalScore != 60 {
		t.Errorf("expected total score 60, got %d", report.TotalScore)
	}
	if report.Passed {
		t.Error("expected Passed=false for score 60 with grade S")
	}
}

func TestMockChecker_AlwaysPasses(t *testing.T) {
	checker := NewMockChecker()
	shot := domain.NewShotSpec(1, 1)
	report, err := checker.Check(context.Background(), "/tmp/test.png", shot, nil)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !report.Passed {
		t.Error("expected MockChecker to always pass")
	}
	if report.TotalScore != 90 {
		t.Errorf("expected score 90, got %d", report.TotalScore)
	}
	if len(report.Dimensions) != 5 {
		t.Errorf("expected 5 dimensions, got %d", len(report.Dimensions))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./internal/quality/ -v`
Expected: compilation error — package does not exist

- [ ] **Step 3: Implement checker types and helpers**

```go
// internal/quality/checker.go
package quality

import (
	"context"
	"math"

	"github.com/west-garden/short-maker/internal/domain"
)

// Dimension represents one evaluation axis with its weight and score.
type Dimension struct {
	Name   string  `json:"name"`
	Weight float64 `json:"weight"`
	Score  int     `json:"score"`
	Notes  string  `json:"notes"`
}

// QualityReport is the structured result of a quality check.
type QualityReport struct {
	ShotNumber  int         `json:"shot_number"`
	Dimensions  []Dimension `json:"dimensions"`
	TotalScore  int         `json:"total_score"`
	Passed      bool        `json:"passed"`
	Suggestions []string    `json:"suggestions"`
}

// Checker evaluates the quality of a generated image or video.
type Checker interface {
	Check(ctx context.Context, filePath string, shotSpec *domain.ShotSpec, characterAssets []*domain.Asset) (*QualityReport, error)
}

// DefaultDimensions returns the 5 standard evaluation dimensions,
// each set to the given score. Weights come from the spec.
func DefaultDimensions(score int) []Dimension {
	return []Dimension{
		{Name: "character_consistency", Weight: 0.30, Score: score},
		{Name: "image_quality", Weight: 0.25, Score: score},
		{Name: "storyboard_fidelity", Weight: 0.20, Score: score},
		{Name: "style_consistency", Weight: 0.15, Score: score},
		{Name: "narrative_accuracy", Weight: 0.10, Score: score},
	}
}

// NewReport computes the weighted total score and pass/fail from dimensions.
func NewReport(shotNumber int, dimensions []Dimension, grade domain.Grade) *QualityReport {
	var weightedSum float64
	for _, d := range dimensions {
		weightedSum += float64(d.Score) * d.Weight
	}
	totalScore := int(math.Round(weightedSum))
	threshold := grade.QualityThreshold()

	return &QualityReport{
		ShotNumber: shotNumber,
		Dimensions: dimensions,
		TotalScore: totalScore,
		Passed:     totalScore >= threshold,
	}
}
```

- [ ] **Step 4: Implement MockChecker**

```go
// internal/quality/mock_checker.go
package quality

import (
	"context"

	"github.com/west-garden/short-maker/internal/domain"
)

// MockChecker always returns a passing quality report with score 90.
type MockChecker struct{}

func NewMockChecker() *MockChecker {
	return &MockChecker{}
}

func (c *MockChecker) Check(_ context.Context, _ string, shotSpec *domain.ShotSpec, _ []*domain.Asset) (*QualityReport, error) {
	// MockChecker uses GradeC threshold (55) so score 90 always passes
	return NewReport(shotSpec.ShotNumber, DefaultDimensions(90), domain.GradeC), nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./internal/quality/ -v`
Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git -C /Users/rain/code/west-garden/ai-drama-research add \
  short-maker/internal/quality/checker.go \
  short-maker/internal/quality/mock_checker.go \
  short-maker/internal/quality/checker_test.go
git -C /Users/rain/code/west-garden/ai-drama-research commit -m "feat(short-maker): QualityChecker interface + MockChecker"
```

---

### Task 5: Extend PipelineState with GeneratedShot

**Files:**
- Modify: `short-maker/internal/agent/agent.go`

- [ ] **Step 1: Read current agent.go**

Read: `short-maker/internal/agent/agent.go`

- [ ] **Step 2: Add GeneratedShot type and new PipelineState fields**

Add after the `PipelineState` struct definition (before the `NewPipelineState` function):

```go
// GeneratedShot tracks the output of image and video generation for one shot.
type GeneratedShot struct {
	ShotNumber int          `json:"shot_number"`
	EpisodeNum int          `json:"episode_number"`
	ImagePath  string       `json:"image_path"`
	VideoPath  string       `json:"video_path"`
	Grade      domain.Grade `json:"grade"`
	ImageScore int          `json:"image_score"`
	VideoScore int          `json:"video_score"`
}
```

Add two new fields to `PipelineState`:

```go
Images     []*GeneratedShot  `json:"images,omitempty"`
Videos     []*GeneratedShot  `json:"videos,omitempty"`
```

These go after the `Storyboard` field and before the `Errors` field.

- [ ] **Step 3: Verify build succeeds**

Run: `go build -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./...`
Expected: success

- [ ] **Step 4: Run all existing tests to verify no regressions**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./... -count=1`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git -C /Users/rain/code/west-garden/ai-drama-research add short-maker/internal/agent/agent.go
git -C /Users/rain/code/west-garden/ai-drama-research commit -m "feat(short-maker): add GeneratedShot type + Images/Videos to PipelineState"
```

---

### Task 6: ImageGenAgent

**Files:**
- Create: `short-maker/internal/agent/imagegen.go`
- Create: `short-maker/internal/agent/imagegen_test.go`

- [ ] **Step 1: Write tests for ImageGenAgent**

```go
// internal/agent/imagegen_test.go
package agent

import (
	"context"
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
	// Storyboard is nil — should error
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

	// Verify first image
	img1 := result.Images[0]
	if img1.ShotNumber != 1 {
		t.Errorf("expected shot number 1, got %d", img1.ShotNumber)
	}
	if img1.EpisodeNum != 1 {
		t.Errorf("expected episode 1, got %d", img1.EpisodeNum)
	}
	if img1.ImagePath == "" {
		t.Error("expected non-empty image path")
	}
	if img1.ImageScore != 90 {
		t.Errorf("expected image score 90 (from mock checker), got %d", img1.ImageScore)
	}

	// Verify file exists on disk
	if _, err := os.Stat(img1.ImagePath); err != nil {
		t.Errorf("image file not found: %v", err)
	}

	// Verify path structure: outputDir/<projectID>/ep01/shot001.png
	if filepath.Ext(img1.ImagePath) != ".png" {
		t.Errorf("expected .png extension, got %s", filepath.Ext(img1.ImagePath))
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

	// C-grade shot: hook × mid_narration × empty = 1.5 × 1.0 × 0.8 = 1.2 → B
	// Actually let's use transition episode: transition × mid_narration × empty = 1.0 × 1.0 × 0.8 = 0.8 → C
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

	// Checker that fails the first check, passes the second
	failOnce := &failOnceChecker{failCount: 1}
	agent := NewImageGenAgent(r, failOnce, outputDir)

	project := domain.NewProject("test", domain.StyleManga, 1)
	bp := domain.NewStoryBlueprint(project.ID)
	bp.AddEpisodeBlueprintWithRole(1, domain.EpisodeRoleHook, "紧张")

	// Grade A shot (maxRetries=2): hook × emotion_peak × dialogue = 1.5 × 1.2 × 1.0 = 1.8 → A
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

	// Should have retried and eventually passed
	if len(result.Images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(result.Images))
	}
	if !result.Images[0].ImageScore > 0 {
		t.Error("expected positive image score after retry")
	}
	// Verify checker was called twice (fail + pass)
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./internal/agent/ -v -run TestImageGenAgent`
Expected: compilation error — NewImageGenAgent not defined

- [ ] **Step 3: Implement ImageGenAgent**

```go
// internal/agent/imagegen.go
package agent

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/west-garden/short-maker/internal/domain"
	"github.com/west-garden/short-maker/internal/quality"
	"github.com/west-garden/short-maker/internal/router"
)

// ImageGenAgent generates images for each shot in the storyboard.
// Uses ModelRouter for model selection and QualityChecker for the
// generate→evaluate→retry loop. Implements Agent for PhaseImageGeneration.
type ImageGenAgent struct {
	router    *router.ModelRouter
	checker   quality.Checker
	outputDir string
}

func NewImageGenAgent(r *router.ModelRouter, checker quality.Checker, outputDir string) *ImageGenAgent {
	return &ImageGenAgent{router: r, checker: checker, outputDir: outputDir}
}

func (a *ImageGenAgent) Phase() Phase { return PhaseImageGeneration }

func (a *ImageGenAgent) Run(ctx context.Context, state *PipelineState) (*PipelineState, error) {
	if len(state.Storyboard) == 0 {
		return nil, fmt.Errorf("image gen agent requires a non-empty Storyboard")
	}

	for _, shot := range state.Storyboard {
		episodeRole := findEpisodeRole(state.Blueprint, shot.EpisodeNumber)
		importance := domain.NewImportanceScore(episodeRole, shot.RhythmPosition, shot.ContentType)
		grade := importance.Grade()
		maxRetries := importance.MaxRetries()

		charAssets := findCharacterAssets(state.Assets, shot.CharacterRefs)
		outPath := shotImagePath(a.outputDir, state.Project.ID, shot.EpisodeNumber, shot.ShotNumber)

		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return nil, fmt.Errorf("create output dir: %w", err)
		}

		var charRefPaths []string
		for _, ca := range charAssets {
			if ca.FilePath != "" {
				charRefPaths = append(charRefPaths, ca.FilePath)
			}
		}

		req := router.GenerateRequest{
			Prompt:        shot.Prompt,
			Style:         string(state.Project.Style),
			CharacterRefs: charRefPaths,
			OutputPath:    outPath,
		}

		var lastReport *quality.QualityReport
		for attempt := 0; attempt <= maxRetries; attempt++ {
			resp, err := a.router.Generate(ctx, grade, string(state.Project.Style), router.ModelTypeImage, req)
			if err != nil {
				if attempt < maxRetries {
					log.Printf("[image-gen] shot %d generate failed, retrying (%d/%d): %v", shot.ShotNumber, attempt+1, maxRetries, err)
					continue
				}
				return nil, fmt.Errorf("generate image for shot %d: %w", shot.ShotNumber, err)
			}

			report, err := a.checker.Check(ctx, resp.FilePath, shot, charAssets)
			if err != nil {
				return nil, fmt.Errorf("quality check shot %d: %w", shot.ShotNumber, err)
			}
			lastReport = report

			if report.Passed {
				log.Printf("[image-gen] shot %d passed quality check (score: %d, grade: %s)", shot.ShotNumber, report.TotalScore, grade)
				break
			}

			if attempt < maxRetries {
				log.Printf("[image-gen] shot %d failed quality check (score: %d, threshold: %d), retrying (%d/%d)",
					shot.ShotNumber, report.TotalScore, grade.QualityThreshold(), attempt+1, maxRetries)
			} else {
				log.Printf("[image-gen] shot %d failed quality check after %d attempts (score: %d)", shot.ShotNumber, maxRetries+1, report.TotalScore)
			}
		}

		score := 0
		if lastReport != nil {
			score = lastReport.TotalScore
		}

		state.Images = append(state.Images, &GeneratedShot{
			ShotNumber: shot.ShotNumber,
			EpisodeNum: shot.EpisodeNumber,
			ImagePath:  outPath,
			Grade:      grade,
			ImageScore: score,
		})
	}

	log.Printf("[image-gen] generated %d images", len(state.Images))
	return state, nil
}

// findEpisodeRole looks up the EpisodeRole for a given episode number.
func findEpisodeRole(bp *domain.StoryBlueprint, episodeNumber int) domain.EpisodeRole {
	if bp == nil {
		return domain.EpisodeRoleTransition
	}
	for _, ep := range bp.Episodes {
		if ep.Number == episodeNumber {
			return ep.Role
		}
	}
	return domain.EpisodeRoleTransition
}

// findCharacterAssets returns assets whose character_id matches the given refs.
func findCharacterAssets(assets []*domain.Asset, characterRefs []string) []*domain.Asset {
	if len(characterRefs) == 0 {
		return nil
	}
	refSet := make(map[string]bool, len(characterRefs))
	for _, ref := range characterRefs {
		refSet[ref] = true
	}
	var result []*domain.Asset
	for _, a := range assets {
		if refSet[a.Metadata["character_id"]] {
			result = append(result, a)
		}
	}
	return result
}

func shotImagePath(outputDir, projectID string, episodeNum, shotNum int) string {
	return filepath.Join(outputDir, projectID, fmt.Sprintf("ep%02d", episodeNum), fmt.Sprintf("shot%03d.png", shotNum))
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./internal/agent/ -v -run TestImageGenAgent`
Expected: all PASS

- [ ] **Step 5: Run ALL tests**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./... -count=1`
Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git -C /Users/rain/code/west-garden/ai-drama-research add \
  short-maker/internal/agent/imagegen.go \
  short-maker/internal/agent/imagegen_test.go
git -C /Users/rain/code/west-garden/ai-drama-research commit -m "feat(short-maker): ImageGenAgent — shot → router → quality check → retry loop"
```

---

### Task 7: VideoGenAgent

**Files:**
- Create: `short-maker/internal/agent/videogen.go`
- Create: `short-maker/internal/agent/videogen_test.go`

- [ ] **Step 1: Write tests for VideoGenAgent**

```go
// internal/agent/videogen_test.go
package agent

import (
	"context"
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
	// Images is nil — should error
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

	// Verify first video
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

	// Verify video file exists on disk
	if _, err := os.Stat(vid1.VideoPath); err != nil {
		t.Errorf("video file not found: %v", err)
	}
	if filepath.Ext(vid1.VideoPath) != ".mp4" {
		t.Errorf("expected .mp4 extension, got %s", filepath.Ext(vid1.VideoPath))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./internal/agent/ -v -run TestVideoGenAgent`
Expected: compilation error — NewVideoGenAgent not defined

- [ ] **Step 3: Implement VideoGenAgent**

```go
// internal/agent/videogen.go
package agent

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/west-garden/short-maker/internal/domain"
	"github.com/west-garden/short-maker/internal/quality"
	"github.com/west-garden/short-maker/internal/router"
)

// VideoGenAgent generates video clips from static images + camera directives.
// Uses ModelRouter for model selection and QualityChecker for the
// generate→evaluate→retry loop. Implements Agent for PhaseVideoGeneration.
type VideoGenAgent struct {
	router    *router.ModelRouter
	checker   quality.Checker
	outputDir string
}

func NewVideoGenAgent(r *router.ModelRouter, checker quality.Checker, outputDir string) *VideoGenAgent {
	return &VideoGenAgent{router: r, checker: checker, outputDir: outputDir}
}

func (a *VideoGenAgent) Phase() Phase { return PhaseVideoGeneration }

func (a *VideoGenAgent) Run(ctx context.Context, state *PipelineState) (*PipelineState, error) {
	if len(state.Images) == 0 {
		return nil, fmt.Errorf("video gen agent requires non-empty Images")
	}

	// Build shot number → ShotSpec lookup
	shotSpecByNum := make(map[int]*domain.ShotSpec, len(state.Storyboard))
	for _, shot := range state.Storyboard {
		shotSpecByNum[shot.ShotNumber] = shot
	}

	for _, img := range state.Images {
		shotSpec := shotSpecByNum[img.ShotNumber]
		if shotSpec == nil {
			log.Printf("[video-gen] warning: no ShotSpec for shot %d, skipping", img.ShotNumber)
			continue
		}

		// Recompute ImportanceScore to get MaxRetries (Grade alone doesn't have this method)
		episodeRole := findEpisodeRole(state.Blueprint, img.EpisodeNum)
		importance := domain.NewImportanceScore(episodeRole, shotSpec.RhythmPosition, shotSpec.ContentType)
		grade := importance.Grade()
		maxRetries := importance.MaxRetries()
		charAssets := findCharacterAssets(state.Assets, shotSpec.CharacterRefs)

		outPath := shotVideoPath(a.outputDir, state.Project.ID, img.EpisodeNum, img.ShotNumber)
		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return nil, fmt.Errorf("create output dir: %w", err)
		}

		req := router.GenerateRequest{
			Prompt:      shotSpec.Prompt,
			Style:       string(state.Project.Style),
			CameraMove:  shotSpec.CameraMove,
			SourceImage: img.ImagePath,
			OutputPath:  outPath,
		}

		var lastReport *quality.QualityReport
		for attempt := 0; attempt <= maxRetries; attempt++ {
			resp, err := a.router.Generate(ctx, grade, string(state.Project.Style), router.ModelTypeVideo, req)
			if err != nil {
				if attempt < maxRetries {
					log.Printf("[video-gen] shot %d generate failed, retrying (%d/%d): %v", img.ShotNumber, attempt+1, maxRetries, err)
					continue
				}
				return nil, fmt.Errorf("generate video for shot %d: %w", img.ShotNumber, err)
			}

			report, err := a.checker.Check(ctx, resp.FilePath, shotSpec, charAssets)
			if err != nil {
				return nil, fmt.Errorf("quality check video shot %d: %w", img.ShotNumber, err)
			}
			lastReport = report

			if report.Passed {
				log.Printf("[video-gen] shot %d passed quality check (score: %d, grade: %s)", img.ShotNumber, report.TotalScore, grade)
				break
			}

			if attempt < maxRetries {
				log.Printf("[video-gen] shot %d failed quality check (score: %d), retrying (%d/%d)",
					img.ShotNumber, report.TotalScore, attempt+1, maxRetries)
			}
		}

		score := 0
		if lastReport != nil {
			score = lastReport.TotalScore
		}

		state.Videos = append(state.Videos, &GeneratedShot{
			ShotNumber: img.ShotNumber,
			EpisodeNum: img.EpisodeNum,
			ImagePath:  img.ImagePath,
			VideoPath:  outPath,
			Grade:      grade,
			ImageScore: img.ImageScore,
			VideoScore: score,
		})
	}

	log.Printf("[video-gen] generated %d videos", len(state.Videos))
	return state, nil
}

func shotVideoPath(outputDir, projectID string, episodeNum, shotNum int) string {
	return filepath.Join(outputDir, projectID, fmt.Sprintf("ep%02d", episodeNum), fmt.Sprintf("shot%03d.mp4", shotNum))
}
```

Note: `videogen.go` uses `findEpisodeRole` and `findCharacterAssets` which are defined in `imagegen.go`. Both files are in the `agent` package, so no cross-package dependency is needed for these helpers.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./internal/agent/ -v -run TestVideoGenAgent`
Expected: all PASS

- [ ] **Step 5: Run ALL tests**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./... -count=1`
Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git -C /Users/rain/code/west-garden/ai-drama-research add \
  short-maker/internal/agent/videogen.go \
  short-maker/internal/agent/videogen_test.go
git -C /Users/rain/code/west-garden/ai-drama-research commit -m "feat(short-maker): VideoGenAgent — image+camera → video via router + quality check"
```

---

### Task 8: Wire into CLI

**Files:**
- Modify: `short-maker/cmd/shortmaker/main.go`

- [ ] **Step 1: Read current main.go**

Read: `short-maker/cmd/shortmaker/main.go`

- [ ] **Step 2: Add --output flag to init()**

Add this line after the existing `runCmd.Flags()` calls in `init()`:

```go
runCmd.Flags().String("output", "./output", "Output directory for generated files")
```

- [ ] **Step 3: Read the flag in runCmd.RunE**

Add after `strategyPath, _ := cmd.Flags().GetString("strategies")`:

```go
outputDir, _ := cmd.Flags().GetString("output")
```

Update the `buildAgents` call to pass `outputDir`:

```go
agents, cleanup, err := buildAgents(useMock, llmModel, dbPath, strategyPath, outputDir)
```

- [ ] **Step 4: Update buildAgents signature and implementation**

New signature:

```go
func buildAgents(useMock bool, llmModel, dbPath, strategyPath, outputDir string) (map[agent.Phase]agent.Agent, func(), error)
```

Add imports for:

```go
"github.com/west-garden/short-maker/internal/quality"
"github.com/west-garden/short-maker/internal/router"
```

Replace the mock-fallback loop at the end (lines 116-125) with:

```go
// Image and video generation agents with model router
imageAdapter := router.NewMockImageAdapter()
videoAdapter := router.NewMockVideoAdapter()
modelRouter := router.NewModelRouter(imageAdapter, videoAdapter)
checker := quality.NewMockChecker()

agents[agent.PhaseImageGeneration] = agent.NewImageGenAgent(modelRouter, checker, outputDir)
agents[agent.PhaseVideoGeneration] = agent.NewVideoGenAgent(modelRouter, checker, outputDir)
```

Also update the mock branch (inside `if useMock {`) — it already creates mock agents for all phases, so no change needed there.

- [ ] **Step 5: Update printSummary**

Add after the storyboard line:

```go
log.Printf("Generated images: %d", len(result.Images))
for _, img := range result.Images {
	log.Printf("  - ep%02d/shot%03d [%s] score:%d %s", img.EpisodeNum, img.ShotNumber, img.Grade, img.ImageScore, img.ImagePath)
}
log.Printf("Generated videos: %d", len(result.Videos))
for _, vid := range result.Videos {
	log.Printf("  - ep%02d/shot%03d [%s] score:%d %s", vid.EpisodeNum, vid.ShotNumber, vid.Grade, vid.VideoScore, vid.VideoPath)
}
```

- [ ] **Step 6: Verify build succeeds**

Run: `go build -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./...`
Expected: success

- [ ] **Step 7: Verify mock mode still works**

Run: `go run -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./cmd/shortmaker run testdata/sample-script.txt --style manga --episodes 2`
Expected: pipeline completes with all mock agents

- [ ] **Step 8: Commit**

```bash
git -C /Users/rain/code/west-garden/ai-drama-research add short-maker/cmd/shortmaker/main.go
git -C /Users/rain/code/west-garden/ai-drama-research commit -m "feat(short-maker): wire ImageGen + VideoGen agents into CLI with --output flag"
```

---

### Task 9: Integration Test — Full 5-Stage Pipeline

**Files:**
- Modify: `short-maker/internal/agent/integration_test.go`

- [ ] **Step 1: Read current integration_test.go**

Read: `short-maker/internal/agent/integration_test.go`

- [ ] **Step 2: Add full pipeline integration test with generation**

Append this test to `integration_test.go`. Do NOT remove existing tests. Add the new imports `"github.com/west-garden/short-maker/internal/quality"` and `"github.com/west-garden/short-maker/internal/router"` to the import block.

```go
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
```

- [ ] **Step 3: Run the new integration test**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./internal/agent/ -v -run TestIntegration_FullPipelineWithGeneration`
Expected: PASS

- [ ] **Step 4: Run ALL tests**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./... -count=1`
Expected: all PASS across all packages

- [ ] **Step 5: Commit**

```bash
git -C /Users/rain/code/west-garden/ai-drama-research add short-maker/internal/agent/integration_test.go
git -C /Users/rain/code/west-garden/ai-drama-research commit -m "feat(short-maker): integration test — full 5-stage pipeline with image+video generation"
```

---

### Task 10: go mod tidy

- [ ] **Step 1: Run go mod tidy**

Run: `GOWORK=off go mod tidy -C /Users/rain/code/west-garden/ai-drama-research/short-maker`

- [ ] **Step 2: Check if go.mod/go.sum changed**

Run: `git -C /Users/rain/code/west-garden/ai-drama-research diff --name-only short-maker/go.mod short-maker/go.sum`

- [ ] **Step 3: Commit if changed**

If there are changes:

```bash
git -C /Users/rain/code/west-garden/ai-drama-research add short-maker/go.mod short-maker/go.sum
git -C /Users/rain/code/west-garden/ai-drama-research commit -m "chore(short-maker): go mod tidy"
```

- [ ] **Step 4: Final full test run**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./... -count=1 -v`
Expected: all PASS
