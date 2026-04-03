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
