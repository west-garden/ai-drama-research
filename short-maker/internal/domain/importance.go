// internal/domain/importance.go
package domain

import "math"

// Dimension 2: Intra-episode rhythm position
type RhythmPosition string

const (
	RhythmOpenHook     RhythmPosition = "open_hook"     // 开场钩子 ×1.4
	RhythmEmotionPeak  RhythmPosition = "emotion_peak"  // 情绪高点 ×1.2
	RhythmTailHook     RhythmPosition = "tail_hook"     // 尾部钩子 ×1.2
	RhythmMidNarration RhythmPosition = "mid_narration" // 中段叙事 ×1.0
)

func (r RhythmPosition) Weight() float64 {
	switch r {
	case RhythmOpenHook:
		return 1.4
	case RhythmEmotionPeak, RhythmTailHook:
		return 1.2
	default:
		return 1.0
	}
}

// Dimension 3: Content type
type ContentType string

const (
	ContentFirstAppear ContentType = "first_appear" // 角色首次出场 ×1.3
	ContentFight       ContentType = "fight"        // 打斗/特效 ×1.2
	ContentDialogue    ContentType = "dialogue"     // 对话/特写 ×1.0
	ContentEmpty       ContentType = "empty"        // 空镜/环境 ×0.8
)

func (c ContentType) Weight() float64 {
	switch c {
	case ContentFirstAppear:
		return 1.3
	case ContentFight:
		return 1.2
	case ContentDialogue:
		return 1.0
	case ContentEmpty:
		return 0.8
	default:
		return 1.0
	}
}

type Grade string

const (
	GradeS Grade = "S" // ≥ 2.0
	GradeA Grade = "A" // 1.4 ~ 2.0
	GradeB Grade = "B" // 1.0 ~ 1.4
	GradeC Grade = "C" // < 1.0
)

func (g Grade) QualityThreshold() int {
	switch g {
	case GradeS:
		return 85
	case GradeA:
		return 75
	case GradeB:
		return 65
	default:
		return 55
	}
}

type ImportanceScore struct {
	EpisodeRole    EpisodeRole    `json:"episode_role"`
	RhythmPosition RhythmPosition `json:"rhythm_position"`
	ContentType    ContentType    `json:"content_type"`
}

func NewImportanceScore(ep EpisodeRole, rhythm RhythmPosition, content ContentType) ImportanceScore {
	return ImportanceScore{
		EpisodeRole:    ep,
		RhythmPosition: rhythm,
		ContentType:    content,
	}
}

func (s ImportanceScore) Score() float64 {
	raw := s.EpisodeRole.Weight() * s.RhythmPosition.Weight() * s.ContentType.Weight()
	return math.Round(raw*100) / 100
}

func (s ImportanceScore) Grade() Grade {
	score := s.Score()
	switch {
	case score >= 2.0:
		return GradeS
	case score >= 1.4:
		return GradeA
	case score >= 1.0:
		return GradeB
	default:
		return GradeC
	}
}

func (s ImportanceScore) MaxRetries() int {
	switch s.Grade() {
	case GradeS:
		return 3
	case GradeA:
		return 2
	case GradeB:
		return 1
	default:
		return 0
	}
}
