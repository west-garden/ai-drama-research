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

type VideoGenAgent struct {
	router    *router.ModelRouter
	checker   quality.Checker
	outputDir string
}

func NewVideoGenAgent(r *router.ModelRouter, checker quality.Checker, outputDir string) *VideoGenAgent {
	return &VideoGenAgent{router: r, checker: checker, outputDir: outputDir}
}

func (a *VideoGenAgent) Phase() Phase { return PhaseVideoGeneration }

func (a *VideoGenAgent) Run(ctx context.Context, state *PipelineState) (*PipelineState, error) {
	if len(state.Images) == 0 {
		return nil, fmt.Errorf("video gen agent requires non-empty Images")
	}

	shotSpecByNum := make(map[int]*domain.ShotSpec, len(state.Storyboard))
	for _, shot := range state.Storyboard {
		shotSpecByNum[shot.ShotNumber] = shot
	}

	for _, img := range state.Images {
		shotSpec := shotSpecByNum[img.ShotNumber]
		if shotSpec == nil {
			log.Printf("[video-gen] warning: no ShotSpec for shot %d, skipping", img.ShotNumber)
			continue
		}

		episodeRole := findEpisodeRole(state.Blueprint, img.EpisodeNum)
		importance := domain.NewImportanceScore(episodeRole, shotSpec.RhythmPosition, shotSpec.ContentType)
		grade := importance.Grade()
		maxRetries := importance.MaxRetries()
		charAssets := findCharacterAssets(state.Assets, shotSpec.CharacterRefs)

		outPath := shotVideoPath(a.outputDir, state.Project.ID, img.EpisodeNum, img.ShotNumber)
		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return nil, fmt.Errorf("create output dir: %w", err)
		}

		srcImagePath := shotImagePath(a.outputDir, state.Project.ID, img.EpisodeNum, img.ShotNumber)
		req := router.GenerateRequest{
			Prompt:      shotSpec.Prompt,
			Style:       string(state.Project.Style),
			CameraMove:  shotSpec.CameraMove,
			SourceImage: srcImagePath,
			OutputPath:  outPath,
		}

		var lastReport *quality.QualityReport
		for attempt := 0; attempt <= maxRetries; attempt++ {
			resp, err := a.router.Generate(ctx, grade, string(state.Project.Style), router.ModelTypeVideo, req)
			if err != nil {
				if attempt < maxRetries {
					log.Printf("[video-gen] shot %d generate failed, retrying (%d/%d): %v", img.ShotNumber, attempt+1, maxRetries, err)
					continue
				}
				return nil, fmt.Errorf("generate video for shot %d: %w", img.ShotNumber, err)
			}

			report, err := a.checker.Check(ctx, resp.FilePath, shotSpec, charAssets)
			if err != nil {
				return nil, fmt.Errorf("quality check video shot %d: %w", img.ShotNumber, err)
			}
			lastReport = report

			if report.Passed {
				log.Printf("[video-gen] shot %d passed quality check (score: %d, grade: %s)", img.ShotNumber, report.TotalScore, grade)
				break
			}

			if attempt < maxRetries {
				log.Printf("[video-gen] shot %d failed quality check (score: %d), retrying (%d/%d)",
					img.ShotNumber, report.TotalScore, attempt+1, maxRetries)
			}
		}

		score := 0
		if lastReport != nil {
			score = lastReport.TotalScore
		}

		state.Videos = append(state.Videos, &GeneratedShot{
			ShotNumber: img.ShotNumber,
			EpisodeNum: img.EpisodeNum,
			ImagePath:  img.ImagePath,
			VideoPath:  shotVideoURL(state.Project.ID, img.EpisodeNum, img.ShotNumber),
			Grade:      grade,
			ImageScore: img.ImageScore,
			VideoScore: score,
		})
	}

	log.Printf("[video-gen] generated %d videos", len(state.Videos))
	return state, nil
}

func shotVideoPath(outputDir, projectID string, episodeNum, shotNum int) string {
	return filepath.Join(outputDir, projectID, fmt.Sprintf("ep%02d", episodeNum), fmt.Sprintf("shot%03d.mp4", shotNum))
}

// shotVideoURL returns the URL path for serving a video via the /output/ file server.
func shotVideoURL(projectID string, episodeNum, shotNum int) string {
	return fmt.Sprintf("/output/%s/ep%02d/shot%03d.mp4", projectID, episodeNum, shotNum)
}
