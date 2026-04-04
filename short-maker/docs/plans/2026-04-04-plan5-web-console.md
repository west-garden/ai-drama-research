# Plan 5: Web Console Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a Web console (Go API + React frontend) to Short Maker so users can upload scripts, run the pipeline, watch progress via SSE, and browse generated images/videos in the browser.

**Architecture:** Go API layer (`internal/api/`) using chi router serves REST endpoints and SSE. Pipeline runs asynchronously in a background goroutine, with the Orchestrator's checkpoint hook pushing events through a channel to SSE connections. SQLite stores pipeline run state and results for restart persistence. React + Vite + TailwindCSS frontend in `web/` with 3 pages (list, create, detail).

**Tech Stack:** Go (chi, cors), React 18, Vite, TailwindCSS, TypeScript, SQLite

**Spec reference:** `short-maker/docs/specs/2026-04-04-web-console-design.md`

---

## File Structure

```
short-maker/
├── internal/
│   ├── api/
│   │   ├── server.go              # Server struct, NewServer, route setup, Start
│   │   ├── server_test.go         # API handler tests
│   │   ├── handler_project.go     # POST/GET /api/projects, GET /api/projects/{id}
│   │   ├── handler_sse.go         # GET /api/projects/{id}/events — SSE
│   │   └── middleware.go          # CORS middleware
│   └── store/
│       ├── store.go               # (modify) add PipelineRunStore + ListProjects to Store
│       ├── sqlite.go              # (modify) add pipeline tables + ListProjects impl
│       └── pipeline_test.go       # Tests for pipeline run/result persistence
├── cmd/shortmaker/
│   └── main.go                    # (modify) add serve subcommand
└── web/
    ├── package.json
    ├── vite.config.ts
    ├── tsconfig.json
    ├── index.html
    ├── postcss.config.js
    ├── tailwind.config.js
    └── src/
        ├── main.tsx
        ├── App.tsx
        ├── api.ts
        ├── pages/
        │   ├── ProjectList.tsx
        │   ├── NewProject.tsx
        │   └── ProjectDetail.tsx
        └── components/
            ├── Layout.tsx
            ├── PipelineProgress.tsx
            └── ShotGallery.tsx
```

---

### Task 1: SQLite Store — PipelineRunStore

**Files:**
- Modify: `short-maker/internal/store/store.go`
- Modify: `short-maker/internal/store/sqlite.go`
- Create: `short-maker/internal/store/pipeline_test.go`

- [ ] **Step 1: Write tests for pipeline run persistence**

```go
// internal/store/pipeline_test.go
package store

import (
	"context"
	"testing"
)

func TestSavePipelineRun(t *testing.T) {
	s := setupTestDB(t)
	ctx := context.Background()

	run := &PipelineRunRecord{
		ProjectID:    "proj_test_1",
		Status:       "running",
		CurrentPhase: "story_understanding",
	}
	if err := s.SavePipelineRun(ctx, run); err != nil {
		t.Fatalf("SavePipelineRun: %v", err)
	}

	got, err := s.GetPipelineRun(ctx, "proj_test_1")
	if err != nil {
		t.Fatalf("GetPipelineRun: %v", err)
	}
	if got.Status != "running" {
		t.Errorf("expected status 'running', got '%s'", got.Status)
	}
	if got.CurrentPhase != "story_understanding" {
		t.Errorf("expected phase 'story_understanding', got '%s'", got.CurrentPhase)
	}
}

func TestUpdatePipelineRun(t *testing.T) {
	s := setupTestDB(t)
	ctx := context.Background()

	run := &PipelineRunRecord{
		ProjectID:    "proj_test_2",
		Status:       "running",
		CurrentPhase: "story_understanding",
	}
	s.SavePipelineRun(ctx, run)

	if err := s.UpdatePipelineRun(ctx, "proj_test_2", "completed", "video_generation", ""); err != nil {
		t.Fatalf("UpdatePipelineRun: %v", err)
	}

	got, _ := s.GetPipelineRun(ctx, "proj_test_2")
	if got.Status != "completed" {
		t.Errorf("expected status 'completed', got '%s'", got.Status)
	}
	if got.CurrentPhase != "video_generation" {
		t.Errorf("expected phase 'video_generation', got '%s'", got.CurrentPhase)
	}
}

func TestSaveAndGetPipelineResult(t *testing.T) {
	s := setupTestDB(t)
	ctx := context.Background()

	resultJSON := []byte(`{"project":{"id":"proj_test_3"},"storyboard":[]}`)
	if err := s.SavePipelineResult(ctx, "proj_test_3", resultJSON); err != nil {
		t.Fatalf("SavePipelineResult: %v", err)
	}

	got, err := s.GetPipelineResult(ctx, "proj_test_3")
	if err != nil {
		t.Fatalf("GetPipelineResult: %v", err)
	}
	if string(got) != string(resultJSON) {
		t.Errorf("expected '%s', got '%s'", resultJSON, got)
	}
}

func TestGetPipelineRun_NotFound(t *testing.T) {
	s := setupTestDB(t)
	ctx := context.Background()

	_, err := s.GetPipelineRun(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent pipeline run")
	}
}

func TestListProjects(t *testing.T) {
	s := setupTestDB(t)
	ctx := context.Background()

	p1 := domain.NewProject("项目A", domain.StyleManga, 5)
	p2 := domain.NewProject("项目B", domain.Style3D, 10)
	s.SaveProject(ctx, p1)
	s.SaveProject(ctx, p2)

	projects, err := s.ListProjects(ctx)
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(projects))
	}
}

func TestRecoverRunningPipelines(t *testing.T) {
	s := setupTestDB(t)
	ctx := context.Background()

	run := &PipelineRunRecord{
		ProjectID:    "proj_running",
		Status:       "running",
		CurrentPhase: "storyboard",
	}
	s.SavePipelineRun(ctx, run)

	if err := s.RecoverRunningPipelines(ctx); err != nil {
		t.Fatalf("RecoverRunningPipelines: %v", err)
	}

	got, _ := s.GetPipelineRun(ctx, "proj_running")
	if got.Status != "failed" {
		t.Errorf("expected status 'failed', got '%s'", got.Status)
	}
	if got.Error != "server restarted" {
		t.Errorf("expected error 'server restarted', got '%s'", got.Error)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./internal/store/ -v -run "TestSavePipelineRun|TestUpdatePipelineRun|TestSaveAndGetPipelineResult|TestGetPipelineRun_NotFound|TestListProjects|TestRecoverRunningPipelines" -count=1`
