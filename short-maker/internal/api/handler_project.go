// internal/api/handler_project.go
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

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

	promptLanguage := r.FormValue("prompt_language")
	if promptLanguage == "" {
		promptLanguage = "zh" // default to Chinese
	}

	style := domain.Style(styleName)
	project := domain.NewProject(r.FormValue("name"), style, episodes)
	project.PromptLanguage = promptLanguage
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

// --- Phase helpers ---

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

// mergeStoryboardShots replaces storyboard shots for the given episode and keeps the rest.
func mergeStoryboardShots(existing, incoming []*domain.ShotSpec, episode int) []*domain.ShotSpec {
	var result []*domain.ShotSpec
	for _, s := range existing {
		if s.EpisodeNumber != episode {
			result = append(result, s)
		}
	}
	result = append(result, incoming...)
	return result
}

// filterBlueprintEpisodes returns a copy of the blueprint with only the target episode.
func filterBlueprintEpisodes(bp *domain.StoryBlueprint, episode int) *domain.StoryBlueprint {
	filtered := *bp
	filtered.Episodes = nil
	for _, ep := range bp.Episodes {
		if ep.Number == episode {
			filtered.Episodes = append(filtered.Episodes, ep)
		}
	}
	return &filtered
}

// --- Prerequisite checking ---

// parseNodeKey splits "phase:epN" into phase and episode number.
func parseNodeKey(nodeKey string) (agent.Phase, int) {
	if idx := strings.Index(nodeKey, ":ep"); idx > 0 {
		phase := agent.Phase(nodeKey[:idx])
		if ep, err := strconv.Atoi(nodeKey[idx+3:]); err == nil {
			return phase, ep
		}
	}
	return agent.Phase(nodeKey), 0
}

// checkPrerequisites returns an error if prerequisites for running this node are not met.
func checkPrerequisites(state *agent.PipelineState, phase agent.Phase, episode int) error {
	switch phase {
	case agent.PhaseStoryUnderstanding:
		// No prerequisites
		return nil
	case agent.PhaseCharacterAsset:
		if state.GetNodeStatus("story_understanding") != domain.NodeStatusCompleted {
			return fmt.Errorf("prerequisite not met: story_understanding must be completed")
		}
	case agent.PhaseStoryboard:
		if state.GetNodeStatus("character_asset") != domain.NodeStatusCompleted {
			return fmt.Errorf("prerequisite not met: character_asset must be completed")
		}
	case agent.PhaseImageGeneration:
		sbKey := agent.NodeKey(agent.PhaseStoryboard, episode)
		if state.GetNodeStatus(sbKey) != domain.NodeStatusCompleted {
			return fmt.Errorf("prerequisite not met: %s must be completed", sbKey)
		}
	case agent.PhaseVideoGeneration:
		imgKey := agent.NodeKey(agent.PhaseImageGeneration, episode)
		if state.GetNodeStatus(imgKey) != domain.NodeStatusCompleted {
			return fmt.Errorf("prerequisite not met: %s must be completed", imgKey)
		}
	}
	return nil
}

// invalidateDownstream resets downstream node statuses and clears their data.
func invalidateDownstream(state *agent.PipelineState, phase agent.Phase, episode int) {
	switch phase {
	case agent.PhaseStoryUnderstanding:
		// Clear all downstream
		state.Blueprint = nil
		state.Assets = nil
		state.Storyboard = nil
		state.Images = nil
		state.Videos = nil
		// Reset all nodes to pending
		for key := range state.NodeStatuses {
			if key != "story_understanding" {
				state.SetNodeStatus(key, domain.NodeStatusPending, "")
			}
		}
	case agent.PhaseCharacterAsset:
		state.Assets = nil
		// Reset all storyboard, image, video nodes
		for key := range state.NodeStatuses {
			p, _ := parseNodeKey(key)
			if p == agent.PhaseStoryboard || p == agent.PhaseImageGeneration || p == agent.PhaseVideoGeneration {
				state.SetNodeStatus(key, domain.NodeStatusPending, "")
			}
		}
	case agent.PhaseStoryboard:
		if episode > 0 {
			state.Storyboard = filterOutEpisodeShots(state.Storyboard, episode)
			state.Images = filterOutEpisodeGenerated(state.Images, episode)
			state.Videos = filterOutEpisodeGenerated(state.Videos, episode)
			state.SetNodeStatus(agent.NodeKey(agent.PhaseImageGeneration, episode), domain.NodeStatusPending, "")
			state.SetNodeStatus(agent.NodeKey(agent.PhaseVideoGeneration, episode), domain.NodeStatusPending, "")
		}
	case agent.PhaseImageGeneration:
		if episode > 0 {
			state.Images = filterOutEpisodeGenerated(state.Images, episode)
			state.Videos = filterOutEpisodeGenerated(state.Videos, episode)
			state.SetNodeStatus(agent.NodeKey(agent.PhaseVideoGeneration, episode), domain.NodeStatusPending, "")
		}
	case agent.PhaseVideoGeneration:
		if episode > 0 {
			state.Videos = filterOutEpisodeGenerated(state.Videos, episode)
		}
	}
}

