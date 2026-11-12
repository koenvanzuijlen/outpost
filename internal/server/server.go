// Package server implements Outpost's HTTP layer: routing, request handlers, and response rendering.
package server

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/koenvanzuijlen/outpost/internal/database"
	"github.com/koenvanzuijlen/outpost/internal/igdb"
)

// defaultUserID is a hardcoded user ID until user management is implemented.
const defaultUserID = 1

// IGDBClient is the subset of the IGDB API the server uses. *igdb.Client implements it.
type IGDBClient interface {
	SearchGames(ctx context.Context, query string) ([]igdb.Game, error)
	GameByID(ctx context.Context, id int) (igdb.Game, error)
	DLCs(ctx context.Context, parentID int) ([]igdb.Game, error)
}

// Server holds the dependencies shared by all HTTP handlers and routes requests to them. It implements http.Handler.
type Server struct {
	mux        *http.ServeMux
	handler    http.Handler
	logger     *slog.Logger
	store      *database.Store
	igdbClient IGDBClient
	version    string
}

// New creates a Server with the given logger, store and IGDB client. Every page shows version in the footer. New
// registers the routes and the middleware.
func New(logger *slog.Logger, store *database.Store, igdbClient IGDBClient, version string) *Server {
	server := &Server{
		version:    version,
		mux:        http.NewServeMux(),
		logger:     logger,
		store:      store,
		igdbClient: igdbClient,
	}
	server.routes()

	crossOrigin := http.NewCrossOriginProtection()
	server.handler = chainMiddleware(
		server.mux,
		crossOrigin.Handler,
		server.recoverPanic,
		server.requestLogging,
		contentSecurityPolicy,
	)
	return server
}

// ServeHTTP passes the request through the middleware to the routes.
func (server *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	server.handler.ServeHTTP(w, r)
}
