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
