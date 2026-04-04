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
