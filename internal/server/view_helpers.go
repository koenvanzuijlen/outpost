package server

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

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

// ratingLabel converts a numeric rating into an accessible textual representation.
func ratingLabel(rating int64) string {
	return ratingScore(rating) + " out of 5"
}

// outcomeLabel returns the badge text for a playthrough outcome: the label of the game status with the same value. The
// stored value "playing" reads oddly as an outcome, so it is shown as "In progress".
func outcomeLabel(outcome string) string {
	if outcome == string(database.PlaythroughOutcomePlaying) {
		return "In progress"
	}
	return gameStatusLabel(database.GameStatus(outcome))
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

// playthroughPlatformOptions returns the platforms for the platform select of the playthrough in entry: the owned
// platforms, followed by the platform of the playthrough if it is not owned. The platform of the playthrough must be
// in the list, because the browser selects the first option when no option matches, and saving would then clear the
// platform. The result never shares its backing array with owned.
func playthroughPlatformOptions(owned []dbgen.Platform, entry dbgen.ListPlaythroughsForUserGameIDRow) []dbgen.Platform {
	if entry.Playthrough.PlatformID == nil || entry.PlatformName == nil || entry.PlatformIgdbID == nil {
		return owned
	}
	if slices.ContainsFunc(owned, func(p dbgen.Platform) bool {
		return p.ID == *entry.Playthrough.PlatformID
	}) {
		return owned
	}

	return append(slices.Clip(owned), dbgen.Platform{
		ID:     *entry.Playthrough.PlatformID,
		Name:   *entry.PlatformName,
		IgdbID: *entry.PlatformIgdbID,
	})
}

// dateInputValue returns the date as the value of a date input (2006-01-02). It returns an empty string if the date is
// nil, so that the input is empty.
func dateInputValue(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
}

// playthroughAnchor returns the HTML id of the card of a playthrough, for example "playthrough-7". The redirects of the
// playthrough handlers use the id as the URL fragment.
func playthroughAnchor(id int64) string {
	return "playthrough-" + strconv.FormatInt(id, 10)
}
