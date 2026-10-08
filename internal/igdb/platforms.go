package igdb

// Platform represents a platform retrieved from the IGDB API.
type Platform struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}
