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
