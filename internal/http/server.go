package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/etnperlong/neko-webrtc-companion/internal/refresh"
)

// Dependencies encapsulates the collaborators required by the HTTP handlers.
type Dependencies struct {
	Ready   func() bool
	Trigger func(context.Context) refresh.Result
}

type triggerResponse struct {
	Status    string   `json:"status"`
	Changed   bool     `json:"changed"`
	Restarted []string `json:"restarted"`
	Message   string   `json:"message"`
}

// Server exposes the HTTP endpoints for health and trigger control.
type Server struct {
	deps   Dependencies
	router *http.ServeMux
}

// New wires the HTTP handlers with the provided dependencies.
func New(deps Dependencies) http.Handler {
	if deps.Ready == nil {
		deps.Ready = func() bool { return true }
	}
	s := &Server{
		deps:   deps,
		router: http.NewServeMux(),
	}
	s.router.HandleFunc("/healthz", s.healthHandler)
	s.router.HandleFunc("/readyz", s.readyHandler)
	s.router.HandleFunc("/trigger", s.triggerHandler)
	return s
}

// ServeHTTP dispatches requests to the configured handlers.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) readyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.deps.Ready() {
		http.Error(w, "service not ready", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) triggerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.deps.Trigger == nil {
		http.Error(w, "trigger not configured", http.StatusServiceUnavailable)
		return
	}
	result := s.deps.Trigger(r.Context())
	response := triggerResponse{Changed: result.Changed, Restarted: result.Restarted}
	w.Header().Set("Content-Type", "application/json")
	switch {
	case result.Busy || result.Skipped:
		response.Status = "busy"
		response.Message = "refresh already running"
		w.WriteHeader(http.StatusConflict)
	case result.Err != nil:
		response.Status = "failed"
		response.Message = result.Err.Error()
		w.WriteHeader(http.StatusInternalServerError)
	case result.NoOp:
		response.Status = "noop"
		response.Message = "no config changes"
		w.WriteHeader(http.StatusOK)
	default:
		response.Status = "ok"
		response.Message = "refresh completed"
		w.WriteHeader(http.StatusOK)
	}
	_ = json.NewEncoder(w).Encode(response)
}
