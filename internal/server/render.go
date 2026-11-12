package server

import (
	"bytes"
	"net/http"

	"github.com/a-h/templ"
)

// renderComponent renders component and writes the result. If that fails, it logs the error and responds with a 500,
// without writing a partial page.
func (server *Server) renderComponent(w http.ResponseWriter, r *http.Request, component templ.Component) {
	var buf bytes.Buffer
	if err := component.Render(withVersion(r.Context(), server.version), &buf); err != nil {
		server.logger.Error("failed to render component", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = buf.WriteTo(w)
}
