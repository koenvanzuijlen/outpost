package server

import (
	"fmt"
	"strings"

	"github.com/koenvanzuijlen/outpost/internal/database"
	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
)

// gameURL returns the path of the game page for the game with the given IGDB ID.
func gameURL(igdbID int64) string {
	return fmt.Sprintf("/games/%d", igdbID)
}

// ratingScore returns the rating as a number of stars from 0.5 to 5, without trailing zeros ("4.5", "5").
func ratingScore(rating int64) string {
	starCount := float64(rating) / 2
	return fmt.Sprintf("%g", starCount)
}

// ratingText converts a numeric rating into a textual star representation.
func ratingText(rating int64) string {
	return ratingScore(rating) + " ⭐"
}

// ratingLabel converts a numeric rating into an accessible textual representation.
func ratingLabel(rating int64) string {
	return ratingScore(rating) + " out of 5"
}

// outcomeLabel returns the display text for a playthrough outcome. The stored value "playing" reads oddly as an
// outcome, so it is shown as "in progress"; other outcomes are shown as they are.
func outcomeLabel(outcome string) string {
	if outcome == string(database.PlaythroughOutcomePlaying) {
		return "in progress"
	}
	return outcome
}

// platformNames returns the names of the platforms, separated by commas, for example "PC, Nintendo Switch".
func platformNames(platforms []dbgen.Platform) string {
	names := make([]string, 0, len(platforms))
	for _, platform := range platforms {
		names = append(names, platform.Name)
	}
	return strings.Join(names, ", ")
}

// dlcTitle returns the title of a DLC without the title of its parent game and the separator after it, for example
// "Hearts of Stone" for "The Witcher 3: Wild Hunt - Hearts of Stone". It returns the full title if the title does not
// start with the parent title and a separator, or if nothing is left after them.
func dlcTitle(parentTitle, title string) string {
	rest, ok := strings.CutPrefix(title, parentTitle)
	if !ok {
		return title
	}
	trimmed := strings.TrimLeft(rest, " -–:")
	if trimmed == rest || trimmed == "" {
		return title
	}
	return trimmed
}

// copySummary returns the title of a copy row: the platform name, followed by the number of owned DLC if the user owns
// at least one DLC on the copy, for example "PC (1 DLC owned)".
func copySummary(platformName string, dlcs []database.CopyDLC) string {
	ownedCount := 0
	for _, dlc := range dlcs {
		if dlc.Owned {
			ownedCount++
		}
	}
	if ownedCount == 0 {
		return platformName
	}
	return fmt.Sprintf("%s (%d DLC owned)", platformName, ownedCount)
}
