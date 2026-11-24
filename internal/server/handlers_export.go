package server

import (
	"mime"
	"net/http"
	"time"
)

// handleExportJSON sends the library and the wishlist of the user as a JSON file download. The file name holds the
// date of the export in the server time zone. It reads no input.
func (server *Server) handleExportJSON(w http.ResponseWriter, r *http.Request) {
	data, err := server.store.Export(r.Context(), currentUser(r).ID)
	if err != nil {
		server.logger.Error("failed to export", "error", err)
		server.renderServerError(w, r)
		return
	}
	exportedAt := time.Now()

	body, err := marshalExportJSON(newExportFile(data, server.version, exportedAt))
	if err != nil {
		server.logger.Error("failed to encode export", "error", err)
		server.renderServerError(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": exportFilename(exportedAt)}))
	_, _ = w.Write(body)
}

// handleExportAllJSON sends the library and the wishlist of every user as a JSON file download. The file name holds
// the date of the export in the server time zone. It reads no input.
func (server *Server) handleExportAllJSON(w http.ResponseWriter, r *http.Request) {
	users, err := server.store.ExportAll(r.Context())
	if err != nil {
		server.logger.Error("failed to export all users", "error", err)
		server.renderServerError(w, r)
		return
	}
	exportedAt := time.Now()

	body, err := marshalExportJSON(newExportAllFile(users, server.version, exportedAt))
	if err != nil {
		server.logger.Error("failed to encode export", "error", err)
		server.renderServerError(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": exportAllFilename(exportedAt)}))
	_, _ = w.Write(body)
}
