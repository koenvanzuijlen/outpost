// Package server implements Outpost's HTTP layer: routing, request handlers, and response rendering.
package server

import (
	"bytes"
	"embed"
	"html/template"
	"log/slog"
	"net/http"
)

// Server holds the dependencies shared by all HTTP handlers and routes requests to them. It implements http.Handler.
type Server struct {
	mux       *http.ServeMux
	logger    *slog.Logger
	templates *template.Template
}

//go:embed templates
var templateFS embed.FS

// New creates a Server with the given logger and registers its routes. The returned Server is ready to serve requests.
func New(logger *slog.Logger) *Server {
	tmpl := template.Must(template.ParseFS(templateFS, "templates/*.html"))
	server := &Server{
		mux:       http.NewServeMux(),
		logger:    logger,
		templates: tmpl,
	}
	server.routes()
	return server
}

// ServeHTTP dispatches the request to the handler registered for its method and path, satisfying http.Handler.
func (server *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	server.mux.ServeHTTP(w, r)
}

// render executes the specified template with the given data and writes the result to the response writer. If an error
// occurs, it logs the error and returns a 500 status code.
func (server *Server) render(w http.ResponseWriter, name string, data any) {
	var buf bytes.Buffer
	if err := server.templates.ExecuteTemplate(&buf, name, data); err != nil {
		server.logger.Error("failed to execute template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	buf.WriteTo(w)
}
