package server

import (
	"embed"
	"io/fs"
	"net/http"
)

// staticFiles holds the files in the static folder. They are part of the binary.
//
//go:embed static
var staticFiles embed.FS

// blockDirsFS wraps a file system and hides its directories. A file server on top of it cannot list files.
type blockDirsFS struct {
	fs.FS
}

// Open opens the named file. Returns fs.ErrNotExist if the name is a directory, or the error of the wrapped file
// system.
func (fsys blockDirsFS) Open(name string) (fs.File, error) {
	f, err := fsys.FS.Open(name)
	if err != nil {
		return nil, err
	}
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if fi.IsDir() {
		_ = f.Close()
		return nil, fs.ErrNotExist
	}
	return f, nil
}

// staticHandler returns the handler that serves the files in the static folder under /static/. It sets a
// Cache-Control header on every response and answers 404 for directories.
func (server *Server) staticHandler() http.Handler {
	subFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		// The path is constant, so fs.Sub cannot fail.
		panic(err)
	}

	staticServer := http.StripPrefix("/static/", http.FileServerFS(blockDirsFS{FS: subFS}))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=3600")
		staticServer.ServeHTTP(w, r)
	})
}
