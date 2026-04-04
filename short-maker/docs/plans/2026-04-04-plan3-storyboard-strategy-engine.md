# Plan 3: Storyboard Agent + Strategy Engine

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Storyboard Agent (pipeline phase 3) and the Strategy Engine that powers it. Given a StoryBlueprint and character assets, produce a complete storyboard — structured ShotSpecs for every scene with shot formulas, character references, and importance annotations (RhythmPosition + ContentType).

**Architecture:** A new `internal/strategy` package provides the Strategy Engine: strategy data types, a JSON-based repository, and a tag-matching coarse filter. The StoryboardAgent uses the Strategy Engine for candidate selection, then calls the LLM to finalize shot details per scene. Each ShotSpec is annotated with RhythmPosition and ContentType, enabling the existing ImportanceScore system from Plan 1 to compute grades for Plan 4's model routing.

**Tech Stack:** Go 1.22+, existing internal packages, `encoding/json` for strategy loading

**Spec reference:** `short-maker/docs/specs/2026-04-03-core-pipeline-design.md` — Sections 三 (阶段 3), 五 (三维度重要性评分, 维度二+三 标注), 六 (爆款策略引擎)

**Depends on:** Plan 1 (domain types, Agent interface, Orchestrator, ImportanceScore), Plan 2 (StoryAgent, CharacterAgent, llm.Client, llm.ParseJSON)

---

## File Structure

```
short-maker/
├── internal/
│   ├── strategy/
│   │   ├── strategy.go           # Strategy, ShotFormula, StrategyTags types
│   │   ├── strategy_test.go
│   │   ├── repository.go         # Repository — load strategies from JSON
│   │   ├── repository_test.go
│   │   ├── matcher.go            # TagMatcher — coarse filter for scene → candidates
│   │   └── matcher_test.go
│   └── agent/
│       ├── storyboard.go         # StoryboardAgent — scenes → ShotSpecs via strategy + LLM
│       ├── storyboard_test.go
│       └── integration_test.go   # (modify) add storyboard integration test
├── cmd/
│   └── shortmaker/
│       └── main.go               # (modify) wire StoryboardAgent
└── data/
    └── strategies.json           # Seed strategy data for MVP
```

---

### Task 1: Strategy Data Types + JSON Repository

**Files:**
- Create: `short-maker/internal/strategy/strategy.go`
- Create: `short-maker/internal/strategy/strategy_test.go`
- Create: `short-maker/internal/strategy/repository.go`
- Create: `short-maker/internal/strategy/repository_test.go`

- [ ] **Step 1: Write tests for Strategy types**

```go
// internal/strategy/strategy_test.go
package strategy

import "testing"

func TestShotFormula_String(t *testing.T) {
	f := ShotFormula{
		FrameType:   "close_up",
		Composition: "center",
		CameraMove:  "zoom_in",
		Duration:    "short",
	}
	s := f.String()
	if s == "" {
		t.Error("expected non-empty string")
	}
}

func TestStrategy_HasTag(t *testing.T) {
	s := &Strategy{
		ID:   "test_001",
		Name: "Test Strategy",
		Tags: StrategyTags{
			Pacing: []string{"fast", "medium"},
		},
		Weight: 1.0,
	}
	if !s.Tags.HasPacing("fast") {
		t.Error("expected HasPacing('fast') to be true")
	}
	if s.Tags.HasPacing("slow") {
		t.Error("expected HasPacing('slow') to be false")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./internal/strategy/ -v -run TestShotFormula`
Expected: compilation error — package does not exist

- [ ] **Step 3: Implement strategy types**

