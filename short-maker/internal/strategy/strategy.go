package strategy

import "fmt"

type ShotFormula struct {
	FrameType   string `json:"frame_type"`
	Composition string `json:"composition"`
	CameraMove  string `json:"camera_move"`
	Duration    string `json:"duration"`
}

func (f ShotFormula) String() string {
	return fmt.Sprintf("%s / %s / %s / %s", f.FrameType, f.Composition, f.CameraMove, f.Duration)
}

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

type Strategy struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Tags        StrategyTags `json:"tags"`
	ShotFormula ShotFormula  `json:"shot_formula"`
	Examples    []string     `json:"examples"`
	Weight      float64      `json:"weight"`
}