Expected: compilation error — PipelineRunRecord, SavePipelineRun, etc. not defined

- [ ] **Step 3: Add PipelineRunStore interface and types to store.go**

Add to `internal/store/store.go` — new import `"time"`, new interface and type, add `PipelineRunStore` to `Store` composite:

```go
// internal/store/store.go — add after BlueprintStore interface

type PipelineRunRecord struct {
	ProjectID    string    `json:"project_id"`
	Status       string    `json:"status"`
	CurrentPhase string    `json:"current_phase"`
	Error        string    `json:"error"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type PipelineRunStore interface {
	SavePipelineRun(ctx context.Context, run *PipelineRunRecord) error
	GetPipelineRun(ctx context.Context, projectID string) (*PipelineRunRecord, error)
	UpdatePipelineRun(ctx context.Context, projectID string, status string, phase string, errMsg string) error
	SavePipelineResult(ctx context.Context, projectID string, resultJSON []byte) error
	GetPipelineResult(ctx context.Context, projectID string) ([]byte, error)
	ListProjects(ctx context.Context) ([]*domain.Project, error)
	RecoverRunningPipelines(ctx context.Context) error
}
```

Update the `Store` composite interface:

```go
type Store interface {
	ProjectStore
	AssetStore
	BlueprintStore
	PipelineRunStore
	Close() error
}
```

- [ ] **Step 4: Implement PipelineRunStore methods in sqlite.go**

Add to `internal/store/sqlite.go` — add pipeline tables to `migrate()`:

```go
// Add inside migrate(), after existing CREATE TABLE statements:
CREATE TABLE IF NOT EXISTS pipeline_runs (
    project_id   TEXT PRIMARY KEY,
    status       TEXT NOT NULL DEFAULT 'running',
    current_phase TEXT NOT NULL DEFAULT '',
    error        TEXT NOT NULL DEFAULT '',
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS pipeline_results (
    project_id   TEXT PRIMARY KEY,
    result_json  TEXT NOT NULL,
    updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

Add these methods to `SQLiteStore`:

```go
func (s *SQLiteStore) SavePipelineRun(ctx context.Context, run *PipelineRunRecord) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT OR REPLACE INTO pipeline_runs (project_id, status, current_phase, error, created_at, updated_at) VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)",
		run.ProjectID, run.Status, run.CurrentPhase, run.Error)
	return err
}

func (s *SQLiteStore) GetPipelineRun(ctx context.Context, projectID string) (*PipelineRunRecord, error) {
	r := &PipelineRunRecord{}
	err := s.db.QueryRowContext(ctx,
		"SELECT project_id, status, current_phase, error, created_at, updated_at FROM pipeline_runs WHERE project_id = ?", projectID).
		Scan(&r.ProjectID, &r.Status, &r.CurrentPhase, &r.Error, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get pipeline run %s: %w", projectID, err)
	}
	return r, nil
}

func (s *SQLiteStore) UpdatePipelineRun(ctx context.Context, projectID string, status string, phase string, errMsg string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE pipeline_runs SET status = ?, current_phase = ?, error = ?, updated_at = CURRENT_TIMESTAMP WHERE project_id = ?",
		status, phase, errMsg, projectID)
	return err
}

func (s *SQLiteStore) SavePipelineResult(ctx context.Context, projectID string, resultJSON []byte) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT OR REPLACE INTO pipeline_results (project_id, result_json, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)",
		projectID, string(resultJSON))
	return err
}

func (s *SQLiteStore) GetPipelineResult(ctx context.Context, projectID string) ([]byte, error) {
	var data string
	err := s.db.QueryRowContext(ctx,
		"SELECT result_json FROM pipeline_results WHERE project_id = ?", projectID).Scan(&data)
	if err != nil {
		return nil, fmt.Errorf("get pipeline result %s: %w", projectID, err)
	}
	return []byte(data), nil
}

func (s *SQLiteStore) ListProjects(ctx context.Context) ([]*domain.Project, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, name, style, episode_count, status, created_at FROM projects ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*domain.Project
	for rows.Next() {
		p := &domain.Project{}
		if err := rows.Scan(&p.ID, &p.Name, &p.Style, &p.EpisodeCount, &p.Status, &p.CreatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, nil
}

func (s *SQLiteStore) RecoverRunningPipelines(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE pipeline_runs SET status = 'failed', error = 'server restarted', updated_at = CURRENT_TIMESTAMP WHERE status = 'running'")
	return err
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./internal/store/ -v -count=1`
Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git -C /Users/rain/code/west-garden/ai-drama-research add \
  short-maker/internal/store/store.go \
  short-maker/internal/store/sqlite.go \
  short-maker/internal/store/pipeline_test.go
git -C /Users/rain/code/west-garden/ai-drama-research commit -m "feat(short-maker): PipelineRunStore — persistence for pipeline runs and results"
```

---

### Task 2: Add chi and cors dependencies

**Files:**
- Modify: `short-maker/go.mod`

- [ ] **Step 1: Add chi and cors**

```bash
cd /Users/rain/code/west-garden/ai-drama-research/short-maker && go get github.com/go-chi/chi/v5@latest github.com/go-chi/cors@latest
```

- [ ] **Step 2: Verify build**

Run: `go build -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./...`
Expected: success

- [ ] **Step 3: Commit**

```bash
git -C /Users/rain/code/west-garden/ai-drama-research add short-maker/go.mod short-maker/go.sum
git -C /Users/rain/code/west-garden/ai-drama-research commit -m "chore(short-maker): add chi router and cors middleware dependencies"
```

---

### Task 3: API Server Skeleton + CORS Middleware

**Files:**
- Create: `short-maker/internal/api/server.go`
- Create: `short-maker/internal/api/middleware.go`

- [ ] **Step 1: Create server.go**

```go
// internal/api/server.go
package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/west-garden/short-maker/internal/agent"
	"github.com/west-garden/short-maker/internal/store"
)

type SSEEvent struct {
	Type    string `json:"type"`
	Phase   string `json:"phase,omitempty"`
	Message string `json:"message,omitempty"`
}

type PipelineRun struct {
	ProjectID string
	Status    string
	Phase     agent.Phase
	Events    chan SSEEvent
	Error     string
}

type Server struct {
	router    chi.Router
	agents    map[agent.Phase]agent.Agent
	store     store.Store
	outputDir string
	runs      map[string]*PipelineRun
	mu        sync.RWMutex
}

func NewServer(agents map[agent.Phase]agent.Agent, st store.Store, outputDir string) *Server {
	s := &Server{
		agents:    agents,
		store:     st,
		outputDir: outputDir,
		runs:      make(map[string]*PipelineRun),
	}

	r := chi.NewRouter()
	r.Use(CORSMiddleware())

	r.Post("/api/projects", s.handleCreateProject)
	r.Get("/api/projects", s.handleListProjects)
	r.Get("/api/projects/{id}", s.handleGetProject)
	r.Get("/api/projects/{id}/events", s.handleSSE)

	// Serve generated files
	fileServer := http.StripPrefix("/output/", http.FileServer(http.Dir(outputDir)))
	r.Handle("/output/*", fileServer)

	s.router = r
	return s
}

func (s *Server) Start(port int) error {
	addr := fmt.Sprintf(":%d", port)
	log.Printf("[api] starting server on %s", addr)
	return http.ListenAndServe(addr, s.router)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
```

- [ ] **Step 2: Create middleware.go**

```go
// internal/api/middleware.go
package api

import (
	"github.com/go-chi/cors"
)

func CORSMiddleware() func(next http.Handler) http.Handler {
	return cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	})
}
```

Note: middleware.go needs an import for `"net/http"` — the `cors.Handler` returns `func(next http.Handler) http.Handler` which uses `net/http`.

- [ ] **Step 3: Verify build**

Run: `go build -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./...`
Expected: may fail because handler methods don't exist yet. Add stub handlers.

- [ ] **Step 4: Add stub handler files**

Create `internal/api/handler_project.go`:

```go
// internal/api/handler_project.go
package api

