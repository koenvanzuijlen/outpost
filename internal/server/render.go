package server

import (
	"bytes"
	"net/http"

	"github.com/a-h/templ"
)

// renderComponent renders component and writes the result with the given status code. If rendering fails, it logs the
// error and responds with a plain-text 500, without writing a partial page. The fallback is plain text, so a failure
// of the error page itself cannot loop.
func (server *Server) renderComponent(w http.ResponseWriter, r *http.Request, status int, component templ.Component) {
	var buf bytes.Buffer
	if err := component.Render(withVersion(r.Context(), server.version), &buf); err != nil {
		server.logger.Error("failed to render component", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = buf.WriteTo(w)
}
