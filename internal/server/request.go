package server

import (
	"net/http"
	"strconv"
)

// pathID extracts an integer ID from the URL path parameters. It returns the ID and a boolean indicating success.
func pathID(r *http.Request, name string) (int64, bool) {
	value, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || value <= 0 {
		return 0, false
	}
	return value, true
}
