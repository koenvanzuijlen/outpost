package server

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/auth"
	"github.com/koenvanzuijlen/outpost/internal/database"
)

// settingsAccountPage is the data for settingsAccountView. Saved is the message of a successful change.
// UsernameError and PasswordError are the messages of a failed change in that section. Username is the value of the
// username field.
type settingsAccountPage struct {
	Saved         string
	User          database.User
	Username      string
	UsernameError string
	PasswordError string
}

// accountSavedMessages maps the saved query value to the message that the account page shows after a successful change.
var accountSavedMessages = map[string]string{
	"username": "Username saved.",
	"password": "Password changed. Other sessions are signed out.",
}

// renderSettingsAccount renders the account page of the settings of the signed-in user with status. It fills
// page.User, and page.Username when it is empty.
func (server *Server) renderSettingsAccount(w http.ResponseWriter, r *http.Request, status int, page settingsAccountPage) {
	page.User = currentUser(r)
	if page.Username == "" {
		page.Username = page.User.Username
	}
	server.renderComponent(w, r, status, settingsAccountView(page))
}

// handleSettingsAccount renders the account page of the settings. It reads query value saved, and shows the message
// of a successful change for a known value.
func (server *Server) handleSettingsAccount(w http.ResponseWriter, r *http.Request) {
	saved := r.URL.Query().Get("saved")
	page := settingsAccountPage{
		Saved: accountSavedMessages[saved],
	}
	server.renderSettingsAccount(w, r, http.StatusOK, page)
}

// handleChangeUsername changes the username of the signed-in user. It reads form value username, and redirects to the
// account page of the settings. The own username counts as success.
func (server *Server) handleChangeUsername(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)

	username := normalizeUsername(r.PostFormValue("username"))
	if !validUsername(username) {
		server.renderSettingsAccount(w, r, http.StatusBadRequest, settingsAccountPage{
			Username:      username,
			UsernameError: usernameRule,
		})
		return
	}

	err := server.store.RenameUser(r.Context(), user.ID, username)
	if err != nil {
		if errors.Is(err, database.ErrUsernameTaken) {
			server.renderSettingsAccount(w, r, http.StatusBadRequest, settingsAccountPage{
				Username:      username,
				UsernameError: "This username is taken",
			})
			return
		}
		server.logger.Error("failed to rename user", "error", err)
		server.renderServerError(w, r)
		return
	}

	http.Redirect(w, r, "/settings/account?saved=username", http.StatusSeeOther)
}

// handleChangePassword changes the password of the signed-in user, after it checks the current password. It ends all
// other sessions of the user. It reads form values current_password, new_password and new_password_confirm, and
// redirects to the account page of the settings.
func (server *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)

	currentPassword := r.PostFormValue("current_password")
	newPassword := r.PostFormValue("new_password")
	newPasswordConfirm := r.PostFormValue("new_password_confirm")

	if problem := passwordProblem(newPassword); problem != "" {
		server.renderSettingsAccount(w, r, http.StatusBadRequest, settingsAccountPage{
			PasswordError: problem,
		})
		return
	}

	if newPassword != newPasswordConfirm {
		server.renderSettingsAccount(w, r, http.StatusBadRequest, settingsAccountPage{
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
		server.renderSettingsAccount(w, r, http.StatusBadRequest, settingsAccountPage{
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

	http.Redirect(w, r, "/settings/account?saved=password", http.StatusSeeOther)
}

// settingsPreferencesPage is the data for settingsPreferencesView. Saved is the message of a successful change.
// TimeZoneError is the message of a failed time zone change. TimeZone is the value of the time zone field.
// ServerTimeZone is the name of the server time zone, or "" if it has no name.
type settingsPreferencesPage struct {
	Saved          string
	TimeZone       string
	ServerTimeZone string
	TimeZoneError  string
}

// preferencesSavedMessages maps the saved query value to the message that the preferences page shows after a
// successful change.
var preferencesSavedMessages = map[string]string{
	"time-zone": "Time zone saved.",
}

// renderSettingsPreferences renders the preferences page of the settings of the signed-in user with status. It fills
// page.ServerTimeZone, and page.TimeZone when it is empty and no time zone error is set.
func (server *Server) renderSettingsPreferences(w http.ResponseWriter, r *http.Request, status int, page settingsPreferencesPage) {
	user := currentUser(r)
	if page.TimeZone == "" && page.TimeZoneError == "" && user.TimeZone != nil {
		page.TimeZone = *user.TimeZone
	}
	page.ServerTimeZone = serverTimeZoneLabel()
	server.renderComponent(w, r, status, settingsPreferencesView(page))
}

// handleSettingsPreferences renders the preferences page of the settings. It reads query value saved, and shows the
// message of a successful change for a known value.
func (server *Server) handleSettingsPreferences(w http.ResponseWriter, r *http.Request) {
	saved := r.URL.Query().Get("saved")
	page := settingsPreferencesPage{
		Saved: preferencesSavedMessages[saved],
	}
	server.renderSettingsPreferences(w, r, http.StatusOK, page)
}

// handleSetTimeZone stores the time zone of the signed-in user. An empty value selects the time zone of the server. It
// reads form value time_zone, and redirects to the preferences page of the settings.
func (server *Server) handleSetTimeZone(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.PostFormValue("time_zone"))
	var timeZone *string
	if name != "" {
		_, err := time.LoadLocation(name)
		if err != nil || name == "Local" {
			server.renderSettingsPreferences(w, r, http.StatusBadRequest, settingsPreferencesPage{TimeZone: name, TimeZoneError: "Choose a time zone from the list"})
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

	http.Redirect(w, r, "/settings/preferences?saved=time-zone", http.StatusSeeOther)
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
