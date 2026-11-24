package server

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/koenvanzuijlen/outpost/internal/auth"
	"github.com/koenvanzuijlen/outpost/internal/database"
)

// TestHandleCreateUser checks which users the admin can create, and that a created user has the role of the form.
func TestHandleCreateUser(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		wantStatus   int
		wantLocation string
		wantUsername string
		wantRole     database.UserRole
	}{
		{
			name:       "user",
			body:       "username=Anna&password=correct+horse+battery&role=user",
			wantStatus: http.StatusSeeOther, wantLocation: "/admin/users",
			wantUsername: "anna", wantRole: database.UserRoleUser,
		},
		{
			name:       "admin",
			body:       "username=bram&password=correct+horse+battery&role=admin",
			wantStatus: http.StatusSeeOther, wantLocation: "/admin/users",
			wantUsername: "bram", wantRole: database.UserRoleAdmin,
		},
		{
			name:       "taken",
			body:       "username=admin&password=correct+horse+battery&role=user",
			wantStatus: http.StatusBadRequest, wantLocation: "",
		},
		{
			name:       "invalid username",
			body:       "username=a%21&password=correct+horse+battery&role=user",
			wantStatus: http.StatusBadRequest, wantLocation: "",
		},
		{
			name:       "short password",
			body:       "username=anna&password=short&role=user",
			wantStatus: http.StatusBadRequest, wantLocation: "",
		},
		{
			name:       "invalid role",
			body:       "username=anna&password=correct+horse+battery&role=owner",
			wantStatus: http.StatusBadRequest, wantLocation: "",
		},
		{
			name:       "no role",
			body:       "username=anna&password=correct+horse+battery",
			wantStatus: http.StatusBadRequest, wantLocation: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)

			w := postForm(t, server, "/admin/users", tt.body)
			if w.Code != tt.wantStatus {
				t.Errorf("POST /admin/users status = %d, want %d", w.Code, tt.wantStatus)
			}
			assertLocation(t, w, tt.wantLocation)
			if tt.wantUsername == "" {
				return
			}

			user, _, err := server.store.UserByUsername(t.Context(), tt.wantUsername)
			if err != nil {
				t.Fatalf("UserByUsername(%q) error = %v", tt.wantUsername, err)
			}
			if user.Role != tt.wantRole {
				t.Errorf("UserByUsername(%q) role = %q, want %q", tt.wantUsername, user.Role, tt.wantRole)
			}
		})
	}
}

// TestAdminUserActions checks which admin actions on a user succeed, the protection of the own account and the
// answers for unknown users. The admin has ID 1. "{anna}" in a path is the ID of the user anna.
func TestAdminUserActions(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		body         string
		wantStatus   int
		wantLocation string
	}{
		{"rename", "/admin/users/{anna}/username", "username=Anna2", http.StatusSeeOther, "/admin/users"},
		{"rename taken", "/admin/users/{anna}/username", "username=admin", http.StatusBadRequest, ""},
		{"rename invalid", "/admin/users/{anna}/username", "username=a%21", http.StatusBadRequest, ""},
		{"rename self", "/admin/users/1/username", "username=boss", http.StatusSeeOther, "/admin/users"},
		{"make admin", "/admin/users/{anna}/role", "role=admin", http.StatusSeeOther, "/admin/users"},
		{"invalid role", "/admin/users/{anna}/role", "role=owner", http.StatusBadRequest, ""},
		{"own role", "/admin/users/1/role", "role=user", http.StatusBadRequest, ""},
		{"reset password", "/admin/users/{anna}/password", "password=new+password+123", http.StatusSeeOther, "/admin/users"},
		{"short password", "/admin/users/{anna}/password", "password=short", http.StatusBadRequest, ""},
		{"own password", "/admin/users/1/password", "password=new+password+123", http.StatusBadRequest, ""},
		{"delete", "/admin/users/{anna}/delete", "", http.StatusSeeOther, "/admin/users"},
		{"own account", "/admin/users/1/delete", "", http.StatusBadRequest, ""},
		{"unknown rename", "/admin/users/99/username", "username=nobody", http.StatusNotFound, ""},
		{"unknown role", "/admin/users/99/role", "role=user", http.StatusNotFound, ""},
		{"unknown password", "/admin/users/99/password", "password=new+password+123", http.StatusNotFound, ""},
		{"unknown delete", "/admin/users/99/delete", "", http.StatusSeeOther, "/admin/users"},
		{"bad ID", "/admin/users/abc/delete", "", http.StatusNotFound, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)
			anna, _ := addSignedInUser(t, server, "anna")
			path := strings.ReplaceAll(tt.path, "{anna}", strconv.FormatInt(anna.ID, 10))

			w := postForm(t, server, path, tt.body)
			if w.Code != tt.wantStatus {
				t.Errorf("POST %s status = %d, want %d", path, w.Code, tt.wantStatus)
			}
			assertLocation(t, w, tt.wantLocation)
		})
	}
}

