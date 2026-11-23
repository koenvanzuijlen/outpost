package server

import (
	"errors"
	"net/http"

	"github.com/koenvanzuijlen/outpost/internal/auth"
	"github.com/koenvanzuijlen/outpost/internal/database"
)

// adminUsersPage is the data for adminUsersView. Users are all users, sorted by username. SelfID is the ID of the
// signed-in admin. Error is the message of a failed action. NewUsername and NewRole are the values of the form
// "Create user" after a failed attempt.
type adminUsersPage struct {
	Users       []database.User
	SelfID      int64
	Error       string
	NewUsername string
	NewRole     database.UserRole
}

// renderAdminUsers renders the admin page with all users and status. It fills page.Users and page.SelfID, and sets
// page.NewRole to the role user when it is empty.
func (server *Server) renderAdminUsers(w http.ResponseWriter, r *http.Request, status int, page adminUsersPage) {
	users, err := server.store.ListUsers(r.Context())
	if err != nil {
		server.logger.Error("failed to list users", "error", err)
		server.renderServerError(w, r)
		return
	}

	page.Users = users
	page.SelfID = currentUser(r).ID
	if page.NewRole == "" {
		page.NewRole = database.UserRoleUser
	}

	server.renderComponent(w, r, status, adminUsersView(page))
}

// handleAdminUsers renders all users with the forms to manage them. It reads no input.
func (server *Server) handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	server.renderAdminUsers(w, r, http.StatusOK, adminUsersPage{})
}

// handleCreateUser creates a user. It reads form values username, password and role, and redirects to the admin page.
func (server *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	username := normalizeUsername(r.PostFormValue("username"))
	password := r.PostFormValue("password")
	role := database.UserRole(r.PostFormValue("role"))

	page := adminUsersPage{
		NewUsername: username,
		NewRole:     role,
	}

	if !validUsername(username) {
		page.Error = usernameRule
		server.renderAdminUsers(w, r, http.StatusBadRequest, page)
		return
	}

	if problem := passwordProblem(password); problem != "" {
		page.Error = problem
		server.renderAdminUsers(w, r, http.StatusBadRequest, page)
		return
	}

	if !role.Valid() {
		page.Error = "Choose a valid role"
		server.renderAdminUsers(w, r, http.StatusBadRequest, page)
		return
	}

	_, err := server.store.CreateUser(r.Context(), username, auth.HashPassword(password), role)
	if err != nil {
		if errors.Is(err, database.ErrUsernameTaken) {
			page.Error = "This username is taken"
			server.renderAdminUsers(w, r, http.StatusBadRequest, page)
			return
		}
		server.logger.Error("failed to create user", "error", err)
		server.renderServerError(w, r)
		return
	}

	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

// adminTargetUserID returns the user ID in the path of an admin action. It renders a 404 for an ID that is not valid.
// When the ID is the own account and ownAccountError is not empty, it renders the admin page with ownAccountError and
// a 400. ok is false after an error page.
func (server *Server) adminTargetUserID(w http.ResponseWriter, r *http.Request, ownAccountError string) (int64, bool) {
	userID, ok := pathID(r, "userID")
	if !ok {
		server.renderError(w, r, http.StatusNotFound, "User not found")
		return 0, false
	}

	if ownAccountError != "" && userID == currentUser(r).ID {
		server.renderAdminUsers(w, r, http.StatusBadRequest, adminUsersPage{
			Error: ownAccountError,
		})
		return 0, false
	}

	return userID, true
}

// handleAdminRenameUser changes the username of a user. It reads path value userID and form value username, and
// redirects to the admin page. Renders a 404 for an unknown user.
func (server *Server) handleAdminRenameUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := server.adminTargetUserID(w, r, "")
	if !ok {
		return
	}

	username := normalizeUsername(r.PostFormValue("username"))
	if !validUsername(username) {
		server.renderAdminUsers(w, r, http.StatusBadRequest, adminUsersPage{
			Error: usernameRule,
		})
		return
	}

	err := server.store.RenameUser(r.Context(), userID, username)
	if err != nil {
		if errors.Is(err, database.ErrUsernameTaken) {
			server.renderAdminUsers(w, r, http.StatusBadRequest, adminUsersPage{
				Error: "This username is taken",
			})
			return
		}
		if errors.Is(err, database.ErrUserNotFound) {
			server.renderAdminUsers(w, r, http.StatusNotFound, adminUsersPage{
				Error: "User not found",
			})
			return
		}
		server.logger.Error("failed to rename user", "error", err)
		server.renderServerError(w, r)
		return
	}

	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

// handleSetUserRole sets the role of another user. It reads path value userID and form value role, and redirects to
// the admin page. The current role counts as success. Renders a 404 for an unknown user.
func (server *Server) handleSetUserRole(w http.ResponseWriter, r *http.Request) {
	userID, ok := server.adminTargetUserID(w, r, "You cannot change your own role")
	if !ok {
		return
	}

	role := database.UserRole(r.PostFormValue("role"))
	if !role.Valid() {
		server.renderAdminUsers(w, r, http.StatusBadRequest, adminUsersPage{
			Error: "Choose a valid role",
		})
		return
	}

	err := server.store.SetUserRole(r.Context(), userID, role)
	if err != nil {
		if errors.Is(err, database.ErrUserNotFound) {
			server.renderAdminUsers(w, r, http.StatusNotFound, adminUsersPage{
				Error: "User not found",
			})
			return
		}
		server.logger.Error("failed to set user role", "error", err)
		server.renderServerError(w, r)
		return
	}

	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

// handleResetPassword sets a new password for another user and ends all sessions of that user. It reads path value
// userID and form value password, and redirects to the admin page. Renders a 404 for an unknown user.
func (server *Server) handleResetPassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := server.adminTargetUserID(w, r, "Change your own password in the settings")
	if !ok {
		return
	}

	password := r.PostFormValue("password")
	if problem := passwordProblem(password); problem != "" {
		server.renderAdminUsers(w, r, http.StatusBadRequest, adminUsersPage{
			Error: problem,
		})
		return
	}

	err := server.store.ChangePassword(r.Context(), userID, auth.HashPassword(password), "")
	if err != nil {
		if errors.Is(err, database.ErrUserNotFound) {
			server.renderAdminUsers(w, r, http.StatusNotFound, adminUsersPage{
				Error: "User not found",
			})
			return
		}
		server.logger.Error("failed to reset password", "error", err)
		server.renderServerError(w, r)
		return
	}

	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

// handleDeleteUser deletes another user with all data of that user. It reads path value userID, and redirects to the
// admin page. An unknown user counts as success.
func (server *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := server.adminTargetUserID(w, r, "You cannot delete your own account")
	if !ok {
		return
	}

	err := server.store.DeleteUser(r.Context(), userID)
	if err != nil && !errors.Is(err, database.ErrUserNotFound) {
		server.logger.Error("failed to delete user", "error", err)
		server.renderServerError(w, r)
		return
	}

	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}
