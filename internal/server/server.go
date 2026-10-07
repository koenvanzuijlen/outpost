// Package server implements Outpost's HTTP layer: routing, request handlers, and response rendering.
package server

import (
	"bytes"
	"context"
	"database/sql"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"path"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
	"github.com/koenvanzuijlen/outpost/internal/igdb"
)

// defaultUserID is a hardcoded user ID until user management is implemented.
const defaultUserID = 1

// IGDBClient is the subset of the IGDB API the server uses. *igdb.Client implements it.
type IGDBClient interface {
	SearchGames(ctx context.Context, query string) ([]igdb.Game, error)
	GameByID(ctx context.Context, id int) (igdb.Game, error)
}

// Server holds the dependencies shared by all HTTP handlers and routes requests to them. It implements http.Handler.
type Server struct {
	mux        *http.ServeMux
	handler    http.Handler
	logger     *slog.Logger
	templates  map[string]*template.Template
	db         *sql.DB
	queries    *dbgen.Queries
	igdbClient IGDBClient
}

//go:embed templates
var templateFS embed.FS

// New creates a Server with the given logger, database connection, and IGDB client. It parses the templates, sets up
// the routes, and returns the ready-to-serve Server instance.
func New(logger *slog.Logger, db *sql.DB, igdbClient IGDBClient) (*Server, error) {
	pages, err := fs.Glob(templateFS, "templates/pages/*.html")
	if err != nil {
		return nil, fmt.Errorf("glob page templates: %w", err)
	}

	partials, err := fs.Glob(templateFS, "templates/partials/*.html")
	if err != nil {
		return nil, fmt.Errorf("glob partial templates: %w", err)
	}

	templates := make(map[string]*template.Template)
	for _, page := range pages {
		files := []string{"templates/base.html"}
		files = append(files, partials...)
		files = append(files, page)
		tmpl, err := template.ParseFS(templateFS, files...)
		if err != nil {
			return nil, fmt.Errorf("parse template %s: %w", page, err)
		}
		templates[path.Base(page)] = tmpl
	}

	server := &Server{
		mux:        http.NewServeMux(),
		logger:     logger,
		templates:  templates,
		db:         db,
		queries:    dbgen.New(db),
		igdbClient: igdbClient,
	}
	server.routes()
	server.handler = server.requestLogging(server.recoverPanic(server.mux))
	return server, nil
}

// ServeHTTP passes the request through the middleware to the routers.
func (server *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	server.handler.ServeHTTP(w, r)
}

// render executes the specified template with the given data and writes the result to the response writer. If an error
// occurs, it logs the error and returns a 500 status code.
func (server *Server) render(w http.ResponseWriter, name string, data any) {
	var buf bytes.Buffer
	tmpl, ok := server.templates[name]
	if !ok {
		server.logger.Error("template not found", "name", name)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(&buf, "base.html", data); err != nil {
		server.logger.Error("failed to execute template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = buf.WriteTo(w)
}