```go
// internal/strategy/strategy.go
package strategy

import "fmt"

// ShotFormula describes the cinematic parameters for a shot.
type ShotFormula struct {
	FrameType   string `json:"frame_type"`   // close_up, medium, wide, extreme_wide
	Composition string `json:"composition"`  // center, rule_of_thirds, over_shoulder, low_angle, high_angle
	CameraMove  string `json:"camera_move"`  // static, pan, zoom_in, zoom_out, tracking
	Duration    string `json:"duration"`     // short(1-2s), medium(2-4s), long(4-6s)
}

func (f ShotFormula) String() string {
	return fmt.Sprintf("%s / %s / %s / %s", f.FrameType, f.Composition, f.CameraMove, f.Duration)
}

// StrategyTags defines the multi-dimensional tags for strategy matching.
type StrategyTags struct {
	NarrativeBeat  []string `json:"narrative_beat"`
	EmotionArc     []string `json:"emotion_arc"`
	Pacing         []string `json:"pacing"`
	CharacterCount []int    `json:"character_count"`
}

func (t StrategyTags) HasPacing(p string) bool {
	for _, v := range t.Pacing {
		if v == p {
			return true
		}
	}
	return false
}

// Strategy represents a reusable shot strategy from the strategy library.
type Strategy struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Tags        StrategyTags `json:"tags"`
	ShotFormula ShotFormula  `json:"shot_formula"`
	Examples    []string     `json:"examples"`
	Weight      float64      `json:"weight"`
}
```

- [ ] **Step 4: Write tests for Repository**

```go
// internal/strategy/repository_test.go
package strategy

import "testing"

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

func TestRepository_LoadFromJSON(t *testing.T) {
	repo, err := LoadFromJSON([]byte(testStrategiesJSON))
	if err != nil {
		t.Fatalf("LoadFromJSON: %v", err)
	}
	if len(repo.All()) != 2 {
		t.Fatalf("expected 2 strategies, got %d", len(repo.All()))
	}
}

func TestRepository_Get(t *testing.T) {
	repo, err := LoadFromJSON([]byte(testStrategiesJSON))
	if err != nil {
		t.Fatalf("LoadFromJSON: %v", err)
	}
	s := repo.Get("strat_001")
	if s == nil {
		t.Fatal("expected strategy strat_001")
	}
	if s.Name != "悬念特写" {
		t.Errorf("expected name '悬念特写', got '%s'", s.Name)
	}
}

func TestRepository_GetMissing(t *testing.T) {
	repo, err := LoadFromJSON([]byte(testStrategiesJSON))
	if err != nil {
		t.Fatalf("LoadFromJSON: %v", err)
	}
	s := repo.Get("nonexistent")
	if s != nil {
		t.Error("expected nil for missing strategy")
	}
}

func TestRepository_LoadInvalidJSON(t *testing.T) {
	_, err := LoadFromJSON([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
```

- [ ] **Step 5: Implement Repository**

```go
// internal/strategy/repository.go
package strategy

import (
	"encoding/json"
	"fmt"
	"os"
)

// Repository holds strategies in memory and provides lookup.
type Repository struct {
	strategies []*Strategy
	index      map[string]*Strategy
}

// LoadFromJSON parses a JSON array of strategies.
func LoadFromJSON(data []byte) (*Repository, error) {
	var strategies []*Strategy
	if err := json.Unmarshal(data, &strategies); err != nil {
		return nil, fmt.Errorf("parse strategies JSON: %w", err)
	}
	repo := &Repository{
		strategies: strategies,
		index:      make(map[string]*Strategy, len(strategies)),
	}
	for _, s := range strategies {
		repo.index[s.ID] = s
	}
	return repo, nil
}

// LoadFromFile reads strategies from a JSON file.
func LoadFromFile(path string) (*Repository, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read strategies file: %w", err)
	}
	return LoadFromJSON(data)
}

// All returns all strategies.
func (r *Repository) All() []*Strategy {
	return r.strategies
}

// Get returns a strategy by ID, or nil if not found.
func (r *Repository) Get(id string) *Strategy {
	return r.index[id]
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./internal/strategy/ -v`
Expected: all PASS

- [ ] **Step 7: Commit**

```bash
git add short-maker/internal/strategy/strategy.go short-maker/internal/strategy/strategy_test.go \
       short-maker/internal/strategy/repository.go short-maker/internal/strategy/repository_test.go
git commit -m "feat(short-maker): strategy data types and JSON repository"
```

---

### Task 2: Tag Matcher (Coarse Filter)

**Files:**
- Create: `short-maker/internal/strategy/matcher.go`
- Create: `short-maker/internal/strategy/matcher_test.go`

- [ ] **Step 1: Write tests for MatchScene**