import "net/http"

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented")
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented")
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented")
}
```

Create `internal/api/handler_sse.go`:

```go
// internal/api/handler_sse.go
package api

import "net/http"

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented")
}
```

- [ ] **Step 5: Verify build**

Run: `go build -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./...`
Expected: success

- [ ] **Step 6: Commit**

```bash
git -C /Users/rain/code/west-garden/ai-drama-research add \
  short-maker/internal/api/server.go \
  short-maker/internal/api/middleware.go \
  short-maker/internal/api/handler_project.go \
  short-maker/internal/api/handler_sse.go
git -C /Users/rain/code/west-garden/ai-drama-research commit -m "feat(short-maker): API server skeleton with chi router + CORS"
```

---

### Task 4: Project Handlers (Create, List, Get)

**Files:**
- Modify: `short-maker/internal/api/handler_project.go`
- Modify: `short-maker/internal/api/server.go`

- [ ] **Step 1: Implement handleCreateProject**

Replace the stub in `handler_project.go` with:

```go
// internal/api/handler_project.go
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
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
			// Use pipeline run status if available (more accurate than project status)
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

	// Try to load pipeline result
	type projectDetail struct {
		Project        *domain.Project     `json:"project"`
		PipelineStatus string              `json:"pipeline_status"`
		CurrentPhase   string              `json:"current_phase"`
		Blueprint      json.RawMessage     `json:"blueprint,omitempty"`
		Storyboard     json.RawMessage     `json:"storyboard,omitempty"`
		Images         json.RawMessage     `json:"images,omitempty"`
		Videos         json.RawMessage     `json:"videos,omitempty"`
		Errors         json.RawMessage     `json:"errors,omitempty"`
	}

	detail := projectDetail{
		Project:        project,
		PipelineStatus: pipelineStatus,
		CurrentPhase:   currentPhase,
	}

	resultJSON, err := s.store.GetPipelineResult(r.Context(), id)
	if err == nil {
		// Extract fields from the stored PipelineState JSON
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
```

- [ ] **Step 2: Verify build**

Run: `go build -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./...`
Expected: success

- [ ] **Step 3: Commit**

```bash
git -C /Users/rain/code/west-garden/ai-drama-research add short-maker/internal/api/handler_project.go
git -C /Users/rain/code/west-garden/ai-drama-research commit -m "feat(short-maker): project handlers — create, list, get with async pipeline"
```

---

### Task 5: SSE Handler

**Files:**
- Modify: `short-maker/internal/api/handler_sse.go`

- [ ] **Step 1: Implement SSE handler**

Replace the stub in `handler_sse.go`:

```go
// internal/api/handler_sse.go
package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	s.mu.RLock()
	run, exists := s.runs[id]
	s.mu.RUnlock()

	if !exists {
		// Pipeline already finished — send historical status from DB
		pipelineRun, err := s.store.GetPipelineRun(r.Context(), id)
		if err != nil {
			writeSSEEvent(w, flusher, SSEEvent{Type: "error", Message: "project not found"})
			return
		}
		if pipelineRun.Status == "completed" {
			writeSSEEvent(w, flusher, SSEEvent{Type: "done"})
		} else if pipelineRun.Status == "failed" {
			writeSSEEvent(w, flusher, SSEEvent{Type: "error", Message: pipelineRun.Error})
		}
		return
	}

	// Stream live events from the running pipeline
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			log.Printf("[sse] client disconnected for project %s", id)
			return
		case event, ok := <-run.Events:
			if !ok {
				// Channel closed — pipeline finished
				return
			}
			writeSSEEvent(w, flusher, event)
		}
	}
}

func writeSSEEvent(w http.ResponseWriter, flusher http.Flusher, event SSEEvent) {
	data, _ := json.Marshal(event)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}
```

- [ ] **Step 2: Verify build**

Run: `go build -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./...`
Expected: success

- [ ] **Step 3: Commit**

```bash
git -C /Users/rain/code/west-garden/ai-drama-research add short-maker/internal/api/handler_sse.go
git -C /Users/rain/code/west-garden/ai-drama-research commit -m "feat(short-maker): SSE handler — real-time pipeline progress streaming"
```

---

### Task 6: API Tests

**Files:**
- Create: `short-maker/internal/api/server_test.go`

- [ ] **Step 1: Write API tests**

```go
// internal/api/server_test.go
package api

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/west-garden/short-maker/internal/agent"
	"github.com/west-garden/short-maker/internal/store"
)

