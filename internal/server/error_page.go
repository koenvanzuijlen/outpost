package server

import "net/http"

// errorPage is the data for errorView: the HTTP status code and the message that the handler wrote.
type errorPage struct {
	Status  int
	Message string
}

// renderError renders the error page with status and message in the page layout. The handler writes the message, so
// it never contains internal error details.
func (server *Server) renderError(w http.ResponseWriter, r *http.Request, status int, message string) {
	server.renderComponent(w, r, status, errorView(errorPage{
		Status:  status,
		Message: message,
	}))
}

// renderServerError renders the error page for a 500 with a generic message. The caller logs the details first.
func (server *Server) renderServerError(w http.ResponseWriter, r *http.Request) {
	server.renderError(w, r, http.StatusInternalServerError, "Something went wrong")
}

// handleNotFound renders the error page with a 404 for every GET request that no other route matches. It reads no
// request values.
func (server *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	server.renderError(w, r, http.StatusNotFound, "Page not found")
}