```go
// internal/strategy/matcher_test.go
package strategy

import (
	"testing"

	"github.com/west-garden/short-maker/internal/domain"
)

func TestMatchScene_ReturnsRankedCandidates(t *testing.T) {
	repo, err := LoadFromJSON([]byte(testStrategiesJSON))
	if err != nil {
		t.Fatalf("LoadFromJSON: %v", err)
	}

	scene := domain.SceneTag{
		NarrativeBeat:  "冲突爆发",
		EmotionArc:     "紧张→震惊",
		Pacing:         "fast",
		CharacterCount: 2,
	}

	results := MatchScene(repo, scene, 5)
	if len(results) == 0 {
		t.Fatal("expected at least 1 candidate")
	}
	// strat_001 (悬念特写) should rank higher — matches pacing=fast, emotion=紧张, narrative_beat=冲突
	if results[0].Strategy.ID != "strat_001" {
		t.Errorf("expected strat_001 ranked first, got %s", results[0].Strategy.ID)
	}
	if results[0].Score <= 0 {
		t.Error("expected positive score")
	}
}

func TestMatchScene_LimitsResults(t *testing.T) {
	repo, err := LoadFromJSON([]byte(testStrategiesJSON))
	if err != nil {
		t.Fatalf("LoadFromJSON: %v", err)
	}

	scene := domain.SceneTag{
		Pacing:         "fast",
		CharacterCount: 1,
	}

	results := MatchScene(repo, scene, 1)
	if len(results) > 1 {
		t.Errorf("expected at most 1 result, got %d", len(results))
	}
}

func TestMatchScene_EmptyRepo(t *testing.T) {
	repo, _ := LoadFromJSON([]byte("[]"))
	scene := domain.SceneTag{Pacing: "fast"}
	results := MatchScene(repo, scene, 5)
	if len(results) != 0 {
		t.Errorf("expected 0 results from empty repo, got %d", len(results))
	}
}

func TestMatchScene_CharacterCountMatch(t *testing.T) {
	repo, err := LoadFromJSON([]byte(testStrategiesJSON))
	if err != nil {
		t.Fatalf("LoadFromJSON: %v", err)
	}

	// Both strategies support character_count=2, but scene with pacing=slow
	// should favor strat_002 (全景建立) which has pacing=slow
	scene := domain.SceneTag{
		NarrativeBeat:  "开场引入",
		Pacing:         "slow",
		CharacterCount: 2,
	}
	results := MatchScene(repo, scene, 5)
	if len(results) == 0 {
		t.Fatal("expected at least 1 candidate")
	}
	if results[0].Strategy.ID != "strat_002" {
		t.Errorf("expected strat_002 for slow-paced opening, got %s", results[0].Strategy.ID)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./internal/strategy/ -v -run TestMatchScene`
Expected: compilation error — MatchScene not defined

- [ ] **Step 3: Implement TagMatcher**

