// internal/domain/project.go
package domain

import (
	"fmt"
	"time"
)

type Style string

const (
	StyleManga      Style = "manga"
	Style3D         Style = "3d"
	StyleLiveAction Style = "live_action"
)

type Status string

const (
	StatusCreated    Status = "created"
	StatusProcessing Status = "processing"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
)

// NodeStatus represents the execution status of a workflow node.
type NodeStatus string

const (
	NodeStatusPending   NodeStatus = "pending"
	NodeStatusRunning   NodeStatus = "running"
	NodeStatusCompleted NodeStatus = "completed"
	NodeStatusFailed    NodeStatus = "failed"
	NodeStatusSkipped   NodeStatus = "skipped"
)

type Project struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	Style          Style      `json:"style"`
	EpisodeCount   int        `json:"episode_count"`
	PromptLanguage string     `json:"prompt_language"` // "en" | "zh", default "en"
	Status         Status     `json:"status"`
	Episodes       []*Episode `json:"episodes"`
	CreatedAt      time.Time  `json:"created_at"`
}

func NewProject(name string, style Style, episodeCount int) *Project {
	return &Project{
		ID:             generateID("proj"),
		Name:           name,
		Style:          style,
		EpisodeCount:   episodeCount,
		PromptLanguage: "en",
		Status:         StatusCreated,
		CreatedAt:      time.Now(),
	}
}

func (p *Project) AddEpisode(number int) *Episode {
	ep := &Episode{
		ID:        generateID("ep"),
		ProjectID: p.ID,
		Number:    number,
		Status:    StatusCreated,
	}
	p.Episodes = append(p.Episodes, ep)
	return ep
}

type Episode struct {
	ID        string  `json:"id"`
	ProjectID string  `json:"project_id"`
	Number    int     `json:"number"`
	Status    Status  `json:"status"`
	Shots     []*Shot `json:"shots"`
}

func (e *Episode) AddShot() *Shot {
	shot := &Shot{
		ID:        generateID("shot"),
		EpisodeID: e.ID,
		Number:    len(e.Shots) + 1,
		Status:    StatusCreated,
	}
	e.Shots = append(e.Shots, shot)
	return shot
}

type Shot struct {
	ID        string `json:"id"`
	EpisodeID string `json:"episode_id"`
	Number    int    `json:"number"`
	Status    Status `json:"status"`
	Prompt    string `json:"prompt"`
	ImagePath string `json:"image_path"`
	VideoPath string `json:"video_path"`
}

var idCounter int

func generateID(prefix string) string {
	idCounter++
	return fmt.Sprintf("%s_%d_%d", prefix, time.Now().UnixMilli(), idCounter)
}
