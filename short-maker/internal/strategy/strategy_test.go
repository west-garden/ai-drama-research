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
