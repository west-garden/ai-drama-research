# Plan 1: Project Foundation + Agent Framework

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the runnable pipeline skeleton — domain types, Agent interface, Orchestrator with default flow, storage layer, and CLI — so that mock agents can execute the full 5-stage pipeline end-to-end.

**Architecture:** Go monorepo with `cmd/` for entry points and `internal/` for domain logic. Domain types define the data contracts between Agents. The Orchestrator executes a fixed sequence of Agent phases, passing structured data between them. SQLite for persistence, local filesystem for generated artifacts.

**Tech Stack:** Go 1.22+, SQLite (via modernc.org/sqlite — pure Go, no CGO), Cobra CLI

**Spec reference:** `short-maker/docs/specs/2026-04-03-core-pipeline-design.md`

---

## File Structure

```
short-maker/
├── cmd/
│   └── shortmaker/
│       └── main.go                  # CLI entry point (cobra)
├── internal/
│   ├── domain/
│   │   ├── project.go               # Project, Episode, Shot, Style, Status
│   │   ├── project_test.go
│   │   ├── blueprint.go             # StoryBlueprint, CharacterProfile, EpisodeBlueprint, SceneTag
│   │   ├── blueprint_test.go
│   │   ├── asset.go                 # Asset, AssetType, AssetScope
│   │   ├── asset_test.go
│   │   ├── importance.go            # ImportanceScore, EpisodeRole, RhythmPosition, ContentType
│   │   ├── importance_test.go
│   │   ├── storyboard.go            # ShotSpec (structured storyboard entry)
│   │   └── storyboard_test.go
│   ├── agent/
│   │   ├── agent.go                 # Agent interface, Phase enum, Input/Output types
│   │   ├── orchestrator.go          # Orchestrator — default flow, checkpoint hooks
│   │   ├── orchestrator_test.go
│   │   └── mock.go                  # MockAgent for testing
│   ├── llm/
│   │   ├── client.go                # LLMClient interface
│   │   └── mock.go                  # MockLLMClient — returns canned responses
│   └── store/
│       ├── store.go                 # Store interface (ProjectStore + AssetStore)
│       ├── sqlite.go                # SQLite implementation
│       └── sqlite_test.go
├── go.mod
├── go.sum
└── .gitignore
```

---

### Task 1: Project Scaffolding

**Files:**
- Create: `short-maker/go.mod`
- Create: `short-maker/.gitignore`
- Create: `short-maker/cmd/shortmaker/main.go`

- [ ] **Step 1: Initialize Go module**

```bash
cd /Users/rain/code/west-garden/ai-drama-research/short-maker
go mod init github.com/west-garden/short-maker
```

- [ ] **Step 2: Create .gitignore**

```gitignore
# Binaries
shortmaker
*.exe

# Generated
*.db
*.sqlite
output/

# IDE
.idea/
.vscode/
*.swp

# OS
.DS_Store
```

- [ ] **Step 3: Create minimal main.go**

```go
// cmd/shortmaker/main.go
package main

import "fmt"

func main() {
	fmt.Println("short-maker v0.1.0")
}
```

- [ ] **Step 4: Verify it compiles and runs**

Run: `cd /Users/rain/code/west-garden/ai-drama-research/short-maker && go run ./cmd/shortmaker`
Expected: `short-maker v0.1.0`

- [ ] **Step 5: Create directory structure**

```bash
mkdir -p internal/domain internal/agent internal/llm internal/store
```

- [ ] **Step 6: Commit**

```bash
git add short-maker/
git commit -m "feat(short-maker): project scaffolding with Go module"
```

---

### Task 2: Core Domain Types — Project, Episode, Shot

**Files:**
- Create: `short-maker/internal/domain/project.go`
- Create: `short-maker/internal/domain/project_test.go`

- [ ] **Step 1: Write tests for Project, Episode, Shot**

```go
// internal/domain/project_test.go
package domain

import "testing"

func TestNewProject(t *testing.T) {
	p := NewProject("西游记漫剧", StyleManga, 50)
	if p.Name != "西游记漫剧" {
		t.Errorf("expected name '西游记漫剧', got '%s'", p.Name)
	}
	if p.Style != StyleManga {
		t.Errorf("expected style Manga, got %v", p.Style)
	}
	if p.EpisodeCount != 50 {
		t.Errorf("expected 50 episodes, got %d", p.EpisodeCount)
	}
	if p.Status != StatusCreated {
		t.Errorf("expected status Created, got %v", p.Status)
	}
	if p.ID == "" {
		t.Error("expected non-empty ID")
	}
}

func TestProjectAddEpisode(t *testing.T) {
	p := NewProject("测试剧", StyleManga, 3)
	ep := p.AddEpisode(1)
	if ep.Number != 1 {
		t.Errorf("expected episode number 1, got %d", ep.Number)
	}
	if ep.ProjectID != p.ID {
		t.Errorf("expected episode projectID '%s', got '%s'", p.ID, ep.ProjectID)
	}
	if len(p.Episodes) != 1 {
		t.Errorf("expected 1 episode, got %d", len(p.Episodes))
	}
}

func TestEpisodeAddShot(t *testing.T) {
	p := NewProject("测试剧", StyleManga, 3)
	ep := p.AddEpisode(1)
	shot := ep.AddShot()
	if shot.Number != 1 {
		t.Errorf("expected shot number 1, got %d", shot.Number)
	}
	if shot.EpisodeID != ep.ID {
		t.Error("expected shot to reference episode")
	}
	shot2 := ep.AddShot()
	if shot2.Number != 2 {
		t.Errorf("expected shot number 2, got %d", shot2.Number)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/rain/code/west-garden/ai-drama-research/short-maker && go test ./internal/domain/ -v -run TestNew`
Expected: compilation error — types not defined

- [ ] **Step 3: Implement Project, Episode, Shot**

