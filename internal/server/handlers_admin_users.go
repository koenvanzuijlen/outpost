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
