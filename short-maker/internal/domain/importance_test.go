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