func filterOutEpisodeShots(shots []*domain.ShotSpec, episode int) []*domain.ShotSpec {
	var result []*domain.ShotSpec
	for _, s := range shots {
		if s.EpisodeNumber != episode {
			result = append(result, s)
		}
	}
	return result
}

func filterOutEpisodeGenerated(shots []*agent.GeneratedShot, episode int) []*agent.GeneratedShot {
	var result []*agent.GeneratedShot
	for _, s := range shots {
		if s.EpisodeNum != episode {
			result = append(result, s)
		}
	}
	return result
}

// --- Run phase handler ---

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
	if pipelineRun.Status == "running" {
		writeError(w, http.StatusConflict, "pipeline is already running")
		return
	}

	// Determine phase to run
	var phase agent.Phase
	episode := req.Episode

	if req.Phase != "" {
		// Explicit phase+episode from workflow UI
		phase, episode = parseNodeKey(req.Phase)
		if episode == 0 {
			episode = req.Episode
		}
	} else {
		// Legacy: auto-determine next phase
		nextPhase, ok := determineNextPhase(pipelineRun.CurrentPhase)
		if !ok {
			writeError(w, http.StatusBadRequest, "pipeline already completed all phases")
			return
		}
		phase = nextPhase
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

	// Check prerequisites
	if err := checkPrerequisites(&state, phase, episode); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Invalidate downstream data when re-running a node
	nodeKey := agent.NodeKey(phase, episode)
	currentStatus := state.GetNodeStatus(nodeKey)
	if currentStatus == domain.NodeStatusCompleted || currentStatus == domain.NodeStatusFailed {
		invalidateDownstream(&state, phase, episode)
	}

	// Clear previous errors for this node
	state.Errors = nil

	// Mark node as running
	state.SetNodeStatus(nodeKey, domain.NodeStatusRunning, "")

	// Persist pre-run state
	preRunJSON, _ := json.Marshal(&state)
	s.store.SavePipelineResult(ctx, id, preRunJSON)

	// Update pipeline run status
	s.store.UpdatePipelineRun(ctx, id, "running", pipelineRun.CurrentPhase, "")

	// Create SSE channel for live events
	run := &PipelineRun{
		ProjectID: id,
		Status:    "running",
		Phase:     phase,
		Events:    make(chan SSEEvent, 20),
	}
	s.mu.Lock()
	s.runs[id] = run
	s.mu.Unlock()

	// Run single phase in background
	go s.runSinglePhase(project, &state, run, phase, pipelineRun.CurrentPhase, episode)

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "running",
		"phase":  string(phase),
		"node":   nodeKey,
	})
}

