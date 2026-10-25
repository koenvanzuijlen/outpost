package server

import (
	"fmt"
	"html/template"
)

// templateFuncs returns a map of template functions for use in HTML templates.
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"ratingText":  ratingText,
		"ratingLabel": ratingLabel,
		"derefInt64":  derefOrZero[int64],
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
