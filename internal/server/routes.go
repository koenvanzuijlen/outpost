package server

// routes registers every HTTP route the server handles. It is called by New and must not be called again on a Server
// that is already serving.
func (server *Server) routes() {
	server.mux.HandleFunc("GET /{$}", server.handleHome)
	server.mux.HandleFunc("GET /search", server.handleSearch)
	server.mux.HandleFunc("GET /library", server.handleLibrary)
	server.mux.HandleFunc("GET /wishlist", server.handleWishlist)
	server.mux.HandleFunc("GET /games/{igdbID}", server.handleGame)
	server.mux.Handle("GET /static/", server.staticHandler())
	server.mux.HandleFunc("GET /health", server.handleHealth)
	server.mux.HandleFunc("GET /", server.handleNotFound)

	server.mux.HandleFunc("POST /games/{igdbID}/wishlist", server.handleAddToWishlist)
	server.mux.HandleFunc("POST /games/{igdbID}/wishlist/remove", server.handleRemoveFromWishlist)
	server.mux.HandleFunc("POST /games/{igdbID}/library", server.handleAddToLibrary)
	server.mux.HandleFunc("POST /games/{igdbID}/library/remove", server.handleRemoveFromLibrary)
	server.mux.HandleFunc("POST /games/{igdbID}/details", server.handleUpdateGameDetails)
	server.mux.HandleFunc("POST /games/{igdbID}/copies", server.handleAddOwnedCopy)
	server.mux.HandleFunc("POST /games/{igdbID}/copies/{platformIGDBID}/remove", server.handleRemoveOwnedCopy)
	server.mux.HandleFunc("POST /games/{igdbID}/copies/{platformIGDBID}/dlcs/{dlcIGDBID}", server.handleSetDLCOwned)
	server.mux.HandleFunc("POST /games/{igdbID}/playthroughs", server.handleStartPlaythrough)
	server.mux.HandleFunc("POST /games/{igdbID}/playthroughs/{playthroughID}", server.handleUpdatePlaythrough)
	server.mux.HandleFunc("POST /games/{igdbID}/playthroughs/{playthroughID}/outcome", server.handleSetPlaythroughOutcome)
	server.mux.HandleFunc("POST /games/{igdbID}/playthroughs/{playthroughID}/delete", server.handleDeletePlaythrough)
}
