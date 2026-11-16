package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database"
	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
)

// exportFormatVersion is the version of the export file format. It changes when a change to the format can break a
// reader of older files.
const exportFormatVersion = 1

// exportFile is the content of the JSON export file.
type exportFile struct {
	FormatVersion  int                  `json:"format_version"`
	OutpostVersion string               `json:"outpost_version"`
	ExportedAt     time.Time            `json:"exported_at"`
	Library        []exportLibraryGame  `json:"library"`
	Wishlist       []exportWishlistGame `json:"wishlist"`
}

// exportGame holds the catalog fields of a game in the export. The library and wishlist types embed it, so its fields
// appear at their level.
type exportGame struct {
	IGDBID       int64   `json:"igdb_id"`
	Title        string  `json:"title"`
	ReleasedOn   *string `json:"released_on"`
	ParentIGDBID *int64  `json:"parent_igdb_id"`
}

// exportLibraryGame is a library game or library DLC in the export.
type exportLibraryGame struct {
	exportGame
	Status       string              `json:"status"`
	Rating       *int64              `json:"rating"`
	Notes        *string             `json:"notes"`
	AddedAt      time.Time           `json:"added_at"`
	Copies       []exportCopy        `json:"copies"`
	Playthroughs []exportPlaythrough `json:"playthroughs"`
}

// exportWishlistGame is a wishlisted game in the export.
type exportWishlistGame struct {
	exportGame
	AddedAt time.Time `json:"added_at"`
}

// exportPlatform is a platform in the export.
type exportPlatform struct {
	IGDBID int64  `json:"igdb_id"`
	Name   string `json:"name"`
}

// exportCopy is an owned copy in the export, with the DLC owned on it.
type exportCopy struct {
	Platform exportPlatform `json:"platform"`
	OwnedDLC []exportDLC    `json:"owned_dlc"`
}

// exportDLC is a DLC owned on a copy in the export.
type exportDLC struct {
	IGDBID int64  `json:"igdb_id"`
	Title  string `json:"title"`
}

// exportPlaythrough is a playthrough in the export.
type exportPlaythrough struct {
	Name       *string         `json:"name"`
	Platform   *exportPlatform `json:"platform"`
	Outcome    string          `json:"outcome"`
	StartedOn  *string         `json:"started_on"`
	FinishedOn *string         `json:"finished_on"`
}

// newExportFile converts the user data of the store into the export file format. ExportedAt is exportedAt in UTC,
// truncated to whole seconds. Every list in the result is non-nil, so it encodes as [] when it is empty.
func newExportFile(data database.ExportData, version string, exportedAt time.Time) exportFile {
	library := make([]exportLibraryGame, 0, len(data.Library))
	for _, entry := range data.Library {
		library = append(library, newExportLibraryGame(entry))
	}

	wishlist := make([]exportWishlistGame, 0, len(data.Wishlist))
	for _, entry := range data.Wishlist {
		wishlist = append(wishlist, newExportWishlistGame(entry))
	}

	return exportFile{
		FormatVersion:  exportFormatVersion,
		OutpostVersion: version,
		ExportedAt:     exportedAt.UTC().Truncate(time.Second),
		Library:        library,
		Wishlist:       wishlist,
	}
}

// newExportGame returns the catalog fields of game for the export.
func newExportGame(game dbgen.Game) exportGame {
	return exportGame{
		IGDBID:       game.IgdbID,
		Title:        game.Title,
		ReleasedOn:   exportDate(game.ReleasedAt),
		ParentIGDBID: game.ParentIgdbID,
	}
}

// newExportLibraryGame converts a library entry for the export.
func newExportLibraryGame(entry database.ExportGame) exportLibraryGame {
	copies := make([]exportCopy, 0, len(entry.Copies))
	for _, item := range entry.Copies {
		dlc := make([]exportDLC, 0, len(item.OwnedDLC))
		for _, d := range item.OwnedDLC {
			dlc = append(dlc, exportDLC{
				IGDBID: d.IgdbID,
				Title:  d.Title,
			})
		}
		copies = append(copies, exportCopy{
			Platform: exportPlatform{
				IGDBID: item.Platform.IgdbID,
				Name:   item.Platform.Name,
			},
			OwnedDLC: dlc,
		})
	}

	playthroughs := make([]exportPlaythrough, 0, len(entry.Playthroughs))
	for _, pt := range entry.Playthroughs {
		playthroughs = append(playthroughs, newExportPlaythrough(pt))
	}

	return exportLibraryGame{
		exportGame:   newExportGame(entry.Game),
		Status:       derefOrZero(entry.UserGame.Status),
		Rating:       entry.UserGame.Rating,
		Notes:        entry.UserGame.Notes,
		AddedAt:      entry.UserGame.AddedAt,
		Copies:       copies,
		Playthroughs: playthroughs,
	}
}

// newExportWishlistGame converts a wishlist entry for the export.
func newExportWishlistGame(entry database.ExportGame) exportWishlistGame {
	return exportWishlistGame{
		exportGame: newExportGame(entry.Game),
		AddedAt:    entry.UserGame.AddedAt,
	}
}

// newExportPlaythrough converts a playthrough for the export. Platform is nil for a playthrough without a platform.
func newExportPlaythrough(row dbgen.ListPlaythroughsForUserIDRow) exportPlaythrough {
	var platform *exportPlatform
	if row.PlatformIgdbID != nil {
		platform = &exportPlatform{IGDBID: *row.PlatformIgdbID, Name: derefOrZero(row.PlatformName)}
	}

	return exportPlaythrough{
		Name:       row.Playthrough.Name,
		Platform:   platform,
		Outcome:    row.Playthrough.Outcome,
		StartedOn:  exportDate(row.Playthrough.StartedAt),
		FinishedOn: exportDate(row.Playthrough.FinishedAt),
	}
}

// exportDate formats a calendar date as YYYY-MM-DD in UTC, the time zone in which the database stores dates.
// Returns nil if t is nil.
func exportDate(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format(time.DateOnly)
	return &s
}

// marshalExportFile encodes file as JSON with an indent of two spaces and a final newline. It does not escape HTML
// characters, because the file is not HTML.
func marshalExportFile(file exportFile) ([]byte, error) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	err := enc.Encode(file)
	if err != nil {
		return nil, fmt.Errorf("encode export file: %w", err)
	}
	return buf.Bytes(), nil
}

// exportFilename returns the name of the export file, with the date of exportedAt in its own time zone:
// outpost-2026-11-15.json.
func exportFilename(exportedAt time.Time) string {
	return "outpost-" + exportedAt.Format(time.DateOnly) + ".json"
}