```go
// internal/strategy/matcher.go
package strategy

import (
	"sort"
	"strings"

	"github.com/west-garden/short-maker/internal/domain"
)

// ScoredStrategy pairs a strategy with its match score.
type ScoredStrategy struct {
	Strategy *Strategy
	Score    float64
}

// MatchScene scores all strategies against a scene's tags and returns
// the top candidates sorted by score descending. This is the "coarse
// filter" phase of the strategy engine — fast, deterministic, zero LLM cost.
func MatchScene(repo *Repository, scene domain.SceneTag, maxResults int) []ScoredStrategy {
	var scored []ScoredStrategy

	for _, s := range repo.All() {
		score := scoreStrategy(s, scene)
		if score > 0 {
			scored = append(scored, ScoredStrategy{Strategy: s, Score: score})
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	if len(scored) > maxResults {
		scored = scored[:maxResults]
	}
	return scored
}

// scoreStrategy computes a match score between a strategy and a scene.
// Each matching tag dimension adds to the score:
//   - Pacing exact match: +2.0 (strongest signal)
//   - NarrativeBeat substring match: +1.5
//   - EmotionArc substring match: +1.0
//   - CharacterCount match: +0.5
//
// Final score is multiplied by the strategy's weight.
func scoreStrategy(s *Strategy, scene domain.SceneTag) float64 {
	var score float64

	// Pacing match (exact)
	if scene.Pacing != "" {
		for _, p := range s.Tags.Pacing {
			if p == scene.Pacing {
				score += 2.0
				break
			}
		}
	}

	// NarrativeBeat match (substring — scene's beat may contain strategy's tag)
	if scene.NarrativeBeat != "" {
		for _, nb := range s.Tags.NarrativeBeat {
			if strings.Contains(scene.NarrativeBeat, nb) {
				score += 1.5
				break
			}
		}
	}

	// EmotionArc match (substring — scene's arc may contain strategy's tag)
	if scene.EmotionArc != "" {
		for _, ea := range s.Tags.EmotionArc {
			if strings.Contains(scene.EmotionArc, ea) {
				score += 1.0
				break
			}
		}
	}

	// CharacterCount match
	if scene.CharacterCount > 0 {
		for _, cc := range s.Tags.CharacterCount {
			if cc == scene.CharacterCount {
				score += 0.5
				break
			}
		}
	}

	return score * s.Weight
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./internal/strategy/ -v -run TestMatchScene`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add short-maker/internal/strategy/matcher.go short-maker/internal/strategy/matcher_test.go
git commit -m "feat(short-maker): strategy tag matcher — coarse filter for scene matching"
```

---

### Task 3: Seed Strategy Data

**Files:**
- Create: `short-maker/data/strategies.json`

- [ ] **Step 1: Create seed strategy data**

```json
[
  {
    "id": "strat_001",
    "name": "悬念特写",
    "tags": {
      "narrative_beat": ["冲突", "反转", "悬念"],
      "emotion_arc": ["紧张", "震惊", "恐惧"],
      "pacing": ["fast"],
      "character_count": [1, 2]
    },
    "shot_formula": {
      "frame_type": "close_up",
      "composition": "center",
      "camera_move": "zoom_in",
      "duration": "short"
    },
    "examples": ["角色发现真相的震惊特写", "关键道具被发现的近距离镜头"],
    "weight": 1.0
  },
  {
    "id": "strat_002",
    "name": "全景建立",
    "tags": {
      "narrative_beat": ["开场", "转场", "环境"],
      "emotion_arc": ["平静", "期待", "壮阔"],
      "pacing": ["slow", "medium"],
      "character_count": [0, 1, 2, 3]
    },
    "shot_formula": {
      "frame_type": "extreme_wide",
      "composition": "center",
      "camera_move": "pan",
      "duration": "long"
    },
    "examples": ["远景展示宏大场景", "故事发生地的全貌"],
    "weight": 1.0
  },
  {
    "id": "strat_003",
    "name": "对峙中景",
    "tags": {
      "narrative_beat": ["冲突", "对峙", "谈判"],
      "emotion_arc": ["紧张", "对抗", "压迫"],
      "pacing": ["medium", "fast"],
      "character_count": [2]
    },
    "shot_formula": {
      "frame_type": "medium",
      "composition": "rule_of_thirds",
      "camera_move": "static",
      "duration": "medium"
    },
    "examples": ["两人面对面紧张对话", "师徒之间的分歧"],
    "weight": 1.0
  },
  {
    "id": "strat_004",
    "name": "动作追踪",
    "tags": {
      "narrative_beat": ["打斗", "追逐", "动作"],
      "emotion_arc": ["激动", "紧张", "兴奋"],
      "pacing": ["fast"],
      "character_count": [1, 2, 3]
    },
    "shot_formula": {
      "frame_type": "medium",
      "composition": "rule_of_thirds",
      "camera_move": "tracking",
      "duration": "short"
    },
    "examples": ["角色奔跑或飞行的追踪镜头", "打斗中的动态跟拍"],
    "weight": 1.0
  },
  {
    "id": "strat_005",
    "name": "情感特写",
    "tags": {
      "narrative_beat": ["告白", "离别", "感动", "回忆"],
      "emotion_arc": ["悲伤", "温暖", "感动", "释放"],
      "pacing": ["slow"],
      "character_count": [1, 2]
    },
    "shot_formula": {
      "frame_type": "close_up",
      "composition": "center",
      "camera_move": "static",
      "duration": "medium"
    },
    "examples": ["角色流泪的面部特写", "深情对视"],
    "weight": 1.0
  },
  {
    "id": "strat_006",
    "name": "肩上镜头",
    "tags": {
      "narrative_beat": ["对话", "日常", "交流"],
      "emotion_arc": ["平静", "好奇", "温暖"],
      "pacing": ["medium", "slow"],
      "character_count": [2]
    },
    "shot_formula": {
      "frame_type": "medium",
      "composition": "over_shoulder",
      "camera_move": "static",
      "duration": "medium"
    },
    "examples": ["日常对话的经典过肩镜头", "两人交谈的正反打"],
    "weight": 1.0
  },
  {
    "id": "strat_007",
    "name": "仰拍震撼",
    "tags": {
      "narrative_beat": ["登场", "亮相", "威慑"],
      "emotion_arc": ["震撼", "敬畏", "压迫"],
      "pacing": ["medium", "fast"],
      "character_count": [1]
    },
    "shot_formula": {
      "frame_type": "medium",
      "composition": "low_angle",
      "camera_move": "zoom_out",
      "duration": "medium"
    },
    "examples": ["反派角色首次登场的仰拍", "展现角色力量感"],
    "weight": 1.0
  },
  {
    "id": "strat_008",
    "name": "俯拍孤独",
    "tags": {
      "narrative_beat": ["独处", "思考", "绝望"],
      "emotion_arc": ["孤独", "悲伤", "渺小"],
      "pacing": ["slow"],
      "character_count": [1]
    },
    "shot_formula": {
      "frame_type": "wide",
      "composition": "high_angle",
      "camera_move": "static",
      "duration": "long"
    },
    "examples": ["角色独坐在空旷场景的俯拍", "展现角色的渺小感"],
    "weight": 1.0
  },
  {
    "id": "strat_009",
    "name": "快切节奏",
    "tags": {
      "narrative_beat": ["高潮", "打斗", "混乱"],
      "emotion_arc": ["激动", "混乱", "紧张"],
      "pacing": ["fast"],
      "character_count": [1, 2, 3]
    },
    "shot_formula": {
      "frame_type": "close_up",
      "composition": "rule_of_thirds",
      "camera_move": "static",
      "duration": "short"
    },
    "examples": ["打斗场景的快速切换", "多角度快切展现紧张感"],
    "weight": 1.0
  },
  {
    "id": "strat_010",
    "name": "尾部悬念",
    "tags": {
      "narrative_beat": ["悬念", "尾声", "结尾"],
      "emotion_arc": ["好奇", "紧张", "期待"],
      "pacing": ["medium"],
      "character_count": [0, 1]
    },
    "shot_formula": {
      "frame_type": "medium",
      "composition": "center",
      "camera_move": "zoom_in",
      "duration": "medium"
    },
    "examples": ["结尾画面缓慢推进到悬念画面", "未解之谜的暗示镜头"],
    "weight": 1.0
  }
]
```

- [ ] **Step 2: Write a load test to verify seed data is valid**

Add this test to `short-maker/internal/strategy/repository_test.go`:

```go
func TestRepository_LoadSeedData(t *testing.T) {
	repo, err := LoadFromFile("../../data/strategies.json")
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}
	if len(repo.All()) < 10 {
		t.Errorf("expected at least 10 seed strategies, got %d", len(repo.All()))
	}
	for _, s := range repo.All() {
		if s.ID == "" {
			t.Error("strategy missing ID")
		}
		if s.Name == "" {
			t.Error("strategy missing Name")
		}
		if s.Weight <= 0 {
			t.Errorf("strategy %s has non-positive weight: %f", s.ID, s.Weight)
		}
	}
}
```

- [ ] **Step 3: Run tests**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./internal/strategy/ -v -run TestRepository_LoadSeedData`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add short-maker/data/strategies.json short-maker/internal/strategy/repository_test.go
git commit -m "feat(short-maker): seed strategy data — 10 shot strategies for MVP"
```

---

### Task 4: Storyboard Agent

**Files:**
- Create: `short-maker/internal/agent/storyboard.go`
- Create: `short-maker/internal/agent/storyboard_test.go`

- [ ] **Step 1: Write tests for StoryboardAgent**

```go
// internal/agent/storyboard_test.go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./internal/agent/ -v -run TestStoryboardAgent`
Expected: compilation error — NewStoryboardAgent not defined

