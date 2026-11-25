// Package server implements Outpost's HTTP layer: routing, request handlers, and response rendering.
package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database"
	"github.com/koenvanzuijlen/outpost/internal/igdb"
)

// IGDBClient is the subset of the IGDB API the server uses. *igdb.Client implements it.
type IGDBClient interface {
	SearchGames(ctx context.Context, query string) ([]igdb.Game, error)
	GameByID(ctx context.Context, id int) (igdb.Game, error)
	DLCs(ctx context.Context, parentID int) ([]igdb.Game, error)
}

// Config holds the settings of a Server. Version is shown in the footer of every page. SecureCookies sets the Secure
// flag on the session cookie, so the browser sends it only over HTTPS.
type Config struct {
	Version       string
	SecureCookies bool
}

// Server holds the dependencies shared by all HTTP handlers and routes requests to them. It implements http.Handler.
type Server struct {
	mux           *http.ServeMux
	handler       http.Handler
	logger        *slog.Logger
	store         *database.Store
	igdbClient    IGDBClient
	version       string
	secureCookies bool
	// signInThrottle counts the sign-in attempts per username.
	signInThrottle *signInThrottle
	// now returns the current time. Tests replace it with a fixed time.
	now func() time.Time
}

// New creates a Server with the given logger, store, IGDB client and config. New registers the routes and the
// middleware.
func New(logger *slog.Logger, store *database.Store, igdbClient IGDBClient, config Config) *Server {
	server := &Server{
		mux:            http.NewServeMux(),
		logger:         logger,
		store:          store,
		igdbClient:     igdbClient,
		version:        config.Version,
		secureCookies:  config.SecureCookies,
		signInThrottle: newSignInThrottle(),
		now:            time.Now,
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
