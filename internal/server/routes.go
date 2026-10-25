package server

// routes registers every HTTP route the server handles. It is called by New and must not be called again on a Server
// that is already serving.
func (server *Server) routes() {
	server.mux.HandleFunc("GET /{$}", server.handleLibrary)
	server.mux.HandleFunc("GET /search", server.handleSearch)
	server.mux.HandleFunc("GET /games/{igdbID}", server.handleGame)
	server.mux.HandleFunc("GET /health", server.handleHealth)

	server.mux.HandleFunc("POST /wishlist", server.handleAddToWishlist)
	server.mux.HandleFunc("POST /collection", server.handleAddToCollection)
	server.mux.HandleFunc("POST /games/{igdbID}", server.handleUpdateGame)
	server.mux.HandleFunc("POST /games/{igdbID}/playthroughs", server.handleStartPlaythrough)
	server.mux.HandleFunc("POST /games/{igdbID}/playthroughs/{playthroughID}", server.handleUpdatePlaythrough)
	server.mux.HandleFunc("POST /games/{igdbID}/playthroughs/{playthroughID}/outcome", server.handleSetPlaythroughOutcome)
	server.mux.HandleFunc("POST /games/{igdbID}/playthroughs/{playthroughID}/delete", server.handleDeletePlaythrough)
}
