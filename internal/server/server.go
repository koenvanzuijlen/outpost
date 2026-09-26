// Package server implements Outpost's HTTP layer: routing, request handlers, and response rendering.
package server

import (
	"log/slog"
	"net/http"
)

// Server holds the dependencies shared by all HTTP handlers and routes requests to them. It implements http.Handler.
type Server struct {
	mux    *http.ServeMux
	logger *slog.Logger
}

// New creates a Server with the given logger and registers its routes. The returned Server is ready to serve requests.
func New(logger *slog.Logger) *Server {
	server := &Server{
		mux:    http.NewServeMux(),
		logger: logger,
	}
	server.routes()
	return server
}

// ServeHTTP dispatches r to the handler registered for its method and path, satisfying http.Handler.
func (server *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	server.mux.ServeHTTP(w, r)
}
