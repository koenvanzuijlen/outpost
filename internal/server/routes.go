package server

// routes registers every HTTP route the server handles. It is called by New and must not be called again on a Server
// that is already serving.
func (server *Server) routes() {
	server.mux.HandleFunc("GET /{$}", server.handleIndex)
	server.mux.HandleFunc("GET /search", server.handleSearch)
	server.mux.HandleFunc("GET /health", server.handleHealth)

	server.mux.HandleFunc("POST /collection", server.handleAddToCollection)
}