- [ ] **Step 3: Implement StoryboardAgent**

```go
// internal/agent/storyboard.go
package agent

import (
	"context"
	"fmt"
	"log"

	"github.com/west-garden/short-maker/internal/domain"
	"github.com/west-garden/short-maker/internal/llm"
	"github.com/west-garden/short-maker/internal/strategy"
)

// StoryboardAgent generates structured shot specifications for each scene
// in the blueprint. Uses the strategy engine for candidate selection and
// LLM for final shot details. Implements Agent for PhaseStoryboard.
type StoryboardAgent struct {
	llmClient    llm.Client
	model        string
	strategyRepo *strategy.Repository
}

func NewStoryboardAgent(llmClient llm.Client, model string, repo *strategy.Repository) *StoryboardAgent {
	return &StoryboardAgent{llmClient: llmClient, model: model, strategyRepo: repo}
}

func (a *StoryboardAgent) Phase() Phase { return PhaseStoryboard }

func (a *StoryboardAgent) Run(ctx context.Context, state *PipelineState) (*PipelineState, error) {
	if state.Blueprint == nil {
		return nil, fmt.Errorf("storyboard agent requires a Blueprint")
	}

	// Build character name → ID lookup from blueprint
	nameToID := make(map[string]string, len(state.Blueprint.Characters))
	for _, ch := range state.Blueprint.Characters {
		nameToID[ch.Name] = ch.ID
	}

	shotCounter := 0
	for _, ep := range state.Blueprint.Episodes {
		log.Printf("[storyboard-agent] episode %d (%s): %d scenes", ep.Number, ep.Role, len(ep.Scenes))

		var prevShots []string // brief context of previous shots for continuity

		for _, scene := range ep.Scenes {
			// Phase 1: Coarse filter — get candidate strategies
			var candidates []strategy.ScoredStrategy
			if a.strategyRepo != nil {
				candidates = strategy.MatchScene(a.strategyRepo, scene, 6)
			}

			// Phase 2: LLM — select strategy + generate shot details
			shots, err := a.generateShotsForScene(ctx, ep, scene, candidates, prevShots, state)
			if err != nil {
				return nil, fmt.Errorf("generate shots for ep%d scene '%s': %w", ep.Number, scene.NarrativeBeat, err)
			}

			for _, shot := range shots {
				shotCounter++
				spec := domain.NewShotSpec(ep.Number, shotCounter)
				spec.FrameType = shot.FrameType
				spec.Composition = shot.Composition
				spec.CameraMove = shot.CameraMove
				spec.Emotion = shot.Emotion
				spec.Prompt = shot.Prompt
				spec.SceneRef = shot.SceneRef
				spec.StrategyID = shot.StrategyID
				spec.RhythmPosition = domain.RhythmPosition(shot.RhythmPosition)
				spec.ContentType = domain.ContentType(shot.ContentType)

				// Resolve character names to IDs
				for _, name := range shot.CharacterNames {
					if id, ok := nameToID[name]; ok {
						spec.AddCharacterRef(id)
					}
				}

				state.Storyboard = append(state.Storyboard, spec)
				prevShots = append(prevShots, fmt.Sprintf("shot%d: %s %s", shotCounter, spec.FrameType, spec.Emotion))
			}
		}
	}

	log.Printf("[storyboard-agent] generated %d total shots", shotCounter)
	return state, nil
}

func (a *StoryboardAgent) generateShotsForScene(
	ctx context.Context,
	ep *domain.EpisodeBlueprint,
	scene domain.SceneTag,
	candidates []strategy.ScoredStrategy,
	prevShots []string,
	state *PipelineState,
) ([]storyboardShotResponse, error) {
	systemPrompt := buildStoryboardSystemPrompt()
	userPrompt := buildStoryboardUserPrompt(ep, scene, candidates, prevShots, state)

	resp, err := a.llmClient.Chat(ctx, llm.Request{
		Model: a.model,
		Messages: []llm.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.4,
		MaxTokens:   2048,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM call: %w", err)
	}

	var parsed storyboardResponse
	if err := llm.ParseJSON(resp.Content, &parsed); err != nil {
		return nil, fmt.Errorf("parse storyboard response: %w", err)
	}
	return parsed.Shots, nil
}

// --- Prompt builders ---

func buildStoryboardSystemPrompt() string {
	return `You are a storyboard generation agent for an AI short drama production system.
