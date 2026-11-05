package server

import (
	"fmt"

	"github.com/koenvanzuijlen/outpost/internal/database"
)

// gameURL returns the path of the game page for the game with the given IGDB ID.
func gameURL(igdbID int64) string {
	return fmt.Sprintf("/games/%d", igdbID)
}

// ratingText converts a numeric rating into a textual star representation.
func ratingText(rating int64) string {
	starCount := float64(rating) / 2
	return fmt.Sprintf("%g ⭐", starCount)
}

// ratingLabel converts a numeric rating into an accessible textual representation.
func ratingLabel(rating int64) string {
	starCount := float64(rating) / 2
	return fmt.Sprintf("%g out of 5", starCount)
}

// outcomeLabel returns the display text for a playthrough outcome. The stored value "playing" reads oddly as an
// outcome, so it is shown as "in progress"; other outcomes are shown as they are.
func outcomeLabel(outcome string) string {
	if outcome == string(database.PlaythroughOutcomePlaying) {
		return "in progress"
	}
	return outcome
}