```go
// internal/domain/project.go
package domain

import (
	"fmt"
	"time"
)

type Style string

const (
	StyleManga    Style = "manga"
	Style3D       Style = "3d"
	StyleLiveAction Style = "live_action"
)

type Status string

const (
	StatusCreated    Status = "created"
	StatusProcessing Status = "processing"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
)

type Project struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Style        Style      `json:"style"`
	EpisodeCount int        `json:"episode_count"`
	Status       Status     `json:"status"`
	Episodes     []*Episode `json:"episodes"`
	CreatedAt    time.Time  `json:"created_at"`
}

func NewProject(name string, style Style, episodeCount int) *Project {
	return &Project{
		ID:           generateID("proj"),
		Name:         name,
		Style:        style,
		EpisodeCount: episodeCount,
		Status:       StatusCreated,
		CreatedAt:    time.Now(),
	}
}

func (p *Project) AddEpisode(number int) *Episode {
	ep := &Episode{
		ID:        generateID("ep"),
		ProjectID: p.ID,
		Number:    number,
		Status:    StatusCreated,
	}
	p.Episodes = append(p.Episodes, ep)
	return ep
}

type Episode struct {
	ID        string  `json:"id"`
	ProjectID string  `json:"project_id"`
	Number    int     `json:"number"`
	Status    Status  `json:"status"`
	Shots     []*Shot `json:"shots"`
}

func (e *Episode) AddShot() *Shot {
	shot := &Shot{
		ID:        generateID("shot"),
		EpisodeID: e.ID,
		Number:    len(e.Shots) + 1,
		Status:    StatusCreated,
	}
	e.Shots = append(e.Shots, shot)
	return shot
}

type Shot struct {
	ID        string `json:"id"`
	EpisodeID string `json:"episode_id"`
	Number    int    `json:"number"`
	Status    Status `json:"status"`
	Prompt    string `json:"prompt"`
	ImagePath string `json:"image_path"`
	VideoPath string `json:"video_path"`
}

var idCounter int

func generateID(prefix string) string {
	idCounter++
	return fmt.Sprintf("%s_%d_%d", prefix, time.Now().UnixMilli(), idCounter)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/rain/code/west-garden/ai-drama-research/short-maker && go test ./internal/domain/ -v -run "TestNew|TestProject|TestEpisode"`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add short-maker/internal/domain/project.go short-maker/internal/domain/project_test.go
git commit -m "feat(short-maker): core domain types — Project, Episode, Shot"
```

---

### Task 3: Core Domain Types — Story Blueprint

**Files:**
- Create: `short-maker/internal/domain/blueprint.go`
- Create: `short-maker/internal/domain/blueprint_test.go`

- [ ] **Step 1: Write tests for StoryBlueprint**

```go
// internal/domain/blueprint_test.go
package domain

import "testing"

func TestNewStoryBlueprint(t *testing.T) {
	bp := NewStoryBlueprint("proj_123")
	if bp.ProjectID != "proj_123" {
		t.Errorf("expected projectID 'proj_123', got '%s'", bp.ProjectID)
	}
}

func TestBlueprintAddCharacter(t *testing.T) {
	bp := NewStoryBlueprint("proj_123")
	ch := bp.AddCharacter("孙悟空", "主角，齐天大圣", []string{"好斗", "忠诚"})
	if ch.Name != "孙悟空" {
		t.Errorf("expected name '孙悟空', got '%s'", ch.Name)
	}
	if len(bp.Characters) != 1 {
		t.Errorf("expected 1 character, got %d", len(bp.Characters))
	}
}

func TestBlueprintAddEpisode(t *testing.T) {
	bp := NewStoryBlueprint("proj_123")
	epBP := bp.AddEpisodeBlueprintWithRole(1, EpisodeRoleHook, "紧张")
	if epBP.Number != 1 {
		t.Errorf("expected number 1, got %d", epBP.Number)
	}
	if epBP.Role != EpisodeRoleHook {
		t.Errorf("expected role Hook, got %v", epBP.Role)
	}
}

