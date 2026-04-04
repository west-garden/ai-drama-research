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
