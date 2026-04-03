// internal/domain/blueprint.go
package domain

type EpisodeRole string

const (
	EpisodeRoleHook       EpisodeRole = "hook"       // 钩子集 ×1.5
	EpisodeRolePaywall    EpisodeRole = "paywall"    // 付费卡点 ×1.3
	EpisodeRoleClimax     EpisodeRole = "climax"     // 高潮/大结局 ×1.2
	EpisodeRoleTransition EpisodeRole = "transition" // 过渡集 ×1.0
)

func (r EpisodeRole) Weight() float64 {
	switch r {
	case EpisodeRoleHook:
		return 1.5
	case EpisodeRolePaywall:
		return 1.3
	case EpisodeRoleClimax:
		return 1.2
	default:
		return 1.0
	}
}

type StoryBlueprint struct {
	ProjectID     string              `json:"project_id"`
	WorldView     string              `json:"world_view"`
	Characters    []*CharacterProfile `json:"characters"`
	Episodes      []*EpisodeBlueprint `json:"episodes"`
	Relationships []Relationship      `json:"relationships"`
}

func NewStoryBlueprint(projectID string) *StoryBlueprint {
	return &StoryBlueprint{ProjectID: projectID}
}

func (bp *StoryBlueprint) AddCharacter(name, description string, traits []string) *CharacterProfile {
	ch := &CharacterProfile{
		ID:          generateID("char"),
		Name:        name,
		Description: description,
		Traits:      traits,
	}
	bp.Characters = append(bp.Characters, ch)
	return ch
}

func (bp *StoryBlueprint) AddEpisodeBlueprintWithRole(number int, role EpisodeRole, emotionArc string) *EpisodeBlueprint {
	epBP := &EpisodeBlueprint{
		Number:     number,
		Role:       role,
		EmotionArc: emotionArc,
	}
	bp.Episodes = append(bp.Episodes, epBP)
	return epBP
}

type CharacterProfile struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Traits      []string `json:"traits"`
}

type EpisodeBlueprint struct {
	Number     int         `json:"number"`
	Role       EpisodeRole `json:"role"`
	EmotionArc string      `json:"emotion_arc"`
	Scenes     []SceneTag  `json:"scenes"`
	Synopsis   string      `json:"synopsis"`
}

type SceneTag struct {
	NarrativeBeat  string `json:"narrative_beat"`
	EmotionArc     string `json:"emotion_arc"`
	Setting        string `json:"setting"`
	Pacing         string `json:"pacing"`
	CharacterCount int    `json:"character_count"`
}

type Relationship struct {
	CharacterA string `json:"character_a"`
	CharacterB string `json:"character_b"`
	Type       string `json:"type"`
}
