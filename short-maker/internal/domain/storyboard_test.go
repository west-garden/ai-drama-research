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