Given a scene description and candidate shot strategies, generate structured shot specifications.

For each shot, select the most appropriate strategy and customize it for the scene context.
Annotate each shot with rhythm_position and content_type for importance scoring.

Output ONLY valid JSON with this schema:
{
  "shots": [
    {
      "strategy_id": "id of the selected strategy (or empty if no candidates)",
      "frame_type": "close_up|medium|wide|extreme_wide",
      "composition": "description of shot composition",
      "camera_move": "static|pan|zoom_in|zoom_out|tracking",
      "emotion": "emotional tone of the shot in Chinese",
      "prompt": "detailed visual description for image generation — include art style, specific visual elements, lighting, atmosphere",
      "character_names": ["character names visible in this shot"],
      "scene_ref": "scene or setting name",
      "rhythm_position": "open_hook|emotion_peak|tail_hook|mid_narration",
      "content_type": "first_appear|fight|dialogue|empty"
    }
  ]
}

rhythm_position rules:
- "open_hook": first shot of the episode (must grab attention)
- "emotion_peak": shots at emotional turning points (conflict, revelation, reunion)
- "tail_hook": last shot of the episode (create suspense for next episode)
- "mid_narration": everything else (setup, transition, daily scenes)

content_type rules:
- "first_appear": a character's first appearance in the series (visual anchor point)
- "fight": action, combat, special effects, large-scale scenes
- "dialogue": conversation, close-up on expressions
- "empty": scenery, environment, flashback, no characters