func (s *Server) runSinglePhase(project *domain.Project, state *agent.PipelineState, run *PipelineRun, phase agent.Phase, prevPhase string, episode int) {
	defer close(run.Events)

	nodeKey := agent.NodeKey(phase, episode)

	ag, ok := s.agents[phase]
	if !ok {
		run.Status = "paused"
		run.Error = fmt.Sprintf("no agent for phase %s", phase)
		state.SetNodeStatus(nodeKey, domain.NodeStatusFailed, run.Error)
		run.Events <- SSEEvent{Type: "error", Node: nodeKey, Message: run.Error}
		ctx := context.Background()
		s.store.UpdatePipelineRun(ctx, project.ID, "paused", prevPhase, run.Error)
		s.persistState(ctx, project.ID, state)
		return
	}

	// Send running event
	run.Events <- SSEEvent{Type: "phase_start", Phase: string(phase), Node: nodeKey}

	// For episode-scoped phases, create a scoped state
	inputState := state
	if episode > 0 {
		switch phase {
		case agent.PhaseStoryboard:
			scopedState := *state
			scopedState.Blueprint = filterBlueprintEpisodes(state.Blueprint, episode)
			inputState = &scopedState
		case agent.PhaseImageGeneration:
			scopedState := *state
			scopedState.Storyboard = filterShotsByEpisode(state.Storyboard, episode)
			inputState = &scopedState
		case agent.PhaseVideoGeneration:
			scopedState := *state
			scopedState.Storyboard = filterShotsByEpisode(state.Storyboard, episode)
			scopedState.Images = filterGeneratedShotsByEpisode(state.Images, episode)
			inputState = &scopedState
		}
	}

	result, err := ag.Run(context.Background(), inputState)

	ctx := context.Background()
	if err != nil {
		run.Status = "paused"
		run.Error = err.Error()
		state.SetNodeStatus(nodeKey, domain.NodeStatusFailed, err.Error())
		run.Events <- SSEEvent{Type: "error", Node: nodeKey, Message: err.Error()}
		s.store.UpdatePipelineRun(ctx, project.ID, "paused", prevPhase, err.Error())
		s.persistState(ctx, project.ID, state)
		return
	}

	// Merge results back for episode-scoped runs
	if episode > 0 {
		switch phase {
		case agent.PhaseStoryboard:
			result.Storyboard = mergeStoryboardShots(state.Storyboard, result.Storyboard, episode)
			result.Images = state.Images
			result.Videos = state.Videos
		case agent.PhaseImageGeneration:
			result.Images = mergeGeneratedShots(state.Images, result.Images, episode)
			result.Videos = state.Videos
		case agent.PhaseVideoGeneration:
			result.Videos = mergeGeneratedShots(state.Videos, result.Videos, episode)
			result.Images = state.Images
		}
	}

	// Preserve node statuses from state (result may not have them from agent)
	if result.NodeStatuses == nil {
		result.NodeStatuses = state.NodeStatuses
	} else {
		for k, v := range state.NodeStatuses {
			if _, exists := result.NodeStatuses[k]; !exists {
				result.NodeStatuses[k] = v
			}
		}
	}

	// Determine node status based on errors
	if len(result.Errors) > 0 {
		result.SetNodeStatus(nodeKey, domain.NodeStatusCompleted, strings.Join(result.Errors, "; "))
	} else {
		result.SetNodeStatus(nodeKey, domain.NodeStatusCompleted, "")
	}

	// Update current_phase in pipeline_runs to track latest completed phase
	newCurrentPhase := string(phase)
	// For project-level phases, update directly. For episode phases, keep the phase name.
	if phaseIndex(phase) > phaseIndex(agent.Phase(prevPhase)) {
		newCurrentPhase = string(phase)
	} else {
		newCurrentPhase = prevPhase
		if prevPhase == "" {
			newCurrentPhase = string(phase)
		}
	}

	// Persist updated state
	s.persistState(ctx, project.ID, result)

	run.Status = "paused"
	run.Events <- SSEEvent{Type: "phase_complete", Phase: string(phase), Node: nodeKey}
	s.store.UpdatePipelineRun(ctx, project.ID, "paused", newCurrentPhase, "")
}

func phaseIndex(phase agent.Phase) int {
	for i, p := range agent.DefaultFlow {
		if p == phase {
			return i
		}
	}
	return -1
}

func (s *Server) persistState(ctx context.Context, projectID string, state *agent.PipelineState) {
	resultJSON, err := json.Marshal(state)
	if err == nil {
		s.store.SavePipelineResult(ctx, projectID, resultJSON)
	}
}

// --- Workflow endpoint ---

type workflowNode struct {
	ID      string `json:"id"`
	Phase   string `json:"phase"`
	Episode int    `json:"episode"`
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
	Label   string `json:"label"`
	CanRun  bool   `json:"can_run"`
}

type workflowEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

type workflowResponse struct {
	Nodes []workflowNode `json:"nodes"`
	Edges []workflowEdge `json:"edges"`
}

