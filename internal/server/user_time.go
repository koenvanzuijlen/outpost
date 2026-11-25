package server

import (
	"net/http"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database"
)

// userLocation returns the time zone of user. It returns the time zone of the server if the user has none, or if the
// stored name does not load.
func userLocation(user database.User) *time.Location {
	if user.TimeZone == nil {
		return time.Local
	}
	loc, err := time.LoadLocation(*user.TimeZone)
	if err != nil {
		return time.Local
	}
	return loc
}

// userNow returns the current time in the time zone of the signed-in user, so its date is "today" for that user.
func (server *Server) userNow(r *http.Request) time.Time {
	return server.now().In(userLocation(currentUser(r)))
}