func setupTestServer(t *testing.T) *Server {
	t.Helper()
	dbPath := t.TempDir() + "/test.db"
	st, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	outputDir := t.TempDir()

	// All mock agents
	agents := map[agent.Phase]agent.Agent{}
	for _, phase := range agent.DefaultFlow {
		p := phase
		agents[p] = agent.NewMockAgent(p, func(ctx context.Context, s *agent.PipelineState) (*agent.PipelineState, error) {
			return s, nil
		})
	}

	return NewServer(agents, st, outputDir)
}

func createTestProject(t *testing.T, srv *Server) map[string]any {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("script", "test.txt")
	part.Write([]byte("测试剧本"))
	writer.WriteField("name", "测试项目")
	writer.WriteField("style", "manga")
	writer.WriteField("episodes", "2")
	writer.Close()

	req := httptest.NewRequest("POST", "/api/projects", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]any
	json.Unmarshal(w.Body.Bytes(), &result)
	return result
}

func TestCreateProject(t *testing.T) {
	srv := setupTestServer(t)
	result := createTestProject(t, srv)

	if result["id"] == nil || result["id"] == "" {
		t.Error("expected non-empty project ID")
	}
	if result["name"] != "测试项目" {
		t.Errorf("expected name '测试项目', got '%v'", result["name"])
	}
	if result["style"] != "manga" {
		t.Errorf("expected style 'manga', got '%v'", result["style"])
	}
}

func TestCreateProject_MissingScript(t *testing.T) {
	srv := setupTestServer(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("name", "test")
	writer.Close()

	req := httptest.NewRequest("POST", "/api/projects", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestListProjects(t *testing.T) {
	srv := setupTestServer(t)
	createTestProject(t, srv)
	createTestProject(t, srv)

	// Wait briefly for projects to be saved
	time.Sleep(50 * time.Millisecond)

	req := httptest.NewRequest("GET", "/api/projects", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var projects []map[string]any
	json.Unmarshal(w.Body.Bytes(), &projects)
	if len(projects) < 2 {
		t.Errorf("expected at least 2 projects, got %d", len(projects))
	}
}

func TestGetProject(t *testing.T) {
	srv := setupTestServer(t)
	created := createTestProject(t, srv)
	id := created["id"].(string)

	// Wait for pipeline to finish
	time.Sleep(200 * time.Millisecond)

	req := httptest.NewRequest("GET", "/api/projects/"+id, nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var detail map[string]any
	json.Unmarshal(w.Body.Bytes(), &detail)
	if detail["project"] == nil {
		t.Error("expected project in response")
	}
	if detail["pipeline_status"] == nil {
		t.Error("expected pipeline_status in response")
	}
}

func TestGetProject_NotFound(t *testing.T) {
	srv := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/projects/nonexistent", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestSSE_CompletedProject(t *testing.T) {
	srv := setupTestServer(t)
	created := createTestProject(t, srv)
	id := created["id"].(string)

	// Wait for pipeline to finish
	time.Sleep(200 * time.Millisecond)

	req := httptest.NewRequest("GET", "/api/projects/"+id+"/events", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "\"type\":\"done\"") {
		t.Errorf("expected done event in SSE response, got: %s", body)
	}
}
```

Note: this test file needs `"context"` import for the mock agent setup.

- [ ] **Step 2: Run tests**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./internal/api/ -v -count=1`
Expected: all PASS

- [ ] **Step 3: Run ALL tests**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./... -count=1`
Expected: all PASS

- [ ] **Step 4: Commit**

```bash
git -C /Users/rain/code/west-garden/ai-drama-research add short-maker/internal/api/server_test.go
git -C /Users/rain/code/west-garden/ai-drama-research commit -m "feat(short-maker): API tests — create, list, get, SSE for completed project"
```

---

### Task 7: CLI serve Subcommand

**Files:**
- Modify: `short-maker/cmd/shortmaker/main.go`

- [ ] **Step 1: Add serve command**

Add a new `serveCmd` variable and register it in `init()`. Add import for `"github.com/west-garden/short-maker/internal/api"`.

```go
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the web console server",
	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetInt("port")
		outputDir, _ := cmd.Flags().GetString("output")
		dbPath, _ := cmd.Flags().GetString("db")
		useMock, _ := cmd.Flags().GetBool("mock")
		llmModel, _ := cmd.Flags().GetString("model")
		strategyPath, _ := cmd.Flags().GetString("strategies")

		if dbPath == "" {
			dbPath = "./shortmaker.db"
		}

		agents, cleanup, err := buildAgents(useMock, llmModel, dbPath, strategyPath, outputDir)
		if err != nil {
			return err
		}
		defer cleanup()

		st, err := store.NewSQLiteStore(dbPath)
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer st.Close()

		// Mark any previously running pipelines as failed
		st.RecoverRunningPipelines(context.Background())

		srv := api.NewServer(agents, st, outputDir)
		log.Printf("Starting web console at http://localhost:%d", port)
		return srv.Start(port)
	},
}
```

Add to `init()`:

```go
serveCmd.Flags().Int("port", 8080, "HTTP server port")
serveCmd.Flags().String("output", "./output", "Output directory for generated files")
serveCmd.Flags().String("db", "./shortmaker.db", "SQLite database path")
serveCmd.Flags().Bool("mock", true, "Use mock agents")
serveCmd.Flags().String("model", "gpt-4o-mini", "LLM model name")
serveCmd.Flags().String("strategies", "", "Path to strategies JSON file")
rootCmd.AddCommand(serveCmd)
```

- [ ] **Step 2: Verify build**

Run: `go build -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./...`
Expected: success

- [ ] **Step 3: Run ALL tests**

Run: `go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./... -count=1`
Expected: all PASS

- [ ] **Step 4: Commit**

```bash
git -C /Users/rain/code/west-garden/ai-drama-research add short-maker/cmd/shortmaker/main.go
git -C /Users/rain/code/west-garden/ai-drama-research commit -m "feat(short-maker): serve subcommand — start web console HTTP server"
```

---

### Task 8: Scaffold React + Vite + TailwindCSS

**Files:**
- Create: `short-maker/web/` entire directory

- [ ] **Step 1: Create Vite React project**

```bash
cd /Users/rain/code/west-garden/ai-drama-research/short-maker && npm create vite@latest web -- --template react-ts
```

- [ ] **Step 2: Install dependencies**

```bash
cd /Users/rain/code/west-garden/ai-drama-research/short-maker/web && npm install && npm install -D tailwindcss @tailwindcss/vite react-router-dom
```

- [ ] **Step 3: Configure Vite with TailwindCSS and proxy**

Replace `web/vite.config.ts` with:

```typescript
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    proxy: {
      "/api": "http://localhost:8080",
      "/output": "http://localhost:8080",
    },
  },
});
```

- [ ] **Step 4: Add Tailwind to CSS**

Replace `web/src/index.css` with:

```css
@import "tailwindcss";
```

- [ ] **Step 5: Verify dev server starts**

```bash
cd /Users/rain/code/west-garden/ai-drama-research/short-maker/web && npx vite --host 127.0.0.1 &
sleep 3 && kill %1
```

Expected: Vite dev server starts without errors

- [ ] **Step 6: Commit**

```bash
git -C /Users/rain/code/west-garden/ai-drama-research add short-maker/web/
git -C /Users/rain/code/west-garden/ai-drama-research commit -m "feat(short-maker): scaffold React + Vite + TailwindCSS frontend"
```

---

### Task 9: React — API Client + Layout + Routing

**Files:**
- Create: `short-maker/web/src/api.ts`
- Create: `short-maker/web/src/components/Layout.tsx`
- Modify: `short-maker/web/src/App.tsx`
- Modify: `short-maker/web/src/main.tsx`

- [ ] **Step 1: Create api.ts**

```typescript
// web/src/api.ts
const API_BASE = "/api";