func TestEpisodeRoleWeight(t *testing.T) {
	tests := []struct {
		role   EpisodeRole
		weight float64
	}{
		{EpisodeRoleHook, 1.5},
		{EpisodeRolePaywall, 1.3},
		{EpisodeRoleClimax, 1.2},
		{EpisodeRoleTransition, 1.0},
	}
	for _, tt := range tests {
		if got := tt.role.Weight(); got != tt.weight {
			t.Errorf("EpisodeRole(%s).Weight() = %v, want %v", tt.role, got, tt.weight)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/rain/code/west-garden/ai-drama-research/short-maker && go test ./internal/domain/ -v -run "TestNewStory|TestBlueprint|TestEpisodeRole"`
Expected: compilation error — types not defined

- [ ] **Step 3: Implement StoryBlueprint**

```go
// internal/domain/blueprint.go
package domain

type EpisodeRole string

const (
	EpisodeRoleHook       EpisodeRole = "hook"       // 钩子集 ×1.5
	EpisodeRolePaywall    EpisodeRole = "paywall"     // 付费卡点 ×1.3
	EpisodeRoleClimax     EpisodeRole = "climax"      // 高潮/大结局 ×1.2
	EpisodeRoleTransition EpisodeRole = "transition"  // 过渡集 ×1.0
)

func (r EpisodeRole) Weight() float64 {
	switch r {
	case EpisodeRoleHook:
		return 1.5
	case EpisodeRolePaywall:
		return 1.3
	case EpisodeRoleClimax:
		return 1.2
	default:
		return 1.0
	}
}

type StoryBlueprint struct {
	ProjectID    string              `json:"project_id"`
	WorldView    string              `json:"world_view"`
	Characters   []*CharacterProfile `json:"characters"`
	Episodes     []*EpisodeBlueprint `json:"episodes"`
	Relationships []Relationship     `json:"relationships"`
}

func NewStoryBlueprint(projectID string) *StoryBlueprint {
	return &StoryBlueprint{ProjectID: projectID}
}

func (bp *StoryBlueprint) AddCharacter(name, description string, traits []string) *CharacterProfile {
	ch := &CharacterProfile{
		ID:          generateID("char"),
		Name:        name,
		Description: description,
		Traits:      traits,
	}
	bp.Characters = append(bp.Characters, ch)
	return ch
}

func (bp *StoryBlueprint) AddEpisodeBlueprintWithRole(number int, role EpisodeRole, emotionArc string) *EpisodeBlueprint {
	epBP := &EpisodeBlueprint{
		Number:     number,
		Role:       role,
		EmotionArc: emotionArc,
	}
	bp.Episodes = append(bp.Episodes, epBP)
	return epBP
}

type CharacterProfile struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Traits      []string `json:"traits"`
}

type EpisodeBlueprint struct {
	Number     int         `json:"number"`
	Role       EpisodeRole `json:"role"`
	EmotionArc string      `json:"emotion_arc"`
	Scenes     []SceneTag  `json:"scenes"`
	Synopsis   string      `json:"synopsis"`
}

type SceneTag struct {
	NarrativeBeat  string `json:"narrative_beat"`
	EmotionArc     string `json:"emotion_arc"`
	Setting        string `json:"setting"`
	Pacing         string `json:"pacing"`
	CharacterCount int    `json:"character_count"`
}

type Relationship struct {
	CharacterA string `json:"character_a"`
	CharacterB string `json:"character_b"`
	Type       string `json:"type"`
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/rain/code/west-garden/ai-drama-research/short-maker && go test ./internal/domain/ -v -run "TestNewStory|TestBlueprint|TestEpisodeRole"`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add short-maker/internal/domain/blueprint.go short-maker/internal/domain/blueprint_test.go
git commit -m "feat(short-maker): StoryBlueprint domain types with EpisodeRole weights"
```

---

### Task 4: Core Domain Types — Importance Scoring

**Files:**
- Create: `short-maker/internal/domain/importance.go`
- Create: `short-maker/internal/domain/importance_test.go`

- [ ] **Step 1: Write tests for ImportanceScore**

```go
// internal/domain/importance_test.go
package domain

import "testing"

func TestImportanceScore_Episode1_OpeningFight(t *testing.T) {
	score := NewImportanceScore(EpisodeRoleHook, RhythmOpenHook, ContentFight)
	// 1.5 × 1.4 × 1.2 = 2.52
	if got := score.Score(); got != 2.52 {
		t.Errorf("expected 2.52, got %v", got)
	}
	if got := score.Grade(); got != GradeS {
		t.Errorf("expected grade S, got %v", got)
	}
	if got := score.MaxRetries(); got != 3 {
		t.Errorf("expected 3 retries, got %d", got)
	}
}

func TestImportanceScore_MidEpisode_Dialogue(t *testing.T) {
	score := NewImportanceScore(EpisodeRoleTransition, RhythmMidNarration, ContentDialogue)
	// 1.0 × 1.0 × 1.0 = 1.0
	if got := score.Score(); got != 1.0 {
		t.Errorf("expected 1.0, got %v", got)
	}
	if got := score.Grade(); got != GradeB {
		t.Errorf("expected grade B, got %v", got)
	}
	if got := score.MaxRetries(); got != 1 {
		t.Errorf("expected 1 retry, got %d", got)
	}
}

func TestImportanceScore_Filler_Empty(t *testing.T) {
	score := NewImportanceScore(EpisodeRoleTransition, RhythmMidNarration, ContentEmpty)
	// 1.0 × 1.0 × 0.8 = 0.8
	if got := score.Score(); got != 0.8 {
		t.Errorf("expected 0.8, got %v", got)
	}
	if got := score.Grade(); got != GradeC {
		t.Errorf("expected grade C, got %v", got)
	}
	if got := score.MaxRetries(); got != 0 {
		t.Errorf("expected 0 retries, got %d", got)
	}
}

func TestImportanceScore_QualityThreshold(t *testing.T) {
	tests := []struct {
		grade     Grade
		threshold int
	}{
		{GradeS, 85},
		{GradeA, 75},
		{GradeB, 65},
		{GradeC, 55},
	}
	for _, tt := range tests {
		if got := tt.grade.QualityThreshold(); got != tt.threshold {
			t.Errorf("Grade(%s).QualityThreshold() = %d, want %d", tt.grade, got, tt.threshold)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/rain/code/west-garden/ai-drama-research/short-maker && go test ./internal/domain/ -v -run "TestImportance"`
Expected: compilation error

- [ ] **Step 3: Implement ImportanceScore**

```go
// internal/domain/importance.go
package domain

import "math"

// Dimension 2: Intra-episode rhythm position
type RhythmPosition string

const (
	RhythmOpenHook     RhythmPosition = "open_hook"     // 开场钩子 ×1.4
	RhythmEmotionPeak  RhythmPosition = "emotion_peak"  // 情绪高点 ×1.2
	RhythmTailHook     RhythmPosition = "tail_hook"     // 尾部钩子 ×1.2
	RhythmMidNarration RhythmPosition = "mid_narration" // 中段叙事 ×1.0
)

func (r RhythmPosition) Weight() float64 {
	switch r {
	case RhythmOpenHook:
		return 1.4
	case RhythmEmotionPeak, RhythmTailHook:
		return 1.2
	default:
		return 1.0
	}
}

// Dimension 3: Content type
type ContentType string

const (
	ContentFirstAppear ContentType = "first_appear" // 角色首次出场 ×1.3
	ContentFight       ContentType = "fight"        // 打斗/特效 ×1.2
	ContentDialogue    ContentType = "dialogue"     // 对话/特写 ×1.0
	ContentEmpty       ContentType = "empty"        // 空镜/环境 ×0.8
)

func (c ContentType) Weight() float64 {
	switch c {
	case ContentFirstAppear:
		return 1.3
	case ContentFight:
		return 1.2
	case ContentDialogue:
		return 1.0
	case ContentEmpty:
		return 0.8
	default:
		return 1.0
	}
}

type Grade string

const (
	GradeS Grade = "S" // ≥ 2.0
	GradeA Grade = "A" // 1.4 ~ 2.0
	GradeB Grade = "B" // 1.0 ~ 1.4
	GradeC Grade = "C" // < 1.0
)

func (g Grade) QualityThreshold() int {
	switch g {
	case GradeS:
		return 85
	case GradeA:
		return 75
	case GradeB:
		return 65
	default:
		return 55
	}
}

type ImportanceScore struct {
	EpisodeRole    EpisodeRole    `json:"episode_role"`
	RhythmPosition RhythmPosition `json:"rhythm_position"`
	ContentType    ContentType    `json:"content_type"`
}

func NewImportanceScore(ep EpisodeRole, rhythm RhythmPosition, content ContentType) ImportanceScore {
	return ImportanceScore{
		EpisodeRole:    ep,
		RhythmPosition: rhythm,
		ContentType:    content,
	}
}

func (s ImportanceScore) Score() float64 {
	raw := s.EpisodeRole.Weight() * s.RhythmPosition.Weight() * s.ContentType.Weight()
	return math.Round(raw*100) / 100
}

func (s ImportanceScore) Grade() Grade {
	score := s.Score()
	switch {
	case score >= 2.0:
		return GradeS
	case score >= 1.4:
		return GradeA
	case score >= 1.0:
		return GradeB
	default:
		return GradeC
	}
}

func (s ImportanceScore) MaxRetries() int {
	switch s.Grade() {
	case GradeS:
		return 3
	case GradeA:
		return 2
	case GradeB:
		return 1
	default:
		return 0
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/rain/code/west-garden/ai-drama-research/short-maker && go test ./internal/domain/ -v -run "TestImportance"`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add short-maker/internal/domain/importance.go short-maker/internal/domain/importance_test.go
git commit -m "feat(short-maker): 3-dimension importance scoring with grade thresholds"
```

---

### Task 5: Core Domain Types — Asset & Storyboard

**Files:**
- Create: `short-maker/internal/domain/asset.go`
- Create: `short-maker/internal/domain/asset_test.go`
- Create: `short-maker/internal/domain/storyboard.go`
- Create: `short-maker/internal/domain/storyboard_test.go`

- [ ] **Step 1: Write tests for Asset**

```go
// internal/domain/asset_test.go
package domain

import "testing"

func TestNewAsset(t *testing.T) {
	a := NewAsset("孙悟空正面照", AssetTypeCharacter, AssetScopeProject, "proj_1")
	if a.Name != "孙悟空正面照" {
		t.Errorf("expected name '孙悟空正面照', got '%s'", a.Name)
	}
	if a.Type != AssetTypeCharacter {
		t.Errorf("expected type Character, got %v", a.Type)
	}
	if a.Scope != AssetScopeProject {
		t.Errorf("expected scope Project, got %v", a.Scope)
	}
}

func TestAssetPromoteToGlobal(t *testing.T) {
	a := NewAsset("通用宫殿背景", AssetTypeScene, AssetScopeProject, "proj_1")
	a.PromoteToGlobal()
	if a.Scope != AssetScopeGlobal {
		t.Errorf("expected scope Global after promotion, got %v", a.Scope)
	}
	if a.ProjectID != "" {
		t.Error("expected empty projectID after promotion")
	}
}
```

- [ ] **Step 2: Write tests for ShotSpec**

```go
// internal/domain/storyboard_test.go
package domain

import "testing"

func TestNewShotSpec(t *testing.T) {
	spec := NewShotSpec(1, 3)
	spec.FrameType = "close_up"
	spec.Composition = "rule_of_thirds"
	spec.CameraMove = "push_in"
	spec.Emotion = "tense"
	spec.RhythmPosition = RhythmOpenHook
	spec.ContentType = ContentFirstAppear

	if spec.EpisodeNumber != 1 {
		t.Errorf("expected episode 1, got %d", spec.EpisodeNumber)
	}
	if spec.ShotNumber != 3 {
		t.Errorf("expected shot 3, got %d", spec.ShotNumber)
	}
}

func TestShotSpecCharacterRefs(t *testing.T) {
	spec := NewShotSpec(1, 1)
	spec.AddCharacterRef("char_001")
	spec.AddCharacterRef("char_002")
	if len(spec.CharacterRefs) != 2 {
		t.Errorf("expected 2 character refs, got %d", len(spec.CharacterRefs))
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `cd /Users/rain/code/west-garden/ai-drama-research/short-maker && go test ./internal/domain/ -v -run "TestNewAsset|TestAssetPromote|TestNewShotSpec|TestShotSpecChar"`
Expected: compilation error

- [ ] **Step 4: Implement Asset**

```go
// internal/domain/asset.go
package domain

import "time"

type AssetType string

const (
	AssetTypeCharacter AssetType = "character"
	AssetTypeScene     AssetType = "scene"
	AssetTypeProp      AssetType = "prop"
	AssetTypeCostume   AssetType = "costume"
	AssetTypeStyle     AssetType = "style"
	AssetTypeAudio     AssetType = "audio"
)

type AssetScope string

const (
	AssetScopeGlobal  AssetScope = "global"
	AssetScopeProject AssetScope = "project"
)

type Asset struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      AssetType `json:"type"`
	Scope     AssetScope `json:"scope"`
	ProjectID string    `json:"project_id,omitempty"`
	FilePath  string    `json:"file_path"`
	Tags      []string  `json:"tags"`
	Metadata  map[string]string `json:"metadata"`
	CreatedAt time.Time `json:"created_at"`
}

func NewAsset(name string, assetType AssetType, scope AssetScope, projectID string) *Asset {
	return &Asset{
		ID:        generateID("asset"),
		Name:      name,
		Type:      assetType,
		Scope:     scope,
		ProjectID: projectID,
		Tags:      []string{},
		Metadata:  map[string]string{},
		CreatedAt: time.Now(),
	}
}

func (a *Asset) PromoteToGlobal() {
	a.Scope = AssetScopeGlobal
	a.ProjectID = ""
}
```

- [ ] **Step 5: Implement ShotSpec**

```go
// internal/domain/storyboard.go
package domain

type ShotSpec struct {
	EpisodeNumber  int            `json:"episode_number"`
	ShotNumber     int            `json:"shot_number"`
	FrameType      string         `json:"frame_type"`
	Composition    string         `json:"composition"`
	CameraMove     string         `json:"camera_move"`
	Emotion        string         `json:"emotion"`
	Prompt         string         `json:"prompt"`
	CharacterRefs  []string       `json:"character_refs"`
	SceneRef       string         `json:"scene_ref"`
	RhythmPosition RhythmPosition `json:"rhythm_position"`
	ContentType    ContentType    `json:"content_type"`
	StrategyID     string         `json:"strategy_id,omitempty"`
}

func NewShotSpec(episodeNumber, shotNumber int) *ShotSpec {
	return &ShotSpec{
		EpisodeNumber: episodeNumber,
		ShotNumber:    shotNumber,
		CharacterRefs: []string{},
	}
}

func (s *ShotSpec) AddCharacterRef(characterID string) {
	s.CharacterRefs = append(s.CharacterRefs, characterID)
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd /Users/rain/code/west-garden/ai-drama-research/short-maker && go test ./internal/domain/ -v`
Expected: all PASS

- [ ] **Step 7: Commit**

```bash
git add short-maker/internal/domain/asset.go short-maker/internal/domain/asset_test.go \
       short-maker/internal/domain/storyboard.go short-maker/internal/domain/storyboard_test.go
git commit -m "feat(short-maker): Asset (6 types, dual-scope) and ShotSpec domain types"
```

---

### Task 6: Agent Interface

**Files:**
- Create: `short-maker/internal/agent/agent.go`
- Create: `short-maker/internal/agent/mock.go`

- [ ] **Step 1: Define Agent interface and Phase enum**

```go
// internal/agent/agent.go
package agent

import (
	"context"

	"github.com/west-garden/short-maker/internal/domain"
)

type Phase string

const (
	PhaseStoryUnderstanding Phase = "story_understanding"
	PhaseCharacterAsset     Phase = "character_asset"
	PhaseStoryboard         Phase = "storyboard"
	PhaseImageGeneration    Phase = "image_generation"
	PhaseVideoGeneration    Phase = "video_generation"
	PhaseQualityCheck       Phase = "quality_check"
)

// DefaultFlow is the fixed pipeline order. Orchestrator cannot skip these.
var DefaultFlow = []Phase{
	PhaseStoryUnderstanding,
	PhaseCharacterAsset,
	PhaseStoryboard,
	PhaseImageGeneration,
	PhaseVideoGeneration,
}

type Agent interface {
	Phase() Phase
	Run(ctx context.Context, input *PipelineState) (*PipelineState, error)
}

// PipelineState is the structured data passed between agents.
// Each agent reads what it needs and writes its outputs.
type PipelineState struct {
	Project    *domain.Project        `json:"project"`
	Script     string                 `json:"script"`
	Blueprint  *domain.StoryBlueprint `json:"blueprint,omitempty"`
	Assets     []*domain.Asset        `json:"assets,omitempty"`
	Storyboard []*domain.ShotSpec     `json:"storyboard,omitempty"`
	Errors     []string               `json:"errors,omitempty"`
}

func NewPipelineState(project *domain.Project, script string) *PipelineState {
	return &PipelineState{
		Project: project,
		Script:  script,
	}
}
```

- [ ] **Step 2: Create MockAgent for testing**

```go
// internal/agent/mock.go
package agent

import "context"

// MockAgent returns a canned PipelineState for any phase.
// Used in orchestrator tests.
type MockAgent struct {
	phase   Phase
	runFunc func(ctx context.Context, input *PipelineState) (*PipelineState, error)
}

func NewMockAgent(phase Phase, fn func(ctx context.Context, input *PipelineState) (*PipelineState, error)) *MockAgent {
	return &MockAgent{phase: phase, runFunc: fn}
}

func (m *MockAgent) Phase() Phase { return m.phase }

func (m *MockAgent) Run(ctx context.Context, input *PipelineState) (*PipelineState, error) {
	return m.runFunc(ctx, input)
}
```

- [ ] **Step 3: Verify compilation**

Run: `cd /Users/rain/code/west-garden/ai-drama-research/short-maker && go build ./internal/agent/`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add short-maker/internal/agent/agent.go short-maker/internal/agent/mock.go
git commit -m "feat(short-maker): Agent interface, Phase enum, PipelineState, MockAgent"
```

---

### Task 7: LLM Client Interface + Mock

**Files:**
- Create: `short-maker/internal/llm/client.go`
- Create: `short-maker/internal/llm/mock.go`

- [ ] **Step 1: Define LLMClient interface**

```go
// internal/llm/client.go
package llm

import "context"

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Request struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

type Response struct {
	Content    string `json:"content"`
	TokensUsed int    `json:"tokens_used"`
	Model      string `json:"model"`
}

type Client interface {
	Chat(ctx context.Context, req Request) (*Response, error)
}
```

- [ ] **Step 2: Create MockClient**

```go
// internal/llm/mock.go
package llm

import "context"

type MockClient struct {
	responses map[string]string // model -> canned response
	calls     []Request
}

func NewMockClient() *MockClient {
	return &MockClient{
		responses: map[string]string{},
	}
}

func (m *MockClient) SetResponse(model, response string) {
	m.responses[model] = response
}

func (m *MockClient) SetDefaultResponse(response string) {
	m.responses["*"] = response
}

func (m *MockClient) Chat(ctx context.Context, req Request) (*Response, error) {
	m.calls = append(m.calls, req)
	content, ok := m.responses[req.Model]
	if !ok {
		content = m.responses["*"]
	}
	return &Response{
		Content:    content,
		TokensUsed: len(content),
		Model:      req.Model,
	}, nil
}

func (m *MockClient) Calls() []Request { return m.calls }
```

- [ ] **Step 3: Verify compilation**

Run: `cd /Users/rain/code/west-garden/ai-drama-research/short-maker && go build ./internal/llm/`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add short-maker/internal/llm/client.go short-maker/internal/llm/mock.go
git commit -m "feat(short-maker): LLM client interface and MockClient for testing"
```

---

### Task 8: Storage Interface

**Files:**
- Create: `short-maker/internal/store/store.go`

- [ ] **Step 1: Define Store interface**

```go
// internal/store/store.go
package store

import (
	"context"

	"github.com/west-garden/short-maker/internal/domain"
)

type ProjectStore interface {
	SaveProject(ctx context.Context, project *domain.Project) error
	GetProject(ctx context.Context, id string) (*domain.Project, error)
	UpdateProjectStatus(ctx context.Context, id string, status domain.Status) error
}

type AssetStore interface {
	SaveAsset(ctx context.Context, asset *domain.Asset) error
	GetAsset(ctx context.Context, id string) (*domain.Asset, error)
	ListAssets(ctx context.Context, scope domain.AssetScope, projectID string, assetType domain.AssetType) ([]*domain.Asset, error)
	SearchAssets(ctx context.Context, scope domain.AssetScope, tags []string) ([]*domain.Asset, error)
}

type BlueprintStore interface {
	SaveBlueprint(ctx context.Context, blueprint *domain.StoryBlueprint) error
	GetBlueprint(ctx context.Context, projectID string) (*domain.StoryBlueprint, error)
}

// Store combines all storage interfaces. SQLite implements all of them.
type Store interface {
	ProjectStore
	AssetStore
	BlueprintStore
	Close() error
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd /Users/rain/code/west-garden/ai-drama-research/short-maker && go build ./internal/store/`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add short-maker/internal/store/store.go
git commit -m "feat(short-maker): Store interface — ProjectStore, AssetStore, BlueprintStore"
```

---

### Task 9: SQLite Storage Implementation

**Files:**
- Create: `short-maker/internal/store/sqlite.go`
- Create: `short-maker/internal/store/sqlite_test.go`

- [ ] **Step 1: Add SQLite dependency**

```bash
cd /Users/rain/code/west-garden/ai-drama-research/short-maker
go get modernc.org/sqlite
```

- [ ] **Step 2: Write tests for SQLite store**

```go
// internal/store/sqlite_test.go
package store

import (
	"context"
	"os"
	"testing"

	"github.com/west-garden/short-maker/internal/domain"
)

func setupTestDB(t *testing.T) *SQLiteStore {
	t.Helper()
	tmpFile := t.TempDir() + "/test.db"
	s, err := NewSQLiteStore(tmpFile)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() {
		s.Close()
		os.Remove(tmpFile)
	})
	return s
}

func TestSQLiteStore_SaveAndGetProject(t *testing.T) {
	s := setupTestDB(t)
	ctx := context.Background()

	p := domain.NewProject("测试剧", domain.StyleManga, 10)
	if err := s.SaveProject(ctx, p); err != nil {
		t.Fatalf("SaveProject: %v", err)
	}

	got, err := s.GetProject(ctx, p.ID)
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if got.Name != "测试剧" {
		t.Errorf("expected name '测试剧', got '%s'", got.Name)
	}
	if got.Style != domain.StyleManga {
		t.Errorf("expected style manga, got %v", got.Style)
	}
}

func TestSQLiteStore_UpdateProjectStatus(t *testing.T) {
	s := setupTestDB(t)
	ctx := context.Background()

	p := domain.NewProject("测试剧", domain.StyleManga, 10)
	s.SaveProject(ctx, p)

	if err := s.UpdateProjectStatus(ctx, p.ID, domain.StatusProcessing); err != nil {
		t.Fatalf("UpdateProjectStatus: %v", err)
	}

	got, _ := s.GetProject(ctx, p.ID)
	if got.Status != domain.StatusProcessing {
		t.Errorf("expected status processing, got %v", got.Status)
	}
}

func TestSQLiteStore_SaveAndListAssets(t *testing.T) {
	s := setupTestDB(t)
	ctx := context.Background()

	a1 := domain.NewAsset("孙悟空", domain.AssetTypeCharacter, domain.AssetScopeProject, "proj_1")
	a2 := domain.NewAsset("宫殿", domain.AssetTypeScene, domain.AssetScopeProject, "proj_1")
	a3 := domain.NewAsset("通用森林", domain.AssetTypeScene, domain.AssetScopeGlobal, "")
	s.SaveAsset(ctx, a1)
	s.SaveAsset(ctx, a2)
	s.SaveAsset(ctx, a3)

	// List project characters
	chars, err := s.ListAssets(ctx, domain.AssetScopeProject, "proj_1", domain.AssetTypeCharacter)
	if err != nil {
		t.Fatalf("ListAssets: %v", err)
	}
	if len(chars) != 1 {
		t.Errorf("expected 1 character, got %d", len(chars))
	}

	// List global scenes
	scenes, err := s.ListAssets(ctx, domain.AssetScopeGlobal, "", domain.AssetTypeScene)
	if err != nil {
		t.Fatalf("ListAssets: %v", err)
	}
	if len(scenes) != 1 {
		t.Errorf("expected 1 global scene, got %d", len(scenes))
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `cd /Users/rain/code/west-garden/ai-drama-research/short-maker && go test ./internal/store/ -v`
Expected: compilation error — SQLiteStore not defined

- [ ] **Step 4: Implement SQLiteStore**

```go
// internal/store/sqlite.go
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	_ "modernc.org/sqlite"

	"github.com/west-garden/short-maker/internal/domain"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	s := &SQLiteStore{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *SQLiteStore) Close() error { return s.db.Close() }

func (s *SQLiteStore) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS projects (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		style TEXT NOT NULL,
		episode_count INTEGER NOT NULL,
		status TEXT NOT NULL DEFAULT 'created',
		created_at DATETIME NOT NULL
	);
	CREATE TABLE IF NOT EXISTS assets (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		scope TEXT NOT NULL,
		project_id TEXT DEFAULT '',
		file_path TEXT DEFAULT '',
		tags TEXT DEFAULT '[]',
		metadata TEXT DEFAULT '{}',
		created_at DATETIME NOT NULL
	);
	CREATE TABLE IF NOT EXISTS blueprints (
		project_id TEXT PRIMARY KEY,
		data TEXT NOT NULL
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

func (s *SQLiteStore) SaveProject(ctx context.Context, p *domain.Project) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT OR REPLACE INTO projects (id, name, style, episode_count, status, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		p.ID, p.Name, p.Style, p.EpisodeCount, p.Status, p.CreatedAt)
	return err
}

func (s *SQLiteStore) GetProject(ctx context.Context, id string) (*domain.Project, error) {
	p := &domain.Project{}
	err := s.db.QueryRowContext(ctx,
		"SELECT id, name, style, episode_count, status, created_at FROM projects WHERE id = ?", id).
		Scan(&p.ID, &p.Name, &p.Style, &p.EpisodeCount, &p.Status, &p.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get project %s: %w", id, err)
	}
	return p, nil
}

func (s *SQLiteStore) UpdateProjectStatus(ctx context.Context, id string, status domain.Status) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE projects SET status = ? WHERE id = ?", status, id)
	return err
}

func (s *SQLiteStore) SaveAsset(ctx context.Context, a *domain.Asset) error {
	tagsJSON, _ := json.Marshal(a.Tags)
	metaJSON, _ := json.Marshal(a.Metadata)
	_, err := s.db.ExecContext(ctx,
		"INSERT OR REPLACE INTO assets (id, name, type, scope, project_id, file_path, tags, metadata, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		a.ID, a.Name, a.Type, a.Scope, a.ProjectID, a.FilePath, string(tagsJSON), string(metaJSON), a.CreatedAt)
	return err
}

func (s *SQLiteStore) GetAsset(ctx context.Context, id string) (*domain.Asset, error) {
	a := &domain.Asset{}
	var tagsJSON, metaJSON string
	err := s.db.QueryRowContext(ctx,
		"SELECT id, name, type, scope, project_id, file_path, tags, metadata, created_at FROM assets WHERE id = ?", id).
		Scan(&a.ID, &a.Name, &a.Type, &a.Scope, &a.ProjectID, &a.FilePath, &tagsJSON, &metaJSON, &a.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get asset %s: %w", id, err)
	}
	json.Unmarshal([]byte(tagsJSON), &a.Tags)
	json.Unmarshal([]byte(metaJSON), &a.Metadata)
	return a, nil
}

func (s *SQLiteStore) ListAssets(ctx context.Context, scope domain.AssetScope, projectID string, assetType domain.AssetType) ([]*domain.Asset, error) {
	query := "SELECT id, name, type, scope, project_id, file_path, tags, metadata, created_at FROM assets WHERE scope = ? AND type = ?"
	args := []any{scope, assetType}
	if projectID != "" {
		query += " AND project_id = ?"
		args = append(args, projectID)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []*domain.Asset
	for rows.Next() {
		a := &domain.Asset{}
		var tagsJSON, metaJSON string
		if err := rows.Scan(&a.ID, &a.Name, &a.Type, &a.Scope, &a.ProjectID, &a.FilePath, &tagsJSON, &metaJSON, &a.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(tagsJSON), &a.Tags)
		json.Unmarshal([]byte(metaJSON), &a.Metadata)
		assets = append(assets, a)
	}
	return assets, nil
}

func (s *SQLiteStore) SearchAssets(ctx context.Context, scope domain.AssetScope, tags []string) ([]*domain.Asset, error) {
	// Simple tag search: checks if any of the requested tags appear in the asset's tags JSON
	query := "SELECT id, name, type, scope, project_id, file_path, tags, metadata, created_at FROM assets WHERE scope = ?"
	args := []any{scope}
	for _, tag := range tags {
		query += " AND tags LIKE ?"
		args = append(args, "%"+tag+"%")
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []*domain.Asset
	for rows.Next() {
		a := &domain.Asset{}
		var tagsJSON, metaJSON string
		if err := rows.Scan(&a.ID, &a.Name, &a.Type, &a.Scope, &a.ProjectID, &a.FilePath, &tagsJSON, &metaJSON, &a.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(tagsJSON), &a.Tags)
		json.Unmarshal([]byte(metaJSON), &a.Metadata)
		assets = append(assets, a)
	}
	return assets, nil
}

func (s *SQLiteStore) SaveBlueprint(ctx context.Context, bp *domain.StoryBlueprint) error {
	data, err := json.Marshal(bp)
	if err != nil {
		return fmt.Errorf("marshal blueprint: %w", err)
	}
	_, err = s.db.ExecContext(ctx,
		"INSERT OR REPLACE INTO blueprints (project_id, data) VALUES (?, ?)",
		bp.ProjectID, string(data))
	return err
}

func (s *SQLiteStore) GetBlueprint(ctx context.Context, projectID string) (*domain.StoryBlueprint, error) {
	var data string
	err := s.db.QueryRowContext(ctx,
		"SELECT data FROM blueprints WHERE project_id = ?", projectID).Scan(&data)
	if err != nil {
		return nil, fmt.Errorf("get blueprint for %s: %w", projectID, err)
	}
	bp := &domain.StoryBlueprint{}
	if err := json.Unmarshal([]byte(data), bp); err != nil {
		return nil, fmt.Errorf("unmarshal blueprint: %w", err)
	}
	return bp, nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd /Users/rain/code/west-garden/ai-drama-research/short-maker && go test ./internal/store/ -v`
Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add short-maker/internal/store/
git commit -m "feat(short-maker): SQLite storage — projects, assets, blueprints"
```

---

### Task 10: Orchestrator with Default Flow

**Files:**
- Create: `short-maker/internal/agent/orchestrator.go`
- Create: `short-maker/internal/agent/orchestrator_test.go`

- [ ] **Step 1: Write Orchestrator tests**

```go
// internal/agent/orchestrator_test.go
package agent

import (
	"context"
	"testing"

	"github.com/west-garden/short-maker/internal/domain"
)

func TestOrchestrator_RunsDefaultFlow(t *testing.T) {
	var executedPhases []Phase

	makeAgent := func(phase Phase) Agent {
		return NewMockAgent(phase, func(ctx context.Context, state *PipelineState) (*PipelineState, error) {
			executedPhases = append(executedPhases, phase)
			return state, nil
		})
	}

	agents := map[Phase]Agent{
		PhaseStoryUnderstanding: makeAgent(PhaseStoryUnderstanding),
		PhaseCharacterAsset:     makeAgent(PhaseCharacterAsset),
		PhaseStoryboard:         makeAgent(PhaseStoryboard),
		PhaseImageGeneration:    makeAgent(PhaseImageGeneration),
		PhaseVideoGeneration:    makeAgent(PhaseVideoGeneration),
	}

	orch := NewOrchestrator(agents, nil)
	project := domain.NewProject("测试剧", domain.StyleManga, 10)
	state := NewPipelineState(project, "这是一个测试剧本...")

	result, err := orch.Run(context.Background(), state)
	if err != nil {
		t.Fatalf("Orchestrator.Run: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Verify all 5 phases ran in order
	if len(executedPhases) != 5 {
		t.Fatalf("expected 5 phases, got %d", len(executedPhases))
	}
	for i, expected := range DefaultFlow {
		if executedPhases[i] != expected {
			t.Errorf("phase %d: expected %s, got %s", i, expected, executedPhases[i])
		}
	}
}

func TestOrchestrator_CannotSkipPhases(t *testing.T) {
	// Missing PhaseStoryUnderstanding — orchestrator should error
	agents := map[Phase]Agent{
		PhaseCharacterAsset:  NewMockAgent(PhaseCharacterAsset, func(ctx context.Context, s *PipelineState) (*PipelineState, error) { return s, nil }),
		PhaseStoryboard:      NewMockAgent(PhaseStoryboard, func(ctx context.Context, s *PipelineState) (*PipelineState, error) { return s, nil }),
		PhaseImageGeneration: NewMockAgent(PhaseImageGeneration, func(ctx context.Context, s *PipelineState) (*PipelineState, error) { return s, nil }),
		PhaseVideoGeneration: NewMockAgent(PhaseVideoGeneration, func(ctx context.Context, s *PipelineState) (*PipelineState, error) { return s, nil }),
	}

	orch := NewOrchestrator(agents, nil)
	project := domain.NewProject("测试剧", domain.StyleManga, 10)
	state := NewPipelineState(project, "剧本")

	_, err := orch.Run(context.Background(), state)
	if err == nil {
		t.Error("expected error for missing phase, got nil")
	}
}

func TestOrchestrator_CheckpointHook(t *testing.T) {
	var checkpoints []Phase

	agents := map[Phase]Agent{}
	for _, p := range DefaultFlow {
		phase := p
		agents[phase] = NewMockAgent(phase, func(ctx context.Context, s *PipelineState) (*PipelineState, error) { return s, nil })
	}

	hook := func(phase Phase, state *PipelineState) error {
		checkpoints = append(checkpoints, phase)
		return nil
	}

	orch := NewOrchestrator(agents, hook)
	project := domain.NewProject("测试剧", domain.StyleManga, 10)
	state := NewPipelineState(project, "剧本")

	orch.Run(context.Background(), state)

	if len(checkpoints) != 5 {
		t.Errorf("expected 5 checkpoint calls, got %d", len(checkpoints))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/rain/code/west-garden/ai-drama-research/short-maker && go test ./internal/agent/ -v`
Expected: compilation error — NewOrchestrator not defined

- [ ] **Step 3: Implement Orchestrator**

```go
// internal/agent/orchestrator.go
package agent

import (
	"context"
	"fmt"
	"log"
)

// CheckpointHook is called after each phase completes.
// Return a non-nil error to halt the pipeline.
type CheckpointHook func(phase Phase, state *PipelineState) error

type Orchestrator struct {
	agents     map[Phase]Agent
	checkpoint CheckpointHook
}

func NewOrchestrator(agents map[Phase]Agent, checkpoint CheckpointHook) *Orchestrator {
	return &Orchestrator{
		agents:     agents,
		checkpoint: checkpoint,
	}
}

func (o *Orchestrator) Run(ctx context.Context, state *PipelineState) (*PipelineState, error) {
	for _, phase := range DefaultFlow {
		agent, ok := o.agents[phase]
		if !ok {
			return nil, fmt.Errorf("missing required agent for phase: %s", phase)
		}

		log.Printf("[orchestrator] starting phase: %s", phase)

		result, err := agent.Run(ctx, state)
		if err != nil {
			return nil, fmt.Errorf("phase %s failed: %w", phase, err)
		}
		state = result

		if o.checkpoint != nil {
			if err := o.checkpoint(phase, state); err != nil {
				return nil, fmt.Errorf("checkpoint halted at phase %s: %w", phase, err)
			}
		}

		log.Printf("[orchestrator] completed phase: %s", phase)
	}

	return state, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/rain/code/west-garden/ai-drama-research/short-maker && go test ./internal/agent/ -v`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add short-maker/internal/agent/orchestrator.go short-maker/internal/agent/orchestrator_test.go
git commit -m "feat(short-maker): Orchestrator with default flow and checkpoint hooks"
```

---

### Task 11: CLI Entry Point

**Files:**
- Modify: `short-maker/cmd/shortmaker/main.go`

- [ ] **Step 1: Add cobra dependency**

```bash
cd /Users/rain/code/west-garden/ai-drama-research/short-maker
go get github.com/spf13/cobra
```

- [ ] **Step 2: Implement CLI with run command**

```go
// cmd/shortmaker/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/west-garden/short-maker/internal/agent"
	"github.com/west-garden/short-maker/internal/domain"
)

var rootCmd = &cobra.Command{
	Use:   "shortmaker",
	Short: "AI short drama production pipeline",
}

var runCmd = &cobra.Command{
	Use:   "run [script-file]",
	Short: "Run the production pipeline on a script file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		scriptPath := args[0]
		script, err := os.ReadFile(scriptPath)
		if err != nil {
			return fmt.Errorf("read script: %w", err)
		}

		styleName, _ := cmd.Flags().GetString("style")
		episodes, _ := cmd.Flags().GetInt("episodes")

		style := domain.Style(styleName)
		project := domain.NewProject(scriptPath, style, episodes)
		state := agent.NewPipelineState(project, string(script))

		// Build mock agents for now — real agents come in Plan 2-4
		agents := map[agent.Phase]agent.Agent{}
		for _, phase := range agent.DefaultFlow {
			p := phase
			agents[p] = agent.NewMockAgent(p, func(ctx context.Context, s *agent.PipelineState) (*agent.PipelineState, error) {
				log.Printf("  [mock-%s] processing...", p)
				return s, nil
			})
		}

		checkpoint := func(phase agent.Phase, s *agent.PipelineState) error {
			log.Printf("  [checkpoint] phase %s completed", phase)
			return nil
		}

		orch := agent.NewOrchestrator(agents, checkpoint)
		result, err := orch.Run(context.Background(), state)
		if err != nil {
			return fmt.Errorf("pipeline failed: %w", err)
		}

		log.Printf("Pipeline completed for project: %s (errors: %d)", result.Project.Name, len(result.Errors))
		return nil
	},
}

func init() {
	runCmd.Flags().String("style", "manga", "Content style: manga, 3d, live_action")
	runCmd.Flags().Int("episodes", 10, "Number of episodes")
	rootCmd.AddCommand(runCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
```

- [ ] **Step 3: Create a sample test script**

```bash
mkdir -p /Users/rain/code/west-garden/ai-drama-research/short-maker/testdata
```

Write file `short-maker/testdata/sample-script.txt`:
```
第一集：初遇

孙悟空从五行山下被唐僧解救。

场景一：五行山脚下
唐僧骑着白龙马，缓缓走向五行山。山上贴着一张封印符。
唐僧：（抬头看向山顶）这就是被压了五百年的齐天大圣吗？

场景二：山顶
孙悟空从石缝中探出头来。
孙悟空：师父！快救我出来！

第二集：收服

孙悟空获得自由后大闹一番，唐僧用紧箍咒制服他。
```

- [ ] **Step 4: Verify CLI runs end-to-end**

Run: `cd /Users/rain/code/west-garden/ai-drama-research/short-maker && go run ./cmd/shortmaker run testdata/sample-script.txt --style manga --episodes 2`
Expected output includes:
```
[mock-story_understanding] processing...
[checkpoint] phase story_understanding completed
[mock-character_asset] processing...
...
Pipeline completed for project: testdata/sample-script.txt
```

- [ ] **Step 5: Commit**

```bash
git add short-maker/cmd/shortmaker/main.go short-maker/testdata/sample-script.txt
git commit -m "feat(short-maker): CLI entry point with cobra — 'shortmaker run' command"
```

---

### Task 12: Integration Test — Full Pipeline with Mock Agents

**Files:**
- Create: `short-maker/internal/agent/integration_test.go`

- [ ] **Step 1: Write integration test**

```go
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
```

- [ ] **Step 2: Run integration test**

Run: `cd /Users/rain/code/west-garden/ai-drama-research/short-maker && go test ./internal/agent/ -v -run TestIntegration`
Expected: PASS

- [ ] **Step 3: Run all tests across the project**

Run: `cd /Users/rain/code/west-garden/ai-drama-research/short-maker && go test ./... -v`
Expected: all PASS

- [ ] **Step 4: Commit**

```bash
git add short-maker/internal/agent/integration_test.go
git commit -m "feat(short-maker): integration test — full pipeline with mock agents"
```

- [ ] **Step 5: Run go mod tidy and final commit**

```bash
cd /Users/rain/code/west-garden/ai-drama-research/short-maker && go mod tidy
git add short-maker/go.mod short-maker/go.sum
git commit -m "chore(short-maker): go mod tidy"
```
