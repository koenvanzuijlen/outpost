package server

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/auth"
	"github.com/koenvanzuijlen/outpost/internal/database"
)

// settingsPage is the data for settingsView. Saved is the message of a successful change. UsernameError, PasswordError
// and TimeZoneError are the messages of a failed change in that section. Username and TimeZone are the values of the
// username and time zone fields. ServerTimeZone is the name of the server time zone, or "" if it has no name.
type settingsPage struct {
	User           database.User
	Username       string
	TimeZone       string
	ServerTimeZone string
	Saved          string
	UsernameError  string
	PasswordError  string
	TimeZoneError  string
}

// savedMessages maps the saved query value to the message that the settings page shows after a successful change.
var savedMessages = map[string]string{
	"username":  "Username saved.",
	"password":  "Password changed. Other sessions are signed out.",
	"time-zone": "Time zone saved.",
}

// renderSettings renders the settings page of the signed-in user with status. It fills page.User and
// page.ServerTimeZone, page.Username when it is empty, and page.TimeZone when it is empty and no time zone error is
// set.
func (server *Server) renderSettings(w http.ResponseWriter, r *http.Request, status int, page settingsPage) {
	page.User = currentUser(r)
	if page.Username == "" {
		page.Username = page.User.Username
	}
	if page.TimeZone == "" && page.TimeZoneError == "" && page.User.TimeZone != nil {
		page.TimeZone = *page.User.TimeZone
	}
	page.ServerTimeZone = serverTimeZoneLabel()
	server.renderComponent(w, r, status, settingsView(page))
}

// handleSettings renders the settings of the signed-in user. It reads query value saved, and shows the message of a
// successful change for a known value.
func (server *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	saved := r.URL.Query().Get("saved")
	page := settingsPage{
		Saved: savedMessages[saved],
	}
	server.renderSettings(w, r, http.StatusOK, page)
}

// handleChangeUsername changes the username of the signed-in user. It reads form value username, and redirects to the
// settings page. The own username counts as success.
func (server *Server) handleChangeUsername(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)

	username := normalizeUsername(r.PostFormValue("username"))
	if !validUsername(username) {
		server.renderSettings(w, r, http.StatusBadRequest, settingsPage{
			Username:      username,
			UsernameError: usernameRule,
		})
		return
	}

	err := server.store.RenameUser(r.Context(), user.ID, username)
	if err != nil {
		if errors.Is(err, database.ErrUsernameTaken) {
			server.renderSettings(w, r, http.StatusBadRequest, settingsPage{
				Username:      username,
				UsernameError: "This username is taken",
			})
			return
		}
		server.logger.Error("failed to rename user", "error", err)
		server.renderServerError(w, r)
		return
	}

	http.Redirect(w, r, "/settings?saved=username", http.StatusSeeOther)
}

// handleChangePassword changes the password of the signed-in user, after it checks the current password. It ends all
// other sessions of the user. It reads form values current_password, new_password and new_password_confirm, and
// redirects to the settings page.
func (server *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)

	currentPassword := r.PostFormValue("current_password")
	newPassword := r.PostFormValue("new_password")
	newPasswordConfirm := r.PostFormValue("new_password_confirm")

	if problem := passwordProblem(newPassword); problem != "" {
		server.renderSettings(w, r, http.StatusBadRequest, settingsPage{
			PasswordError: problem,
		})
		return
	}

	if newPassword != newPasswordConfirm {
		server.renderSettings(w, r, http.StatusBadRequest, settingsPage{
			PasswordError: "The passwords do not match",
		})
		return
	}

	currentHash, err := server.store.PasswordHash(r.Context(), user.ID)
	if err != nil {
		server.logger.Error("failed to get password hash", "error", err)
		server.renderServerError(w, r)
		return
	}

	ok, err := auth.VerifyPassword(currentHash, currentPassword)
	if err != nil || !ok {
		server.renderSettings(w, r, http.StatusBadRequest, settingsPage{
			PasswordError: "The current password is wrong",
		})
		return
	}

	err = server.store.ChangePassword(r.Context(), user.ID, auth.HashPassword(newPassword), sessionTokenHash(r))
	if err != nil {
		server.logger.Error("failed to change password", "error", err)
		server.renderServerError(w, r)
		return
	}

	http.Redirect(w, r, "/settings?saved=password", http.StatusSeeOther)
}

// handleSetTimeZone stores the time zone of the signed-in user. An empty value selects the time zone of the server. It
// reads form value time_zone, and redirects to the settings page.
func (server *Server) handleSetTimeZone(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.PostFormValue("time_zone"))
	var timeZone *string
	if name != "" {
		_, err := time.LoadLocation(name)
		if err != nil || name == "Local" {
			server.renderSettings(w, r, http.StatusBadRequest, settingsPage{TimeZone: name, TimeZoneError: "Choose a time zone from the list"})
			return
		}
		timeZone = &name
	}

	err := server.store.SetUserTimeZone(r.Context(), currentUser(r).ID, timeZone)
	if err != nil {
		server.logger.Error("failed to set time zone", "error", err)
		server.renderServerError(w, r)
		return
	}

	http.Redirect(w, r, "/settings?saved=time-zone", http.StatusSeeOther)
}

// serverTimeZoneLabel returns the name of the server time zone for the settings page, or "" if Go knows the zone only
// as Local.
func serverTimeZoneLabel() string {
	local := time.Local.String()
	if local == "Local" {
		return ""
	}
	return local
}