export interface ProjectSummary {
  id: string;
  name: string;
  style: string;
  episode_count: number;
  status: string;
  current_phase: string;
  created_at: string;
}

export interface GeneratedShot {
  shot_number: number;
  episode_number: number;
  image_path: string;
  video_path: string;
  grade: string;
  image_score: number;
  video_score: number;
}

export interface ProjectDetail {
  project: {
    id: string;
    name: string;
    style: string;
    episode_count: number;
    status: string;
  };
  pipeline_status: string;
  current_phase: string;
  blueprint?: any;
  storyboard?: any[];
  images?: GeneratedShot[];
  videos?: GeneratedShot[];
  errors?: string[];
}

export interface SSEEvent {
  type: "phase_start" | "phase_complete" | "done" | "error";
  phase?: string;
  message?: string;
}

export async function createProject(form: FormData): Promise<any> {
  const res = await fetch(`${API_BASE}/projects`, {
    method: "POST",
    body: form,
  });
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

export async function listProjects(): Promise<ProjectSummary[]> {
  const res = await fetch(`${API_BASE}/projects`);
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

export async function getProject(id: string): Promise<ProjectDetail> {
  const res = await fetch(`${API_BASE}/projects/${id}`);
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

export function subscribeToEvents(
  id: string,
  onEvent: (e: SSEEvent) => void
): EventSource {
  const es = new EventSource(`${API_BASE}/projects/${id}/events`);
  es.onmessage = (e) => onEvent(JSON.parse(e.data));
  return es;
}
```

- [ ] **Step 2: Create Layout.tsx**

```tsx
// web/src/components/Layout.tsx
import { Link, Outlet } from "react-router-dom";

export default function Layout() {
  return (
    <div className="min-h-screen bg-gray-950 text-gray-100">
      <header className="border-b border-gray-800 px-6 py-4">
        <div className="flex items-center justify-between max-w-6xl mx-auto">
          <Link to="/" className="text-lg font-bold text-white">
            Short Maker
          </Link>
          <Link
            to="/new"
            className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg text-sm"
          >
            + 新建项目
          </Link>
        </div>
      </header>
      <main className="max-w-6xl mx-auto px-6 py-8">
        <Outlet />
      </main>
    </div>
  );
}
```

- [ ] **Step 3: Update App.tsx**

```tsx
// web/src/App.tsx
import { BrowserRouter, Routes, Route } from "react-router-dom";
import Layout from "./components/Layout";
import ProjectList from "./pages/ProjectList";
import NewProject from "./pages/NewProject";
import ProjectDetail from "./pages/ProjectDetail";

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<Layout />}>
          <Route path="/" element={<ProjectList />} />
          <Route path="/new" element={<NewProject />} />
          <Route path="/projects/:id" element={<ProjectDetail />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}
```

- [ ] **Step 4: Update main.tsx**

```tsx
// web/src/main.tsx
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "./index.css";
import App from "./App";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>
);
```

- [ ] **Step 5: Create placeholder page files**

Create `web/src/pages/ProjectList.tsx`:
```tsx
export default function ProjectList() {
  return <div>ProjectList — TODO</div>;
}
```

Create `web/src/pages/NewProject.tsx`:
```tsx
export default function NewProject() {
  return <div>NewProject — TODO</div>;
}
```

Create `web/src/pages/ProjectDetail.tsx`:
```tsx
export default function ProjectDetail() {
  return <div>ProjectDetail — TODO</div>;
}
```

- [ ] **Step 6: Verify build**

```bash
cd /Users/rain/code/west-garden/ai-drama-research/short-maker/web && npx tsc --noEmit && npx vite build
```

Expected: no errors

- [ ] **Step 7: Commit**

```bash
git -C /Users/rain/code/west-garden/ai-drama-research add short-maker/web/src/
git -C /Users/rain/code/west-garden/ai-drama-research commit -m "feat(short-maker): React routing + API client + Layout shell"
```

---

### Task 10: React — ProjectList Page

**Files:**
- Modify: `short-maker/web/src/pages/ProjectList.tsx`

- [ ] **Step 1: Implement ProjectList**

```tsx
// web/src/pages/ProjectList.tsx
import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { listProjects, ProjectSummary } from "../api";

const statusColors: Record<string, string> = {
  completed: "bg-green-500 text-black",
  processing: "bg-blue-500 text-white",
  running: "bg-blue-500 text-white",
  failed: "bg-red-500 text-white",
  created: "bg-gray-500 text-white",
};

export default function ProjectList() {
  const [projects, setProjects] = useState<ProjectSummary[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    listProjects()
      .then(setProjects)
      .catch(console.error)
      .finally(() => setLoading(false));
  }, []);

  if (loading) {
    return <div className="text-gray-500">加载中...</div>;
  }

  if (projects.length === 0) {
    return (
      <div className="text-center py-20">
        <p className="text-gray-500 mb-4">还没有项目</p>
        <Link
          to="/new"
          className="bg-blue-600 hover:bg-blue-700 text-white px-6 py-3 rounded-lg"
        >
          创建第一个项目
        </Link>
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {projects.map((p) => (
        <Link
          key={p.id}
          to={`/projects/${p.id}`}
          className="block border border-gray-800 rounded-lg p-4 hover:border-gray-600 transition-colors"
        >
          <div className="flex justify-between items-center mb-2">
            <span className="font-bold">{p.name}</span>
            <span
              className={`text-xs px-2 py-0.5 rounded ${statusColors[p.status] || statusColors.created}`}
            >
              {p.status}
            </span>
          </div>
          <div className="text-sm text-gray-500">
            {p.style} · {p.episode_count} 集
          </div>
          <div className="text-xs text-gray-600 mt-1">
            {new Date(p.created_at).toLocaleString()}
          </div>
        </Link>
      ))}
    </div>
  );
}
```

- [ ] **Step 2: Verify build**

```bash
cd /Users/rain/code/west-garden/ai-drama-research/short-maker/web && npx tsc --noEmit
```

- [ ] **Step 3: Commit**

```bash
git -C /Users/rain/code/west-garden/ai-drama-research add short-maker/web/src/pages/ProjectList.tsx
git -C /Users/rain/code/west-garden/ai-drama-research commit -m "feat(short-maker): ProjectList page — card grid with status badges"
```

---

### Task 11: React — NewProject Page

**Files:**
- Modify: `short-maker/web/src/pages/NewProject.tsx`

- [ ] **Step 1: Implement NewProject**

```tsx
// web/src/pages/NewProject.tsx
import { useState, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { createProject } from "../api";

export default function NewProject() {
  const navigate = useNavigate();
  const fileRef = useRef<HTMLInputElement>(null);
  const [name, setName] = useState("");
  const [style, setStyle] = useState("manga");
  const [episodes, setEpisodes] = useState(10);
  const [fileName, setFileName] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");

    const file = fileRef.current?.files?.[0];
    if (!file) {
      setError("请上传剧本文件");
      return;
    }

    setSubmitting(true);
    try {
      const form = new FormData();
      form.append("script", file);
      form.append("name", name || file.name.replace(/\.\w+$/, ""));
      form.append("style", style);
      form.append("episodes", String(episodes));

      const project = await createProject(form);
      navigate(`/projects/${project.id}`);
    } catch (err: any) {
      setError(err.message || "创建失败");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="max-w-md mx-auto">
      <h2 className="text-xl font-bold mb-6">新建项目</h2>
      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label className="block text-sm text-gray-400 mb-1">项目名称</label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="留空则使用文件名"
            className="w-full bg-gray-900 border border-gray-700 rounded-lg px-3 py-2 text-white"
          />
        </div>

        <div>
          <label className="block text-sm text-gray-400 mb-1">剧本文件</label>
          <div
            onClick={() => fileRef.current?.click()}
            className="border-2 border-dashed border-gray-700 rounded-lg p-6 text-center cursor-pointer hover:border-gray-500 transition-colors"
          >
            {fileName ? (
              <span className="text-white">{fileName}</span>
            ) : (
              <span className="text-gray-500">点击上传 .txt 文件</span>
            )}
          </div>
          <input
            ref={fileRef}
            type="file"
            accept=".txt"
            className="hidden"
            onChange={(e) => setFileName(e.target.files?.[0]?.name || "")}
          />
        </div>

        <div>
          <label className="block text-sm text-gray-400 mb-1">风格</label>
          <select
            value={style}
            onChange={(e) => setStyle(e.target.value)}
            className="w-full bg-gray-900 border border-gray-700 rounded-lg px-3 py-2 text-white"
          >
            <option value="manga">manga（漫画）</option>
            <option value="3d">3D</option>
            <option value="live_action">live_action（真人）</option>
          </select>
        </div>

        <div>
          <label className="block text-sm text-gray-400 mb-1">集数</label>
          <input
            type="number"
            value={episodes}
            onChange={(e) => setEpisodes(Number(e.target.value))}
            min={1}
            max={100}
            className="w-full bg-gray-900 border border-gray-700 rounded-lg px-3 py-2 text-white"
          />
        </div>

        {error && <div className="text-red-400 text-sm">{error}</div>}

        <button
          type="submit"
          disabled={submitting}
          className="w-full bg-blue-600 hover:bg-blue-700 disabled:bg-gray-700 text-white py-3 rounded-lg font-medium"
        >
          {submitting ? "创建中..." : "开始生成"}
        </button>
      </form>
    </div>
  );
}
```

- [ ] **Step 2: Verify build**

```bash
cd /Users/rain/code/west-garden/ai-drama-research/short-maker/web && npx tsc --noEmit
```

- [ ] **Step 3: Commit**

```bash
git -C /Users/rain/code/west-garden/ai-drama-research add short-maker/web/src/pages/NewProject.tsx
git -C /Users/rain/code/west-garden/ai-drama-research commit -m "feat(short-maker): NewProject page — upload form with style/episodes"
```

---

### Task 12: React — PipelineProgress + ShotGallery Components

**Files:**
- Create: `short-maker/web/src/components/PipelineProgress.tsx`
- Create: `short-maker/web/src/components/ShotGallery.tsx`

- [ ] **Step 1: Create PipelineProgress**

```tsx
// web/src/components/PipelineProgress.tsx
const PHASES = [
  { key: "story_understanding", label: "剧本理解" },
  { key: "character_asset", label: "角色资产" },
  { key: "storyboard", label: "分镜" },
  { key: "image_generation", label: "图片生成" },
  { key: "video_generation", label: "视频生成" },
];

interface Props {
  currentPhase: string;
  status: string; // "running" | "completed" | "failed"
  completedPhases: Set<string>;
}

export default function PipelineProgress({
  currentPhase,
  status,
  completedPhases,
}: Props) {
  return (
    <div className="flex gap-1 mb-6">
      {PHASES.map((phase) => {
        const isCompleted = completedPhases.has(phase.key);
        const isCurrent = phase.key === currentPhase && status === "running";
        const isFailed = phase.key === currentPhase && status === "failed";

        let bg = "bg-gray-800 border-gray-700";
        let icon = "○";
        let iconColor = "text-gray-600";

        if (isCompleted) {
          bg = "bg-green-900/30 border-green-700";
          icon = "✓";
          iconColor = "text-green-400";
        } else if (isCurrent) {
          bg = "bg-blue-900/30 border-blue-600";
          icon = "●";
          iconColor = "text-blue-400 animate-pulse";
        } else if (isFailed) {
          bg = "bg-red-900/30 border-red-700";
          icon = "✗";
          iconColor = "text-red-400";
        }

        return (
          <div
            key={phase.key}
            className={`flex-1 text-center p-2 border rounded-lg ${bg}`}
          >
            <div className={`text-sm ${iconColor}`}>{icon}</div>
            <div className="text-xs text-gray-400 mt-1">{phase.label}</div>
          </div>
        );
      })}
    </div>
  );
}
```

- [ ] **Step 2: Create ShotGallery**

```tsx
// web/src/components/ShotGallery.tsx
import { useState } from "react";
import { GeneratedShot } from "../api";

const gradeColors: Record<string, string> = {
  S: "bg-amber-500 text-black",
  A: "bg-blue-500 text-white",
  B: "bg-green-500 text-black",
  C: "bg-gray-500 text-white",
};

interface Props {
  images: GeneratedShot[];
  videos: GeneratedShot[];
}

export default function ShotGallery({ images, videos }: Props) {
  const [selectedShot, setSelectedShot] = useState<GeneratedShot | null>(null);
  const [showVideo, setShowVideo] = useState(false);

  // Group by episode
  const episodes = new Map<number, GeneratedShot[]>();
  for (const img of images) {
    if (!episodes.has(img.episode_number)) {
      episodes.set(img.episode_number, []);
    }
    episodes.get(img.episode_number)!.push(img);
  }

  const videoMap = new Map<string, GeneratedShot>();
  for (const vid of videos) {
    videoMap.set(`${vid.episode_number}-${vid.shot_number}`, vid);
  }

  return (
    <div>
      {Array.from(episodes.entries())
        .sort(([a], [b]) => a - b)
        .map(([epNum, shots]) => (
          <div key={epNum} className="mb-6">
            <h3 className="font-bold mb-3 text-gray-300">第 {epNum} 集</h3>
            <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-3">
              {shots
                .sort((a, b) => a.shot_number - b.shot_number)
                .map((shot) => (
                  <div
                    key={shot.shot_number}
                    onClick={() => {
                      setSelectedShot(shot);
                      setShowVideo(false);
                    }}
                    className="border border-gray-800 rounded-lg overflow-hidden cursor-pointer hover:border-gray-600 transition-colors"
                  >
                    <div className="aspect-video bg-gray-900 flex items-center justify-center">
                      <img
                        src={shot.image_path}
                        alt={`Shot ${shot.shot_number}`}
                        className="w-full h-full object-cover"
                        onError={(e) => {
                          (e.target as HTMLImageElement).style.display = "none";
                        }}
                      />
                    </div>
                    <div className="p-2 text-xs">
                      <div className="flex justify-between items-center">
                        <span>Shot {shot.shot_number}</span>
                        <span
                          className={`px-1.5 py-0.5 rounded text-[10px] font-bold ${gradeColors[shot.grade] || "bg-gray-700"}`}
                        >
                          {shot.grade}
                        </span>
                      </div>
                      <div className="text-gray-500 mt-0.5">
                        {shot.image_score}分
                      </div>
                    </div>
                  </div>
                ))}
            </div>
          </div>
        ))}

      {/* Modal */}
      {selectedShot && (
        <div
          className="fixed inset-0 bg-black/80 flex items-center justify-center z-50 p-8"
          onClick={() => setSelectedShot(null)}
        >
          <div
            className="max-w-3xl w-full bg-gray-900 rounded-xl overflow-hidden"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="aspect-video bg-black flex items-center justify-center">
              {showVideo ? (
                <video
                  src={
                    videoMap.get(
                      `${selectedShot.episode_number}-${selectedShot.shot_number}`
                    )?.video_path
                  }
                  controls
                  autoPlay
                  className="w-full h-full"
                />
              ) : (
                <img
                  src={selectedShot.image_path}
                  alt={`Shot ${selectedShot.shot_number}`}
                  className="w-full h-full object-contain"
                />
              )}
            </div>
            <div className="p-4 flex justify-between items-center">
              <div>
                <span className="font-bold">
                  EP{selectedShot.episode_number} Shot{" "}
                  {selectedShot.shot_number}
                </span>
                <span
                  className={`ml-2 px-2 py-0.5 rounded text-xs font-bold ${gradeColors[selectedShot.grade] || ""}`}
                >
                  {selectedShot.grade}
                </span>
                <span className="ml-2 text-gray-500 text-sm">
                  图片 {selectedShot.image_score}分
                </span>
              </div>
              <div className="flex gap-2">
                <button
                  onClick={() => setShowVideo(false)}
                  className={`px-3 py-1 rounded text-sm ${!showVideo ? "bg-blue-600" : "bg-gray-700"}`}
                >
                  图片
                </button>
                <button
                  onClick={() => setShowVideo(true)}
                  className={`px-3 py-1 rounded text-sm ${showVideo ? "bg-blue-600" : "bg-gray-700"}`}
                >
                  视频
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
```

- [ ] **Step 3: Verify build**

```bash
cd /Users/rain/code/west-garden/ai-drama-research/short-maker/web && npx tsc --noEmit
```

- [ ] **Step 4: Commit**

```bash
git -C /Users/rain/code/west-garden/ai-drama-research add \
  short-maker/web/src/components/PipelineProgress.tsx \
  short-maker/web/src/components/ShotGallery.tsx
git -C /Users/rain/code/west-garden/ai-drama-research commit -m "feat(short-maker): PipelineProgress + ShotGallery components"
```

---

### Task 13: React — ProjectDetail Page

**Files:**
- Modify: `short-maker/web/src/pages/ProjectDetail.tsx`

- [ ] **Step 1: Implement ProjectDetail**

```tsx
// web/src/pages/ProjectDetail.tsx
import { useEffect, useState, useRef } from "react";
import { useParams } from "react-router-dom";
import { getProject, subscribeToEvents, ProjectDetail as ProjectDetailType, SSEEvent } from "../api";
import PipelineProgress from "../components/PipelineProgress";
import ShotGallery from "../components/ShotGallery";

const PHASE_ORDER = [
  "story_understanding",
  "character_asset",
  "storyboard",
  "image_generation",
  "video_generation",
];

export default function ProjectDetail() {
  const { id } = useParams<{ id: string }>();
  const [detail, setDetail] = useState<ProjectDetailType | null>(null);
  const [loading, setLoading] = useState(true);
  const [completedPhases, setCompletedPhases] = useState<Set<string>>(
    new Set()
  );
  const [pipelineStatus, setPipelineStatus] = useState("unknown");
  const [currentPhase, setCurrentPhase] = useState("");
  const esRef = useRef<EventSource | null>(null);

  useEffect(() => {
    if (!id) return;

    // Load project detail
    getProject(id)
      .then((d) => {
        setDetail(d);
        setPipelineStatus(d.pipeline_status);
        setCurrentPhase(d.current_phase);

        // Determine completed phases from current_phase
        if (d.pipeline_status === "completed") {
          setCompletedPhases(new Set(PHASE_ORDER));
        } else if (d.current_phase) {
          const idx = PHASE_ORDER.indexOf(d.current_phase);
          if (idx >= 0) {
            setCompletedPhases(new Set(PHASE_ORDER.slice(0, idx + 1)));
          }
        }
      })
      .catch(console.error)
      .finally(() => setLoading(false));

    // Subscribe to SSE
    const es = subscribeToEvents(id, (event: SSEEvent) => {
      if (event.type === "phase_complete" && event.phase) {
        setCompletedPhases((prev) => new Set([...prev, event.phase!]));
        setCurrentPhase(event.phase);
      }
      if (event.type === "done") {
        setPipelineStatus("completed");
        setCompletedPhases(new Set(PHASE_ORDER));
        // Reload full detail
        getProject(id).then(setDetail);
        es.close();
      }
      if (event.type === "error") {
        setPipelineStatus("failed");
        es.close();
      }
    });
    esRef.current = es;

    return () => {
      es.close();
    };
  }, [id]);

  if (loading) {
    return <div className="text-gray-500">加载中...</div>;
  }

  if (!detail) {
    return <div className="text-red-400">项目未找到</div>;
  }

  const statusLabel: Record<string, string> = {
    running: "运行中",
    completed: "已完成",
    failed: "失败",
    unknown: "未知",
  };

  const statusColor: Record<string, string> = {
    running: "bg-blue-500",
    completed: "bg-green-500",
    failed: "bg-red-500",
    unknown: "bg-gray-500",
  };

  return (
    <div>
      {/* Header */}
      <div className="flex items-center gap-3 mb-6">
        <h2 className="text-xl font-bold">{detail.project.name}</h2>
        <span
          className={`text-xs px-2 py-0.5 rounded text-white ${statusColor[pipelineStatus] || "bg-gray-500"}`}
        >
          {statusLabel[pipelineStatus] || pipelineStatus}
        </span>
        <span className="text-sm text-gray-500">
          {detail.project.style} · {detail.project.episode_count} 集
        </span>
      </div>

      {/* Pipeline Progress */}
      <PipelineProgress
        currentPhase={currentPhase}
        status={pipelineStatus}
        completedPhases={completedPhases}
      />

      {/* Shot Gallery */}
      {detail.images && detail.images.length > 0 && (
        <ShotGallery
          images={detail.images}
          videos={detail.videos || []}
        />
      )}

      {/* Empty state while running */}
      {pipelineStatus === "running" && (!detail.images || detail.images.length === 0) && (
        <div className="text-center py-12 text-gray-500">
          Pipeline 运行中，请等待...
        </div>
      )}

      {/* Error state */}
      {pipelineStatus === "failed" && (
        <div className="text-center py-12 text-red-400">
          Pipeline 执行失败
        </div>
      )}
    </div>
  );
}
```

- [ ] **Step 2: Verify build**

```bash
cd /Users/rain/code/west-garden/ai-drama-research/short-maker/web && npx tsc --noEmit && npx vite build
```

Expected: both type check and production build succeed

- [ ] **Step 3: Commit**

```bash
git -C /Users/rain/code/west-garden/ai-drama-research add short-maker/web/src/pages/ProjectDetail.tsx
git -C /Users/rain/code/west-garden/ai-drama-research commit -m "feat(short-maker): ProjectDetail page — SSE progress + shot gallery"
```

---

### Task 14: go mod tidy + Final Verification

- [ ] **Step 1: go mod tidy**

```bash
GOWORK=off go mod tidy -C /Users/rain/code/west-garden/ai-drama-research/short-maker
```

- [ ] **Step 2: Run all Go tests**

```bash
go test -C /Users/rain/code/west-garden/ai-drama-research/short-maker ./... -count=1
```

Expected: all PASS

- [ ] **Step 3: Build Go binary**

```bash
go build -C /Users/rain/code/west-garden/ai-drama-research/short-maker -o /dev/null ./cmd/shortmaker
```

- [ ] **Step 4: Build React frontend**

```bash
cd /Users/rain/code/west-garden/ai-drama-research/short-maker/web && npx vite build
```

- [ ] **Step 5: Commit if any changes**

```bash
git -C /Users/rain/code/west-garden/ai-drama-research diff --name-only short-maker/go.mod short-maker/go.sum
# If changes:
git -C /Users/rain/code/west-garden/ai-drama-research add short-maker/go.mod short-maker/go.sum
git -C /Users/rain/code/west-garden/ai-drama-research commit -m "chore(short-maker): go mod tidy"
```
