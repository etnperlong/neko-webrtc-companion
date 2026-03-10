package http

import (
	"context"
	"net/http"
)

// Dependencies encapsulates the collaborators required by the HTTP handlers.
type Dependencies struct {
	Ready   func() bool
	Trigger func(context.Context) error
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
	if err := s.deps.Trigger(r.Context()); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