Generate 1-3 shots per scene. Each shot should have a distinct purpose in the narrative.`
}

func buildStoryboardUserPrompt(
	ep *domain.EpisodeBlueprint,
	scene domain.SceneTag,
	candidates []strategy.ScoredStrategy,
	prevShots []string,
	state *PipelineState,
) string {
	prompt := fmt.Sprintf("Episode %d (role: %s), emotion arc: %s\n", ep.Number, ep.Role, ep.EmotionArc)
	prompt += fmt.Sprintf("Synopsis: %s\n\n", ep.Synopsis)
	prompt += fmt.Sprintf("Scene: %s\n", scene.NarrativeBeat)
	prompt += fmt.Sprintf("Scene emotion: %s\n", scene.EmotionArc)
	prompt += fmt.Sprintf("Setting: %s\n", scene.Setting)
	prompt += fmt.Sprintf("Pacing: %s\n", scene.Pacing)
	prompt += fmt.Sprintf("Character count: %d\n", scene.CharacterCount)
	prompt += fmt.Sprintf("Style: %s\n\n", state.Project.Style)

	// Characters available
	if state.Blueprint != nil && len(state.Blueprint.Characters) > 0 {
		prompt += "Characters:\n"
		for _, ch := range state.Blueprint.Characters {
			prompt += fmt.Sprintf("- %s: %s\n", ch.Name, ch.Description)
		}
		prompt += "\n"
	}

	// Candidate strategies
	if len(candidates) > 0 {
		prompt += "Candidate strategies (pick the most appropriate for each shot):\n"
		for i, c := range candidates {
			prompt += fmt.Sprintf("%d. [%s] %s — %s (score: %.1f)\n",
				i+1, c.Strategy.ID, c.Strategy.Name, c.Strategy.ShotFormula.String(), c.Score)
			if len(c.Strategy.Examples) > 0 {
				prompt += fmt.Sprintf("   Example: %s\n", c.Strategy.Examples[0])
			}
		}
		prompt += "\n"
	}

	// Previous shots for continuity
	if len(prevShots) > 0 {
		prompt += "Previous shots in this episode (for continuity):\n"
		limit := 3
		if len(prevShots) < limit {
			limit = len(prevShots)
		}
		for _, ps := range prevShots[len(prevShots)-limit:] {
			prompt += fmt.Sprintf("- %s\n", ps)
		}
	}

	return prompt
}

// --- Response types ---

type storyboardResponse struct {
	Shots []storyboardShotResponse `json:"shots"`
}

type storyboardShotResponse struct {
	StrategyID    string   `json:"strategy_id"`
	FrameType     string   `json:"frame_type"`
	Composition   string   `json:"composition"`
	CameraMove    string   `json:"camera_move"`
	Emotion       string   `json:"emotion"`
	Prompt        string   `json:"prompt"`
	CharacterNames []string `json:"character_names"`
	SceneRef      string   `json:"scene_ref"`
	RhythmPosition string  `json:"rhythm_position"`
	ContentType   string   `json:"content_type"`
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./internal/agent/ -v -run TestStoryboardAgent`
Expected: all PASS

