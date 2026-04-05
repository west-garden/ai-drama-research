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

	if err := s.store.SavePipelineRun(ctx, &store.PipelineRunRecord{
		ProjectID: project.ID,
		Status:    "paused",
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save pipeline run")
		return
	}

	// Save initial PipelineState with script so run-phase can load it later
	state := agent.NewPipelineState(project, string(scriptBytes))
	resultJSON, err := json.Marshal(state)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to serialize initial state")
		return
	}
	if err := s.store.SavePipelineResult(ctx, project.ID, resultJSON); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save initial state")
		return
	}

	writeJSON(w, http.StatusCreated, project)
}

// determineNextPhase returns the next phase after currentPhase.
// If currentPhase is empty, it returns the first phase.
func determineNextPhase(currentPhase string) (agent.Phase, bool) {
	if currentPhase == "" {
		return agent.DefaultFlow[0], true
	}
	for i, p := range agent.DefaultFlow {
		if string(p) == currentPhase {
			if i+1 < len(agent.DefaultFlow) {
				return agent.DefaultFlow[i+1], true
			}
			return "", false // already at last phase
		}
	}
	return "", false
}

func isLastPhase(phase agent.Phase) bool {
	return phase == agent.DefaultFlow[len(agent.DefaultFlow)-1]
}

func filterShotsByEpisode(shots []*domain.ShotSpec, episode int) []*domain.ShotSpec {
	var filtered []*domain.ShotSpec
	for _, s := range shots {
		if s.EpisodeNumber == episode {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

func filterGeneratedShotsByEpisode(shots []*agent.GeneratedShot, episode int) []*agent.GeneratedShot {
	var filtered []*agent.GeneratedShot
	for _, s := range shots {
		if s.EpisodeNum == episode {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

// mergeGeneratedShots replaces shots for the given episode and keeps the rest.
func mergeGeneratedShots(existing, incoming []*agent.GeneratedShot, episode int) []*agent.GeneratedShot {
	var result []*agent.GeneratedShot
	for _, s := range existing {
		if s.EpisodeNum != episode {
			result = append(result, s)
		}
	}
	result = append(result, incoming...)
	return result
}

type runPhaseRequest struct {
	Phase   string `json:"phase"`
	Episode int    `json:"episode"`
}

func (s *Server) handleRunPhase(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ctx := r.Context()

	var req runPhaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Load pipeline run
	pipelineRun, err := s.store.GetPipelineRun(ctx, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "pipeline run not found")
		return
	}
	if pipelineRun.Status != "paused" {
		writeError(w, http.StatusConflict, fmt.Sprintf("pipeline is %s, not paused", pipelineRun.Status))
		return
	}

	// Determine which phase to run
	nextPhase, ok := determineNextPhase(pipelineRun.CurrentPhase)
	if !ok {
		writeError(w, http.StatusBadRequest, "pipeline already completed all phases")
		return
	}

	// Load project
	project, err := s.store.GetProject(ctx, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	// Load pipeline state
	resultJSON, err := s.store.GetPipelineResult(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load pipeline state")
		return
	}
	var state agent.PipelineState
	if err := json.Unmarshal(resultJSON, &state); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to parse pipeline state")
		return
	}
	state.Project = project

	// Update status to running
	s.store.UpdatePipelineRun(ctx, id, "running", pipelineRun.CurrentPhase, "")

	// Create SSE channel for live events
	run := &PipelineRun{
		ProjectID: id,
		Status:    "running",
		Phase:     nextPhase,
		Events:    make(chan SSEEvent, 20),
	}
	s.mu.Lock()
	s.runs[id] = run
	s.mu.Unlock()

	// Run single phase in background
	go s.runSinglePhase(project, &state, run, nextPhase, req.Episode)

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "running",
		"phase":  string(nextPhase),
	})
}

func (s *Server) runSinglePhase(project *domain.Project, state *agent.PipelineState, run *PipelineRun, phase agent.Phase, episode int) {
	defer close(run.Events)

	ag, ok := s.agents[phase]
	if !ok {
		run.Status = "paused"
		run.Error = fmt.Sprintf("no agent for phase %s", phase)
		run.Events <- SSEEvent{Type: "error", Message: run.Error}
		ctx := context.Background()
		s.store.UpdatePipelineRun(ctx, project.ID, "paused", string(run.Phase), run.Error)
		return
	}

	// For image/video phases with episode filter, create a scoped state
	inputState := state
	if episode > 0 && (phase == agent.PhaseImageGeneration || phase == agent.PhaseVideoGeneration) {
		scopedState := *state
		scopedState.Storyboard = filterShotsByEpisode(state.Storyboard, episode)
		if phase == agent.PhaseVideoGeneration {
			scopedState.Images = filterGeneratedShotsByEpisode(state.Images, episode)
		}
		inputState = &scopedState
	}

	result, err := ag.Run(context.Background(), inputState)

	ctx := context.Background()
	if err != nil {
		run.Status = "paused"
		run.Error = err.Error()
		run.Events <- SSEEvent{Type: "error", Message: err.Error()}
		s.store.UpdatePipelineRun(ctx, project.ID, "paused", string(run.Phase), err.Error())
		return
	}

	// Merge results back for episode-scoped runs
	if episode > 0 && phase == agent.PhaseImageGeneration {
		result.Images = mergeGeneratedShots(state.Images, result.Images, episode)
		// Preserve other fields from original state
		result.Videos = state.Videos
	} else if episode > 0 && phase == agent.PhaseVideoGeneration {
		result.Videos = mergeGeneratedShots(state.Videos, result.Videos, episode)
		result.Images = state.Images
	}

	// Persist updated state
	resultJSON, err := json.Marshal(result)
	if err == nil {
		s.store.SavePipelineResult(ctx, project.ID, resultJSON)
	}

	// Determine final status
	finalStatus := "paused"
	if isLastPhase(phase) {
		finalStatus = "completed"
		s.store.UpdateProjectStatus(ctx, project.ID, domain.StatusCompleted)
	}

	run.Status = finalStatus
	run.Events <- SSEEvent{Type: "phase_complete", Phase: string(phase)}
	s.store.UpdatePipelineRun(ctx, project.ID, finalStatus, string(phase), "")
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
			switch run.Status {
			case "running":
				p.Status = domain.StatusProcessing
			case "failed":
				p.Status = domain.StatusFailed
			case "completed":
				p.Status = domain.StatusCompleted
			case "paused":
				p.Status = "paused"
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

	// Compute next_phase
	nextPhase := ""
	if pipelineStatus == "paused" {
		if np, ok := determineNextPhase(currentPhase); ok {
			nextPhase = string(np)
		}
	}

	type projectDetail struct {
		Project        *domain.Project `json:"project"`
		PipelineStatus string          `json:"pipeline_status"`
		CurrentPhase   string          `json:"current_phase"`
		NextPhase      string          `json:"next_phase"`
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
		NextPhase:      nextPhase,
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
