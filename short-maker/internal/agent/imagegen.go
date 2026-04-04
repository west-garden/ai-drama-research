package agent

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/west-garden/short-maker/internal/domain"
	"github.com/west-garden/short-maker/internal/quality"
	"github.com/west-garden/short-maker/internal/router"
)

type ImageGenAgent struct {
	router    *router.ModelRouter
	checker   quality.Checker
	outputDir string
}

func NewImageGenAgent(r *router.ModelRouter, checker quality.Checker, outputDir string) *ImageGenAgent {
	return &ImageGenAgent{router: r, checker: checker, outputDir: outputDir}
}

func (a *ImageGenAgent) Phase() Phase { return PhaseImageGeneration }

func (a *ImageGenAgent) Run(ctx context.Context, state *PipelineState) (*PipelineState, error) {
	if len(state.Storyboard) == 0 {
		return nil, fmt.Errorf("image gen agent requires a non-empty Storyboard")
	}

	for _, shot := range state.Storyboard {
		episodeRole := findEpisodeRole(state.Blueprint, shot.EpisodeNumber)
		importance := domain.NewImportanceScore(episodeRole, shot.RhythmPosition, shot.ContentType)
		grade := importance.Grade()
		maxRetries := importance.MaxRetries()

		charAssets := findCharacterAssets(state.Assets, shot.CharacterRefs)
		outPath := shotImagePath(a.outputDir, state.Project.ID, shot.EpisodeNumber, shot.ShotNumber)

		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return nil, fmt.Errorf("create output dir: %w", err)
		}

		var charRefPaths []string
		for _, ca := range charAssets {
			if ca.FilePath != "" {
				charRefPaths = append(charRefPaths, ca.FilePath)
			}
		}

		req := router.GenerateRequest{
			Prompt:        shot.Prompt,
			Style:         string(state.Project.Style),
			CharacterRefs: charRefPaths,
			OutputPath:    outPath,
		}

		var lastReport *quality.QualityReport
		for attempt := 0; attempt <= maxRetries; attempt++ {
			resp, err := a.router.Generate(ctx, grade, string(state.Project.Style), router.ModelTypeImage, req)
			if err != nil {
				if attempt < maxRetries {
					log.Printf("[image-gen] shot %d generate failed, retrying (%d/%d): %v", shot.ShotNumber, attempt+1, maxRetries, err)
					continue
				}
				return nil, fmt.Errorf("generate image for shot %d: %w", shot.ShotNumber, err)
			}

			report, err := a.checker.Check(ctx, resp.FilePath, shot, charAssets)
			if err != nil {
				return nil, fmt.Errorf("quality check shot %d: %w", shot.ShotNumber, err)
			}
			lastReport = report

			if report.Passed {
				log.Printf("[image-gen] shot %d passed quality check (score: %d, grade: %s)", shot.ShotNumber, report.TotalScore, grade)
				break
			}

			if attempt < maxRetries {
				log.Printf("[image-gen] shot %d failed quality check (score: %d, threshold: %d), retrying (%d/%d)",
					shot.ShotNumber, report.TotalScore, grade.QualityThreshold(), attempt+1, maxRetries)
			} else {
				log.Printf("[image-gen] shot %d failed quality check after %d attempts (score: %d)", shot.ShotNumber, maxRetries+1, report.TotalScore)
			}
		}

		score := 0
		if lastReport != nil {
			score = lastReport.TotalScore
		}

		state.Images = append(state.Images, &GeneratedShot{
			ShotNumber: shot.ShotNumber,
			EpisodeNum: shot.EpisodeNumber,
			ImagePath:  outPath,
			Grade:      grade,
			ImageScore: score,
		})
	}

	log.Printf("[image-gen] generated %d images", len(state.Images))
	return state, nil
}

func findEpisodeRole(bp *domain.StoryBlueprint, episodeNumber int) domain.EpisodeRole {
	if bp == nil {
		return domain.EpisodeRoleTransition
	}
	for _, ep := range bp.Episodes {
		if ep.Number == episodeNumber {
			return ep.Role
		}
	}
	return domain.EpisodeRoleTransition
}

func findCharacterAssets(assets []*domain.Asset, characterRefs []string) []*domain.Asset {
	if len(characterRefs) == 0 {
		return nil
	}
	refSet := make(map[string]bool, len(characterRefs))
	for _, ref := range characterRefs {
		refSet[ref] = true
	}
	var result []*domain.Asset
	for _, a := range assets {
		if refSet[a.Metadata["character_id"]] {
			result = append(result, a)
		}
	}
	return result
}

func shotImagePath(outputDir, projectID string, episodeNum, shotNum int) string {
	return filepath.Join(outputDir, projectID, fmt.Sprintf("ep%02d", episodeNum), fmt.Sprintf("shot%03d.png", shotNum))
}
