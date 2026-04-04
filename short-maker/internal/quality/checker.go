// internal/quality/checker.go
package quality

import (
	"context"
	"math"

	"github.com/west-garden/short-maker/internal/domain"
)

type Dimension struct {
	Name   string  `json:"name"`
	Weight float64 `json:"weight"`
	Score  int     `json:"score"`
	Notes  string  `json:"notes"`
}

type QualityReport struct {
	ShotNumber  int         `json:"shot_number"`
	Dimensions  []Dimension `json:"dimensions"`
	TotalScore  int         `json:"total_score"`
	Passed      bool        `json:"passed"`
	Suggestions []string    `json:"suggestions"`
}

type Checker interface {
	Check(ctx context.Context, filePath string, shotSpec *domain.ShotSpec, characterAssets []*domain.Asset) (*QualityReport, error)
}

func DefaultDimensions(score int) []Dimension {
	return []Dimension{
		{Name: "character_consistency", Weight: 0.30, Score: score},
		{Name: "image_quality", Weight: 0.25, Score: score},
		{Name: "storyboard_fidelity", Weight: 0.20, Score: score},
		{Name: "style_consistency", Weight: 0.15, Score: score},
		{Name: "narrative_accuracy", Weight: 0.10, Score: score},
	}
}

func NewReport(shotNumber int, dimensions []Dimension, grade domain.Grade) *QualityReport {
	var weightedSum float64
	for _, d := range dimensions {
		weightedSum += float64(d.Score) * d.Weight
	}
	totalScore := int(math.Round(weightedSum))
	threshold := grade.QualityThreshold()

	return &QualityReport{
		ShotNumber: shotNumber,
		Dimensions: dimensions,
		TotalScore: totalScore,
		Passed:     totalScore >= threshold,
	}
}
