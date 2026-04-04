// internal/api/handler_sse.go
package api

import "net/http"

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented")
}
