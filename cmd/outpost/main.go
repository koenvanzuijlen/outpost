// Command outpost runs the Outpost game-tracking server.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database"
	"github.com/koenvanzuijlen/outpost/internal/igdb"
	"github.com/koenvanzuijlen/outpost/internal/server"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	if err := run(logger); err != nil {
		logger.Error("Fatal error", "error", err)
		os.Exit(1)
	}
}

// run starts the Outpost server and handles graceful shutdown.
func run(logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dbPath := envOrDefault("DATABASE_PATH", "outpost.db")
	db, err := database.Open(ctx, logger, dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer func() {
		_ = db.Close()
	}()

	clientID, ok := os.LookupEnv("TWITCH_CLIENT_ID")
	if !ok || clientID == "" {
		return errors.New("TWITCH_CLIENT_ID is required (see .env.example)")
	}
	clientSecret, ok := os.LookupEnv("TWITCH_CLIENT_SECRET")
	if !ok || clientSecret == "" {
		return errors.New("TWITCH_CLIENT_SECRET is required (see .env.example)")
	}
	client := igdb.New(clientID, clientSecret)

	app, err := server.New(logger, db, client)
	if err != nil {
		return fmt.Errorf("create server: %w", err)
	}

	port := envOrDefault("PORT", "8080")
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           app,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	logger.Info("Server listening", "addr", ln.Addr())

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ln)
	}()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("listen: %w", err)
		}
	case <-ctx.Done():
		logger.Info("Server shutting down")
		// Shutdown the server with a timeout for ongoing requests.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}
	}
	return nil
}

// envOrDefault returns the value of the environment variable specified by key, or the provided defaultValue if the
// environment variable is not set.
func envOrDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
