package server

import (
	"errors"
	"net/http"
	"sync"

	"github.com/koenvanzuijlen/outpost/internal/auth"
	"github.com/koenvanzuijlen/outpost/internal/database"
)

// setupPage is the data for setupView: the submitted username and the message of a failed attempt.
type setupPage struct {
	Username string
	Error    string
}

// signInPage is the data for signInView: the submitted username and the message of a failed attempt.
type signInPage struct {
	Username string
	Error    string
}

// dummyPasswordHash is a valid hash that no account uses. A sign-in with an unknown username verifies against it, so
// the response time does not show whether the username exists. It is computed on first use.
var dummyPasswordHash = sync.OnceValue(func() string { return auth.HashPassword("outpost dummy password") })

// handleSetupPage renders the form that creates the first admin account. It renders a 404 when an admin already
// exists. It reads no input.
func (server *Server) handleSetupPage(w http.ResponseWriter, r *http.Request) {
	needed, err := server.store.SetupNeeded(r.Context())
	if err != nil {
		server.logger.Error("setup needed check failed", "error", err)
		server.renderServerError(w, r)
		return
	}
	if !needed {
		server.renderError(w, r, http.StatusNotFound, "Page not found")
		return
	}
	server.renderComponent(w, r, http.StatusOK, setupView(setupPage{}))
}

// handleSetup creates the first admin account and signs it in. The account keeps the library and the wishlist from
// before user accounts. It reads form values username, password and password_confirm, and redirects to the home page.
// It renders a 404 when an admin already exists.
func (server *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	username := normalizeUsername(r.FormValue("username"))
	password := r.FormValue("password")
	confirm := r.FormValue("password_confirm")

	if !validUsername(username) {
		server.renderComponent(w, r, http.StatusBadRequest, setupView(setupPage{
			Username: username,
			Error:    usernameRule,
		}))
		return
	}
	problem := passwordProblem(password)
	if problem != "" {
		server.renderComponent(w, r, http.StatusBadRequest, setupView(setupPage{
			Username: username,
			Error:    problem,
		}))
		return
	}
	if password != confirm {
		server.renderComponent(w, r, http.StatusBadRequest, setupView(setupPage{
			Username: username,
			Error:    "The passwords do not match",
		}))
		return
	}

	user, err := server.store.CompleteSetup(r.Context(), username, auth.HashPassword(password))
	if err != nil {
		if errors.Is(err, database.ErrSetupDone) {
			server.renderError(w, r, http.StatusNotFound, "Page not found")
			return
		}
		server.logger.Error("setup completion failed", "error", err)
		server.renderServerError(w, r)
		return
	}

	err = server.startSession(r.Context(), w, user.ID)
	if err != nil {
		server.logger.Error("start session failed", "error", err)
		server.renderServerError(w, r)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// handleSignInPage renders the sign-in form. Before the first admin exists, it redirects to the setup page. It reads
// no input.
func (server *Server) handleSignInPage(w http.ResponseWriter, r *http.Request) {
	needed, err := server.store.SetupNeeded(r.Context())
	if err != nil {
		server.logger.Error("setup needed check failed", "error", err)
		server.renderServerError(w, r)
		return
	}
	if needed {
		http.Redirect(w, r, "/setup", http.StatusSeeOther)
		return
	}
	server.renderComponent(w, r, http.StatusOK, signInView(signInPage{}))
}

// handleSignIn signs in the user with the username and the password. A wrong username or password renders the form
// again with a 401. It reads form values username and password, and redirects to the home page.
func (server *Server) handleSignIn(w http.ResponseWriter, r *http.Request) {
	username := normalizeUsername(r.FormValue("username"))
	password := r.FormValue("password")

	if username == "" || password == "" {
		server.renderComponent(w, r, http.StatusBadRequest, signInView(signInPage{
			Username: username,
			Error:    "Enter your username and password",
		}))
		return
	}

	user, hash, err := server.store.UserByUsername(r.Context(), username)
	if err != nil {
		if errors.Is(err, database.ErrUserNotFound) {
			// Perform dummy password hash to mitigate timing attacks
			_, _ = auth.VerifyPassword(dummyPasswordHash(), password)
			server.renderComponent(w, r, http.StatusUnauthorized, signInView(signInPage{
				Username: username,
				Error:    "Invalid username or password",
			}))
			return
		}
		server.logger.Error("user lookup failed", "error", err)
		server.renderServerError(w, r)
		return
	}

	ok, err := auth.VerifyPassword(hash, password)
	if err != nil || !ok {
		server.renderComponent(w, r, http.StatusUnauthorized, signInView(signInPage{
			Username: username,
			Error:    "Invalid username or password",
		}))
		return
	}

	err = server.startSession(r.Context(), w, user.ID)
	if err != nil {
		server.logger.Error("start session failed", "error", err)
		server.renderServerError(w, r)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// handleSignOut ends the session of the request and deletes the session cookie. It reads the session cookie and
// redirects to the sign-in page. A request without a valid session counts as success.
func (server *Server) handleSignOut(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil {
		err := server.store.DeleteSession(r.Context(), auth.HashSessionToken(cookie.Value))
		if err != nil {
			server.logger.Error("delete session failed", "error", err)
			server.renderServerError(w, r)
			return
		}
	}
	server.clearSessionCookie(w)
	http.Redirect(w, r, "/sign-in", http.StatusSeeOther)
}

// redirectToSignIn redirects with 303 to the setup page when setup is not done, and to the sign-in page otherwise. It
// renders the server error page if the setup check fails.
func (server *Server) redirectToSignIn(w http.ResponseWriter, r *http.Request) {
	needed, err := server.store.SetupNeeded(r.Context())
	if err != nil {
		server.logger.Error("setup needed check failed", "error", err)
		server.renderServerError(w, r)
		return
	}
	if needed {
		http.Redirect(w, r, "/setup", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/sign-in", http.StatusSeeOther)
}