// TestAdminUserActionsState checks the stored state after an admin action, because a handler can redirect without a
// change.
func TestAdminUserActionsState(t *testing.T) {
	ctx := t.Context()

	t.Run("make admin", func(t *testing.T) {
		server := newTestServer(t)
		anna, _ := addSignedInUser(t, server, "anna")
		postForm(t, server, "/admin/users/"+strconv.FormatInt(anna.ID, 10)+"/role", "role=admin")

		user, _, err := server.store.UserByUsername(ctx, "anna")
		if err != nil {
			t.Fatalf("UserByUsername() error = %v", err)
		}
		if user.Role != database.UserRoleAdmin {
			t.Errorf("UserByUsername() role = %q, want %q", user.Role, database.UserRoleAdmin)
		}
	})

	t.Run("own role", func(t *testing.T) {
		server := newTestServer(t)
		postForm(t, server, "/admin/users/1/role", "role=user")

		user, _, err := server.store.UserByUsername(ctx, "admin")
		if err != nil {
			t.Fatalf("UserByUsername() error = %v", err)
		}
		if user.Role != database.UserRoleAdmin {
			t.Errorf("UserByUsername() role = %q, want %q", user.Role, database.UserRoleAdmin)
		}
	})

	t.Run("rename", func(t *testing.T) {
		server := newTestServer(t)
		anna, _ := addSignedInUser(t, server, "anna")
		postForm(t, server, "/admin/users/"+strconv.FormatInt(anna.ID, 10)+"/username", "username=Anna2")

		user, _, err := server.store.UserByUsername(ctx, "anna2")
		if err != nil {
			t.Fatalf("UserByUsername() error = %v", err)
		}
		if user.ID != anna.ID {
			t.Errorf("UserByUsername() ID = %d, want %d", user.ID, anna.ID)
		}
	})

	t.Run("reset password", func(t *testing.T) {
		server := newTestServer(t)
		anna, _ := addSignedInUser(t, server, "anna")
		postForm(t, server, "/admin/users/"+strconv.FormatInt(anna.ID, 10)+"/password", "password=new+password+123")

		hash, err := server.store.PasswordHash(ctx, anna.ID)
		if err != nil {
			t.Fatalf("PasswordHash() error = %v", err)
		}
		ok, err := auth.VerifyPassword(hash, "new password 123")
		if err != nil {
			t.Fatalf("VerifyPassword() error = %v", err)
		}
		if !ok {
			t.Errorf("VerifyPassword() = %v, want %v", ok, true)
		}
	})

	t.Run("delete", func(t *testing.T) {
		server := newTestServer(t)
		anna, _ := addSignedInUser(t, server, "anna")
		postForm(t, server, "/admin/users/"+strconv.FormatInt(anna.ID, 10)+"/delete", "")

		_, _, err := server.store.UserByUsername(ctx, "anna")
		if !errors.Is(err, database.ErrUserNotFound) {
			t.Errorf("UserByUsername() error = %v, want %v", err, database.ErrUserNotFound)
		}
	})
}

// TestResetPasswordEndsSessions checks that a password reset signs the user out: the old session cookie no longer
// works.
func TestResetPasswordEndsSessions(t *testing.T) {
	server := newTestServer(t)
	anna, cookie := addSignedInUser(t, server, "anna")

	w := postForm(t, server, "/admin/users/"+strconv.FormatInt(anna.ID, 10)+"/password", "password=new+password+123")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("POST password status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	w = serveWithCookie(t, server, http.MethodGet, "/library", "", cookie)
	if w.Code != http.StatusSeeOther {
		t.Errorf("GET /library status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	assertLocation(t, w, "/sign-in?next=%2Flibrary")
}
