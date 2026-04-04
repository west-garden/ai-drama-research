// internal/api/handler_project.go
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/west-garden/short-maker/internal/agent"
	"github.com/west-garden/short-maker/internal/domain"
	"github.com/west-garden/short-maker/internal/store"
)

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid form data")
		return
	}

	file, _, err := r.FormFile("script")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing script file")
		return
	}
	defer file.Close()

	scriptBytes, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read script")
		return
	}

	styleName := r.FormValue("style")
	if styleName == "" {
		styleName = "manga"
	}
	episodesStr := r.FormValue("episodes")
	episodes := 10
	if episodesStr != "" {
		if n, err := strconv.Atoi(episodesStr); err == nil {
			episodes = n
		}
	}

	style := domain.Style(styleName)
	project := domain.NewProject(r.FormValue("name"), style, episodes)
	if project.Name == "" {
		project.Name = "Untitled"
	}

	ctx := r.Context()
	if err := s.store.SaveProject(ctx, project); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save project")
		return
	}

	run := &PipelineRun{
		ProjectID: project.ID,
		Status:    "running",
		Events:    make(chan SSEEvent, 20),
	}
	s.mu.Lock()
	s.runs[project.ID] = run
	s.mu.Unlock()

	if err := s.store.SavePipelineRun(ctx, &store.PipelineRunRecord{
		ProjectID: project.ID,
		Status:    "running",
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save pipeline run")
		return
	}

	go s.runPipeline(project, string(scriptBytes), run)

	writeJSON(w, http.StatusCreated, project)
}

func (s *Server) runPipeline(project *domain.Project, script string, run *PipelineRun) {
	defer close(run.Events)

	state := agent.NewPipelineState(project, script)

	checkpoint := func(phase agent.Phase, st *agent.PipelineState) error {
		run.Phase = phase

		// Send SSE event
		run.Events <- SSEEvent{Type: "phase_complete", Phase: string(phase)}

		// Persist to SQLite
		ctx := context.Background()
		s.store.UpdatePipelineRun(ctx, project.ID, "running", string(phase), "")

		resultJSON, err := json.Marshal(st)
		if err == nil {
			s.store.SavePipelineResult(ctx, project.ID, resultJSON)
		}
		return nil
	}

	orch := agent.NewOrchestrator(s.agents, checkpoint)
	result, err := orch.Run(context.Background(), state)

	ctx := context.Background()
	if err != nil {
		run.Status = "failed"
		run.Error = err.Error()
		run.Events <- SSEEvent{Type: "error", Message: err.Error()}
		s.store.UpdatePipelineRun(ctx, project.ID, "failed", string(run.Phase), err.Error())
		s.store.UpdateProjectStatus(ctx, project.ID, domain.StatusFailed)
		return
	}

	run.Status = "completed"
	run.Events <- SSEEvent{Type: "done"}

	resultJSON, _ := json.Marshal(result)
	s.store.SavePipelineResult(ctx, project.ID, resultJSON)
	s.store.UpdatePipelineRun(ctx, project.ID, "completed", "video_generation", "")
	s.store.UpdateProjectStatus(ctx, project.ID, domain.StatusCompleted)
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := s.store.ListProjects(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list projects")
		return
	}
	if projects == nil {
		projects = []*domain.Project{}
	}

	type projectSummary struct {
		ID           string        `json:"id"`
		Name         string        `json:"name"`
		Style        domain.Style  `json:"style"`
		EpisodeCount int           `json:"episode_count"`
		Status       domain.Status `json:"status"`
		CurrentPhase string        `json:"current_phase"`
		CreatedAt    string        `json:"created_at"`
	}

	summaries := make([]projectSummary, 0, len(projects))
	for _, p := range projects {
		phase := ""
		if run, err := s.store.GetPipelineRun(r.Context(), p.ID); err == nil {
			phase = run.CurrentPhase
			if run.Status == "running" {
				p.Status = domain.StatusProcessing
			} else if run.Status == "failed" {
				p.Status = domain.StatusFailed
			} else if run.Status == "completed" {
				p.Status = domain.StatusCompleted
			}
		}
		summaries = append(summaries, projectSummary{
			ID:           p.ID,
			Name:         p.Name,
			Style:        p.Style,
			EpisodeCount: p.EpisodeCount,
			Status:       p.Status,
			CurrentPhase: phase,
			CreatedAt:    p.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	writeJSON(w, http.StatusOK, summaries)
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	project, err := s.store.GetProject(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project not found: %s", id))
		return
	}

	pipelineStatus := "unknown"
	currentPhase := ""
	if run, err := s.store.GetPipelineRun(r.Context(), id); err == nil {
		pipelineStatus = run.Status
		currentPhase = run.CurrentPhase
	}

	type projectDetail struct {
		Project        *domain.Project `json:"project"`
		PipelineStatus string          `json:"pipeline_status"`
		CurrentPhase   string          `json:"current_phase"`
		Blueprint      json.RawMessage `json:"blueprint,omitempty"`
		Storyboard     json.RawMessage `json:"storyboard,omitempty"`
		Images         json.RawMessage `json:"images,omitempty"`
		Videos         json.RawMessage `json:"videos,omitempty"`
		Errors         json.RawMessage `json:"errors,omitempty"`
	}

	detail := projectDetail{
		Project:        project,
		PipelineStatus: pipelineStatus,
		CurrentPhase:   currentPhase,
	}

	resultJSON, err := s.store.GetPipelineResult(r.Context(), id)
	if err == nil {
		var raw map[string]json.RawMessage
		if json.Unmarshal(resultJSON, &raw) == nil {
			detail.Blueprint = raw["blueprint"]
			detail.Storyboard = raw["storyboard"]
			detail.Images = raw["images"]
			detail.Videos = raw["videos"]
			detail.Errors = raw["errors"]
		}
	}

	writeJSON(w, http.StatusOK, detail)
}
