package server

import (
	"fmt"
	"html/template"

	"github.com/koenvanzuijlen/outpost/internal/igdb"
)

// templateFuncs returns a map of template functions for use in HTML templates.
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"derefInt64":  derefOrZero[int64],
		"ratingText":  ratingText,
		"ratingLabel": ratingLabel,
		"gameType":    gameType,
	}
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

// gameType converts a numeric game type into its string representation.
func gameType(t int64) string {
	return igdb.GameType(t).String()
}
