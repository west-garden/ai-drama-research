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
