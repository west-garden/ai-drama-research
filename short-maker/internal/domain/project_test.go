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