var phaseLabels = map[agent.Phase]string{
	agent.PhaseStoryUnderstanding: "剧本理解",
	agent.PhaseCharacterAsset:    "角色资产",
	agent.PhaseStoryboard:        "分镜",
	agent.PhaseImageGeneration:   "图片生成",
	agent.PhaseVideoGeneration:   "视频生成",
}

func (s *Server) handleGetWorkflow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	project, err := s.store.GetProject(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project not found: %s", id))
		return
	}

	// Load pipeline state for node statuses
	var state agent.PipelineState
	resultJSON, err := s.store.GetPipelineResult(r.Context(), id)
	if err == nil {
		json.Unmarshal(resultJSON, &state)
	}
	state.Project = project

	pipelineStatus := "paused"
	if run, err := s.store.GetPipelineRun(r.Context(), id); err == nil {
		pipelineStatus = run.Status
	}

	isRunning := pipelineStatus == "running"
	episodeCount := project.EpisodeCount

	resp := workflowResponse{
		Nodes: []workflowNode{},
		Edges: []workflowEdge{},
	}

	// Project-level nodes
	projectPhases := []agent.Phase{agent.PhaseStoryUnderstanding, agent.PhaseCharacterAsset}
	for _, phase := range projectPhases {
		nodeKey := agent.NodeKey(phase, 0)
		status := string(state.GetNodeStatus(nodeKey))
		canRun := !isRunning && checkPrerequisites(&state, phase, 0) == nil
		resp.Nodes = append(resp.Nodes, workflowNode{
			ID:     nodeKey,
			Phase:  string(phase),
			Status: status,
			Label:  phaseLabels[phase],
			CanRun: canRun,
		})
	}

	// Project config -> story_understanding
	resp.Edges = append(resp.Edges, workflowEdge{
		Source: "project_config",
		Target: "story_understanding",
	})
	resp.Edges = append(resp.Edges, workflowEdge{
		Source: "story_understanding",
		Target: "character_asset",
	})

	// Episode-level nodes
	episodePhases := []agent.Phase{agent.PhaseStoryboard, agent.PhaseImageGeneration, agent.PhaseVideoGeneration}
	for ep := 1; ep <= episodeCount; ep++ {
		// character_asset -> storyboard:epN
		resp.Edges = append(resp.Edges, workflowEdge{
			Source: "character_asset",
			Target: agent.NodeKey(agent.PhaseStoryboard, ep),
		})

		for i, phase := range episodePhases {
			nodeKey := agent.NodeKey(phase, ep)
			status := string(state.GetNodeStatus(nodeKey))
			errMsg := ""
			if entry, ok := state.NodeStatuses[nodeKey]; ok {
				errMsg = entry.Error
			}
			canRun := !isRunning && checkPrerequisites(&state, phase, ep) == nil
			label := fmt.Sprintf("EP%d %s", ep, phaseLabels[phase])

			resp.Nodes = append(resp.Nodes, workflowNode{
				ID:      nodeKey,
				Phase:   string(phase),
				Episode: ep,
				Status:  status,
				Error:   errMsg,
				Label:   label,
				CanRun:  canRun,
			})

			// Edges between episode phases
			if i > 0 {
				prevKey := agent.NodeKey(episodePhases[i-1], ep)
				resp.Edges = append(resp.Edges, workflowEdge{
					Source: prevKey,
					Target: nodeKey,
				})
			}
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// --- List / Get project ---

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

	// Compute next_phase (legacy support)
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
		Assets         json.RawMessage `json:"assets,omitempty"`
		Storyboard     json.RawMessage `json:"storyboard,omitempty"`
		Images         json.RawMessage `json:"images,omitempty"`
		Videos         json.RawMessage `json:"videos,omitempty"`
		Errors         json.RawMessage `json:"errors,omitempty"`
		NodeStatuses   json.RawMessage `json:"node_statuses,omitempty"`
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
			detail.Assets = raw["assets"]
			detail.Storyboard = raw["storyboard"]
			detail.Images = raw["images"]
			detail.Videos = raw["videos"]
			detail.Errors = raw["errors"]
			detail.NodeStatuses = raw["node_statuses"]
		}
	}

	writeJSON(w, http.StatusOK, detail)
}