- [ ] **Step 5: Run ALL tests**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./... -count=1`
Expected: all PASS across all packages

- [ ] **Step 6: Commit**

```bash
git add short-maker/internal/agent/storyboard.go short-maker/internal/agent/storyboard_test.go
git commit -m "feat(short-maker): Storyboard Agent — scene → shot specs via strategy engine + LLM"
```

---

### Task 5: Wire StoryboardAgent into CLI

**Files:**
- Modify: `short-maker/cmd/shortmaker/main.go`

- [ ] **Step 1: Read current main.go**

Read: `short-maker/cmd/shortmaker/main.go`

- [ ] **Step 2: Update buildAgents to use real StoryboardAgent**

In the `buildAgents` function, add a `strategyPath` parameter and wire the real StoryboardAgent. Update the `runCmd` to read the new `--strategies` flag.

Changes to `init()` — add the strategies flag:

```go
runCmd.Flags().String("strategies", "", "Path to strategies JSON file (enables real storyboard agent)")
```

Changes to `runCmd.RunE` — read the flag:

```go
strategyPath, _ := cmd.Flags().GetString("strategies")
```

Pass `strategyPath` to `buildAgents`.

Changes to `buildAgents` function signature:

```go
func buildAgents(useMock bool, llmModel, dbPath, strategyPath string) (map[agent.Phase]agent.Agent, func(), error)
```

In the real-agent branch (after creating llmClient), add:

```go
// Storyboard agent with strategy engine
if strategyPath != "" {
    repo, err := strategy.LoadFromFile(strategyPath)
    if err != nil {
        return nil, nil, fmt.Errorf("load strategies: %w", err)
    }
    agents[agent.PhaseStoryboard] = agent.NewStoryboardAgent(llmClient, llmModel, repo)
} else {
    agents[agent.PhaseStoryboard] = agent.NewStoryboardAgent(llmClient, llmModel, nil)
}
```

Add import for `"github.com/west-garden/short-maker/internal/strategy"`.

Remove `PhaseStoryboard` from the mock fallback loop (it's now always real in non-mock mode).

- [ ] **Step 3: Verify mock mode still works**

Run: `go run -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./cmd/shortmaker run testdata/sample-script.txt --style manga --episodes 2`
Expected: all mock agents, pipeline completes

- [ ] **Step 4: Verify build succeeds**

Run: `go build -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./...`
Expected: success

- [ ] **Step 5: Commit**

```bash
git add short-maker/cmd/shortmaker/main.go
git commit -m "feat(short-maker): wire StoryboardAgent into CLI with --strategies flag"
```

---

### Task 6: Integration Test

**Files:**
- Modify: `short-maker/internal/agent/integration_test.go`

- [ ] **Step 1: Read current integration_test.go**

Read: `short-maker/internal/agent/integration_test.go`

- [ ] **Step 2: Add a new integration test using Story + Character + Storyboard agents**

Add this test to the existing file. Do NOT remove existing tests.

```go
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
```

Add import for `"github.com/west-garden/short-maker/internal/strategy"` if not already present.

- [ ] **Step 3: Run the new integration test**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./internal/agent/ -v -run TestIntegration_FullPipelineWithStoryboardAgent`
Expected: PASS

- [ ] **Step 4: Run ALL tests**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./... -count=1`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add short-maker/internal/agent/integration_test.go
git commit -m "feat(short-maker): integration test — full pipeline with Storyboard Agent + strategy engine"
```

- [ ] **Step 6: Run go mod tidy and commit if changed**

```bash
GOWORK=off go mod tidy -C /Users/rain/code/west-garden/ai-drama-research/short-maker
git diff --name-only short-maker/go.mod short-maker/go.sum
```
If changed:
```bash
git add short-maker/go.mod short-maker/go.sum
git commit -m "chore(short-maker): go mod tidy"
```
