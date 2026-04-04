package strategy

import (
	"sort"
	"strings"

	"github.com/west-garden/short-maker/internal/domain"
)

type ScoredStrategy struct {
	Strategy *Strategy
	Score    float64
}

func MatchScene(repo *Repository, scene domain.SceneTag, maxResults int) []ScoredStrategy {
	var scored []ScoredStrategy

	for _, s := range repo.All() {
		score := scoreStrategy(s, scene)
		if score > 0 {
			scored = append(scored, ScoredStrategy{Strategy: s, Score: score})
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	if len(scored) > maxResults {
		scored = scored[:maxResults]
	}
	return scored
}

func scoreStrategy(s *Strategy, scene domain.SceneTag) float64 {
	var score float64

	// Pacing match (exact)
	if scene.Pacing != "" {
		for _, p := range s.Tags.Pacing {
			if p == scene.Pacing {
				score += 2.0
				break
			}
		}
	}

	// NarrativeBeat match (substring)
	if scene.NarrativeBeat != "" {
		for _, nb := range s.Tags.NarrativeBeat {
			if strings.Contains(scene.NarrativeBeat, nb) {
				score += 1.5
				break
			}
		}
	}

	// EmotionArc match (substring)
	if scene.EmotionArc != "" {
		for _, ea := range s.Tags.EmotionArc {
			if strings.Contains(scene.EmotionArc, ea) {
				score += 1.0
				break
			}
		}
	}

	// CharacterCount match
	if scene.CharacterCount > 0 {
		for _, cc := range s.Tags.CharacterCount {
			if cc == scene.CharacterCount {
				score += 0.5
				break
			}
		}
	}

	return score * s.Weight
}
