// internal/api/server_test.go
package api

import (
	"bytes"
	"context"
	"encoding/json"
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
	// Wait for first pipeline to finish before creating second project
	// to avoid SQLite write contention
	time.Sleep(300 * time.Millisecond)
	createTestProject(t, srv)

	// Wait for second pipeline to finish and persist
	time.Sleep(300 * time.Millisecond)

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

func TestSSE_PausedProject(t *testing.T) {
	srv := setupTestServer(t)
	created := createTestProject(t, srv)
	id := created["id"].(string)

	req := httptest.NewRequest("GET", "/api/projects/"+id+"/events", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "\"type\":\"paused\"") {
		t.Errorf("expected paused event in SSE response, got: %s", body)
	}
}
